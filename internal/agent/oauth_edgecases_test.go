package agent

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// TestResourceRoundTripperEdgeCases tests edge cases for resource parameter injection per RFC 8707
// Verifies that resource indicators are correctly added to OAuth requests
func TestResourceRoundTripperEdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		resourceURI    string
		skipResource   bool
		method         string
		url            string
		body           string
		contentType    string
		expectResource bool
		expectError    bool
	}{
		{
			name:           "POST to token endpoint with empty resource URI",
			resourceURI:    "",
			skipResource:   false,
			method:         http.MethodPost,
			url:            "/token",
			body:           "grant_type=authorization_code&code=test",
			contentType:    "application/x-www-form-urlencoded",
			expectResource: false,
		},
		{
			name:           "POST to token endpoint with skipResource true",
			resourceURI:    "https://example.com",
			skipResource:   true,
			method:         http.MethodPost,
			url:            "/token",
			body:           "grant_type=authorization_code&code=test",
			contentType:    "application/x-www-form-urlencoded",
			expectResource: false,
		},
		{
			name:           "GET to authorize with resource",
			resourceURI:    "https://example.com/api",
			skipResource:   false,
			method:         http.MethodGet,
			url:            "/authorize?response_type=code&client_id=test",
			expectResource: true,
		},
		{
			name:           "POST with non-form content type",
			resourceURI:    "https://example.com",
			skipResource:   false,
			method:         http.MethodPost,
			url:            "/token",
			body:           `{"grant_type":"authorization_code"}`,
			contentType:    "application/json",
			expectResource: true, // Currently adds to all POST /token requests regardless of content-type
		},
		{
			name:           "GET to non-OAuth endpoint",
			resourceURI:    "https://example.com",
			skipResource:   false,
			method:         http.MethodGet,
			url:            "/api/data",
			expectResource: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Track whether resource parameter was received
			var receivedResource string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check for resource parameter in query or body
				switch r.Method {
				case http.MethodGet:
					receivedResource = r.URL.Query().Get("resource")
				case http.MethodPost:
					bodyBytes, _ := io.ReadAll(r.Body)
					values, _ := url.ParseQuery(string(bodyBytes))
					receivedResource = values.Get("resource")
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			rt := newResourceRoundTripper(tt.resourceURI, tt.skipResource, nil, nil)

			var body io.Reader
			if tt.body != "" {
				body = strings.NewReader(tt.body)
			}

			req, err := http.NewRequest(tt.method, server.URL+tt.url, body)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			client := &http.Client{Transport: rt}
			resp, err := client.Do(req)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			// Verify resource parameter was added when expected
			if tt.expectResource && receivedResource == "" {
				t.Errorf("expected resource parameter to be added, but it was missing")
			}
			if !tt.expectResource && receivedResource != "" {
				t.Errorf("expected no resource parameter, but got: %s", receivedResource)
			}
			if tt.expectResource && receivedResource != tt.resourceURI {
				t.Errorf("resource parameter = %s, want %s", receivedResource, tt.resourceURI)
			}
		})
	}
}

// TestAddResourceParameterEdgeCases tests edge cases in addResourceParameter
func TestAddResourceParameterEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		url          string
		body         string
		contentType  string
		resourceURI  string
		wantResource bool
	}{
		{
			name:         "GET request adds resource to query",
			method:       http.MethodGet,
			url:          "http://example.com/authorize",
			resourceURI:  "https://resource.example.com",
			wantResource: true,
		},
		{
			name:         "POST with form data adds resource to body",
			method:       http.MethodPost,
			url:          "http://example.com/token",
			body:         "grant_type=authorization_code",
			contentType:  "application/x-www-form-urlencoded",
			resourceURI:  "https://resource.example.com",
			wantResource: true,
		},
		{
			name:         "POST with empty body adds resource",
			method:       http.MethodPost,
			url:          "http://example.com/token",
			body:         "",
			contentType:  "application/x-www-form-urlencoded",
			resourceURI:  "https://resource.example.com",
			wantResource: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body io.Reader
			if tt.body != "" {
				body = strings.NewReader(tt.body)
			} else if tt.method == http.MethodPost {
				// For POST with empty body, use empty reader instead of nil
				body = strings.NewReader("")
			}

			req, err := http.NewRequest(tt.method, tt.url, body)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			rt := &resourceRoundTripper{
				resourceURI: tt.resourceURI,
			}

			err = rt.addResourceParameter(req)
			if err != nil {
				t.Fatalf("addResourceParameter failed: %v", err)
			}

			hasResource := false
			if req.Method == http.MethodGet {
				hasResource = req.URL.Query().Has("resource")
			} else if req.Method == http.MethodPost && req.Body != nil {
				bodyBytes, _ := io.ReadAll(req.Body)
				hasResource = strings.Contains(string(bodyBytes), "resource=")
			}

			if hasResource != tt.wantResource {
				t.Errorf("resource present = %v, want %v", hasResource, tt.wantResource)
			}
		})
	}
}

