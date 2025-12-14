package agent

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

// Test timeout constants
const (
	testTimeoutShort  = 1 * time.Millisecond
	testTimeoutNormal = 1 * time.Second
	testTimeoutLong   = 5 * time.Second
	testDelayLong     = 100 * time.Millisecond
)

// OAuth response type constant
const responseTypeCode = "code"

// testEnv encapsulates a complete test environment with mock servers
type testEnv struct {
	AS      *MockAuthServer
	MCP     *MockMCPServer
	cleanup func()
}

// setupTestEnvironment creates a complete test environment with mock AS and MCP servers
func setupTestEnvironment(t *testing.T) *testEnv {
	t.Helper()

	mockAS := NewMockAuthServer(t)
	mockMCP := NewMockMCPServer(t, mockAS.URL)

	return &testEnv{
		AS:  mockAS,
		MCP: mockMCP,
		cleanup: func() {
			mockMCP.Close()
			mockAS.Close()
		},
	}
}

// MockAuthServer provides a mock OAuth 2.1 authorization server for testing
//
// SECURITY NOTE: This is a TEST-ONLY implementation.
// Production implementations must:
// - Use constant-time comparison (subtle.ConstantTimeCompare) for token validation
// - Never store code verifiers after validation (they should be immediately discarded)
// - Implement proper rate limiting and request validation
type MockAuthServer struct {
	*httptest.Server
	t *testing.T

	// Configuration
	supportsPKCE               bool
	supportsResourceIndicators bool
	supportsClientRegistration bool
	supportsClientIDMetadata   bool
	registrationToken          string
	scopesSupported            []string
	codeChallengeMethods       []string
	issuerURL                  string

	// State tracking
	mu                   sync.Mutex
	authorizedClients    map[string]bool   // client_id -> authorized
	issuedCodes          map[string]string // code -> client_id
	issuedTokens         map[string]TokenInfo
	registeredClients    map[string]ClientInfo
	requestCount         int
	authRequestCount     int
	tokenRequestCount    int
	registrationRequests []RegistrationRequest
}

// TokenInfo stores information about issued tokens
//
// SECURITY NOTE: The CodeVerifier field is stored here for test validation purposes only.
// In production OAuth servers, code verifiers MUST be discarded immediately after validation
// and should NEVER be stored alongside access tokens.
type TokenInfo struct {
	ClientID     string
	Scopes       []string
	Resource     string
	CodeVerifier string // For PKCE validation (TEST ONLY - never store in production)
}

// ClientInfo stores registered client information
type ClientInfo struct {
	ClientID     string
	ClientSecret string
	RedirectURIs []string
	Scopes       []string
}

// RegistrationRequest captures a client registration request
type RegistrationRequest struct {
	ClientName    string   `json:"client_name"`
	RedirectURIs  []string `json:"redirect_uris"`
	GrantTypes    []string `json:"grant_types"`
	ResponseTypes []string `json:"response_types"`
}

// NewMockAuthServer creates a new mock authorization server
func NewMockAuthServer(t *testing.T) *MockAuthServer {
	t.Helper()

	mas := &MockAuthServer{
		t:                          t,
		supportsPKCE:               true,
		supportsResourceIndicators: true,
		supportsClientRegistration: true,
		supportsClientIDMetadata:   false,
		scopesSupported:            []string{"mcp:read", "mcp:write", "mcp:admin"},
		codeChallengeMethods:       []string{"S256", "plain"},
		authorizedClients:          make(map[string]bool),
		issuedCodes:                make(map[string]string),
		issuedTokens:               make(map[string]TokenInfo),
		registeredClients:          make(map[string]ClientInfo),
		registrationRequests:       make([]RegistrationRequest, 0),
	}

	mux := http.NewServeMux()

	// AS Metadata Discovery Endpoints
	mux.HandleFunc("/.well-known/oauth-authorization-server", mas.handleASMetadata)
	mux.HandleFunc("/.well-known/openid-configuration", mas.handleASMetadata)

	// OAuth Endpoints
	mux.HandleFunc("/authorize", mas.handleAuthorize)
	mux.HandleFunc("/token", mas.handleToken)
	mux.HandleFunc("/register", mas.handleRegister)

	mas.Server = httptest.NewServer(mux)
	mas.issuerURL = mas.URL

	return mas
}

