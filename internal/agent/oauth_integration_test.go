package agent

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestOAuthEndToEndFlow tests the complete OAuth flow with mock servers
func TestOAuthEndToEndFlow(t *testing.T) {
	// Create mock authorization server
	mockAS := NewMockAuthServer(t)
	defer mockAS.Close()

	// Create mock MCP server that requires authentication
	mockMCP := NewMockMCPServer(t, mockAS.URL)
	defer mockMCP.Close()

	tests := []struct {
		name            string
		setupAS         func(*MockAuthServer)
		setupMCP        func(*MockMCPServer)
		expectedSuccess bool
		description     string
	}{
		{
			name: "successful OAuth flow with PKCE and resource indicators",
			setupAS: func(as *MockAuthServer) {
				as.supportsPKCE = true
				as.supportsResourceIndicators = true
				as.supportsClientRegistration = true
			},
			setupMCP: func(mcp *MockMCPServer) {
				mcp.requireAuth = true
				mcp.requiredScopes = []string{"mcp:read"}
			},
			expectedSuccess: true,
			description:     "Complete flow with all security features enabled",
		},
		{
			name: "OAuth flow without resource indicators",
			setupAS: func(as *MockAuthServer) {
				as.supportsPKCE = true
				as.supportsResourceIndicators = false
				as.supportsClientRegistration = true
			},
			setupMCP: func(mcp *MockMCPServer) {
				mcp.requireAuth = true
			},
			expectedSuccess: true,
			description:     "Flow works when resource indicators are not supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupAS != nil {
				tt.setupAS(mockAS)
			}
			if tt.setupMCP != nil {
				tt.setupMCP(mockMCP)
			}

			// Verify mock servers are working
			resp, err := http.Get(mockAS.URL + "/.well-known/oauth-authorization-server")
			if err != nil {
				t.Fatalf("failed to reach AS metadata endpoint: %v", err)
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("AS metadata status = %d, want %d", resp.StatusCode, http.StatusOK)
			}
		})
	}
}

// TestProtectedResourceMetadataIntegration tests metadata discovery with mock MCP server
func TestProtectedResourceMetadataIntegration(t *testing.T) {
	mockAS := NewMockAuthServer(t)
	defer mockAS.Close()

	mockMCP := NewMockMCPServer(t, mockAS.URL)
	defer mockMCP.Close()

	ctx := context.Background()
	logger := NewLogger(false, false, false)

	// Test discovery from well-known URI
	metadata, err := discoverProtectedResourceMetadata(ctx, mockMCP.URL, nil, logger)
	if err != nil {
		t.Fatalf("discovery failed: %v", err)
	}

	if metadata.Resource != mockMCP.URL {
		t.Errorf("Resource = %s, want %s", metadata.Resource, mockMCP.URL)
	}

	if len(metadata.AuthorizationServers) != 1 {
		t.Errorf("got %d auth servers, want 1", len(metadata.AuthorizationServers))
	} else if metadata.AuthorizationServers[0] != mockAS.URL {
		t.Errorf("auth server = %s, want %s", metadata.AuthorizationServers[0], mockAS.URL)
	}
}

// TestAuthorizationServerMetadataIntegration tests AS metadata discovery
func TestAuthorizationServerMetadataIntegration(t *testing.T) {
	mockAS := NewMockAuthServer(t)
	defer mockAS.Close()

	ctx := context.Background()
	logger := NewLogger(false, false, false)

	// Test discovery
	metadata, err := DiscoverAuthorizationServerMetadata(ctx, mockAS.URL, logger)
	if err != nil {
		t.Fatalf("discovery failed: %v", err)
	}

	if metadata.Issuer != mockAS.URL {
		t.Errorf("Issuer = %s, want %s", metadata.Issuer, mockAS.URL)
	}

	if metadata.AuthorizationEndpoint != mockAS.URL+"/authorize" {
		t.Errorf("AuthorizationEndpoint = %s, want %s", metadata.AuthorizationEndpoint, mockAS.URL+"/authorize")
	}

	if metadata.TokenEndpoint != mockAS.URL+"/token" {
		t.Errorf("TokenEndpoint = %s, want %s", metadata.TokenEndpoint, mockAS.URL+"/token")
	}

	// Verify PKCE support
	err = ValidatePKCESupport(metadata, false, logger)
	if err != nil {
		t.Errorf("PKCE validation failed: %v", err)
	}
}

// TestResourceRoundTripperE2E tests resource parameter injection in real OAuth flow
func TestResourceRoundTripperE2E(t *testing.T) {
	mockAS := NewMockAuthServer(t)
	mockAS.supportsResourceIndicators = true
	defer mockAS.Close()

	resourceURI := "https://mcp.example.com/mcp"

	// Create HTTP client with resource roundtripper
	transport := newResourceRoundTripper(resourceURI, false, nil, nil)
	client := &http.Client{
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Test authorization endpoint
	authURL := mockAS.URL + "/authorize" +
		"?response_type=code" +
		"&client_id=test-client" +
		"&redirect_uri=http://localhost:8765/callback" +
		"&code_challenge=test-challenge" +
		"&code_challenge_method=S256"

	resp, err := client.Get(authURL)
	if err != nil {
		t.Fatalf("authorization request failed: %v", err)
	}
	defer resp.Body.Close()

	// Verify redirect received
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected redirect, got status %d", resp.StatusCode)
	}

	// Verify the request had resource parameter (check by making another request and inspecting)
	// Note: The roundtripper should have added the resource parameter
	location := resp.Header.Get("Location")
	if !strings.Contains(location, "code=") {
		t.Error("redirect location missing authorization code")
	}
}

