package agent

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRegistrationTokenRoundTripper(t *testing.T) {
	testToken := "test-registration-token-12345"

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server that captures the request
			var capturedAuthHeader string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedAuthHeader = r.Header.Get("Authorization")
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			// Create a RoundTripper with the test token
			rt := newRegistrationTokenRoundTripper(tt.registrationToken, nil)

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
			defer resp.Body.Close()

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
	rt := newRegistrationTokenRoundTripper("test-token", nil)
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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	rt := newRegistrationTokenRoundTripper(testToken, nil)
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
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)

	// Verify the original request was not modified
	newAuth := req.Header.Get("Authorization")
	if newAuth != originalAuth {
		t.Errorf("Original request was modified: original=%q, new=%q", originalAuth, newAuth)
	}
}
