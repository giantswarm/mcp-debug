package agent

import (
	"strings"
	"testing"
	"time"
)

func TestOAuthConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *OAuthConfig
		wantErr bool
	}{
		{
			name: "valid config with client ID",
			config: &OAuthConfig{
				Enabled:              true,
				ClientID:             "test-client-id",
				RedirectURL:          "http://localhost:8765/callback",
				Scopes:               []string{"mcp:tools"},
				AuthorizationTimeout: 5 * time.Minute,
			},
			wantErr: false,
		},
		{
			name: "valid config without client ID (DCR)",
			config: &OAuthConfig{
				Enabled:              true,
				RedirectURL:          "http://localhost:8765/callback",
				Scopes:               []string{"mcp:tools"},
				AuthorizationTimeout: 5 * time.Minute,
			},
			wantErr: false,
		},
		{
			name: "disabled config",
			config: &OAuthConfig{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "missing redirect URL",
			config: &OAuthConfig{
				Enabled: true,
			},
			wantErr: true,
		},
		{
			name: "empty scopes are allowed",
			config: &OAuthConfig{
				Enabled:              true,
				RedirectURL:          "http://localhost:8765/callback",
				Scopes:               []string{},
				AuthorizationTimeout: 5 * time.Minute,
			},
			wantErr: false,
		},
		{
			name: "http redirect for localhost is allowed",
			config: &OAuthConfig{
				Enabled:              true,
				RedirectURL:          "http://localhost:8765/callback",
				Scopes:               []string{"mcp:tools"},
				AuthorizationTimeout: 5 * time.Minute,
			},
			wantErr: false,
		},
		{
			name: "http redirect for 127.0.0.1 is allowed",
			config: &OAuthConfig{
				Enabled:              true,
				RedirectURL:          "http://127.0.0.1:8765/callback",
				Scopes:               []string{"mcp:tools"},
				AuthorizationTimeout: 5 * time.Minute,
			},
			wantErr: false,
		},
		{
			name: "http redirect for IPv6 loopback is allowed",
			config: &OAuthConfig{
				Enabled:              true,
				RedirectURL:          "http://[::1]:8765/callback",
				Scopes:               []string{"mcp:tools"},
				AuthorizationTimeout: 5 * time.Minute,
			},
			wantErr: false,
		},
		{
			name: "http redirect for non-localhost is rejected",
			config: &OAuthConfig{
				Enabled:     true,
				RedirectURL: "http://example.com/callback",
				Scopes:      []string{"mcp:tools"},
			},
			wantErr: true,
		},
		{
			name: "https redirect is rejected (not supported)",
			config: &OAuthConfig{
				Enabled:     true,
				RedirectURL: "https://example.com/callback",
				Scopes:      []string{"mcp:tools"},
			},
			wantErr: true,
		},
		{
			name: "https redirect for localhost is also rejected",
			config: &OAuthConfig{
				Enabled:     true,
				RedirectURL: "https://localhost:8765/callback",
				Scopes:      []string{"mcp:tools"},
			},
			wantErr: true,
		},
		{
			name: "invalid redirect URL scheme",
			config: &OAuthConfig{
				Enabled:     true,
				RedirectURL: "ftp://localhost/callback",
				Scopes:      []string{"mcp:tools"},
			},
			wantErr: true,
		},
		{
			name: "malformed redirect URL",
			config: &OAuthConfig{
				Enabled:     true,
				RedirectURL: "://invalid",
				Scopes:      []string{"mcp:tools"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultOAuthConfig(t *testing.T) {
	config := DefaultOAuthConfig()

	if config.Enabled {
		t.Error("Default config should have Enabled = false")
	}

	if config.RedirectURL == "" {
		t.Error("Default config should have a redirect URL")
	}

	if !config.UsePKCE {
		t.Error("Default config should have UsePKCE = true")
	}

	if len(config.Scopes) != 0 {
		t.Error("Default config should have no scopes")
	}

	if config.UseOIDC {
		t.Error("Default config should have UseOIDC = false")
	}
}

func TestOAuthConfig_WithDefaults(t *testing.T) {
	tests := []struct {
		name     string
		config   *OAuthConfig
		expected *OAuthConfig
	}{
		{
			name: "empty config gets all defaults",
			config: &OAuthConfig{
				Enabled: true,
			},
			expected: &OAuthConfig{
				Enabled:              true,
				Scopes:               []string{},
				RedirectURL:          "http://localhost:8765/callback",
				AuthorizationTimeout: 5 * time.Minute,
			},
		},
		{
			name: "partial config keeps custom values",
			config: &OAuthConfig{
				Enabled:     true,
				ClientID:    "custom-client",
				RedirectURL: "http://localhost:9999/callback",
			},
			expected: &OAuthConfig{
				Enabled:              true,
				ClientID:             "custom-client",
				RedirectURL:          "http://localhost:9999/callback",
				Scopes:               []string{},
				AuthorizationTimeout: 5 * time.Minute,
			},
		},
		{
			name: "fully specified config unchanged",
			config: &OAuthConfig{
				Enabled:              true,
				ClientID:             "custom-client",
				ClientSecret:         "custom-secret",
				Scopes:               []string{"custom:scope"},
				RedirectURL:          "http://localhost:9999/callback",
				UsePKCE:              false,
				AuthorizationTimeout: 10 * time.Minute,
				UseOIDC:              true,
			},
			expected: &OAuthConfig{
				Enabled:              true,
				ClientID:             "custom-client",
				ClientSecret:         "custom-secret",
				Scopes:               []string{"custom:scope"},
				RedirectURL:          "http://localhost:9999/callback",
				UsePKCE:              false,
				AuthorizationTimeout: 10 * time.Minute,
				UseOIDC:              true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.WithDefaults()

			if result.Enabled != tt.expected.Enabled {
				t.Errorf("Enabled = %v, want %v", result.Enabled, tt.expected.Enabled)
			}
			if result.ClientID != tt.expected.ClientID {
				t.Errorf("ClientID = %v, want %v", result.ClientID, tt.expected.ClientID)
			}
			if result.ClientSecret != tt.expected.ClientSecret {
				t.Errorf("ClientSecret = %v, want %v", result.ClientSecret, tt.expected.ClientSecret)
			}
			if result.RedirectURL != tt.expected.RedirectURL {
				t.Errorf("RedirectURL = %v, want %v", result.RedirectURL, tt.expected.RedirectURL)
			}
			if result.UsePKCE != tt.expected.UsePKCE {
				t.Errorf("UsePKCE = %v, want %v", result.UsePKCE, tt.expected.UsePKCE)
			}
			if result.AuthorizationTimeout != tt.expected.AuthorizationTimeout {
				t.Errorf("AuthorizationTimeout = %v, want %v", result.AuthorizationTimeout, tt.expected.AuthorizationTimeout)
			}
			if result.UseOIDC != tt.expected.UseOIDC {
				t.Errorf("UseOIDC = %v, want %v", result.UseOIDC, tt.expected.UseOIDC)
			}
			if len(result.Scopes) != len(tt.expected.Scopes) {
				t.Errorf("Scopes length = %v, want %v", len(result.Scopes), len(tt.expected.Scopes))
			}
		})
	}
}

func TestOAuthConfig_FieldValues(t *testing.T) {
	config := &OAuthConfig{
		Enabled:      true,
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		Scopes:       []string{"scope1", "scope2"},
		RedirectURL:  "http://localhost:9999/callback",
		UsePKCE:      false,
		UseOIDC:      true,
	}

	if config.ClientID != "test-id" {
		t.Errorf("ClientID = %v, want test-id", config.ClientID)
	}

	if config.ClientSecret != "test-secret" {
		t.Errorf("ClientSecret = %v, want test-secret", config.ClientSecret)
	}

	if len(config.Scopes) != 2 {
		t.Errorf("Scopes length = %v, want 2", len(config.Scopes))
	}

	if config.RedirectURL != "http://localhost:9999/callback" {
		t.Errorf("RedirectURL = %v, want http://localhost:9999/callback", config.RedirectURL)
	}

	if config.UsePKCE {
		t.Errorf("UsePKCE = %v, want false", config.UsePKCE)
	}

	if !config.UseOIDC {
		t.Errorf("UseOIDC = %v, want true", config.UseOIDC)
	}
}

func TestOAuthConfig_HTTPSValidation(t *testing.T) {
	tests := []struct {
		name        string
		redirectURL string
		wantErr     bool
		errContains string
	}{
		{
			name:        "HTTPS localhost rejected",
			redirectURL: "https://localhost:8765/callback",
			wantErr:     true,
			errContains: "HTTPS redirect URIs are not supported",
		},
		{
			name:        "HTTPS 127.0.0.1 rejected",
			redirectURL: "https://127.0.0.1:8765/callback",
			wantErr:     true,
			errContains: "HTTPS redirect URIs are not supported",
		},
		{
			name:        "HTTPS external rejected",
			redirectURL: "https://example.com/callback",
			wantErr:     true,
			errContains: "HTTPS redirect URIs are not supported",
		},
		{
			name:        "HTTP localhost allowed",
			redirectURL: "http://localhost:8765/callback",
			wantErr:     false,
		},
		{
			name:        "HTTP 127.0.0.1 allowed",
			redirectURL: "http://127.0.0.1:8765/callback",
			wantErr:     false,
		},
		{
			name:        "HTTP IPv6 loopback allowed",
			redirectURL: "http://[::1]:8765/callback",
			wantErr:     false,
		},
		{
			name:        "HTTP non-localhost rejected",
			redirectURL: "http://example.com/callback",
			wantErr:     true,
			errContains: "HTTP redirect URIs are only allowed for localhost",
		},
		{
			name:        "FTP scheme rejected",
			redirectURL: "ftp://localhost/callback",
			wantErr:     true,
			errContains: "redirect URI scheme must be http",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &OAuthConfig{
				Enabled:     true,
				RedirectURL: tt.redirectURL,
				Scopes:      []string{"mcp:tools"},
			}

			// Apply defaults for non-error cases to ensure all required fields are set
			if !tt.wantErr {
				config = config.WithDefaults()
			}

			err := config.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got nil")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Error %q does not contain %q", err.Error(), tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestOAuthConfig_OIDCFeatures(t *testing.T) {
	config := &OAuthConfig{
		Enabled:     true,
		RedirectURL: "http://localhost:8765/callback",
		UseOIDC:     true,
	}

	// Apply defaults before validation
	config = config.WithDefaults()

	err := config.Validate()
	if err != nil {
		t.Errorf("Validation failed for OIDC config: %v", err)
	}

	if !config.UseOIDC {
		t.Error("Expected UseOIDC to be true")
	}
}

func TestOAuthConfig_RegistrationToken(t *testing.T) {
	tests := []struct {
		name              string
		registrationToken string
		wantErr           bool
	}{
		{
			name:              "valid config with registration token",
			registrationToken: "test-registration-token-12345",
			wantErr:           false,
		},
		{
			name:              "valid config without registration token",
			registrationToken: "",
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &OAuthConfig{
				Enabled:           true,
				RedirectURL:       "http://localhost:8765/callback",
				RegistrationToken: tt.registrationToken,
			}

			// Apply defaults before validation
			config = config.WithDefaults()

			err := config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if config.RegistrationToken != tt.registrationToken {
				t.Errorf("RegistrationToken = %v, want %v", config.RegistrationToken, tt.registrationToken)
			}
		})
	}
}
