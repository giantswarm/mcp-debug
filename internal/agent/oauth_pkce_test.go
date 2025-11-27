package agent

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/mark3labs/mcp-go/client"
)

// TestGeneratePKCEChallenge tests PKCE code verifier and challenge generation
// Verifies RFC 7636 Section 4.1 (code verifier) and Section 4.2 (code challenge) requirements
func TestGeneratePKCEChallenge(t *testing.T) {
	tests := []struct {
		name string
		runs int // Run multiple times to test randomness
	}{
		{
			name: "generate unique verifiers",
			runs: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifiers := make(map[string]bool)
			challenges := make(map[string]bool)

			for i := 0; i < tt.runs; i++ {
				verifier, err := client.GenerateCodeVerifier()
				if err != nil {
					t.Fatalf("GenerateCodeVerifier() error = %v", err)
				}
				challenge := client.GenerateCodeChallenge(verifier)

				// Verify verifier requirements (RFC 7636)
				if len(verifier) < 43 || len(verifier) > 128 {
					t.Errorf("verifier length = %d, want between 43 and 128", len(verifier))
				}

				// Verify verifier contains only allowed characters
				for _, c := range verifier {
					if !isUnreservedChar(c) {
						t.Errorf("verifier contains invalid character: %c", c)
					}
				}

				// Verify challenge is base64url encoded
				if len(challenge) == 0 {
					t.Error("challenge is empty")
				}

				// Verify challenge is unique
				if challenges[challenge] {
					t.Error("generated duplicate challenge")
				}
				challenges[challenge] = true

				// Verify verifier is unique
				if verifiers[verifier] {
					t.Error("generated duplicate verifier")
				}
				verifiers[verifier] = true

				// Verify challenge is correct SHA256 of verifier
				expectedChallenge := computeS256Challenge(verifier)
				if challenge != expectedChallenge {
					t.Errorf("challenge mismatch: got %s, expected %s", challenge, expectedChallenge)
				}
			}
		})
	}
}

// computeS256Challenge computes the S256 code challenge from a verifier
func computeS256Challenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// isUnreservedChar checks if a character is in the unreserved character set
// as defined in RFC 3986 section 2.3: A-Z, a-z, 0-9, -, ., _, ~
func isUnreservedChar(c rune) bool {
	return (c >= 'A' && c <= 'Z') ||
		(c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') ||
		c == '-' || c == '.' || c == '_' || c == '~'
}

// TestGeneratePKCEChallengeLength tests verifier length requirements
func TestGeneratePKCEChallengeLength(t *testing.T) {
	for i := 0; i < 100; i++ {
		verifier, err := client.GenerateCodeVerifier()
		if err != nil {
			t.Fatalf("GenerateCodeVerifier() error = %v", err)
		}
		challenge := client.GenerateCodeChallenge(verifier)

		// Per RFC 7636: verifier must be 43-128 characters
		if len(verifier) < 43 {
			t.Errorf("verifier too short: %d characters, want at least 43", len(verifier))
		}
		if len(verifier) > 128 {
			t.Errorf("verifier too long: %d characters, want at most 128", len(verifier))
		}

		// Challenge should be base64url of SHA256 hash (32 bytes)
		// Base64url encoding of 32 bytes = 43 characters (without padding)
		if len(challenge) != 43 {
			t.Errorf("challenge length = %d, want 43 (base64url of SHA256)", len(challenge))
		}
	}
}

// TestPKCEChallengeVerification tests that challenge can be verified against verifier
func TestPKCEChallengeVerification(t *testing.T) {
	tests := []struct {
		name     string
		verifier string
		wantLen  int
	}{
		{
			name:     "minimum length verifier (43 chars)",
			verifier: "0123456789012345678901234567890123456789012", // 43 chars
			wantLen:  43,
		},
		{
			name:     "medium length verifier",
			verifier: "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~", // 66 chars
			wantLen:  43,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Compute challenge using S256 method
			hash := sha256.Sum256([]byte(tt.verifier))
			challenge := base64.RawURLEncoding.EncodeToString(hash[:])

			if len(challenge) != tt.wantLen {
				t.Errorf("challenge length = %d, want %d", len(challenge), tt.wantLen)
			}

			// Verify challenge doesn't contain padding
			if challenge[len(challenge)-1] == '=' {
				t.Error("challenge contains padding, want base64url without padding")
			}

			// Verify we can decode it
			decoded, err := base64.RawURLEncoding.DecodeString(challenge)
			if err != nil {
				t.Errorf("failed to decode challenge: %v", err)
			}

			if len(decoded) != 32 {
				t.Errorf("decoded challenge length = %d, want 32 (SHA256 hash size)", len(decoded))
			}
		})
	}
}

