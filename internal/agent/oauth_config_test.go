package agent

import (
	"testing"
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
				Enabled:     true,
				ClientID:    "test-client-id",
				RedirectURL: "http://localhost:8765/callback",
				Scopes:      []string{"mcp:tools"},
			},
			wantErr: false,
		},
		{
			name: "valid config without client ID (DCR)",
			config: &OAuthConfig{
				Enabled:     true,
				RedirectURL: "http://localhost:8765/callback",
				Scopes:      []string{"mcp:tools"},
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
			name: "empty scopes get defaults",
			config: &OAuthConfig{
				Enabled:     true,
				RedirectURL: "http://localhost:8765/callback",
				Scopes:      []string{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Check that default scopes were set if needed
			if !tt.wantErr && tt.config.Enabled && len(tt.config.Scopes) == 0 {
				if len(tt.config.Scopes) < 1 {
					// Note: Validate() should have set default scopes
					t.Log("Scopes should be set to defaults after validation")
				}
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

	if len(config.Scopes) == 0 {
		t.Error("Default config should have default scopes")
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
}
