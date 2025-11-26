package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestValidateClientIDURL tests client_id URL validation per CIMD spec
func TestValidateClientIDURL(t *testing.T) {
	tests := []struct {
		name      string
		clientURL string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid HTTPS URL with path",
			clientURL: "https://app.example.com/oauth/client-metadata.json",
			wantErr:   false,
		},
		{
			name:      "valid HTTPS URL with deep path",
			clientURL: "https://example.com/apps/my-app/metadata.json",
			wantErr:   false,
		},
		{
			name:      "empty URL",
			clientURL: "",
			wantErr:   true,
			errMsg:    "cannot be empty",
		},
		{
			name:      "HTTP scheme not allowed",
			clientURL: "http://app.example.com/oauth/client-metadata.json",
			wantErr:   true,
			errMsg:    "must use https scheme",
		},
		{
			name:      "HTTP localhost not allowed (CIMD requires HTTPS)",
			clientURL: "http://localhost/oauth/client-metadata.json",
			wantErr:   true,
			errMsg:    "must use https scheme",
		},
		{
			name:      "missing path component",
			clientURL: "https://example.com",
			wantErr:   true,
			errMsg:    "must contain a path component",
		},
		{
			name:      "root path not sufficient",
			clientURL: "https://example.com/",
			wantErr:   true,
			errMsg:    "must contain a path component",
		},
		{
			name:      "relative URL",
			clientURL: "/oauth/client-metadata.json",
			wantErr:   true,
			errMsg:    "must be absolute",
		},
		{
			name:      "missing scheme",
			clientURL: "example.com/oauth/client-metadata.json",
			wantErr:   true,
			errMsg:    "must be absolute",
		},
		{
			name:      "custom scheme not allowed",
			clientURL: "ftp://example.com/client-metadata.json",
			wantErr:   true,
			errMsg:    "must use https scheme",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateClientIDURL(tt.clientURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateClientIDURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateClientIDURL() error = %v, expected to contain %q", err, tt.errMsg)
				}
			}
		})
	}
}

// TestGenerateClientMetadata tests client metadata generation
func TestGenerateClientMetadata(t *testing.T) {
	tests := []struct {
		name      string
		config    *OAuthConfig
		wantErr   bool
		errMsg    string
		checkFunc func(*testing.T, *ClientMetadataDocument)
	}{
		{
			name: "valid configuration",
			config: &OAuthConfig{
				ClientIDMetadataURL: "https://app.example.com/oauth/client-metadata.json",
				RedirectURL:         "http://localhost:8765/callback",
			},
			wantErr: false,
			checkFunc: func(t *testing.T, doc *ClientMetadataDocument) {
				if doc.ClientID != "https://app.example.com/oauth/client-metadata.json" {
					t.Errorf("ClientID = %v, want %v", doc.ClientID, "https://app.example.com/oauth/client-metadata.json")
				}
				if doc.ClientName != "mcp-debug" {
					t.Errorf("ClientName = %v, want mcp-debug", doc.ClientName)
				}
				if doc.ClientURI != "https://github.com/giantswarm/mcp-debug" {
					t.Errorf("ClientURI = %v, want https://github.com/giantswarm/mcp-debug", doc.ClientURI)
				}
				if len(doc.RedirectURIs) != 1 || doc.RedirectURIs[0] != "http://localhost:8765/callback" {
					t.Errorf("RedirectURIs = %v, want [http://localhost:8765/callback]", doc.RedirectURIs)
				}
				if len(doc.GrantTypes) != 1 || doc.GrantTypes[0] != "authorization_code" {
					t.Errorf("GrantTypes = %v, want [authorization_code]", doc.GrantTypes)
				}
				if len(doc.ResponseTypes) != 1 || doc.ResponseTypes[0] != "code" {
					t.Errorf("ResponseTypes = %v, want [code]", doc.ResponseTypes)
				}
				if doc.TokenEndpointAuthMethod != "none" {
					t.Errorf("TokenEndpointAuthMethod = %v, want none", doc.TokenEndpointAuthMethod)
				}
			},
		},
		{
			name: "missing client_id URL",
			config: &OAuthConfig{
				RedirectURL: "http://localhost:8765/callback",
			},
			wantErr: true,
			errMsg:  "ClientIDMetadataURL is required",
		},
		{
			name: "invalid client_id URL (HTTP)",
			config: &OAuthConfig{
				ClientIDMetadataURL: "http://app.example.com/oauth/client-metadata.json",
				RedirectURL:         "http://localhost:8765/callback",
			},
			wantErr: true,
			errMsg:  "must use https scheme",
		},
		{
			name: "invalid client_id URL (no path)",
			config: &OAuthConfig{
				ClientIDMetadataURL: "https://app.example.com",
				RedirectURL:         "http://localhost:8765/callback",
			},
			wantErr: true,
			errMsg:  "must contain a path component",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := GenerateClientMetadata(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateClientMetadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if err != nil && tt.errMsg != "" {
					if !strings.Contains(err.Error(), tt.errMsg) {
						t.Errorf("GenerateClientMetadata() error = %v, expected to contain %q", err, tt.errMsg)
					}
				}
				return
			}
			if tt.checkFunc != nil {
				tt.checkFunc(t, doc)
			}
		})
	}
}