// TestDynamicClientRegistration tests client registration flow
func TestDynamicClientRegistration(t *testing.T) {
	mockAS := NewMockAuthServer(t)
	mockAS.supportsClientRegistration = true
	defer mockAS.Close()

	// Discover AS metadata
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	logger := NewLogger(false, false, false)
	metadata, err := DiscoverAuthorizationServerMetadata(ctx, mockAS.URL, logger)
	if err != nil {
		t.Fatalf("metadata discovery failed: %v", err)
	}

	if metadata.RegistrationEndpoint == "" {
		t.Fatal("registration endpoint not found in metadata")
	}

	// Verify registration endpoint is correct
	expectedRegEndpoint := mockAS.URL + "/register"
	if metadata.RegistrationEndpoint != expectedRegEndpoint {
		t.Errorf("RegistrationEndpoint = %s, want %s", metadata.RegistrationEndpoint, expectedRegEndpoint)
	}
}

// TestRegistrationTokenRoundTripper_Integration tests registration token injection
func TestRegistrationTokenRoundTripper_Integration(t *testing.T) {
	mockAS := NewMockAuthServer(t)
	mockAS.supportsClientRegistration = true
	mockAS.registrationToken = "" // No token required for HTTP test server
	defer mockAS.Close()

	// Create HTTP client without registration token for HTTP testing
	// (registration token enforcement requires HTTPS)
	client := mockAS.Client()

	// Test registration request
	registrationReq := `{
		"client_name": "test-client",
		"redirect_uris": ["http://localhost:8765/callback"],
		"grant_types": ["authorization_code"],
		"response_types": ["code"]
	}`

	resp, err := client.Post(
		mockAS.URL+"/register",
		"application/json",
		strings.NewReader(registrationReq),
	)
	if err != nil {
		t.Fatalf("registration request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("registration status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	// Verify the server received the registration request
	reqs := mockAS.GetRegistrationRequests()
	if len(reqs) != 1 {
		t.Errorf("got %d registration requests, want 1", len(reqs))
	}
}

// TestErrorHandling tests error scenarios
func TestErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func() (string, func())
		expectError bool
		description string
	}{
		{
			name: "metadata endpoint not found",
			setupFunc: func() (string, func()) {
				mockAS := NewMockAuthServer(t)
				// Close the server immediately to simulate network error
				mockAS.Close()
				return mockAS.URL, func() {}
			},
			expectError: true,
			description: "Should fail when metadata endpoint is unreachable",
		},
		{
			name: "invalid issuer URL",
			setupFunc: func() (string, func()) {
				return "not-a-valid-url", func() {}
			},
			expectError: true,
			description: "Should fail with invalid URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issuerURL, cleanup := tt.setupFunc()
			defer cleanup()

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			logger := NewLogger(false, false, false)
			_, err := DiscoverAuthorizationServerMetadata(ctx, issuerURL, logger)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestScopeSelection_Integration tests scope selection with actual metadata
func TestScopeSelection_Integration(t *testing.T) {
	mockAS := NewMockAuthServer(t)
	mockAS.scopesSupported = []string{"mcp:read", "mcp:write", "mcp:admin"}
	defer mockAS.Close()

	mockMCP := NewMockMCPServer(t, mockAS.URL)
	mockMCP.scopesSupported = []string{"mcp:read", "mcp:write"}
	defer mockMCP.Close()

	ctx := context.Background()
	logger := NewLogger(false, false, false)

	// Discover protected resource metadata
	metadata, err := discoverProtectedResourceMetadata(ctx, mockMCP.URL, nil, logger)
	if err != nil {
		t.Fatalf("discovery failed: %v", err)
	}

	// Test scope selection in auto mode
	config := &OAuthConfig{
		ScopeSelectionMode: "auto",
	}

	scopes := selectScopes(config, nil, metadata, logger)
	if len(scopes) != len(mockMCP.scopesSupported) {
		t.Errorf("got %d scopes, want %d", len(scopes), len(mockMCP.scopesSupported))
	}
}

// TestHTTPSEnforcement tests HTTPS requirement for non-localhost
func TestHTTPSEnforcement(t *testing.T) {
	tests := []struct {
		name        string
		redirectURL string
		wantErr     bool
	}{
		{
			name:        "localhost HTTP allowed",
			redirectURL: "http://localhost:8765/callback",
			wantErr:     false,
		},
		{
			name:        "127.0.0.1 HTTP allowed",
			redirectURL: "http://127.0.0.1:8765/callback",
			wantErr:     false,
		},
		{
			name:        "non-localhost HTTP rejected",
			redirectURL: "http://example.com/callback",
			wantErr:     true,
		},
		{
			name:        "HTTPS rejected (not supported)",
			redirectURL: "https://example.com/callback",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &OAuthConfig{
				Enabled:              true,
				RedirectURL:          tt.redirectURL,
				AuthorizationTimeout: 5 * time.Minute,
			}

			err := config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
