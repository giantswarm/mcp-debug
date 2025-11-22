package agent

import (
	"fmt"
	"net/url"
)

// OAuthConfig contains OAuth 2.1 configuration for authenticating with MCP servers
type OAuthConfig struct {
	// Enabled indicates whether OAuth authentication should be used
	Enabled bool

	// ClientID is the OAuth client identifier (optional - will use DCR if not provided)
	ClientID string

	// ClientSecret is the OAuth client secret (optional for public clients)
	ClientSecret string

	// Scopes are the OAuth scopes to request (default: mcp:tools, mcp:resources)
	Scopes []string

	// RedirectURL is the callback URL for OAuth flow (default: http://localhost:8765/callback)
	RedirectURL string

	// UsePKCE enables Proof Key for Code Exchange (recommended, enabled by default)
	UsePKCE bool
}

// DefaultOAuthConfig returns a default OAuth configuration
func DefaultOAuthConfig() *OAuthConfig {
	return &OAuthConfig{
		Enabled:     false,
		Scopes:      []string{"mcp:tools", "mcp:resources"},
		RedirectURL: "http://localhost:8765/callback",
		UsePKCE:     true,
	}
}

// Validate checks if the OAuth configuration is valid
func (c *OAuthConfig) Validate() error {
	if !c.Enabled {
		return nil
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
	if parsedURL.Scheme == "http" {
		hostname := parsedURL.Hostname()
		// Note: Hostname() strips brackets from IPv6 addresses, so [::1] becomes ::1
		if hostname != "localhost" && hostname != "127.0.0.1" && hostname != "::1" {
			return fmt.Errorf("HTTP redirect URIs are only allowed for localhost/127.0.0.1/[::1], use HTTPS for other hosts")
		}
	} else if parsedURL.Scheme != "https" {
		return fmt.Errorf("redirect URI scheme must be http (localhost only) or https, got: %s", parsedURL.Scheme)
	}

	// Set default scopes if none provided
	if len(c.Scopes) == 0 {
		c.Scopes = []string{"mcp:tools", "mcp:resources"}
	}

	return nil
}