// handleASMetadata returns authorization server metadata
func (mas *MockAuthServer) handleASMetadata(w http.ResponseWriter, r *http.Request) {
	mas.mu.Lock()
	mas.requestCount++
	mas.mu.Unlock()

	metadata := &AuthorizationServerMetadata{
		Issuer:                            mas.issuerURL,
		AuthorizationEndpoint:             mas.issuerURL + "/authorize",
		TokenEndpoint:                     mas.issuerURL + "/token",
		ScopesSupported:                   mas.scopesSupported,
		ResponseTypesSupported:            []string{responseTypeCode},
		GrantTypesSupported:               []string{"authorization_code", "refresh_token"},
		CodeChallengeMethods:              mas.codeChallengeMethods,
		ClientIDMetadataDocumentSupported: mas.supportsClientIDMetadata,
	}

	if mas.supportsClientRegistration {
		metadata.RegistrationEndpoint = mas.issuerURL + "/register"
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(metadata)
}

// handleAuthorize handles authorization requests
func (mas *MockAuthServer) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	mas.mu.Lock()
	mas.requestCount++
	mas.authRequestCount++
	mas.mu.Unlock()

	query := r.URL.Query()

	// Validate required parameters
	clientID := query.Get("client_id")
	redirectURI := query.Get("redirect_uri")
	responseType := query.Get("response_type")
	state := query.Get("state")
	codeChallenge := query.Get("code_challenge")
	codeChallengeMethod := query.Get("code_challenge_method")
	resource := query.Get("resource")

	if clientID == "" || redirectURI == "" || responseType != responseTypeCode {
		http.Error(w, "invalid_request", http.StatusBadRequest)
		return
	}

	// Validate PKCE if supported
	if mas.supportsPKCE && codeChallenge == "" {
		http.Error(w, "code_challenge_required", http.StatusBadRequest)
		return
	}

	if mas.supportsPKCE && codeChallenge != "" && codeChallengeMethod != "S256" {
		http.Error(w, "invalid_code_challenge_method", http.StatusBadRequest)
		return
	}

	// Validate resource parameter if required
	if mas.supportsResourceIndicators && resource == "" {
		http.Error(w, "missing_resource_parameter", http.StatusBadRequest)
		return
	}

	// Generate authorization code
	code := fmt.Sprintf("AUTH_CODE_%d", mas.authRequestCount)
	mas.mu.Lock()
	mas.issuedCodes[code] = clientID
	mas.mu.Unlock()

	// Redirect back with code
	redirectURL, err := url.Parse(redirectURI)
	if err != nil {
		http.Error(w, "invalid_redirect_uri", http.StatusBadRequest)
		return
	}

	params := url.Values{}
	params.Set("code", code)
	if state != "" {
		params.Set("state", state)
	}
	redirectURL.RawQuery = params.Encode()

	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}

