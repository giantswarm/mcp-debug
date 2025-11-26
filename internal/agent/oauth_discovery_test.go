package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestParseWWWAuthenticate(t *testing.T) {
	tests := []struct {
		name                 string
		header               string
		wantScheme           string
		wantResourceMetadata string
		wantScopes           []string
		wantError            string
		wantErrorDesc        string
		expectError          bool
	}{
		{
			name:                 "complete challenge with all parameters",
			header:               `Bearer resource_metadata="https://mcp.example.com/.well-known/oauth-protected-resource", scope="files:read files:write", error="insufficient_scope", error_description="Additional permissions required"`,
			wantScheme:           "Bearer",
			wantResourceMetadata: "https://mcp.example.com/.well-known/oauth-protected-resource",
			wantScopes:           []string{"files:read", "files:write"},
			wantError:            "insufficient_scope",
			wantErrorDesc:        "Additional permissions required",
		},
		{
			name:                 "minimal challenge with resource_metadata only",
			header:               `Bearer resource_metadata="https://auth.example.com/.well-known/oauth-protected-resource"`,
			wantScheme:           "Bearer",
			wantResourceMetadata: "https://auth.example.com/.well-known/oauth-protected-resource",
		},
		{
			name:       "challenge with scope only",
			header:     `Bearer scope="read write"`,
			wantScheme: "Bearer",
			wantScopes: []string{"read", "write"},
		},
		{
			name:       "scheme only, no parameters",
			header:     "Bearer",
			wantScheme: "Bearer",
		},
		{
			name:        "empty header",
			header:      "",
			expectError: true,
		},
		{
			name:          "challenge with error",
			header:        `Bearer error="invalid_token", error_description="The access token expired"`,
			wantScheme:    "Bearer",
			wantError:     "invalid_token",
			wantErrorDesc: "The access token expired",
		},
		{
			name:       "single scope",
			header:     `Bearer scope="openid"`,
			wantScheme: "Bearer",
			wantScopes: []string{"openid"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			challenge, err := parseWWWAuthenticate(tt.header)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if challenge.Scheme != tt.wantScheme {
				t.Errorf("Scheme = %q, want %q", challenge.Scheme, tt.wantScheme)
			}

			if challenge.ResourceMetadataURL != tt.wantResourceMetadata {
				t.Errorf("ResourceMetadataURL = %q, want %q", challenge.ResourceMetadataURL, tt.wantResourceMetadata)
			}

			if len(challenge.Scopes) != len(tt.wantScopes) {
				t.Errorf("Scopes count = %d, want %d", len(challenge.Scopes), len(tt.wantScopes))
			} else {
				for i, scope := range challenge.Scopes {
					if scope != tt.wantScopes[i] {
						t.Errorf("Scopes[%d] = %q, want %q", i, scope, tt.wantScopes[i])
					}
				}
			}

			if challenge.Error != tt.wantError {
				t.Errorf("Error = %q, want %q", challenge.Error, tt.wantError)
			}

			if challenge.ErrorDescription != tt.wantErrorDesc {
				t.Errorf("ErrorDescription = %q, want %q", challenge.ErrorDescription, tt.wantErrorDesc)
			}
		})
	}
}