// TestPKCEIntegrationWithMockServer tests PKCE flow with mock authorization server
// Verifies RFC 7636 PKCE compliance:
// - Code verifier must be 43-128 characters (Section 4.1)
// - Code challenge must be base64url(SHA256(verifier)) for S256 (Section 4.2)
// - Server must validate code verifier against challenge (Section 4.6)
func TestPKCEIntegrationWithMockServer(t *testing.T) {
	// Create mock authorization server with PKCE enabled
	mockAS := NewMockAuthServer(t)
	mockAS.supportsPKCE = true
	mockAS.codeChallengeMethods = []string{"S256"}
	defer mockAS.Close()

	// Generate PKCE challenge
	verifier, err := client.GenerateCodeVerifier()
	if err != nil {
		t.Fatalf("GenerateCodeVerifier() error = %v", err)
	}
	challenge := client.GenerateCodeChallenge(verifier)

	// Test authorization request with PKCE
	clientID := "test-client"
	redirectURI := "http://localhost:8765/callback"
	state := "test-state"

	authURL := mockAS.URL + "/authorize" +
		"?response_type=code" +
		"&client_id=" + clientID +
		"&redirect_uri=" + redirectURI +
		"&state=" + state +
		"&code_challenge=" + challenge +
		"&code_challenge_method=S256" +
		"&resource=https://mcp.example.com/mcp"

	// Make authorization request without following redirects
	httpClient := mockAS.ClientWithoutRedirect()
	resp, err := httpClient.Get(authURL)
	if err != nil {
		t.Fatalf("authorization request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Verify we got a redirect
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected redirect (302), got %d", resp.StatusCode)
	}

	// Parse the redirect location to get the code
	location := resp.Header.Get("Location")
	if location == "" {
		t.Fatal("no redirect location in response")
	}

	redirectURL, err := url.Parse(location)
	if err != nil {
		t.Fatalf("failed to parse redirect URL: %v", err)
	}

	code := redirectURL.Query().Get("code")
	if code == "" {
		t.Fatal("no authorization code in redirect")
	}

	returnedState := redirectURL.Query().Get("state")
	if returnedState != state {
		t.Errorf("state mismatch: got %s, want %s", returnedState, state)
	}

	// Test token request with code verifier
	tokenResp, err := mockAS.Client().PostForm(mockAS.URL+"/token", url.Values{
		"grant_type":    []string{"authorization_code"},
		"code":          []string{code},
		"client_id":     []string{clientID},
		"code_verifier": []string{verifier},
		"resource":      []string{"https://mcp.example.com/mcp"},
	})
	if err != nil {
		t.Fatalf("token request failed: %v", err)
	}
	defer func() { _ = tokenResp.Body.Close() }()

	if tokenResp.StatusCode != http.StatusOK {
		t.Errorf("token request status = %d, want %d", tokenResp.StatusCode, http.StatusOK)
	}

	// Verify token response contains access_token
	var tokenData map[string]interface{}
	err = json.NewDecoder(tokenResp.Body).Decode(&tokenData)
	if err != nil {
		t.Fatalf("failed to decode token response: %v", err)
	}

	if tokenData["access_token"] == nil {
		t.Error("token response missing access_token")
	}

	// Verify the server stored the verifier for this token
	token := tokenData["access_token"].(string)
	mockAS.mu.Lock()
	tokenInfo, ok := mockAS.issuedTokens[token]
	mockAS.mu.Unlock()

	if !ok {
		t.Fatal("token not found in server's issued tokens")
	}

	if tokenInfo.CodeVerifier != verifier {
		t.Errorf("stored code verifier = %s, want %s", tokenInfo.CodeVerifier, verifier)
	}
}

// TestPKCERequiredByServer tests that server rejects requests without PKCE when required
func TestPKCERequiredByServer(t *testing.T) {
	mockAS := NewMockAuthServer(t)
	mockAS.supportsPKCE = true
	mockAS.codeChallengeMethods = []string{"S256"}
	defer mockAS.Close()

	// Try authorization without PKCE
	authURL := mockAS.URL + "/authorize" +
		"?response_type=code" +
		"&client_id=test-client" +
		"&redirect_uri=http://localhost:8765/callback" +
		"&resource=https://mcp.example.com/mcp"

	resp, err := mockAS.Client().Get(authURL)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Should get error response
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d (bad request for missing PKCE)", resp.StatusCode, http.StatusBadRequest)
	}
}

