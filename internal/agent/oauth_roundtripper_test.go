package agent

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// testRegistrationToken is a test registration token used across multiple tests
const testRegistrationToken = "test-registration-token-12345"

func TestRegistrationTokenRoundTripper(t *testing.T) {
	testToken := testRegistrationToken

	tests := []struct {
		name              string
		method            string
		path              string
		registrationToken string
		expectAuthHeader  bool
		expectedToken     string
	}{
		{
			name:              "adds token to POST /register",
			method:            http.MethodPost,
			path:              "/oauth/register",
			registrationToken: testToken,
			expectAuthHeader:  true,
			expectedToken:     "Bearer " + testToken,
		},
		{
			name:              "adds token to POST /registration",
			method:            http.MethodPost,
			path:              "/oauth/registration",
			registrationToken: testToken,
			expectAuthHeader:  true,
			expectedToken:     "Bearer " + testToken,
		},
		{
			name:              "adds token to POST with mixed case path",
			method:            http.MethodPost,
			path:              "/oauth/Register",
			registrationToken: testToken,
			expectAuthHeader:  true,
			expectedToken:     "Bearer " + testToken,
		},
		{
			name:              "adds token to POST /oauth2/register",
			method:            http.MethodPost,
			path:              "/oauth2/register",
			registrationToken: testToken,
			expectAuthHeader:  true,
			expectedToken:     "Bearer " + testToken,
		},
		{
			name:              "adds token to POST /connect/register",
			method:            http.MethodPost,
			path:              "/connect/register",
			registrationToken: testToken,
			expectAuthHeader:  true,
			expectedToken:     "Bearer " + testToken,
		},
		{
			name:              "does not add token to GET /register",
			method:            http.MethodGet,
			path:              "/oauth/register",
			registrationToken: testToken,
			expectAuthHeader:  false,
			expectedToken:     "",
		},
		{
			name:              "does not add token to POST /token",
			method:            http.MethodPost,
			path:              "/oauth/token",
			registrationToken: testToken,
			expectAuthHeader:  false,
			expectedToken:     "",
		},
		{
			name:              "does not add token when token is empty",
			method:            http.MethodPost,
			path:              "/oauth/register",
			registrationToken: "",
			expectAuthHeader:  false,
			expectedToken:     "",
		},
		{
			name:              "does not add token to POST /user/registration-stats (security)",
			method:            http.MethodPost,
			path:              "/api/user/registration-stats",
			registrationToken: testToken,
			expectAuthHeader:  false,
			expectedToken:     "",
		},
		{
			name:              "does not add token to POST /deregister-device (security)",
			method:            http.MethodPost,
			path:              "/v1/deregister-device",
			registrationToken: testToken,
			expectAuthHeader:  false,
			expectedToken:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server that captures the request
			var capturedAuthHeader string
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedAuthHeader = r.Header.Get("Authorization")
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			// Create a RoundTripper with the test token
			rt := newRegistrationTokenRoundTripper(tt.registrationToken, server.Client().Transport, nil)

			// Create a client with our custom RoundTripper
			client := &http.Client{
				Transport: rt,
			}

			// Build the request URL using the test server
			url := server.URL + tt.path

			// Create and execute the request
			req, err := http.NewRequest(tt.method, url, strings.NewReader("test body"))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			// Read and discard the response body
			_, _ = io.ReadAll(resp.Body)

			// Verify the Authorization header
			if tt.expectAuthHeader {
				if capturedAuthHeader == "" {
					t.Errorf("Expected Authorization header but none was set")
				} else if capturedAuthHeader != tt.expectedToken {
					t.Errorf("Expected Authorization header %q, got %q", tt.expectedToken, capturedAuthHeader)
				}
			} else {
				if capturedAuthHeader != "" {
					t.Errorf("Did not expect Authorization header, but got %q", capturedAuthHeader)
				}
			}
		})
	}
}

func TestRegistrationTokenRoundTripper_WithNilBaseTransport(t *testing.T) {
	// Test that the RoundTripper works with nil base transport (should use default)
	rt := newRegistrationTokenRoundTripper("test-token", nil, nil)
	if rt == nil {
		t.Fatal("Expected non-nil RoundTripper")
	}

	// Verify it uses the internal transport
	rtt, ok := rt.(*registrationTokenRoundTripper)
	if !ok {
		t.Fatal("Expected registrationTokenRoundTripper type")
	}

	if rtt.transport == nil {
		t.Error("Expected non-nil internal transport (should use default)")
	}
}

func TestRegistrationTokenRoundTripper_RequestCloning(t *testing.T) {
	// Test that the original request is not modified
	testToken := "test-token"

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	rt := newRegistrationTokenRoundTripper(testToken, server.Client().Transport, nil)
	client := &http.Client{Transport: rt}

	url := server.URL + "/oauth/register"
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Store original header value
	originalAuth := req.Header.Get("Authorization")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.ReadAll(resp.Body)

	// Verify the original request was not modified
	newAuth := req.Header.Get("Authorization")
	if newAuth != originalAuth {
		t.Errorf("Original request was modified: original=%q, new=%q", originalAuth, newAuth)
	}
}

