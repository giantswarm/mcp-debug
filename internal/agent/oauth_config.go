package agent

import (
	"fmt"
	"net/url"
	"time"
)

// OAuthConfig contains OAuth 2.1 configuration for authenticating with MCP servers
type OAuthConfig struct {
	// Enabled indicates whether OAuth authentication should be used
	Enabled bool

	// ClientID is the OAuth client identifier (optional - will use DCR if not provided)
	ClientID string

	// ClientSecret is the OAuth client secret (optional for public clients)
	ClientSecret string

	// Scopes are the OAuth scopes to request (optional, no default scopes)
	// When ScopeSelectionMode is "manual", these scopes are used directly.
	// When ScopeSelectionMode is "auto", these scopes are used as a fallback
	// if no scopes can be determined from WWW-Authenticate or resource metadata.
	Scopes []string

	// ScopeSelectionMode controls automatic scope selection per MCP spec
	// - "auto" (default): Follow MCP spec priority (secure by default)
	//   Priority 1: Use scope from WWW-Authenticate header
	//   Priority 2: Use scopes_supported from Protected Resource Metadata
	//   Priority 3: Omit scope parameter entirely
	// - "manual": Use only the Scopes field value
	ScopeSelectionMode string

	// RedirectURL is the callback URL for OAuth flow (default: http://localhost:8765/callback)
	//
	// IMPORTANT SECURITY LIMITATION:
	// - HTTPS redirect URIs are NOT currently supported
	// - HTTP is only allowed for localhost/127.0.0.1/[::1] addresses
	// - The callback server runs on localhost only with HTTP
	// - For production OAuth flows requiring HTTPS callbacks, this is a known limitation
	//
	// This design follows OAuth 2.1 best practices for native/desktop applications
	// which use localhost loopback for authorization callbacks.
	RedirectURL string

	// UsePKCE enables Proof Key for Code Exchange (recommended, enabled by default)
	UsePKCE bool

	// AuthorizationTimeout is the maximum time to wait for user authorization (default: 5 minutes)
	AuthorizationTimeout time.Duration

	// UseOIDC enables OpenID Connect features including nonce validation (optional)
	UseOIDC bool

	// RegistrationToken is the OAuth registration access token for Dynamic Client Registration
	// Required if the authorization server has DCR authentication enabled
	RegistrationToken string

	// ResourceURI is the target MCP server URI for RFC 8707 Resource Indicators
	// If empty, automatically derived from endpoint URL
	// This parameter is included in authorization and token requests for better security
	ResourceURI string

	// SkipResourceParam disables RFC 8707 resource parameter inclusion
	// Only use this for testing with older servers that don't support RFC 8707
	// Security: Disabling this weakens token audience binding
	SkipResourceParam bool

	// SkipResourceMetadata disables RFC 9728 Protected Resource Metadata discovery
	// Only use this for testing with older servers that don't support RFC 9728
	// Security: Disabling this may require manual authorization server configuration
	SkipResourceMetadata bool

	// PreferredAuthServer allows selecting a specific authorization server
	// when multiple servers are available in the protected resource metadata
	// If empty, the first server in the list is used
	PreferredAuthServer string

	// SkipPKCEValidation disables checking for PKCE support in AS metadata
	// DANGEROUS: Only use for testing servers that support PKCE but don't advertise it
	// Security: This weakens security by allowing connections to servers without PKCE
	SkipPKCEValidation bool

	// SkipAuthServerDiscovery disables automatic AS metadata discovery
	// Useful for testing with pre-configured endpoints or older servers
	// When enabled, relies on mcp-go's internal discovery mechanisms
	SkipAuthServerDiscovery bool

	// EnableStepUpAuth enables automatic step-up authorization when the server
	// returns 403 Forbidden with insufficient_scope error per MCP spec
	// Default: true (secure by default, provides better UX)
	EnableStepUpAuth bool

	// StepUpMaxRetries limits the number of retry attempts per resource/operation
	// when handling insufficient_scope errors during step-up authorization
	// Default: 2 (prevents infinite authorization loops)
	StepUpMaxRetries int

	// StepUpUserPrompt asks user before requesting additional scopes
	// When false, step-up authorization happens automatically
	// Default: false (automatic for better UX, user can always decline in browser)
	StepUpUserPrompt bool
}