// TestPKCEMethodValidation tests that server validates code_challenge_method
func TestPKCEMethodValidation(t *testing.T) {
	mockAS := NewMockAuthServer(t)
	mockAS.supportsPKCE = true
	mockAS.codeChallengeMethods = []string{"S256"} // Only S256 supported
	defer mockAS.Close()

	tests := []struct {
		name         string
		method       string
		wantAccepted bool
	}{
		{
			name:         "S256 method accepted",
			method:       "S256",
			wantAccepted: true,
		},
		{
			name:         "plain method rejected",
			method:       "plain",
			wantAccepted: false,
		},
		{
			name:         "invalid method rejected",
			method:       "invalid",
			wantAccepted: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifier, err := client.GenerateCodeVerifier()
			if err != nil {
				t.Fatalf("GenerateCodeVerifier() error = %v", err)
			}
			challenge := client.GenerateCodeChallenge(verifier)

			authURL := mockAS.URL + "/authorize" +
				"?response_type=code" +
				"&client_id=test-client" +
				"&redirect_uri=http://localhost:8765/callback" +
				"&code_challenge=" + challenge +
				"&code_challenge_method=" + tt.method +
				"&resource=https://mcp.example.com/mcp"

			httpClient := mockAS.ClientWithoutRedirect()
			resp, err := httpClient.Get(authURL)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			isAccepted := resp.StatusCode == http.StatusFound
			if isAccepted != tt.wantAccepted {
				t.Errorf("method %s accepted = %v, want %v (status: %d)",
					tt.method, isAccepted, tt.wantAccepted, resp.StatusCode)
			}

			// If accepted, verify we can complete the flow
			if tt.wantAccepted {
				location := resp.Header.Get("Location")
				redirectURL, _ := url.Parse(location)
				code := redirectURL.Query().Get("code")

				// Exchange code for token
				tokenResp, err := mockAS.Client().PostForm(mockAS.URL+"/token", url.Values{
					"grant_type":    []string{"authorization_code"},
					"code":          []string{code},
					"client_id":     []string{"test-client"},
					"code_verifier": []string{verifier},
					"resource":      []string{"https://mcp.example.com/mcp"},
				})
				if err != nil {
					t.Fatalf("token request failed: %v", err)
				}
				defer func() { _ = tokenResp.Body.Close() }()

				if tokenResp.StatusCode != http.StatusOK {
					t.Errorf("token exchange failed with status %d", tokenResp.StatusCode)
				}
			}
		})
	}
}

// TestValidatePKCESupportFromMetadata tests PKCE support detection from AS metadata
func TestValidatePKCESupportFromMetadata(t *testing.T) {
	tests := []struct {
		name          string
		metadata      *AuthorizationServerMetadata
		wantSupported bool
		wantS256      bool
		wantErr       bool
	}{
		{
			name: "supports S256",
			metadata: &AuthorizationServerMetadata{
				CodeChallengeMethods: []string{"S256"},
			},
			wantSupported: true,
			wantS256:      true,
			wantErr:       false,
		},
		{
			name: "supports S256 and plain",
			metadata: &AuthorizationServerMetadata{
				CodeChallengeMethods: []string{"S256", "plain"},
			},
			wantSupported: true,
			wantS256:      true,
			wantErr:       false,
		},
		{
			name: "only supports plain (rejected)",
			metadata: &AuthorizationServerMetadata{
				CodeChallengeMethods: []string{"plain"},
			},
			wantSupported: false,
			wantS256:      false,
			wantErr:       true,
		},
		{
			name: "no PKCE support",
			metadata: &AuthorizationServerMetadata{
				CodeChallengeMethods: []string{},
			},
			wantSupported: false,
			wantS256:      false,
			wantErr:       true,
		},
		{
			name:          "nil methods array",
			metadata:      &AuthorizationServerMetadata{},
			wantSupported: false,
			wantS256:      false,
			wantErr:       true,
		},
	}

	logger := NewLogger(false, false, false)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePKCESupport(tt.metadata, false, logger)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			// Check if S256 is in the supported methods
			hasS256 := false
			for _, method := range tt.metadata.CodeChallengeMethods {
				if method == "S256" {
					hasS256 = true
					break
				}
			}

			if hasS256 != tt.wantS256 {
				t.Errorf("S256 support = %v, want %v", hasS256, tt.wantS256)
			}
		})
	}
}