func TestParseAuthParams(t *testing.T) {
	tests := []struct {
		name   string
		params string
		want   map[string]string
	}{
		{
			name:   "simple key-value pairs",
			params: `key1="value1", key2="value2"`,
			want: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name:   "values with spaces",
			params: `scope="read write", realm="example"`,
			want: map[string]string{
				"scope": "read write",
				"realm": "example",
			},
		},
		{
			name:   "url value",
			params: `resource_metadata="https://example.com/.well-known/oauth"`,
			want: map[string]string{
				"resource_metadata": "https://example.com/.well-known/oauth",
			},
		},
		{
			name:   "mixed spacing",
			params: `key1="value1",key2="value2"  ,  key3="value3"`,
			want: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
		},
		{
			name:   "unquoted values",
			params: `key1=value1, key2=value2`,
			want: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name:   "empty params",
			params: "",
			want:   map[string]string{},
		},
		{
			name:   "malformed - no equals",
			params: "key1 key2",
			want:   map[string]string{},
		},
		{
			name:   "single parameter",
			params: `key="value"`,
			want: map[string]string{
				"key": "value",
			},
		},
		{
			name:   "comma in quoted value",
			params: `desc="value, with comma", key2="val2"`,
			want: map[string]string{
				"desc": "value, with comma",
				"key2": "val2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseAuthParams(tt.params)

			if len(got) != len(tt.want) {
				t.Errorf("got %d params, want %d", len(got), len(tt.want))
			}

			for key, wantValue := range tt.want {
				gotValue, ok := got[key]
				if !ok {
					t.Errorf("missing key %q", key)
					continue
				}
				if gotValue != wantValue {
					t.Errorf("param %q = %q, want %q", key, gotValue, wantValue)
				}
			}
		})
	}
}

func TestSplitPreservingQuotes(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		delimiter byte
		want      []string
	}{
		{
			name:      "simple split",
			input:     "a,b,c",
			delimiter: ',',
			want:      []string{"a", "b", "c"},
		},
		{
			name:      "quoted value with delimiter",
			input:     `a,"b,c",d`,
			delimiter: ',',
			want:      []string{"a", `"b,c"`, "d"},
		},
		{
			name:      "multiple quoted values",
			input:     `"a,b","c,d","e"`,
			delimiter: ',',
			want:      []string{`"a,b"`, `"c,d"`, `"e"`},
		},
		{
			name:      "empty string",
			input:     "",
			delimiter: ',',
			want:      []string{},
		},
		{
			name:      "no delimiter",
			input:     "abc",
			delimiter: ',',
			want:      []string{"abc"},
		},
		{
			name:      "unmatched quotes",
			input:     `a,"b,c`,
			delimiter: ',',
			want:      []string{"a", `"b,c`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitPreservingQuotes(tt.input, tt.delimiter)

			if len(got) != len(tt.want) {
				t.Errorf("got %d parts, want %d\ngot: %v\nwant: %v", len(got), len(tt.want), got, tt.want)
				return
			}

			for i, part := range got {
				if part != tt.want[i] {
					t.Errorf("part[%d] = %q, want %q", i, part, tt.want[i])
				}
			}
		})
	}
}

func TestBuildWellKnownURIs(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		want     []string
		wantErr  bool
	}{
		{
			name:     "root endpoint",
			endpoint: "https://mcp.example.com",
			want: []string{
				"https://mcp.example.com/.well-known/oauth-protected-resource",
			},
		},
		{
			name:     "root endpoint with trailing slash",
			endpoint: "https://mcp.example.com/",
			want: []string{
				"https://mcp.example.com/.well-known/oauth-protected-resource",
			},
		},
		{
			name:     "endpoint with path",
			endpoint: "https://mcp.example.com/api/mcp",
			want: []string{
				"https://mcp.example.com/.well-known/oauth-protected-resource/api/mcp",
				"https://mcp.example.com/.well-known/oauth-protected-resource",
			},
		},
		{
			name:     "endpoint with single path segment",
			endpoint: "https://mcp.example.com/mcp",
			want: []string{
				"https://mcp.example.com/.well-known/oauth-protected-resource/mcp",
				"https://mcp.example.com/.well-known/oauth-protected-resource",
			},
		},
		{
			name:     "non-standard port",
			endpoint: "https://mcp.example.com:8443/mcp",
			want: []string{
				"https://mcp.example.com:8443/.well-known/oauth-protected-resource/mcp",
				"https://mcp.example.com:8443/.well-known/oauth-protected-resource",
			},
		},
		{
			name:     "localhost with port",
			endpoint: "http://localhost:8080/mcp",
			want: []string{
				"http://localhost:8080/.well-known/oauth-protected-resource/mcp",
				"http://localhost:8080/.well-known/oauth-protected-resource",
			},
		},
		{
			name:     "invalid URL - no scheme",
			endpoint: "example.com/mcp",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildWellKnownURIs(tt.endpoint)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(got) != len(tt.want) {
				t.Errorf("got %d URIs, want %d\ngot: %v\nwant: %v", len(got), len(tt.want), got, tt.want)
				return
			}

			for i, uri := range got {
				if uri != tt.want[i] {
					t.Errorf("URI[%d] = %q, want %q", i, uri, tt.want[i])
				}
			}
		})
	}
}

