package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestBuildASMetadataEndpoints(t *testing.T) {
	tests := []struct {
		name        string
		issuerURL   string
		wantCount   int
		want        []string
		wantErr     bool
		errContains string
	}{
		{
			name:      "root domain without path",
			issuerURL: "https://auth.example.com",
			wantCount: 2,
			want: []string{
				"https://auth.example.com/.well-known/oauth-authorization-server",
				"https://auth.example.com/.well-known/openid-configuration",
			},
		},
		{
			name:      "single path component",
			issuerURL: "https://auth.example.com/tenant1",
			wantCount: 3,
			want: []string{
				"https://auth.example.com/.well-known/oauth-authorization-server/tenant1",
				"https://auth.example.com/.well-known/openid-configuration/tenant1",
				"https://auth.example.com/tenant1/.well-known/openid-configuration",
			},
		},
		{
			name:      "multiple path components",
			issuerURL: "https://auth.example.com/org/tenant",
			wantCount: 3,
			want: []string{
				"https://auth.example.com/.well-known/oauth-authorization-server/org/tenant",
				"https://auth.example.com/.well-known/openid-configuration/org/tenant",
				"https://auth.example.com/org/tenant/.well-known/openid-configuration",
			},
		},
		{
			name:      "trailing slash removed",
			issuerURL: "https://auth.example.com/tenant1/",
			wantCount: 3,
			want: []string{
				"https://auth.example.com/.well-known/oauth-authorization-server/tenant1",
				"https://auth.example.com/.well-known/openid-configuration/tenant1",
				"https://auth.example.com/tenant1/.well-known/openid-configuration",
			},
		},
		{
			name:      "HTTP scheme allowed",
			issuerURL: "http://localhost:8080",
			wantCount: 2,
			want: []string{
				"http://localhost:8080/.well-known/oauth-authorization-server",
				"http://localhost:8080/.well-known/openid-configuration",
			},
		},
		{
			name:        "invalid URL",
			issuerURL:   "not a valid url://",
			wantErr:     true,
			errContains: "invalid issuer URL",
		},
		{
			name:        "relative URL",
			issuerURL:   "/path/to/resource",
			wantErr:     true,
			errContains: "must be absolute",
		},
		{
			name:        "invalid scheme",
			issuerURL:   "ftp://auth.example.com",
			wantErr:     true,
			errContains: "must use http or https",
		},
		{
			name:        "missing host",
			issuerURL:   "https://",
			wantErr:     true,
			errContains: "missing host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildASMetadataEndpoints(tt.issuerURL)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got none", tt.errContains)
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(got) != tt.wantCount {
				t.Errorf("got %d endpoints, want %d", len(got), tt.wantCount)
			}

			for i, wantEndpoint := range tt.want {
				if i >= len(got) {
					t.Errorf("missing endpoint[%d]: %s", i, wantEndpoint)
					continue
				}
				if got[i] != wantEndpoint {
					t.Errorf("endpoint[%d] = %q, want %q", i, got[i], wantEndpoint)
				}
			}
		})
	}
}