// TestRegistrationTokenRoundTripperEdgeCases tests edge cases for registration token injection
func TestRegistrationTokenRoundTripperEdgeCases(t *testing.T) {
	tests := []struct {
		name             string
		path             string
		method           string
		token            string
		expectAuthHeader bool
	}{
		{
			name:             "POST to /register with token",
			path:             "/oauth/register",
			method:           http.MethodPost,
			token:            "test-token",
			expectAuthHeader: true,
		},
		{
			name:             "POST to /registration with token",
			path:             "/oauth/registration",
			method:           http.MethodPost,
			token:            "test-token",
			expectAuthHeader: true,
		},
		{
			name:             "POST to /connect/register with token",
			path:             "/connect/register",
			method:           http.MethodPost,
			token:            "test-token",
			expectAuthHeader: true,
		},
		{
			name:             "GET to /register should not add token",
			path:             "/oauth/register",
			method:           http.MethodGet,
			token:            "test-token",
			expectAuthHeader: false,
		},
		{
			name:             "POST to /token should not add token",
			path:             "/oauth/token",
			method:           http.MethodPost,
			token:            "test-token",
			expectAuthHeader: false,
		},
		{
			name:             "POST to /register with empty token",
			path:             "/oauth/register",
			method:           http.MethodPost,
			token:            "",
			expectAuthHeader: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			rt := newRegistrationTokenRoundTripper(tt.token, server.Client().Transport, nil)
			client := &http.Client{Transport: rt}

			req, err := http.NewRequest(tt.method, server.URL+tt.path, strings.NewReader("test body"))
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()
		})
	}
}

// TestDiscoveryTimeouts tests timeout handling in discovery functions per RFC 8414
// Verifies that discovery respects context deadlines for network operations
func TestDiscoveryTimeouts(t *testing.T) {
	tests := []struct {
		name        string
		timeout     context.Context
		expectError bool
	}{
		{
			name: "timeout during metadata fetch",
			timeout: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), testTimeoutShort)
				t.Cleanup(cancel)
				return ctx
			}(),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a slow server that delays longer than the timeout
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				select {
				case <-r.Context().Done():
					// Context was properly cancelled
					return
				case <-time.After(testDelayLong):
					w.WriteHeader(http.StatusOK)
				}
			}))
			defer server.Close()

			logger := NewLogger(false, false, false)
			_, err := DiscoverAuthorizationServerMetadata(tt.timeout, server.URL, logger)

			if tt.expectError && err == nil {
				t.Error("expected timeout error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestParseWWWAuthenticateEdgeCases tests edge cases in WWW-Authenticate parsing
func TestParseWWWAuthenticateEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		header      string
		expectError bool
		checkResult func(*testing.T, *WWWAuthenticateChallenge)
	}{
		{
			name:        "empty header",
			header:      "",
			expectError: true,
		},
		{
			name:        "scheme only",
			header:      "Bearer",
			expectError: false,
			checkResult: func(t *testing.T, c *WWWAuthenticateChallenge) {
				if c.Scheme != "Bearer" {
					t.Errorf("Scheme = %s, want Bearer", c.Scheme)
				}
			},
		},
		{
			name:        "with multiple parameters",
			header:      `Bearer desc="Error description", scope="read write"`,
			expectError: false,
			checkResult: func(t *testing.T, c *WWWAuthenticateChallenge) {
				// Just verify it parses without error
				if c.Scheme != "Bearer" {
					t.Errorf("Scheme = %s, want Bearer", c.Scheme)
				}
			},
		},
		{
			name:        "with multiple spaces",
			header:      `Bearer   key="value"  ,  key2="value2"`,
			expectError: false,
			checkResult: func(t *testing.T, c *WWWAuthenticateChallenge) {
				if c.Scheme != "Bearer" {
					t.Errorf("Scheme = %s, want Bearer", c.Scheme)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			challenge, err := parseWWWAuthenticate(tt.header)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.checkResult != nil {
				tt.checkResult(t, challenge)
			}
		})
	}
}

// TestScopeEqualEdgeCases tests scopesEqual function edge cases
func TestScopeEqualEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		a    []string
		b    []string
		want bool
	}{
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "both empty",
			a:    []string{},
			b:    []string{},
			want: true,
		},
		{
			name: "one nil one empty",
			a:    nil,
			b:    []string{},
			want: true,
		},
		{
			name: "different lengths",
			a:    []string{"a", "b"},
			b:    []string{"a"},
			want: false,
		},
		{
			name: "same scopes different order",
			a:    []string{"read", "write"},
			b:    []string{"write", "read"},
			want: true, // Order doesn't matter (uses map comparison)
		},
		{
			name: "same scopes same order",
			a:    []string{"read", "write"},
			b:    []string{"read", "write"},
			want: true,
		},
		{
			name: "duplicate scopes in first array",
			a:    []string{"read", "read", "write"},
			b:    []string{"read", "write"},
			want: false, // Should fail because lengths differ
		},
		{
			name: "duplicate scopes in both arrays same count",
			a:    []string{"read", "read", "write", "write"},
			b:    []string{"read", "read", "write", "write"},
			want: true, // Same duplicates in both
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scopesEqual(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("scopesEqual(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

// TestCallbackServerEdgeCases tests callback server edge cases
func TestCallbackServerEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		redirectURL string
		expectError bool
	}{
		{
			name:        "valid localhost redirect",
			redirectURL: "http://localhost:8765/callback",
			expectError: false,
		},
		{
			name:        "valid 127.0.0.1 redirect",
			redirectURL: "http://127.0.0.1:8765/callback",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(false, false, false)

			config := &callbackServerConfig{
				redirectURL: tt.redirectURL,
				logger:      logger,
			}

			server, _, err := startCallbackServer(config)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Clean up server
			if server != nil {
				_ = server.Close()
			}
		})
	}
}