func TestFetchProtectedResourceMetadata(t *testing.T) {
	tests := []struct {
		name        string
		metadata    *ProtectedResourceMetadata
		statusCode  int
		contentType string
		wantErr     bool
	}{
		{
			name: "valid metadata",
			metadata: &ProtectedResourceMetadata{
				Resource: "https://mcp.example.com",
				AuthorizationServers: []string{
					"https://auth.example.com",
				},
				ScopesSupported:        []string{"read", "write"},
				BearerMethodsSupported: []string{"header"},
			},
			statusCode:  http.StatusOK,
			contentType: "application/json",
		},
		{
			name: "multiple authorization servers",
			metadata: &ProtectedResourceMetadata{
				Resource: "https://mcp.example.com",
				AuthorizationServers: []string{
					"https://auth1.example.com",
					"https://auth2.example.com",
				},
				ScopesSupported: []string{"read"},
			},
			statusCode:  http.StatusOK,
			contentType: "application/json",
		},
		{
			name:        "404 not found",
			statusCode:  http.StatusNotFound,
			contentType: "application/json",
			wantErr:     true,
		},
		{
			name: "invalid content type",
			metadata: &ProtectedResourceMetadata{
				Resource: "https://mcp.example.com",
				AuthorizationServers: []string{
					"https://auth.example.com",
				},
			},
			statusCode:  http.StatusOK,
			contentType: "text/html",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify Accept header
				if accept := r.Header.Get("Accept"); accept != "application/json" {
					t.Errorf("Accept header = %q, want %q", accept, "application/json")
				}

				w.Header().Set("Content-Type", tt.contentType)
				w.WriteHeader(tt.statusCode)

				if tt.metadata != nil {
					json.NewEncoder(w).Encode(tt.metadata)
				}
			}))
			defer server.Close()

			ctx := context.Background()
			metadata, err := fetchProtectedResourceMetadata(ctx, server.URL)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if metadata.Resource != tt.metadata.Resource {
				t.Errorf("Resource = %q, want %q", metadata.Resource, tt.metadata.Resource)
			}

			if len(metadata.AuthorizationServers) != len(tt.metadata.AuthorizationServers) {
				t.Errorf("got %d authorization servers, want %d", len(metadata.AuthorizationServers), len(tt.metadata.AuthorizationServers))
			}
		})
	}
}

func TestFetchProtectedResourceMetadataTimeout(t *testing.T) {
	// Create server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(15 * time.Second) // Longer than metadataRequestTimeout
	}))
	defer server.Close()

	ctx := context.Background()
	_, err := fetchProtectedResourceMetadata(ctx, server.URL)

	if err == nil {
		t.Errorf("expected timeout error but got none")
	}
}