// TestRegistrationTokenRoundTripper_HTTPSEnforcement tests that tokens are only sent over HTTPS
func TestRegistrationTokenRoundTripper_HTTPSEnforcement(t *testing.T) {
	testToken := testRegistrationToken

	// Create an HTTP (not HTTPS) test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Request should not have reached the server due to HTTPS enforcement")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create a RoundTripper with the test token
	rt := newRegistrationTokenRoundTripper(testToken, nil, nil)
	client := &http.Client{Transport: rt}

	// Try to send a registration request over HTTP (should fail)
	url := server.URL + "/oauth/register"
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader("test body"))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err == nil {
		defer func() { _ = resp.Body.Close() }()
		t.Fatal("Expected error due to HTTP (not HTTPS) scheme, but got none")
	}

	// Verify the error message mentions security and HTTPS
	errMsg := err.Error()
	if !strings.Contains(errMsg, "security") || !strings.Contains(errMsg, "HTTPS") {
		t.Errorf("Expected security-related HTTPS error, got: %v", err)
	}
}

// TestRegistrationTokenRoundTripper_HeaderConflict tests that existing Authorization headers are not overwritten
func TestRegistrationTokenRoundTripper_HeaderConflict(t *testing.T) {
	testToken := testRegistrationToken

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Request should not have reached the server due to header conflict")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	rt := newRegistrationTokenRoundTripper(testToken, server.Client().Transport, nil)
	client := &http.Client{Transport: rt}

	url := server.URL + "/oauth/register"
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader("test body"))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Add an existing Authorization header
	req.Header.Set("Authorization", "Bearer existing-token")

	resp, err := client.Do(req)
	if err == nil {
		defer func() { _ = resp.Body.Close() }()
		t.Fatal("Expected error due to existing Authorization header, but got none")
	}

	// Verify the error message mentions the conflict
	errMsg := err.Error()
	if !strings.Contains(errMsg, "authorization header already present") {
		t.Errorf("Expected authorization header conflict error, got: %v", err)
	}
}

// TestIsRegistrationEndpoint tests the endpoint matching logic
func TestIsRegistrationEndpoint(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		// Should match
		{"/register", true},
		{"/registration", true},
		{"/oauth/register", true},
		{"/oauth2/register", true},
		{"/connect/register", true},
		{"/oauth/registration", true},
		{"/oauth2/registration", true},
		{"/connect/registration", true},
		{"/.well-known/openid-registration", true},
		{"/Register", true},                    // Case insensitive
		{"/OAuth/REGISTER", true},              // Case insensitive
		{"/api/v1/oauth/register", true},       // Suffix match
		{"/api/v1/oauth2/registration/", true}, // With trailing slash

		// Should NOT match (security test - prevent token leakage)
		{"/user/registration-stats", false},
		{"/api/deregister-device", false},
		{"/admin/register-payment", false},
		{"/oauth/token", false},
		{"/oauth/authorize", false},
		{"/api/users", false},
		{"/registration-webhook", false},
		{"/preregister", false},
		{"/api/v1/user-registration", false}, // Contains "registration" but not as endpoint
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isRegistrationEndpoint(tt.path)
			if result != tt.expected {
				t.Errorf("isRegistrationEndpoint(%q) = %v, expected %v", tt.path, result, tt.expected)
			}
		})
	}
}

// TestRegistrationTokenRoundTripper_NonRegistrationEndpoints tests that tokens are NOT injected to non-registration endpoints
func TestRegistrationTokenRoundTripper_NonRegistrationEndpoints(t *testing.T) {
	testToken := testRegistrationToken

	nonRegistrationPaths := []string{
		"/oauth/token",
		"/oauth/authorize",
		"/user/registration-stats",
		"/api/deregister-device",
		"/admin/register-payment",
		"/api/users",
	}

	for _, path := range nonRegistrationPaths {
		t.Run(fmt.Sprintf("no_token_to_%s", path), func(t *testing.T) {
			var capturedAuthHeader string
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedAuthHeader = r.Header.Get("Authorization")
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			rt := newRegistrationTokenRoundTripper(testToken, server.Client().Transport, nil)
			client := &http.Client{Transport: rt}

			url := server.URL + path
			req, err := http.NewRequest(http.MethodPost, url, strings.NewReader("test body"))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()
			_, _ = io.ReadAll(resp.Body)

			// Verify NO Authorization header was added
			if capturedAuthHeader != "" {
				t.Errorf("Expected no Authorization header for %s, but got %q", path, capturedAuthHeader)
			}
		})
	}
}
