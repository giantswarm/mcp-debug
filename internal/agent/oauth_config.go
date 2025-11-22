package agent

import "fmt"

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

	// Set default scopes if none provided
	if len(c.Scopes) == 0 {
		c.Scopes = []string{"mcp:tools", "mcp:resources"}
	}

	return nil
}
