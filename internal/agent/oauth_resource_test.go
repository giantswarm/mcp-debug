package agent

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// TestDeriveResourceURI tests the canonical URI derivation from various endpoint formats
func TestDeriveResourceURI(t *testing.T) {
	tests := []struct {
		name        string
		endpoint    string
		expected    string
		expectError bool
	}{
		{
			name:     "standard HTTPS with path",
			endpoint: "https://mcp.example.com/mcp",
			expected: "https://mcp.example.com/mcp",
		},
		{
			name:     "HTTPS with uppercase host",
			endpoint: "https://MCP.Example.Com/mcp",
			expected: "https://mcp.example.com/mcp",
		},
		{
			name:     "HTTPS with standard port (should be omitted)",
			endpoint: "https://mcp.example.com:443/mcp",
			expected: "https://mcp.example.com/mcp",
		},
		{
			name:     "HTTPS with non-standard port (should be included)",
			endpoint: "https://mcp.example.com:8443/mcp",
			expected: "https://mcp.example.com:8443/mcp",
		},
		{
			name:     "HTTP with standard port (should be omitted)",
			endpoint: "http://localhost:80/mcp",
			expected: "http://localhost/mcp",
		},
		{
			name:     "HTTP with non-standard port (should be included)",
			endpoint: "http://localhost:8090/mcp",
			expected: "http://localhost:8090/mcp",
		},
		{
			name:     "path with trailing slash",
			endpoint: "https://example.com/mcp/",
			expected: "https://example.com/mcp",
		},
		{
			name:     "root path",
			endpoint: "https://example.com/",
			expected: "https://example.com/",
		},
		{
			name:     "root path without trailing slash",
			endpoint: "https://example.com",
			expected: "https://example.com",
		},
		{
			name:     "nested path",
			endpoint: "https://example.com/api/v1/mcp",
			expected: "https://example.com/api/v1/mcp",
		},
		{
			name:     "localhost with port",
			endpoint: "http://localhost:8090/mcp",
			expected: "http://localhost:8090/mcp",
		},
		{
			name:     "IPv4 address",
			endpoint: "http://127.0.0.1:8090/mcp",
			expected: "http://127.0.0.1:8090/mcp",
		},
		{
			name:     "IPv6 address with port",
			endpoint: "http://[::1]:8090/mcp",
			expected: "http://[::1]:8090/mcp",
		},
		{
			name:        "missing scheme",
			endpoint:    "example.com/mcp",
			expectError: true,
		},
		{
			name:        "empty endpoint",
			endpoint:    "",
			expectError: true,
		},
		{
			name:        "invalid URL",
			endpoint:    "ht!tp://invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := deriveResourceURI(tt.endpoint)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for endpoint %q, but got none", tt.endpoint)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error for endpoint %q: %v", tt.endpoint, err)
				return
			}
			if result != tt.expected {
				t.Errorf("deriveResourceURI(%q) = %q, want %q", tt.endpoint, result, tt.expected)
			}
		})
	}
}

// TestResourceRoundTripper tests the HTTP roundtripper that injects resource parameter
func TestResourceRoundTripper(t *testing.T) {
	tests := []struct {
		name             string
		resourceURI      string
		skipResource     bool
		requestMethod    string
		requestURL       string
		requestBody      string
		contentType      string
		expectResource   bool
		expectedResource string
	}{
		{
			name:             "GET authorization request with resource",
			resourceURI:      "https://mcp.example.com/mcp",
			skipResource:     false,
			requestMethod:    http.MethodGet,
			requestURL:       "/authorize?response_type=code&client_id=test-client&redirect_uri=http://localhost:8765/callback",
			expectResource:   true,
			expectedResource: "https://mcp.example.com/mcp",
		},
		{
			name:             "POST token request with resource",
			resourceURI:      "https://mcp.example.com/mcp",
			skipResource:     false,
			requestMethod:    http.MethodPost,
			requestURL:       "/token",
			requestBody:      "grant_type=authorization_code&code=abc123&client_id=test-client",
			contentType:      "application/x-www-form-urlencoded",
			expectResource:   true,
			expectedResource: "https://mcp.example.com/mcp",
		},
		{
			name:           "POST token request with skipResource",
			resourceURI:    "https://mcp.example.com/mcp",
			skipResource:   true,
			requestMethod:  http.MethodPost,
			requestURL:     "/token",
			requestBody:    "grant_type=authorization_code&code=abc123",
			contentType:    "application/x-www-form-urlencoded",
			expectResource: false,
		},
		{
			name:           "POST token request with empty resource URI",
			resourceURI:    "",
			skipResource:   false,
			requestMethod:  http.MethodPost,
			requestURL:     "/token",
			requestBody:    "grant_type=authorization_code&code=abc123",
			contentType:    "application/x-www-form-urlencoded",
			expectResource: false,
		},
		{
			name:           "non-OAuth request should not be modified",
			resourceURI:    "https://mcp.example.com/mcp",
			skipResource:   false,
			requestMethod:  http.MethodGet,
			requestURL:     "/api/data",
			expectResource: false,
		},
		{
			name:             "POST token request with refresh token",
			resourceURI:      "https://mcp.example.com/mcp",
			skipResource:     false,
			requestMethod:    http.MethodPost,
			requestURL:       "/oauth2/token",
			requestBody:      "grant_type=refresh_token&refresh_token=xyz789&client_id=test-client",
			contentType:      "application/x-www-form-urlencoded",
			expectResource:   true,
			expectedResource: "https://mcp.example.com/mcp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server to capture the modified request
			var capturedReq *http.Request
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedReq = r.Clone(r.Context())
				// Read body if present
				if r.Body != nil {
					bodyBytes, _ := io.ReadAll(r.Body)
					capturedReq.Body = io.NopCloser(bytes.NewReader(bodyBytes))
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			// Create roundtripper
			transport := newResourceRoundTripper(tt.resourceURI, tt.skipResource, nil, nil)

			// Create request
			var reqBody io.Reader
			if tt.requestBody != "" {
				reqBody = strings.NewReader(tt.requestBody)
			}
			req, err := http.NewRequest(tt.requestMethod, server.URL+tt.requestURL, reqBody)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			// Execute request through roundtripper
			client := &http.Client{Transport: transport}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			resp.Body.Close()

			// Verify resource parameter
			if tt.expectResource {
				var resourceValue string
				if capturedReq.Method == http.MethodGet {
					// Check query parameters
					resourceValue = capturedReq.URL.Query().Get("resource")
				} else if capturedReq.Method == http.MethodPost {
					// Check form data
					bodyBytes, err := io.ReadAll(capturedReq.Body)
					if err != nil {
						t.Fatalf("failed to read captured body: %v", err)
					}
					values, err := url.ParseQuery(string(bodyBytes))
					if err != nil {
						t.Fatalf("failed to parse form data: %v", err)
					}
					resourceValue = values.Get("resource")
				}

				if resourceValue == "" {
					t.Errorf("expected resource parameter to be present, but it was missing")
				} else if resourceValue != tt.expectedResource {
					t.Errorf("resource parameter = %q, want %q", resourceValue, tt.expectedResource)
				}
			} else {
				// Verify resource parameter is NOT present
				var resourceValue string
				if capturedReq.Method == http.MethodGet {
					resourceValue = capturedReq.URL.Query().Get("resource")
				} else if capturedReq.Method == http.MethodPost && capturedReq.Body != nil {
					bodyBytes, _ := io.ReadAll(capturedReq.Body)
					values, _ := url.ParseQuery(string(bodyBytes))
					resourceValue = values.Get("resource")
				}

				if resourceValue != "" {
					t.Errorf("expected no resource parameter, but found: %q", resourceValue)
				}
			}
		})
	}
}

