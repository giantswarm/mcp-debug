// Package agent implements Authorization Server Metadata Discovery per RFC 8414.
//
// Multi-Endpoint Discovery:
// Probes multiple discovery endpoints in priority order based on issuer URL format.
// Supports both OAuth 2.0 Authorization Server Metadata (RFC 8414) and
// OpenID Connect Discovery 1.0.
//
// PKCE Validation:
// Enforces PKCE support check as required by MCP spec (2025-11-25).
// Authorization servers MUST advertise code_challenge_methods_supported.
// Use SkipPKCEValidation flag only for testing non-compliant servers.
//
// Security:
// - Validates PKCE support by default (fail closed)
// - Supports S256 code challenge method requirement
// - Validates metadata structure and required fields
package agent

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// PKCE code challenge method constant
const pkceMethodS256 = "S256"

// AuthorizationServerMetadata represents OAuth 2.0 Authorization Server Metadata
// as defined in RFC 8414 and OpenID Connect Discovery 1.0.
type AuthorizationServerMetadata struct {
	// Issuer is the authorization server's issuer identifier URL
	Issuer string `json:"issuer"`

	// AuthorizationEndpoint is the URL for the authorization endpoint
	AuthorizationEndpoint string `json:"authorization_endpoint"`

	// TokenEndpoint is the URL for the token endpoint
	TokenEndpoint string `json:"token_endpoint"`

	// RegistrationEndpoint is the URL for Dynamic Client Registration (optional)
	RegistrationEndpoint string `json:"registration_endpoint,omitempty"`

	// CodeChallengeMethods lists supported PKCE code challenge methods
	// MCP spec requires this field to be present and include "S256"
	CodeChallengeMethods []string `json:"code_challenge_methods_supported,omitempty"`

	// ClientIDMetadataDocumentSupported indicates support for Client ID Metadata Documents
	ClientIDMetadataDocumentSupported bool `json:"client_id_metadata_document_supported,omitempty"`

	// ScopesSupported lists supported OAuth scopes (optional)
	ScopesSupported []string `json:"scopes_supported,omitempty"`

	// ResponseTypesSupported lists supported OAuth response types
	ResponseTypesSupported []string `json:"response_types_supported,omitempty"`

	// GrantTypesSupported lists supported OAuth grant types
	GrantTypesSupported []string `json:"grant_types_supported,omitempty"`

	// TokenEndpointAuthMethodsSupported lists supported token endpoint auth methods
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported,omitempty"`
}

const (
	// Maximum size for AS metadata documents (1MB)
	maxASMetadataSize = 1024 * 1024

	// HTTP timeout for AS metadata requests
	asMetadataRequestTimeout = 10 * time.Second

	// User agent string for AS metadata requests
	userAgent = "mcp-debug/1.0"

	// Environment variable to allow insecure operations (testing only)
	allowInsecureEnvVar = "MCP_DEBUG_ALLOW_INSECURE"
)

// DiscoverAuthorizationServerMetadata discovers authorization server metadata
// for the given issuer URL per RFC 8414 and OIDC Discovery 1.0.
//
// Discovery probes endpoints in this priority order:
//
// For issuer URLs with path components (e.g., https://auth.example.com/tenant1):
//  1. OAuth 2.0 with path insertion: https://auth.example.com/.well-known/oauth-authorization-server/tenant1
//  2. OIDC with path insertion: https://auth.example.com/.well-known/openid-configuration/tenant1
//  3. OIDC path appending: https://auth.example.com/tenant1/.well-known/openid-configuration
//
// For issuer URLs without path components (e.g., https://auth.example.com):
//  1. OAuth 2.0: https://auth.example.com/.well-known/oauth-authorization-server
//  2. OIDC: https://auth.example.com/.well-known/openid-configuration
//
// Returns the first successfully retrieved metadata document.
func DiscoverAuthorizationServerMetadata(ctx context.Context, issuerURL string, logger *Logger) (*AuthorizationServerMetadata, error) {
	// Build discovery endpoints based on issuer URL format
	endpoints, err := buildASMetadataEndpoints(issuerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to build AS metadata endpoints: %w", err)
	}

	if logger != nil {
		logger.InfoVerbose("Probing %d AS metadata endpoints for issuer: %s", len(endpoints), issuerURL)
	}

	// Try each endpoint in order
	var lastErr error
	for i, endpoint := range endpoints {
		if logger != nil {
			logger.InfoVerbose("Trying AS metadata endpoint (%d/%d): %s", i+1, len(endpoints), endpoint)
		}

		metadata, err := fetchASMetadata(ctx, endpoint)
		if err != nil {
			if logger != nil {
				logger.WarningVerbose("Failed to fetch from %s: %v", endpoint, err)
			}
			lastErr = err
			continue
		}

		// Validate metadata structure
		if err := validateASMetadata(metadata); err != nil {
			if logger != nil {
				logger.WarningVerbose("Invalid metadata from %s: %v", endpoint, err)
			}
			lastErr = err
			continue
		}

		if logger != nil {
			logger.Info("Successfully discovered AS metadata from: %s", endpoint)
		}

		return metadata, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("no valid AS metadata found (last error: %w)", lastErr)
	}

	return nil, fmt.Errorf("no AS metadata found at any discovery endpoint")
}

