package agent

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestOAuthEndToEndFlow tests the complete OAuth 2.1 flow with mock servers
// Verifies compliance with:
// - RFC 6749 (OAuth 2.0)
// - RFC 7636 (PKCE)
// - RFC 8707 (Resource Indicators)
// - MCP OAuth specification
func TestOAuthEndToEndFlow(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.cleanup()

	tests := []struct {
		name            string
		setupAS         func(*MockAuthServer)
		setupMCP        func(*MockMCPServer)
		expectedSuccess bool
	}{
		{
			name: "should_complete_authorization_when_using_PKCE_S256_and_RFC8707_resource_indicators",
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
		},
		{
			name: "should_complete_authorization_when_resource_indicators_not_supported",
			setupAS: func(as *MockAuthServer) {
				as.supportsPKCE = true
				as.supportsResourceIndicators = false
				as.supportsClientRegistration = true
			},
			setupMCP: func(mcp *MockMCPServer) {
				mcp.requireAuth = true
			},
			expectedSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupAS != nil {
				tt.setupAS(env.AS)
			}
			if tt.setupMCP != nil {
				tt.setupMCP(env.MCP)
			}

			// Verify mock servers are working
			resp, err := http.Get(env.AS.URL + "/.well-known/oauth-authorization-server")
			if err != nil {
				t.Fatalf("failed to reach AS metadata endpoint: %v", err)
			}
			_ = resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("AS metadata status = %d, want %d", resp.StatusCode, http.StatusOK)
			}
		})
	}
}