// TestValidateClientMetadata tests client metadata document validation
func TestValidateClientMetadata(t *testing.T) {
	tests := []struct {
		name     string
		metadata *ClientMetadataDocument
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid metadata",
			metadata: &ClientMetadataDocument{
				ClientID:                "https://app.example.com/oauth/client-metadata.json",
				ClientName:              "mcp-debug",
				RedirectURIs:            []string{"http://localhost:8765/callback"},
				GrantTypes:              []string{"authorization_code"},
				ResponseTypes:           []string{"code"},
				TokenEndpointAuthMethod: "none",
			},
			wantErr: false,
		},
		{
			name: "valid metadata with multiple redirect URIs",
			metadata: &ClientMetadataDocument{
				ClientID: "https://app.example.com/oauth/client-metadata.json",
				RedirectURIs: []string{
					"http://localhost:8765/callback",
					"http://127.0.0.1:8765/callback",
					"https://app.example.com/callback",
				},
				GrantTypes:              []string{"authorization_code"},
				ResponseTypes:           []string{"code"},
				TokenEndpointAuthMethod: "none",
			},
			wantErr: false,
		},
		{
			name: "invalid client_id (HTTP)",
			metadata: &ClientMetadataDocument{
				ClientID:     "http://app.example.com/oauth/client-metadata.json",
				RedirectURIs: []string{"http://localhost:8765/callback"},
			},
			wantErr: true,
			errMsg:  "must use https scheme",
		},
		{
			name: "invalid client_id (no path)",
			metadata: &ClientMetadataDocument{
				ClientID:     "https://app.example.com",
				RedirectURIs: []string{"http://localhost:8765/callback"},
			},
			wantErr: true,
			errMsg:  "must contain a path component",
		},
		{
			name: "missing redirect_uris",
			metadata: &ClientMetadataDocument{
				ClientID:     "https://app.example.com/oauth/client-metadata.json",
				RedirectURIs: []string{},
			},
			wantErr: true,
			errMsg:  "redirect_uris is required",
		},
		{
			name: "invalid redirect_uri scheme",
			metadata: &ClientMetadataDocument{
				ClientID:     "https://app.example.com/oauth/client-metadata.json",
				RedirectURIs: []string{"ftp://localhost/callback"},
			},
			wantErr: true,
			errMsg:  "must use http or https scheme",
		},
		{
			name: "relative redirect_uri",
			metadata: &ClientMetadataDocument{
				ClientID:     "https://app.example.com/oauth/client-metadata.json",
				RedirectURIs: []string{"/callback"},
			},
			wantErr: true,
			errMsg:  "must be absolute",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateClientMetadata(tt.metadata)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateClientMetadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateClientMetadata() error = %v, expected to contain %q", err, tt.errMsg)
				}
			}
		})
	}
}