func TestValidateProtectedResourceMetadata(t *testing.T) {
	tests := []struct {
		name     string
		metadata *ProtectedResourceMetadata
		wantErr  bool
	}{
		{
			name: "valid metadata",
			metadata: &ProtectedResourceMetadata{
				Resource: "https://mcp.example.com",
				AuthorizationServers: []string{
					"https://auth.example.com",
				},
			},
		},
		{
			name: "valid metadata with http scheme",
			metadata: &ProtectedResourceMetadata{
				Resource: "https://mcp.example.com",
				AuthorizationServers: []string{
					"http://localhost:8080",
				},
			},
		},
		{
			name: "missing resource",
			metadata: &ProtectedResourceMetadata{
				AuthorizationServers: []string{
					"https://auth.example.com",
				},
			},
			wantErr: true,
		},
		{
			name: "missing authorization servers",
			metadata: &ProtectedResourceMetadata{
				Resource:             "https://mcp.example.com",
				AuthorizationServers: []string{},
			},
			wantErr: true,
		},
		{
			name: "invalid authorization server URL",
			metadata: &ProtectedResourceMetadata{
				Resource: "https://mcp.example.com",
				AuthorizationServers: []string{
					"not a valid url://",
				},
			},
			wantErr: true,
		},
		{
			name: "relative authorization server URL",
			metadata: &ProtectedResourceMetadata{
				Resource: "https://mcp.example.com",
				AuthorizationServers: []string{
					"/auth/oauth",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid scheme in authorization server URL",
			metadata: &ProtectedResourceMetadata{
				Resource: "https://mcp.example.com",
				AuthorizationServers: []string{
					"ftp://auth.example.com",
				},
			},
			wantErr: true,
		},
		{
			name: "missing host in authorization server URL",
			metadata: &ProtectedResourceMetadata{
				Resource: "https://mcp.example.com",
				AuthorizationServers: []string{
					"https://",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProtectedResourceMetadata(tt.metadata)

			if tt.wantErr && err == nil {
				t.Errorf("expected error but got none")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestSelectAuthorizationServer(t *testing.T) {
	metadata := &ProtectedResourceMetadata{
		Resource: "https://mcp.example.com",
		AuthorizationServers: []string{
			"https://auth1.example.com",
			"https://auth2.example.com",
			"https://auth3.example.com",
		},
	}

	tests := []struct {
		name      string
		preferred string
		want      string
		wantErr   bool
	}{
		{
			name: "no preference - use first",
			want: "https://auth1.example.com",
		},
		{
			name:      "prefer second server",
			preferred: "https://auth2.example.com",
			want:      "https://auth2.example.com",
		},
		{
			name:      "prefer third server",
			preferred: "https://auth3.example.com",
			want:      "https://auth3.example.com",
		},
		{
			name:      "preferred server not found",
			preferred: "https://auth99.example.com",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := selectAuthorizationServer(metadata, tt.preferred)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDiscoverProtectedResourceMetadata(t *testing.T) {
	// Create test server
	wellKnownPath := "/.well-known/oauth-protected-resource"
	metadata := &ProtectedResourceMetadata{
		Resource: "https://mcp.example.com",
		AuthorizationServers: []string{
			"https://auth.example.com",
		},
		ScopesSupported: []string{"read", "write"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only respond to well-known path
		if r.URL.Path == wellKnownPath {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(metadata)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	tests := []struct {
		name      string
		endpoint  string
		challenge *WWWAuthenticateChallenge
		wantErr   bool
	}{
		{
			name:     "discover from well-known URI",
			endpoint: server.URL,
		},
		{
			name:     "challenge with explicit resource_metadata URL (localhost blocked by SSRF protection)",
			endpoint: server.URL,
			challenge: &WWWAuthenticateChallenge{
				ResourceMetadataURL: server.URL + wellKnownPath,
			},
			wantErr: true, // SSRF protection should block localhost URLs from challenge
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			logger := NewLogger(false, false, false)

			got, err := discoverProtectedResourceMetadata(ctx, tt.endpoint, tt.challenge, logger)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.Resource != metadata.Resource {
				t.Errorf("Resource = %q, want %q", got.Resource, metadata.Resource)
			}

			if len(got.AuthorizationServers) != len(metadata.AuthorizationServers) {
				t.Errorf("got %d authorization servers, want %d", len(got.AuthorizationServers), len(metadata.AuthorizationServers))
			}
		})
	}
}

func TestDiscoverProtectedResourceMetadataNoServer(t *testing.T) {
	// Test with endpoint that has no metadata server
	ctx := context.Background()
	logger := NewLogger(false, false, false)

	_, err := discoverProtectedResourceMetadata(ctx, "https://nonexistent.example.com", nil, logger)

	if err == nil {
		t.Errorf("expected error but got none")
	}
}

func TestIsAllowedMetadataURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "valid https URL",
			url:     "https://example.com/.well-known/oauth-protected-resource",
			wantErr: false,
		},
		{
			name:    "valid http URL",
			url:     "http://example.com/.well-known/oauth-protected-resource",
			wantErr: false,
		},
		{
			name:    "localhost - should be denied",
			url:     "http://localhost:8080/.well-known/oauth-protected-resource",
			wantErr: true,
		},
		{
			name:    "127.0.0.1 - should be denied",
			url:     "http://127.0.0.1:8080/.well-known/oauth-protected-resource",
			wantErr: true,
		},
		{
			name:    "127.x.x.x range - should be denied",
			url:     "http://127.5.5.5/.well-known/oauth-protected-resource",
			wantErr: true,
		},
		{
			name:    "IPv6 loopback - should be denied",
			url:     "http://[::1]/.well-known/oauth-protected-resource",
			wantErr: true,
		},
		{
			name:    "private IP 10.x.x.x - should be denied",
			url:     "http://10.0.0.1/.well-known/oauth-protected-resource",
			wantErr: true,
		},
		{
			name:    "private IP 172.16.x.x - should be denied",
			url:     "http://172.16.0.1/.well-known/oauth-protected-resource",
			wantErr: true,
		},
		{
			name:    "private IP 192.168.x.x - should be denied",
			url:     "http://192.168.1.1/.well-known/oauth-protected-resource",
			wantErr: true,
		},
		{
			name:    "link-local 169.254.x.x - should be denied",
			url:     "http://169.254.169.254/.well-known/oauth-protected-resource",
			wantErr: true,
		},
		{
			name:    "0.0.0.0 - should be denied",
			url:     "http://0.0.0.0/.well-known/oauth-protected-resource",
			wantErr: true,
		},
		{
			name:    "multicast 224.x.x.x - should be denied",
			url:     "http://224.0.0.1/.well-known/oauth-protected-resource",
			wantErr: true,
		},
		{
			name:    "invalid scheme - should be denied",
			url:     "ftp://example.com/.well-known/oauth-protected-resource",
			wantErr: true,
		},
		{
			name:    "relative URL - should be denied",
			url:     "/.well-known/oauth-protected-resource",
			wantErr: true,
		},
		{
			name:    "missing hostname",
			url:     "http://",
			wantErr: true,
		},
		{
			name:    "invalid URL",
			url:     "not a valid url://",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := isAllowedMetadataURL(tt.url)

			if tt.wantErr && err == nil {
				t.Errorf("expected error but got none")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestWarnInsecureAuthServers(t *testing.T) {
	tests := []struct {
		name          string
		metadata      *ProtectedResourceMetadata
		expectWarning bool
		warningSubstr string
	}{
		{
			name: "HTTPS server - no warning",
			metadata: &ProtectedResourceMetadata{
				Resource: "https://mcp.example.com",
				AuthorizationServers: []string{
					"https://auth.example.com",
				},
			},
			expectWarning: false,
		},
		{
			name: "HTTP server - should warn",
			metadata: &ProtectedResourceMetadata{
				Resource: "https://mcp.example.com",
				AuthorizationServers: []string{
					"http://auth.example.com",
				},
			},
			expectWarning: true,
			warningSubstr: "HTTP (not HTTPS)",
		},
		{
			name: "mixed HTTP and HTTPS - should warn about HTTP",
			metadata: &ProtectedResourceMetadata{
				Resource: "https://mcp.example.com",
				AuthorizationServers: []string{
					"https://auth1.example.com",
					"http://auth2.example.com",
					"https://auth3.example.com",
				},
			},
			expectWarning: true,
			warningSubstr: "HTTP (not HTTPS)",
		},
		{
			name: "nil logger - no panic",
			metadata: &ProtectedResourceMetadata{
				Resource: "https://mcp.example.com",
				AuthorizationServers: []string{
					"http://auth.example.com",
				},
			},
			expectWarning: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf strings.Builder
			var logger *Logger

			if tt.expectWarning {
				logger = NewLoggerWithWriter(false, false, false, &buf)
			}

			// Should not panic
			warnInsecureAuthServers(tt.metadata, logger)

			if tt.expectWarning {
				output := buf.String()
				if !strings.Contains(output, tt.warningSubstr) {
					t.Errorf("expected warning containing %q, got: %q", tt.warningSubstr, output)
				}
			}
		})
	}
}

func TestSplitPreservingQuotesWithEscapes(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		delimiter byte
		want      []string
	}{
		{
			name:      "escaped quote within quoted value",
			input:     `a,"b\"c",d`,
			delimiter: ',',
			want:      []string{"a", `"b\"c"`, "d"},
		},
		{
			name:      "escaped backslash",
			input:     `a,"b\\c",d`,
			delimiter: ',',
			want:      []string{"a", `"b\\c"`, "d"},
		},
		{
			name:      "escaped quote at end",
			input:     `"value with \" quote"`,
			delimiter: ',',
			want:      []string{`"value with \" quote"`},
		},
		{
			name:      "multiple escaped quotes",
			input:     `key="value \"with\" quotes"`,
			delimiter: ',',
			want:      []string{`key="value \"with\" quotes"`},
		},
		{
			name:      "escaped delimiter inside quotes should not split",
			input:     `"a\,b","c"`,
			delimiter: ',',
			want:      []string{`"a\,b"`, `"c"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitPreservingQuotes(tt.input, tt.delimiter)

			if len(got) != len(tt.want) {
				t.Errorf("got %d parts, want %d\ngot: %v\nwant: %v", len(got), len(tt.want), got, tt.want)
				return
			}

			for i, part := range got {
				if part != tt.want[i] {
					t.Errorf("part[%d] = %q, want %q", i, part, tt.want[i])
				}
			}
		})
	}
}

func TestParseAuthParamsWithEscapes(t *testing.T) {
	tests := []struct {
		name   string
		params string
		want   map[string]string
	}{
		{
			name:   "escaped quote in value",
			params: `desc="value with \" quote"`,
			want: map[string]string{
				"desc": `value with " quote`,
			},
		},
		{
			name:   "escaped backslash in value",
			params: `path="C:\\Users\\test"`,
			want: map[string]string{
				"path": `C:\Users\test`,
			},
		},
		{
			name:   "multiple keys with escapes",
			params: `key1="value \"one\"", key2="value \\two"`,
			want: map[string]string{
				"key1": `value "one"`,
				"key2": `value \two`,
			},
		},
		{
			name:   "error description with quotes",
			params: `error="invalid_token", error_description="The token \"abc\" expired"`,
			want: map[string]string{
				"error":             "invalid_token",
				"error_description": `The token "abc" expired`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseAuthParams(tt.params)

			if len(got) != len(tt.want) {
				t.Errorf("got %d params, want %d", len(got), len(tt.want))
			}

			for key, wantValue := range tt.want {
				gotValue, ok := got[key]
				if !ok {
					t.Errorf("missing key %q", key)
					continue
				}
				if gotValue != wantValue {
					t.Errorf("param %q = %q, want %q", key, gotValue, wantValue)
				}
			}
		})
	}
}
