package agent

import (
	"reflect"
	"testing"
)

func TestSelectScopes(t *testing.T) {
	tests := []struct {
		name      string
		config    *OAuthConfig
		challenge *WWWAuthenticateChallenge
		metadata  *ProtectedResourceMetadata
		want      []string
	}{
		{
			name: "manual mode - returns configured scopes",
			config: &OAuthConfig{
				ScopeSelectionMode: "manual",
				Scopes:             []string{"custom:read", "custom:write"},
			},
			challenge: &WWWAuthenticateChallenge{
				Scopes: []string{"server:read"},
			},
			metadata: &ProtectedResourceMetadata{
				ScopesSupported: []string{"metadata:read"},
			},
			want: []string{"custom:read", "custom:write"},
		},
		{
			name: "manual mode - empty scopes",
			config: &OAuthConfig{
				ScopeSelectionMode: "manual",
				Scopes:             []string{},
			},
			challenge: &WWWAuthenticateChallenge{
				Scopes: []string{"server:read"},
			},
			want: []string{},
		},
		{
			name: "auto mode - priority 1: challenge scopes",
			config: &OAuthConfig{
				ScopeSelectionMode: "auto",
				Scopes:             []string{"config:read"},
			},
			challenge: &WWWAuthenticateChallenge{
				Scopes: []string{"files:read", "files:write"},
			},
			metadata: &ProtectedResourceMetadata{
				ScopesSupported: []string{"metadata:read"},
			},
			want: []string{"files:read", "files:write"},
		},
		{
			name: "auto mode - priority 2: metadata scopes (no challenge)",
			config: &OAuthConfig{
				ScopeSelectionMode: "auto",
				Scopes:             []string{"config:read"},
			},
			challenge: nil,
			metadata: &ProtectedResourceMetadata{
				ScopesSupported: []string{"resource:read", "resource:write"},
			},
			want: []string{"resource:read", "resource:write"},
		},
		{
			name: "auto mode - priority 2: metadata scopes (empty challenge)",
			config: &OAuthConfig{
				ScopeSelectionMode: "auto",
				Scopes:             []string{"config:read"},
			},
			challenge: &WWWAuthenticateChallenge{
				Scopes: []string{},
			},
			metadata: &ProtectedResourceMetadata{
				ScopesSupported: []string{"resource:read"},
			},
			want: []string{"resource:read"},
		},
		{
			name: "auto mode - priority 3: omit scope (no challenge, no metadata)",
			config: &OAuthConfig{
				ScopeSelectionMode: "auto",
				Scopes:             []string{"config:read"},
			},
			challenge: nil,
			metadata:  nil,
			want:      nil,
		},
		{
			name: "auto mode - priority 3: omit scope (empty challenge and metadata)",
			config: &OAuthConfig{
				ScopeSelectionMode: "auto",
				Scopes:             []string{"config:read"},
			},
			challenge: &WWWAuthenticateChallenge{
				Scopes: []string{},
			},
			metadata: &ProtectedResourceMetadata{
				ScopesSupported: []string{},
			},
			want: nil,
		},
		{
			name: "auto mode - priority 3: omit scope (nil scopes in metadata)",
			config: &OAuthConfig{
				ScopeSelectionMode: "auto",
				Scopes:             []string{"config:read"},
			},
			challenge: nil,
			metadata: &ProtectedResourceMetadata{
				Resource:             "https://example.com/mcp",
				AuthorizationServers: []string{"https://auth.example.com"},
				ScopesSupported:      nil,
			},
			want: nil,
		},
		{
			name: "auto mode - empty mode string defaults to auto behavior",
			config: &OAuthConfig{
				ScopeSelectionMode: "",
				Scopes:             []string{"config:read"},
			},
			challenge: &WWWAuthenticateChallenge{
				Scopes: []string{"challenge:scope"},
			},
			metadata: nil,
			want:     []string{"challenge:scope"},
		},
		{
			name: "default mode - nil config uses auto behavior",
			config: &OAuthConfig{
				Scopes: []string{"config:read"},
			},
			challenge: &WWWAuthenticateChallenge{
				Scopes: []string{"challenge:scope"},
			},
			metadata: nil,
			want:     []string{"challenge:scope"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := selectScopes(tt.config, tt.challenge, tt.metadata)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("selectScopes() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestScopeSelectionIntegration tests the complete scope selection flow
// including various configurations and edge cases
func TestScopeSelectionIntegration(t *testing.T) {
	tests := []struct {
		name        string
		config      *OAuthConfig
		challenge   *WWWAuthenticateChallenge
		metadata    *ProtectedResourceMetadata
		want        []string
		description string
	}{
		{
			name: "real world - initial 401 with scope in challenge",
			config: &OAuthConfig{
				ScopeSelectionMode: "auto",
			},
			challenge: &WWWAuthenticateChallenge{
				Scheme:              "Bearer",
				ResourceMetadataURL: "https://mcp.example.com/.well-known/oauth-protected-resource",
				Scopes:              []string{"mcp:read"},
			},
			metadata: &ProtectedResourceMetadata{
				Resource:             "https://mcp.example.com/mcp",
				AuthorizationServers: []string{"https://auth.example.com"},
				ScopesSupported:      []string{"mcp:read", "mcp:write", "admin"},
			},
			want:        []string{"mcp:read"},
			description: "Server specifies exact scope needed via challenge - use it (least privilege)",
		},
		{
			name: "real world - metadata discovery without prior challenge",
			config: &OAuthConfig{
				ScopeSelectionMode: "auto",
			},
			challenge: nil,
			metadata: &ProtectedResourceMetadata{
				Resource:             "https://mcp.example.com/mcp",
				AuthorizationServers: []string{"https://auth.example.com"},
				ScopesSupported:      []string{"mcp:basic"},
			},
			want:        []string{"mcp:basic"},
			description: "Proactive metadata discovery - use all supported scopes for initial access",
		},
		{
			name: "real world - no discovery available",
			config: &OAuthConfig{
				ScopeSelectionMode: "auto",
			},
			challenge:   nil,
			metadata:    nil,
			want:        nil,
			description: "No metadata or challenge - omit scope parameter (let AS determine)",
		},
		{
			name: "manual override - user knows exact scopes needed",
			config: &OAuthConfig{
				ScopeSelectionMode: "manual",
				Scopes:             []string{"custom:admin", "custom:debug"},
			},
			challenge: &WWWAuthenticateChallenge{
				Scopes: []string{"mcp:read"},
			},
			metadata: &ProtectedResourceMetadata{
				ScopesSupported: []string{"mcp:basic"},
			},
			want:        []string{"custom:admin", "custom:debug"},
			description: "Manual mode - user has specific requirements that override discovery",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := selectScopes(tt.config, tt.challenge, tt.metadata)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("selectScopes() = %v, want %v\nDescription: %s", got, tt.want, tt.description)
			}
		})
	}
}

// TestScopeSelectionSecurityProperties tests security-critical properties
func TestScopeSelectionSecurityProperties(t *testing.T) {
	t.Run("principle of least privilege - prefers specific over general scopes", func(t *testing.T) {
		config := &OAuthConfig{ScopeSelectionMode: "auto"}
		challenge := &WWWAuthenticateChallenge{
			Scopes: []string{"resource:read"}, // Specific scope needed
		}
		metadata := &ProtectedResourceMetadata{
			ScopesSupported: []string{"resource:read", "resource:write", "admin"}, // More scopes available
		}

		got := selectScopes(config, challenge, metadata)

		// Should prefer challenge's specific scope over metadata's broader set
		if !reflect.DeepEqual(got, []string{"resource:read"}) {
			t.Errorf("Expected to use specific challenge scope, got %v", got)
		}
	})

	t.Run("no scope escalation without signal", func(t *testing.T) {
		config := &OAuthConfig{
			ScopeSelectionMode: "auto",
			Scopes:             []string{"high:privilege"}, // User configured high privilege
		}
		var challenge *WWWAuthenticateChallenge
		metadata := &ProtectedResourceMetadata{
			ScopesSupported: []string{"basic:read"}, // Server only needs basic
		}

		got := selectScopes(config, challenge, metadata)

		// Should use server's basic scope, not user's high privilege config
		if !reflect.DeepEqual(got, []string{"basic:read"}) {
			t.Errorf("Expected to use server's basic scope, got %v", got)
		}
	})
}