// DefaultOAuthConfig returns a default OAuth configuration
func DefaultOAuthConfig() *OAuthConfig {
	return &OAuthConfig{
		Enabled:              false,
		Scopes:               []string{},
		ScopeSelectionMode:   "auto", // Secure by default: follow MCP spec
		RedirectURL:          "http://localhost:8765/callback",
		UsePKCE:              true,
		AuthorizationTimeout: 5 * time.Minute,
		UseOIDC:              false, // OIDC features disabled by default
		EnableStepUpAuth:     true,  // Secure by default: handle insufficient_scope automatically
		StepUpMaxRetries:     2,     // Prevent infinite authorization loops
		StepUpUserPrompt:     false, // Automatic for better UX
	}
}

// WithDefaults returns a new config with defaults applied for any unset fields
func (c *OAuthConfig) WithDefaults() *OAuthConfig {
	config := *c

	// Set default timeout if not provided
	if config.AuthorizationTimeout == 0 {
		config.AuthorizationTimeout = 5 * time.Minute
	}

	// Set default redirect URL if not provided
	if config.RedirectURL == "" {
		config.RedirectURL = "http://localhost:8765/callback"
	}

	// Set default scope selection mode if not provided
	if config.ScopeSelectionMode == "" {
		config.ScopeSelectionMode = "auto"
	}

	// Set default step-up max retries if not provided
	// Note: EnableStepUpAuth defaults to false (zero value)
	// Callers should explicitly set it to true if desired
	if config.StepUpMaxRetries == 0 {
		config.StepUpMaxRetries = 2
	}

	return &config
}

// Validate checks if the OAuth configuration is valid (read-only, does not modify config)
func (c *OAuthConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	// Validate scope selection mode
	if c.ScopeSelectionMode != "" && c.ScopeSelectionMode != "auto" && c.ScopeSelectionMode != "manual" {
		return fmt.Errorf("invalid scope selection mode: %s (must be 'auto' or 'manual')", c.ScopeSelectionMode)
	}

	// RedirectURL is required for the callback server
	if c.RedirectURL == "" {
		return fmt.Errorf("OAuth redirect URL is required")
	}

	// Validate redirect URL and ensure HTTP is only used for localhost
	parsedURL, err := url.Parse(c.RedirectURL)
	if err != nil {
		return fmt.Errorf("invalid OAuth redirect URL: %w", err)
	}

	// Security: Only allow HTTP for localhost/loopback addresses
	// Note: HTTPS redirect URIs are not supported for the callback server (which only runs on localhost)
	if parsedURL.Scheme == "http" {
		hostname := parsedURL.Hostname()
		// Note: Hostname() strips brackets from IPv6 addresses, so [::1] becomes ::1
		// Accept various forms of localhost: localhost, 127.0.0.1, ::1, and expanded IPv6 0:0:0:0:0:0:0:1
		isLocalhost := hostname == "localhost" ||
			hostname == "127.0.0.1" ||
			hostname == "::1" ||
			hostname == "0:0:0:0:0:0:0:1"
		if !isLocalhost {
			return fmt.Errorf("HTTP redirect URIs are only allowed for localhost/127.0.0.1/[::1], got: %s", hostname)
		}
	} else if parsedURL.Scheme == "https" {
		return fmt.Errorf("HTTPS redirect URIs are not supported - callback server only runs on localhost with HTTP (use http://localhost:PORT/callback)")
	} else {
		return fmt.Errorf("redirect URI scheme must be http, got: %s (only http://localhost:PORT/callback is supported)", parsedURL.Scheme)
	}

	// Validate timeout is set (after defaults have been applied)
	if c.AuthorizationTimeout == 0 {
		return fmt.Errorf("OAuth authorization timeout is required")
	}

	return nil
}