// TestIsOAuthRequest tests the OAuth request detection logic
func TestIsOAuthRequest(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		url         string
		expectOAuth bool
		description string
	}{
		{
			name:        "authorization request",
			method:      http.MethodGet,
			url:         "/authorize?response_type=code&client_id=test",
			expectOAuth: true,
			description: "GET request with response_type=code and client_id",
		},
		{
			name:        "token endpoint",
			method:      http.MethodPost,
			url:         "/token",
			expectOAuth: true,
			description: "POST to /token",
		},
		{
			name:        "oauth2 token endpoint",
			method:      http.MethodPost,
			url:         "/oauth2/token",
			expectOAuth: true,
			description: "POST to /oauth2/token",
		},
		{
			name:        "regular GET request",
			method:      http.MethodGet,
			url:         "/api/data",
			expectOAuth: false,
			description: "regular API call",
		},
		{
			name:        "regular POST request",
			method:      http.MethodPost,
			url:         "/api/submit",
			expectOAuth: false,
			description: "regular POST to non-OAuth endpoint",
		},
		{
			name:        "auth request without client_id",
			method:      http.MethodGet,
			url:         "/authorize?response_type=code",
			expectOAuth: false,
			description: "missing client_id parameter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, tt.url, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			rt := newResourceRoundTripper("https://example.com", false, nil, nil)
			result := rt.isOAuthRequest(req)

			if result != tt.expectOAuth {
				t.Errorf("%s: isOAuthRequest() = %v, want %v", tt.description, result, tt.expectOAuth)
			}
		})
	}
}

// TestResourceRoundTripperIntegration tests end-to-end flow
func TestResourceRoundTripperIntegration(t *testing.T) {
	resourceURI := "https://mcp.example.com/mcp"

	// Create mock server
	capturedRequests := make([]*http.Request, 0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqCopy := r.Clone(r.Context())
		if r.Body != nil {
			bodyBytes, _ := io.ReadAll(r.Body)
			reqCopy.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}
		capturedRequests = append(capturedRequests, reqCopy)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"access_token": "test-token"}`))
	}))
	defer server.Close()

	// Create HTTP client with resource roundtripper
	transport := newResourceRoundTripper(resourceURI, false, nil, nil)
	client := &http.Client{Transport: transport}

	// Simulate authorization request
	authReq, _ := http.NewRequest(http.MethodGet,
		server.URL+"/authorize?response_type=code&client_id=test-client&redirect_uri=http://localhost:8765/callback",
		nil)
	resp1, err := client.Do(authReq)
	if err != nil {
		t.Fatalf("authorization request failed: %v", err)
	}
	resp1.Body.Close()

	// Simulate token request
	tokenReq, _ := http.NewRequest(http.MethodPost,
		server.URL+"/token",
		strings.NewReader("grant_type=authorization_code&code=test-code&client_id=test-client"))
	tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp2, err := client.Do(tokenReq)
	if err != nil {
		t.Fatalf("token request failed: %v", err)
	}
	resp2.Body.Close()

	// Verify both requests have resource parameter
	if len(capturedRequests) != 2 {
		t.Fatalf("expected 2 captured requests, got %d", len(capturedRequests))
	}

	// Check authorization request
	authResource := capturedRequests[0].URL.Query().Get("resource")
	if authResource != resourceURI {
		t.Errorf("authorization request resource = %q, want %q", authResource, resourceURI)
	}

	// Check token request
	bodyBytes, _ := io.ReadAll(capturedRequests[1].Body)
	tokenValues, _ := url.ParseQuery(string(bodyBytes))
	tokenResource := tokenValues.Get("resource")
	if tokenResource != resourceURI {
		t.Errorf("token request resource = %q, want %q", tokenResource, resourceURI)
	}
}