// TestFetchClientMetadata tests fetching client metadata from HTTPS URL
func TestFetchClientMetadata(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func() *httptest.Server
		clientURL string
		wantErr   bool
		errMsg    string
		checkFunc func(*testing.T, *ClientMetadataDocument)
	}{
		{
			name: "successful fetch",
			setupMock: func() *httptest.Server {
				return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method != http.MethodGet {
						t.Errorf("Expected GET request, got %s", r.Method)
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(&ClientMetadataDocument{
						ClientID:                "https://app.example.com/oauth/client-metadata.json",
						ClientName:              "mcp-debug",
						RedirectURIs:            []string{"http://localhost:8765/callback"},
						GrantTypes:              []string{"authorization_code"},
						ResponseTypes:           []string{"code"},
						TokenEndpointAuthMethod: "none",
					})
				}))
			},
			wantErr: false,
			checkFunc: func(t *testing.T, doc *ClientMetadataDocument) {
				if doc.ClientID != "https://app.example.com/oauth/client-metadata.json" {
					t.Errorf("ClientID = %v, want https://app.example.com/oauth/client-metadata.json", doc.ClientID)
				}
				if doc.ClientName != "mcp-debug" {
					t.Errorf("ClientName = %v, want mcp-debug", doc.ClientName)
				}
			},
		},
		{
			name: "404 not found",
			setupMock: func() *httptest.Server {
				return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			wantErr: true,
			errMsg:  "request failed with status 404",
		},
		{
			name: "invalid JSON response",
			setupMock: func() *httptest.Server {
				return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte("invalid json"))
				}))
			},
			wantErr: true,
			errMsg:  "failed to parse JSON",
		},
		{
			name: "invalid metadata structure",
			setupMock: func() *httptest.Server {
				return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					// Missing redirect_uris
					json.NewEncoder(w).Encode(&ClientMetadataDocument{
						ClientID: "https://app.example.com/oauth/client-metadata.json",
					})
				}))
			},
			wantErr: true,
			errMsg:  "redirect_uris is required",
		},
		{
			name: "wrong content type",
			setupMock: func() *httptest.Server {
				return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "text/html")
					w.Write([]byte("<html></html>"))
				}))
			},
			wantErr: true,
			errMsg:  "unexpected Content-Type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip test if no mock server setup (e.g., for validation-only tests)
			if tt.setupMock == nil {
				t.Skip("No mock server setup for this test")
				return
			}

			server := tt.setupMock()
			defer server.Close()

			// Use server URL with path for CIMD compliance
			clientURL := server.URL + "/oauth/client-metadata.json"

			ctx := context.Background()
			// Use the test server's HTTP client which trusts the self-signed certificate
			doc, err := fetchClientMetadataWithClient(ctx, clientURL, server.Client())

			if (err != nil) != tt.wantErr {
				t.Errorf("FetchClientMetadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if err != nil && tt.errMsg != "" {
					if !strings.Contains(err.Error(), tt.errMsg) {
						t.Errorf("FetchClientMetadata() error = %v, expected to contain %q", err, tt.errMsg)
					}
				}
				return
			}
			if tt.checkFunc != nil {
				tt.checkFunc(t, doc)
			}
		})
	}
}

// TestSupportsClientIDMetadata tests AS support detection for CIMD
func TestSupportsClientIDMetadata(t *testing.T) {
	tests := []struct {
		name       string
		asMetadata *AuthorizationServerMetadata
		want       bool
	}{
		{
			name: "AS supports CIMD",
			asMetadata: &AuthorizationServerMetadata{
				Issuer:                            "https://auth.example.com",
				AuthorizationEndpoint:             "https://auth.example.com/authorize",
				TokenEndpoint:                     "https://auth.example.com/token",
				ClientIDMetadataDocumentSupported: true,
			},
			want: true,
		},
		{
			name: "AS does not support CIMD",
			asMetadata: &AuthorizationServerMetadata{
				Issuer:                            "https://auth.example.com",
				AuthorizationEndpoint:             "https://auth.example.com/authorize",
				TokenEndpoint:                     "https://auth.example.com/token",
				ClientIDMetadataDocumentSupported: false,
			},
			want: false,
		},
		{
			name:       "nil metadata",
			asMetadata: nil,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SupportsClientIDMetadata(tt.asMetadata)
			if got != tt.want {
				t.Errorf("SupportsClientIDMetadata() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestOAuthConfigValidation_CIMD tests CIMD-related config validation
func TestOAuthConfigValidation_CIMD(t *testing.T) {
	tests := []struct {
		name    string
		config  *OAuthConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid CIMD config",
			config: &OAuthConfig{
				Enabled:             true,
				ClientIDMetadataURL: "https://app.example.com/oauth/client-metadata.json",
				RedirectURL:         "http://localhost:8765/callback",
			},
			wantErr: false,
		},
		{
			name: "invalid CIMD URL (HTTP)",
			config: &OAuthConfig{
				Enabled:             true,
				ClientIDMetadataURL: "http://app.example.com/oauth/client-metadata.json",
				RedirectURL:         "http://localhost:8765/callback",
			},
			wantErr: true,
			errMsg:  "must use https scheme",
		},
		{
			name: "invalid CIMD URL (no path)",
			config: &OAuthConfig{
				Enabled:             true,
				ClientIDMetadataURL: "https://app.example.com",
				RedirectURL:         "http://localhost:8765/callback",
			},
			wantErr: true,
			errMsg:  "must contain a path component",
		},
		{
			name: "CIMD disabled with URL (should be ignored)",
			config: &OAuthConfig{
				Enabled:             true,
				DisableCIMD:         true,
				ClientIDMetadataURL: "https://app.example.com/oauth/client-metadata.json",
				RedirectURL:         "http://localhost:8765/callback",
			},
			wantErr: false, // DisableCIMD means URL will be ignored, so validation passes
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply defaults before validation
			config := tt.config.WithDefaults()
			err := config.Validate()

			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, expected to contain %q", err, tt.errMsg)
				}
			}
		})
	}
}