// normalizePath removes leading and trailing slashes from a URL path.
func normalizePath(p string) string {
	p = strings.TrimPrefix(p, "/")
	return strings.TrimSuffix(p, "/")
}

// isLocalhost checks if the given host is localhost or 127.0.0.1
func isLocalhost(host string) bool {
	return host == "localhost" ||
		strings.HasPrefix(host, "localhost:") ||
		host == "127.0.0.1" ||
		strings.HasPrefix(host, "127.0.0.1:") ||
		host == "[::1]" ||
		strings.HasPrefix(host, "[::1]:")
}

// buildASMetadataEndpoints constructs AS metadata discovery endpoints
// based on the issuer URL format per RFC 8414 Section 3 and OIDC Discovery Section 4.
func buildASMetadataEndpoints(issuerURL string) ([]string, error) {
	parsed, err := url.Parse(issuerURL)
	if err != nil {
		return nil, fmt.Errorf("invalid issuer URL: %w", err)
	}

	if !parsed.IsAbs() {
		return nil, fmt.Errorf("issuer URL must be absolute")
	}

	// Validate scheme: HTTPS required, HTTP only allowed for localhost
	if parsed.Scheme == schemeHTTP {
		if !isLocalhost(parsed.Host) {
			return nil, fmt.Errorf("issuer URL must use https scheme (http only allowed for localhost, got: %s)", parsed.Host)
		}
	} else if parsed.Scheme != schemeHTTPS {
		return nil, fmt.Errorf("issuer URL must use http or https scheme")
	}

	if parsed.Host == "" {
		return nil, fmt.Errorf("issuer URL missing host")
	}

	var endpoints []string
	baseURL := fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)
	path := normalizePath(parsed.Path)

	// Issuer URL has path components
	if path != "" {
		// Priority 1: OAuth 2.0 with path insertion
		// Format: https://host/.well-known/oauth-authorization-server/path
		endpoints = append(endpoints, fmt.Sprintf("%s/.well-known/oauth-authorization-server/%s", baseURL, path))

		// Priority 2: OIDC with path insertion
		// Format: https://host/.well-known/openid-configuration/path
		endpoints = append(endpoints, fmt.Sprintf("%s/.well-known/openid-configuration/%s", baseURL, path))

		// Priority 3: OIDC path appending
		// Format: https://host/path/.well-known/openid-configuration
		endpoints = append(endpoints, fmt.Sprintf("%s/%s/.well-known/openid-configuration", baseURL, path))
	} else {
		// Issuer URL without path components
		// Priority 1: OAuth 2.0
		endpoints = append(endpoints, fmt.Sprintf("%s/.well-known/oauth-authorization-server", baseURL))

		// Priority 2: OIDC
		endpoints = append(endpoints, fmt.Sprintf("%s/.well-known/openid-configuration", baseURL))
	}

	return endpoints, nil
}