func TestValidateASMetadata(t *testing.T) {
	tests := []struct {
		name        string
		metadata    *AuthorizationServerMetadata
		wantErr     bool
		errContains string
	}{
		{
			name: "valid metadata",
			metadata: &AuthorizationServerMetadata{
				Issuer:                "https://auth.example.com",
				AuthorizationEndpoint: "https://auth.example.com/authorize",
				TokenEndpoint:         "https://auth.example.com/token",
			},
			wantErr: false,
		},
		{
			name: "valid with optional fields",
			metadata: &AuthorizationServerMetadata{
				Issuer:                "https://auth.example.com",
				AuthorizationEndpoint: "https://auth.example.com/authorize",
				TokenEndpoint:         "https://auth.example.com/token",
				RegistrationEndpoint:  "https://auth.example.com/register",
				CodeChallengeMethods:  []string{"S256", "plain"},
			},
			wantErr: false,
		},
		{
			name: "missing issuer",
			metadata: &AuthorizationServerMetadata{
				AuthorizationEndpoint: "https://auth.example.com/authorize",
				TokenEndpoint:         "https://auth.example.com/token",
			},
			wantErr:     true,
			errContains: "missing required field: issuer",
		},
		{
			name: "missing authorization_endpoint",
			metadata: &AuthorizationServerMetadata{
				Issuer:        "https://auth.example.com",
				TokenEndpoint: "https://auth.example.com/token",
			},
			wantErr:     true,
			errContains: "missing required field: authorization_endpoint",
		},
		{
			name: "missing token_endpoint",
			metadata: &AuthorizationServerMetadata{
				Issuer:                "https://auth.example.com",
				AuthorizationEndpoint: "https://auth.example.com/authorize",
			},
			wantErr:     true,
			errContains: "missing required field: token_endpoint",
		},
		{
			name: "invalid issuer URL",
			metadata: &AuthorizationServerMetadata{
				Issuer:                "not a valid url://",
				AuthorizationEndpoint: "https://auth.example.com/authorize",
				TokenEndpoint:         "https://auth.example.com/token",
			},
			wantErr:     true,
			errContains: "invalid issuer URL",
		},
		{
			name: "relative authorization_endpoint",
			metadata: &AuthorizationServerMetadata{
				Issuer:                "https://auth.example.com",
				AuthorizationEndpoint: "/authorize",
				TokenEndpoint:         "https://auth.example.com/token",
			},
			wantErr:     true,
			errContains: "must be absolute URL",
		},
		{
			name: "invalid scheme in token_endpoint",
			metadata: &AuthorizationServerMetadata{
				Issuer:                "https://auth.example.com",
				AuthorizationEndpoint: "https://auth.example.com/authorize",
				TokenEndpoint:         "ftp://auth.example.com/token",
			},
			wantErr:     true,
			errContains: "must use http or https scheme",
		},
		{
			name: "missing host in registration_endpoint",
			metadata: &AuthorizationServerMetadata{
				Issuer:                "https://auth.example.com",
				AuthorizationEndpoint: "https://auth.example.com/authorize",
				TokenEndpoint:         "https://auth.example.com/token",
				RegistrationEndpoint:  "https:///register",
			},
			wantErr:     true,
			errContains: "missing host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateASMetadata(tt.metadata)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got none", tt.errContains)
					return
				}
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidatePKCESupport(t *testing.T) {
	tests := []struct {
		name           string
		metadata       *AuthorizationServerMetadata
		skipValidation bool
		wantErr        bool
		errContains    string
	}{
		{
			name: "S256 supported",
			metadata: &AuthorizationServerMetadata{
				CodeChallengeMethods: []string{"S256"},
			},
			skipValidation: false,
			wantErr:        false,
		},
		{
			name: "S256 and plain supported",
			metadata: &AuthorizationServerMetadata{
				CodeChallengeMethods: []string{"plain", "S256"},
			},
			skipValidation: false,
			wantErr:        false,
		},
		{
			name: "no PKCE methods advertised",
			metadata: &AuthorizationServerMetadata{
				CodeChallengeMethods: []string{},
			},
			skipValidation: false,
			wantErr:        true,
			errContains:    "does not advertise PKCE support",
		},
		{
			name: "nil PKCE methods",
			metadata: &AuthorizationServerMetadata{
				CodeChallengeMethods: nil,
			},
			skipValidation: false,
			wantErr:        true,
			errContains:    "does not advertise PKCE support",
		},
		{
			name: "only plain method (no S256)",
			metadata: &AuthorizationServerMetadata{
				CodeChallengeMethods: []string{"plain"},
			},
			skipValidation: false,
			wantErr:        true,
			errContains:    "does not support S256",
		},
		{
			name: "skip validation with no PKCE",
			metadata: &AuthorizationServerMetadata{
				CodeChallengeMethods: []string{},
			},
			skipValidation: true,
			wantErr:        false,
		},
		{
			name: "skip validation with only plain",
			metadata: &AuthorizationServerMetadata{
				CodeChallengeMethods: []string{"plain"},
			},
			skipValidation: true,
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePKCESupport(tt.metadata, tt.skipValidation)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got none", tt.errContains)
					return
				}
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestFetchASMetadata(t *testing.T) {
	tests := []struct {
		name        string
		setupServer func() *httptest.Server
		wantErr     bool
		errContains string
		checkResult func(*testing.T, *AuthorizationServerMetadata)
	}{
		{
			name: "successful fetch OAuth 2.0 metadata",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					metadata := AuthorizationServerMetadata{
						Issuer:                "https://auth.example.com",
						AuthorizationEndpoint: "https://auth.example.com/authorize",
						TokenEndpoint:         "https://auth.example.com/token",
						CodeChallengeMethods:  []string{"S256", "plain"},
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(metadata)
				}))
			},
			wantErr: false,
			checkResult: func(t *testing.T, metadata *AuthorizationServerMetadata) {
				if metadata.Issuer != "https://auth.example.com" {
					t.Errorf("issuer = %q, want %q", metadata.Issuer, "https://auth.example.com")
				}
				if len(metadata.CodeChallengeMethods) != 2 {
					t.Errorf("got %d code challenge methods, want 2", len(metadata.CodeChallengeMethods))
				}
			},
		},
		{
			name: "successful fetch OIDC metadata",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					metadata := AuthorizationServerMetadata{
						Issuer:                            "https://auth.example.com",
						AuthorizationEndpoint:             "https://auth.example.com/authorize",
						TokenEndpoint:                     "https://auth.example.com/token",
						CodeChallengeMethods:              []string{"S256"},
						ClientIDMetadataDocumentSupported: true,
						ScopesSupported:                   []string{"openid", "profile", "email"},
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(metadata)
				}))
			},
			wantErr: false,
			checkResult: func(t *testing.T, metadata *AuthorizationServerMetadata) {
				if !metadata.ClientIDMetadataDocumentSupported {
					t.Error("expected ClientIDMetadataDocumentSupported = true")
				}
				if len(metadata.ScopesSupported) != 3 {
					t.Errorf("got %d scopes, want 3", len(metadata.ScopesSupported))
				}
			},
		},
		{
			name: "404 not found",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			wantErr:     true,
			errContains: "status 404",
		},
		{
			name: "invalid JSON",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte("not valid json"))
				}))
			},
			wantErr:     true,
			errContains: "failed to parse JSON",
		},
		{
			name: "wrong content type",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "text/plain")
					w.Write([]byte("plain text"))
				}))
			},
			wantErr:     true,
			errContains: "unexpected Content-Type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			metadata, err := fetchASMetadata(ctx, server.URL)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got none", tt.errContains)
					return
				}
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.checkResult != nil {
				tt.checkResult(t, metadata)
			}
		})
	}
}