// handleToken handles token requests
func (mas *MockAuthServer) handleToken(w http.ResponseWriter, r *http.Request) {
	mas.mu.Lock()
	mas.requestCount++
	mas.tokenRequestCount++
	mas.mu.Unlock()

	if r.Method != http.MethodPost {
		http.Error(w, "method_not_allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "invalid_request", http.StatusBadRequest)
		return
	}

	grantType := r.Form.Get("grant_type")
	code := r.Form.Get("code")
	clientID := r.Form.Get("client_id")
	codeVerifier := r.Form.Get("code_verifier")
	resource := r.Form.Get("resource")

	// Validate grant type
	if grantType != "authorization_code" {
		http.Error(w, "unsupported_grant_type", http.StatusBadRequest)
		return
	}

	// Validate code
	mas.mu.Lock()
	storedClientID, codeValid := mas.issuedCodes[code]
	mas.mu.Unlock()

	if !codeValid || storedClientID != clientID {
		http.Error(w, "invalid_grant", http.StatusBadRequest)
		return
	}

	// Validate PKCE
	if mas.supportsPKCE && codeVerifier == "" {
		http.Error(w, "code_verifier_required", http.StatusBadRequest)
		return
	}

	// Validate resource parameter if required
	if mas.supportsResourceIndicators && resource == "" {
		http.Error(w, "missing_resource_parameter", http.StatusBadRequest)
		return
	}

	// Generate access token
	accessToken := fmt.Sprintf("ACCESS_TOKEN_%d", mas.tokenRequestCount)
	mas.mu.Lock()
	mas.issuedTokens[accessToken] = TokenInfo{
		ClientID:     clientID,
		Scopes:       strings.Fields(r.Form.Get("scope")),
		Resource:     resource,
		CodeVerifier: codeVerifier,
	}
	// Invalidate the code after use
	delete(mas.issuedCodes, code)
	mas.mu.Unlock()

	response := map[string]interface{}{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   3600,
		"scope":        r.Form.Get("scope"),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// handleRegister handles dynamic client registration
func (mas *MockAuthServer) handleRegister(w http.ResponseWriter, r *http.Request) {
	mas.mu.Lock()
	mas.requestCount++
	mas.mu.Unlock()

	if !mas.supportsClientRegistration {
		http.Error(w, "not_supported", http.StatusNotFound)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "method_not_allowed", http.StatusMethodNotAllowed)
		return
	}

	// Validate registration token if required
	// SECURITY NOTE: This uses simple string comparison for testing only.
	// Production implementations MUST use subtle.ConstantTimeCompare() to prevent timing attacks.
	if mas.registrationToken != "" {
		authHeader := r.Header.Get("Authorization")
		expectedAuth := "Bearer " + mas.registrationToken
		if authHeader != expectedAuth {
			http.Error(w, "invalid_token", http.StatusUnauthorized)
			return
		}
	}

	var req RegistrationRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "invalid_request", http.StatusBadRequest)
		return
	}

	mas.mu.Lock()
	mas.registrationRequests = append(mas.registrationRequests, req)
	mas.mu.Unlock()

	// Generate client credentials
	clientID := fmt.Sprintf("registered_client_%d", len(mas.registrationRequests))
	clientSecret := fmt.Sprintf("secret_%d", len(mas.registrationRequests))

	clientInfo := ClientInfo{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURIs: req.RedirectURIs,
		Scopes:       mas.scopesSupported,
	}

	mas.mu.Lock()
	mas.registeredClients[clientID] = clientInfo
	mas.mu.Unlock()

	response := map[string]interface{}{
		"client_id":      clientID,
		"client_secret":  clientSecret,
		"redirect_uris":  req.RedirectURIs,
		"grant_types":    req.GrantTypes,
		"response_types": req.ResponseTypes,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(response)
}

// GetRequestCount returns the total number of requests received
func (mas *MockAuthServer) GetRequestCount() int {
	mas.mu.Lock()
	defer mas.mu.Unlock()
	return mas.requestCount
}

// GetRegistrationRequests returns all registration requests received
func (mas *MockAuthServer) GetRegistrationRequests() []RegistrationRequest {
	mas.mu.Lock()
	defer mas.mu.Unlock()
	return append([]RegistrationRequest{}, mas.registrationRequests...)
}

// ClientWithoutRedirect returns an HTTP client that doesn't follow redirects
func (mas *MockAuthServer) ClientWithoutRedirect() *http.Client {
	client := mas.Client()
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return client
}

// MockMCPServer provides a mock MCP server for testing
type MockMCPServer struct {
	*httptest.Server
	t *testing.T

	// Configuration
	requireAuth             bool
	requiredScopes          []string
	authorizationServers    []string
	scopesSupported         []string
	returnInsufficientScope bool // For testing step-up
	validTokens             map[string]bool

	// State tracking
	mu           sync.Mutex
	requestCount int
	requests     []*http.Request
}

// NewMockMCPServer creates a new mock MCP server
func NewMockMCPServer(t *testing.T, authServerURL string) *MockMCPServer {
	t.Helper()

	mms := &MockMCPServer{
		t:               t,
		requireAuth:     true,
		requiredScopes:  []string{"mcp:read"},
		scopesSupported: []string{"mcp:read", "mcp:write"},
		validTokens:     make(map[string]bool),
		requests:        make([]*http.Request, 0),
	}

	if authServerURL != "" {
		mms.authorizationServers = []string{authServerURL}
	}

	mux := http.NewServeMux()

	// Protected Resource Metadata Discovery
	mux.HandleFunc("/.well-known/oauth-protected-resource", mms.handleResourceMetadata)

	// MCP Endpoints
	mux.HandleFunc("/", mms.handleRequest)

	mms.Server = httptest.NewServer(mux)
	return mms
}

// handleResourceMetadata returns protected resource metadata
func (mms *MockMCPServer) handleResourceMetadata(w http.ResponseWriter, r *http.Request) {
	mms.mu.Lock()
	mms.requestCount++
	mms.mu.Unlock()

	metadata := &ProtectedResourceMetadata{
		Resource:               mms.URL,
		AuthorizationServers:   mms.authorizationServers,
		ScopesSupported:        mms.scopesSupported,
		BearerMethodsSupported: []string{"header"},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(metadata)
}

// handleRequest handles MCP requests with auth validation
func (mms *MockMCPServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	mms.mu.Lock()
	mms.requestCount++
	mms.requests = append(mms.requests, r)
	mms.mu.Unlock()

	// Return 401 if auth is required and no token provided
	if mms.requireAuth {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.Header().Set("WWW-Authenticate", fmt.Sprintf(
				`Bearer resource_metadata="%s/.well-known/oauth-protected-resource", scope="%s"`,
				mms.URL,
				strings.Join(mms.requiredScopes, " "),
			))
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "unauthorized",
			})
			return
		}

		// Validate token
		token := strings.TrimPrefix(authHeader, "Bearer ")
		mms.mu.Lock()
		valid := mms.validTokens[token]
		mms.mu.Unlock()

		if !valid && !mms.returnInsufficientScope {
			w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "invalid_token",
			})
			return
		}

		// Return 403 for insufficient scope if configured
		if mms.returnInsufficientScope {
			w.Header().Set("WWW-Authenticate", fmt.Sprintf(
				`Bearer error="insufficient_scope", scope="%s"`,
				strings.Join(append(mms.requiredScopes, "mcp:write"), " "),
			))
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error":             "insufficient_scope",
				"error_description": "Additional permissions required",
			})
			return
		}
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"jsonrpc": "2.0",
		"result":  map[string]string{"status": "ok"},
		"id":      1,
	})
}

// AddValidToken registers a token as valid
func (mms *MockMCPServer) AddValidToken(token string) {
	mms.mu.Lock()
	defer mms.mu.Unlock()
	mms.validTokens[token] = true
}

// GetRequestCount returns the total number of requests received
func (mms *MockMCPServer) GetRequestCount() int {
	mms.mu.Lock()
	defer mms.mu.Unlock()
	return mms.requestCount
}

// GetRequests returns all requests received
func (mms *MockMCPServer) GetRequests() []*http.Request {
	mms.mu.Lock()
	defer mms.mu.Unlock()
	return append([]*http.Request{}, mms.requests...)
}