// fetchASMetadata fetches and parses authorization server metadata from the specified URL.
func fetchASMetadata(ctx context.Context, metadataURL string) (*AuthorizationServerMetadata, error) {
	// Create HTTP client with timeout and secure TLS configuration
	client := &http.Client{
		Timeout: asMetadataRequestTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
				// InsecureSkipVerify: false (explicit default for security)
			},
		},
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, metadataURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set appropriate headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	// Validate Content-Type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(strings.ToLower(contentType), "application/json") {
		return nil, fmt.Errorf("unexpected Content-Type: %s", contentType)
	}

	// Read response body with size limit
	limitedReader := io.LimitReader(resp.Body, maxASMetadataSize)
	bodyBytes, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check if response was truncated
	if int64(len(bodyBytes)) >= maxASMetadataSize {
		return nil, fmt.Errorf("response exceeds maximum size of %d bytes", maxASMetadataSize)
	}

	// Parse JSON
	var metadata AuthorizationServerMetadata
	if err := json.Unmarshal(bodyBytes, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &metadata, nil
}

// validateASMetadata validates authorization server metadata structure
// and checks for required fields per RFC 8414 Section 3.
func validateASMetadata(metadata *AuthorizationServerMetadata) error {
	if metadata.Issuer == "" {
		return fmt.Errorf("missing required field: issuer")
	}

	if metadata.AuthorizationEndpoint == "" {
		return fmt.Errorf("missing required field: authorization_endpoint")
	}

	if metadata.TokenEndpoint == "" {
		return fmt.Errorf("missing required field: token_endpoint")
	}

	// Validate endpoint URLs are absolute HTTP(S) URLs
	endpoints := map[string]string{
		"issuer":                 metadata.Issuer,
		"authorization_endpoint": metadata.AuthorizationEndpoint,
		"token_endpoint":         metadata.TokenEndpoint,
	}

	if metadata.RegistrationEndpoint != "" {
		endpoints["registration_endpoint"] = metadata.RegistrationEndpoint
	}

	for name, endpoint := range endpoints {
		parsed, err := url.Parse(endpoint)
		if err != nil {
			return fmt.Errorf("invalid %s URL: %w", name, err)
		}

		if !parsed.IsAbs() {
			return fmt.Errorf("%s must be absolute URL: %s", name, endpoint)
		}

		// Validate scheme: HTTPS required, HTTP only allowed for localhost
		if parsed.Scheme == "http" {
			if !isLocalhost(parsed.Host) {
				return fmt.Errorf("%s must use https scheme (http only allowed for localhost): %s", name, endpoint)
			}
		} else if parsed.Scheme != "https" {
			return fmt.Errorf("%s must use http or https scheme: %s", name, endpoint)
		}

		if parsed.Host == "" {
			return fmt.Errorf("%s missing host: %s", name, endpoint)
		}
	}

	return nil
}

// ValidatePKCESupport checks if the authorization server advertises PKCE support.
// Per MCP spec (2025-11-25), authorization servers MUST support PKCE and
// MUST advertise it in code_challenge_methods_supported.
//
// Returns an error if:
//   - code_challenge_methods_supported is absent
//   - code_challenge_methods_supported is empty
//   - S256 method is not supported
//
// Use skipValidation=true only for testing with non-compliant servers.
// Requires MCP_DEBUG_ALLOW_INSECURE=true environment variable when skipValidation is true.
func ValidatePKCESupport(metadata *AuthorizationServerMetadata, skipValidation bool, logger *Logger) error {
	if skipValidation {
		// Require explicit environment variable to allow insecure operations
		if os.Getenv(allowInsecureEnvVar) != "true" {
			return fmt.Errorf("PKCE validation skip requires %s=true environment variable for safety", allowInsecureEnvVar)
		}

		// Log security warning
		if logger != nil {
			logger.Warning("⚠️  SECURITY WARNING: PKCE validation is disabled. This should only be used for testing!")
		}

		return nil
	}

	if len(metadata.CodeChallengeMethods) == 0 {
		return fmt.Errorf("authorization server does not advertise PKCE support (code_challenge_methods_supported missing or empty) - this is required by MCP spec. Use --oauth-skip-pkce-validation flag for testing only")
	}

	// Check for S256 support (MUST use S256 when capable per OAuth 2.1)
	hasS256 := false
	for _, method := range metadata.CodeChallengeMethods {
		if method == pkceMethodS256 {
			hasS256 = true
			break
		}
	}

	if !hasS256 {
		return fmt.Errorf("authorization server does not support S256 PKCE method (only: %v) - S256 is required", metadata.CodeChallengeMethods)
	}

	return nil
}