func TestDiscoverAuthorizationServerMetadata(t *testing.T) {
	tests := []struct {
		name        string
		issuerPath  string
		setupServer func(mux *http.ServeMux, issuerPath string)
		wantErr     bool
		errContains string
		checkResult func(*testing.T, *AuthorizationServerMetadata, string)
	}{
		{
			name:       "OAuth 2.0 discovery at root",
			issuerPath: "",
			setupServer: func(mux *http.ServeMux, issuerPath string) {
				mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, r *http.Request) {
					metadata := AuthorizationServerMetadata{
						Issuer:                "https://auth.example.com",
						AuthorizationEndpoint: "https://auth.example.com/authorize",
						TokenEndpoint:         "https://auth.example.com/token",
						CodeChallengeMethods:  []string{"S256"},
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(metadata)
				})
			},
			wantErr: false,
			checkResult: func(t *testing.T, metadata *AuthorizationServerMetadata, issuerURL string) {
				if metadata.Issuer != "https://auth.example.com" {
					t.Errorf("issuer = %q, want %q", metadata.Issuer, "https://auth.example.com")
				}
			},
		},
		{
			name:       "OIDC discovery fallback at root",
			issuerPath: "",
			setupServer: func(mux *http.ServeMux, issuerPath string) {
				mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				})
				mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
					metadata := AuthorizationServerMetadata{
						Issuer:                "https://auth.example.com",
						AuthorizationEndpoint: "https://auth.example.com/authorize",
						TokenEndpoint:         "https://auth.example.com/token",
						CodeChallengeMethods:  []string{"S256"},
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(metadata)
				})
			},
			wantErr: false,
		},
		{
			name:       "path-based OAuth 2.0 discovery",
			issuerPath: "/tenant1",
			setupServer: func(mux *http.ServeMux, issuerPath string) {
				mux.HandleFunc("/.well-known/oauth-authorization-server/tenant1", func(w http.ResponseWriter, r *http.Request) {
					metadata := AuthorizationServerMetadata{
						Issuer:                "https://auth.example.com/tenant1",
						AuthorizationEndpoint: "https://auth.example.com/tenant1/authorize",
						TokenEndpoint:         "https://auth.example.com/tenant1/token",
						CodeChallengeMethods:  []string{"S256"},
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(metadata)
				})
			},
			wantErr: false,
			checkResult: func(t *testing.T, metadata *AuthorizationServerMetadata, issuerURL string) {
				if metadata.Issuer != "https://auth.example.com/tenant1" {
					t.Errorf("issuer = %q, want %q", metadata.Issuer, "https://auth.example.com/tenant1")
				}
			},
		},
		{
			name:       "path-based OIDC insertion discovery",
			issuerPath: "/tenant1",
			setupServer: func(mux *http.ServeMux, issuerPath string) {
				mux.HandleFunc("/.well-known/oauth-authorization-server/tenant1", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				})
				mux.HandleFunc("/.well-known/openid-configuration/tenant1", func(w http.ResponseWriter, r *http.Request) {
					metadata := AuthorizationServerMetadata{
						Issuer:                "https://auth.example.com/tenant1",
						AuthorizationEndpoint: "https://auth.example.com/tenant1/authorize",
						TokenEndpoint:         "https://auth.example.com/tenant1/token",
						CodeChallengeMethods:  []string{"S256"},
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(metadata)
				})
			},
			wantErr: false,
		},
		{
			name:       "path-based OIDC appending discovery",
			issuerPath: "/tenant1",
			setupServer: func(mux *http.ServeMux, issuerPath string) {
				mux.HandleFunc("/.well-known/oauth-authorization-server/tenant1", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				})
				mux.HandleFunc("/.well-known/openid-configuration/tenant1", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				})
				mux.HandleFunc("/tenant1/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
					metadata := AuthorizationServerMetadata{
						Issuer:                "https://auth.example.com/tenant1",
						AuthorizationEndpoint: "https://auth.example.com/tenant1/authorize",
						TokenEndpoint:         "https://auth.example.com/tenant1/token",
						CodeChallengeMethods:  []string{"S256"},
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(metadata)
				})
			},
			wantErr: false,
		},
		{
			name:       "no metadata found",
			issuerPath: "",
			setupServer: func(mux *http.ServeMux, issuerPath string) {
				mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				})
			},
			wantErr:     true,
			errContains: "no valid AS metadata found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			tt.setupServer(mux, tt.issuerPath)

			server := httptest.NewServer(mux)
			defer server.Close()

			issuerURL := server.URL + tt.issuerPath

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			logger := NewLogger(false, false, false)
			metadata, err := DiscoverAuthorizationServerMetadata(ctx, issuerURL, logger)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got none", tt.errContains)
					return
				}
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.checkResult != nil {
				tt.checkResult(t, metadata, issuerURL)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