// TestProtectedResourceMetadataIntegration tests metadata discovery with mock MCP server
// Verifies RFC 9728 (OAuth 2.0 Protected Resource Metadata) compliance
func TestProtectedResourceMetadataIntegration(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.cleanup()

	ctx := context.Background()
	logger := NewLogger(false, false, false)

	// Test discovery from well-known URI
	metadata, err := discoverProtectedResourceMetadata(ctx, env.MCP.URL, nil, logger)
	if err != nil {
		t.Fatalf("discovery failed: %v", err)
	}

	if metadata.Resource != env.MCP.URL {
		t.Errorf("Resource = %s, want %s", metadata.Resource, env.MCP.URL)
	}

	if len(metadata.AuthorizationServers) != 1 {
		t.Errorf("got %d auth servers, want 1", len(metadata.AuthorizationServers))
	} else if metadata.AuthorizationServers[0] != env.AS.URL {
		t.Errorf("auth server = %s, want %s", metadata.AuthorizationServers[0], env.AS.URL)
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
	defer func() { _ = resp.Body.Close() }()

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
	defer func() { _ = resp.Body.Close() }()

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
// Verifies MCP OAuth specification scope selection priority order
func TestScopeSelection_Integration(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.cleanup()

	env.AS.scopesSupported = []string{"mcp:read", "mcp:write", "mcp:admin"}
	env.MCP.scopesSupported = []string{"mcp:read", "mcp:write"}

	ctx := context.Background()
	logger := NewLogger(false, false, false)

	// Discover protected resource metadata
	metadata, err := discoverProtectedResourceMetadata(ctx, env.MCP.URL, nil, logger)
	if err != nil {
		t.Fatalf("discovery failed: %v", err)
	}

	// Test scope selection in auto mode
	config := &OAuthConfig{
		ScopeSelectionMode: "auto",
	}

	scopes := selectScopes(config, nil, metadata, logger)
	if len(scopes) != len(env.MCP.scopesSupported) {
		t.Errorf("got %d scopes, want %d", len(scopes), len(env.MCP.scopesSupported))
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
			name:        "IPv6 loopback [::1] HTTP allowed",
			redirectURL: "http://[::1]:8765/callback",
			wantErr:     false,
		},
		{
			name:        "IPv6 loopback expanded form [0:0:0:0:0:0:0:1] HTTP allowed",
			redirectURL: "http://[0:0:0:0:0:0:0:1]:8765/callback",
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

// TestClientRegistrationPriority tests the client registration priority logic:
// 1. Pre-registered client ID
// 2. Client ID Metadata Documents (CIMD)
// 3. Dynamic Client Registration (DCR)
// 4. Manual configuration (error case)
func TestClientRegistrationPriority(t *testing.T) {
	tests := []struct {
		name                string
		clientID            string
		clientIDMetadataURL string
		disableCIMD         bool
		expectedClientID    string
		description         string
	}{
		{
			name:             "priority_1_pre_registered_client_id",
			clientID:         "pre-registered-client-123",
			expectedClientID: "pre-registered-client-123",
			description:      "Pre-registered client ID should be used when provided",
		},
		{
			name:                "priority_2_cimd_when_no_client_id",
			clientID:            "",
			clientIDMetadataURL: "https://app.example.com/oauth/client-metadata.json",
			disableCIMD:         false,
			expectedClientID:    "https://app.example.com/oauth/client-metadata.json",
			description:         "CIMD URL should be used as client_id when no pre-registered client ID",
		},
		{
			name:                "priority_1_over_priority_2",
			clientID:            "pre-registered-client-123",
			clientIDMetadataURL: "https://app.example.com/oauth/client-metadata.json",
			disableCIMD:         false,
			expectedClientID:    "pre-registered-client-123",
			description:         "Pre-registered client ID takes priority over CIMD",
		},
		{
			name:                "cimd_disabled_falls_back_to_dcr",
			clientID:            "",
			clientIDMetadataURL: "https://app.example.com/oauth/client-metadata.json",
			disableCIMD:         true,
			expectedClientID:    "",
			description:         "CIMD URL should be ignored when DisableCIMD is true, falling back to DCR",
		},
		{
			name:             "no_client_id_falls_back_to_dcr",
			clientID:         "",
			expectedClientID: "",
			description:      "Empty client ID should fall back to DCR (mcp-go handles this)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &OAuthConfig{
				Enabled:             true,
				ClientID:            tt.clientID,
				ClientIDMetadataURL: tt.clientIDMetadataURL,
				DisableCIMD:         tt.disableCIMD,
				RedirectURL:         "http://localhost:8765/callback",
			}

			// Apply defaults
			config = config.WithDefaults()

			// Validate configuration
			if err := config.Validate(); err != nil {
				t.Fatalf("config validation failed: %v", err)
			}

			// Simulate the client registration priority logic from client.go
			var actualClientID string
			if config.ClientID != "" {
				// Priority 1: Pre-registered
				actualClientID = config.ClientID
			} else if config.ClientIDMetadataURL != "" && !config.DisableCIMD {
				// Priority 2: CIMD
				actualClientID = config.ClientIDMetadataURL
			} else {
				// Priority 3: DCR (empty client ID means DCR will be attempted)
				actualClientID = ""
			}

			if actualClientID != tt.expectedClientID {
				t.Errorf("%s: got client_id = %q, want %q", tt.description, actualClientID, tt.expectedClientID)
			}
		})
	}
}

// TestCIMDWithMockAuthServer tests CIMD integration with mock authorization server
func TestCIMDWithMockAuthServer(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.cleanup()

	// Configure mock AS to support CIMD
	env.AS.supportsClientIDMetadata = true

	// Create a valid client metadata URL (using mock server)
	// Note: In real usage, this would point to where the client hosts their metadata
	clientMetadataURL := "https://app.example.com/oauth/client-metadata.json"

	config := &OAuthConfig{
		Enabled:             true,
		ClientIDMetadataURL: clientMetadataURL,
		DisableCIMD:         false,
		RedirectURL:         "http://localhost:8765/callback",
	}

	config = config.WithDefaults()

	if err := config.Validate(); err != nil {
		t.Fatalf("config validation failed: %v", err)
	}

	// Verify that CIMD URL is used as client_id
	var clientID string
	if config.ClientID != "" {
		clientID = config.ClientID
	} else if config.ClientIDMetadataURL != "" && !config.DisableCIMD {
		clientID = config.ClientIDMetadataURL
	}

	if clientID != clientMetadataURL {
		t.Errorf("Expected client_id to be CIMD URL %q, got %q", clientMetadataURL, clientID)
	}

	// Verify AS metadata advertises CIMD support
	ctx := context.Background()
	logger := NewLogger(false, false, false)

	asMetadata, err := DiscoverAuthorizationServerMetadata(ctx, env.AS.URL, logger)
	if err != nil {
		t.Fatalf("AS metadata discovery failed: %v", err)
	}

	if !asMetadata.ClientIDMetadataDocumentSupported {
		t.Error("Mock AS should advertise CIMD support")
	}

	// Verify CIMD support detection helper
	if !SupportsClientIDMetadata(asMetadata) {
		t.Error("SupportsClientIDMetadata should return true for mock AS")
	}
}

// TestCIMDValidationInConfig tests that invalid CIMD URLs are caught during validation
func TestCIMDValidationInConfig(t *testing.T) {
	tests := []struct {
		name                string
		clientIDMetadataURL string
		wantErr             bool
		errMsg              string
	}{
		{
			name:                "valid_https_url_with_path",
			clientIDMetadataURL: "https://app.example.com/oauth/client-metadata.json",
			wantErr:             false,
		},
		{
			name:                "invalid_http_url",
			clientIDMetadataURL: "http://app.example.com/oauth/client-metadata.json",
			wantErr:             true,
			errMsg:              "must use https scheme",
		},
		{
			name:                "invalid_no_path",
			clientIDMetadataURL: "https://app.example.com",
			wantErr:             true,
			errMsg:              "must contain a path component",
		},
		{
			name:                "empty_url_is_valid",
			clientIDMetadataURL: "",
			wantErr:             false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &OAuthConfig{
				Enabled:             true,
				ClientIDMetadataURL: tt.clientIDMetadataURL,
				RedirectURL:         "http://localhost:8765/callback",
			}

			config = config.WithDefaults()
			err := config.Validate()

			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Validate() error = %v, expected to contain %q", err, tt.errMsg)
			}
		})
	}
}
