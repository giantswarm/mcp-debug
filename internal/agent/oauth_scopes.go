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
func selectScopes(config *OAuthConfig, challenge *WWWAuthenticateChallenge, metadata *ProtectedResourceMetadata) []string {
	// Manual mode: always use configured scopes
	if config.ScopeSelectionMode == "manual" {
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