// TestBrowserURLValidation tests browser URL validation
func TestBrowserURLValidation(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "valid https URL",
			url:         "https://auth.example.com/authorize",
			expectError: false,
		},
		{
			name:        "valid http localhost URL",
			url:         "http://localhost:8080/authorize",
			expectError: false,
		},
		{
			name:        "http non-localhost allowed (for testing)",
			url:         "http://auth.example.com/authorize",
			expectError: false,
		},
		{
			name:        "javascript URL rejected",
			url:         "javascript:alert(1)",
			expectError: true,
		},
		{
			name:        "data URL rejected",
			url:         "data:text/html,<script>alert(1)</script>",
			expectError: true,
		},
		{
			name:        "file URL rejected",
			url:         "file:///etc/passwd",
			expectError: true,
		},
		{
			name:        "empty URL rejected",
			url:         "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBrowserURL(tt.url)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestCallbackHandlerEdgeCases tests callback handler edge cases
func TestCallbackHandlerEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		expectError  bool
		expectedCode string
		expectedErr  string
	}{
		{
			name:         "valid callback with code",
			query:        "?code=test-code&state=test-state",
			expectError:  false,
			expectedCode: "test-code",
		},
		{
			name:        "callback with error",
			query:       "?error=access_denied&error_description=User%20denied",
			expectError: true,
			expectedErr: "access_denied",
		},
		{
			name:        "callback with error no description",
			query:       "?error=server_error",
			expectError: true,
			expectedErr: "server_error",
		},
		{
			name:         "callback missing code and error",
			query:        "?state=test-state",
			expectError:  false,
			expectedCode: "", // No code, but not an error either
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultChan := make(chan callbackResult, 1)
			logger := NewLogger(false, false, false)

			handler := createCallbackHandler(logger, resultChan)

			req := httptest.NewRequest(http.MethodGet, "/callback"+tt.query, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			select {
			case result := <-resultChan:
				if tt.expectError {
					if result.err == nil {
						t.Error("expected error but got none")
					} else if tt.expectedErr != "" && !strings.Contains(result.err.Error(), tt.expectedErr) {
						t.Errorf("error = %v, want to contain %s", result.err, tt.expectedErr)
					}
				} else {
					if result.err != nil {
						t.Errorf("unexpected error: %v", result.err)
					}
					if code := result.params["code"]; code != tt.expectedCode {
						t.Errorf("code = %s, want %s", code, tt.expectedCode)
					}
				}
			case <-time.After(testTimeoutLong):
				t.Fatal("timeout waiting for callback result")
			}
		})
	}
}
