package agent

// selectScopes selects OAuth scopes based on the MCP spec priority order.
//
// Priority order (when ScopeSelectionMode is "auto"):
//  1. Use scope parameter from WWW-Authenticate header (challenge.Scopes)
//  2. Use scopes_supported from Protected Resource Metadata
//  3. Omit scope parameter entirely (return nil)
//
// When ScopeSelectionMode is "manual", always returns config.Scopes.
//
// Security: This implements the principle of least privilege by requesting
// only the scopes specified by the server or none at all.
func selectScopes(config *OAuthConfig, challenge *WWWAuthenticateChallenge, metadata *ProtectedResourceMetadata, logger *Logger) []string {
	// Manual mode: always use configured scopes
	if config.ScopeSelectionMode == ScopeModeManual {
		// Warn if manual scopes diverge from discovered scopes
		if logger != nil {
			var discoveredScopes []string
			if challenge != nil && len(challenge.Scopes) > 0 {
				discoveredScopes = challenge.Scopes
			} else if metadata != nil && len(metadata.ScopesSupported) > 0 {
				discoveredScopes = metadata.ScopesSupported
			}

			if len(discoveredScopes) > 0 {
				// Check if scopes differ
				if !scopesEqual(config.Scopes, discoveredScopes) {
					logger.Warning("Manual scope mode: requested scopes %v differ from server-discovered scopes %v", config.Scopes, discoveredScopes)
					logger.Warning("This may lead to authorization failures or over-privileged tokens")
				}
			}
		}
		return config.Scopes
	}

	// Auto mode: follow MCP spec priority

	// Priority 1: Scope from WWW-Authenticate header
	if challenge != nil && len(challenge.Scopes) > 0 {
		return challenge.Scopes
	}

	// Priority 2: Scopes from Protected Resource Metadata
	if metadata != nil && len(metadata.ScopesSupported) > 0 {
		return metadata.ScopesSupported
	}

	// Priority 3: Omit scope parameter
	// Return nil to indicate no scope parameter should be sent
	return nil
}

// scopesEqual checks if two scope slices contain the same elements (order-independent)
func scopesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	// Create a map for quick lookup
	scopeMap := make(map[string]bool)
	for _, scope := range a {
		scopeMap[scope] = true
	}

	// Check if all scopes in b exist in a
	for _, scope := range b {
		if !scopeMap[scope] {
			return false
		}
	}

	return true
}
