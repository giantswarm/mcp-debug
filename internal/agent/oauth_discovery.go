// Package agent implements Protected Resource Metadata Discovery per RFC 9728.
//
// Security Features:
//
// SSRF Protection: Validates metadata URLs to prevent Server-Side Request Forgery attacks.
// Blocks requests to localhost, private IP ranges, link-local addresses, and special-use IPs.
//
// HTTP Detection: Warns when authorization servers use HTTP instead of HTTPS,
// as this exposes credentials to network interception.
//
// Escaped Characters: Properly handles escaped quotes and backslashes in
// WWW-Authenticate header parameters to prevent injection attacks.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ProtectedResourceMetadata represents OAuth 2.0 Protected Resource Metadata
// as defined in RFC 9728.
type ProtectedResourceMetadata struct {
	// Resource is the protected resource identifier
	Resource string `json:"resource"`

	// AuthorizationServers lists the authorization servers for this resource
	AuthorizationServers []string `json:"authorization_servers"`

	// ScopesSupported lists the OAuth scopes supported by this resource
	ScopesSupported []string `json:"scopes_supported,omitempty"`

	// BearerMethodsSupported indicates how bearer tokens can be presented
	BearerMethodsSupported []string `json:"bearer_methods_supported,omitempty"`

	// ResourceDocumentation provides human-readable documentation URL
	ResourceDocumentation string `json:"resource_documentation,omitempty"`
}

// WWWAuthenticateChallenge represents parsed WWW-Authenticate header information
type WWWAuthenticateChallenge struct {
	// Scheme is the authentication scheme (typically "Bearer")
	Scheme string

	// ResourceMetadataURL is the URL to fetch protected resource metadata
	ResourceMetadataURL string

	// Scopes are the required scopes for this resource/operation
	Scopes []string

	// Error indicates the error type (e.g., "insufficient_scope")
	Error string

	// ErrorDescription provides human-readable error details
	ErrorDescription string
}

const (
	// Maximum size for metadata documents (1MB)
	maxMetadataSize = 1024 * 1024

	// HTTP timeout for metadata requests
	metadataRequestTimeout = 10 * time.Second
)

// parseWWWAuthenticate parses a WWW-Authenticate header value and extracts
// OAuth challenge parameters per RFC 6750 and RFC 9728.
//
// Example header:
//
//	WWW-Authenticate: Bearer resource_metadata="https://mcp.example.com/.well-known/oauth-protected-resource",
//	                         scope="files:read",
//	                         error="insufficient_scope"
func parseWWWAuthenticate(header string) (*WWWAuthenticateChallenge, error) {
	if header == "" {
		return nil, fmt.Errorf("empty WWW-Authenticate header")
	}

	// Split scheme and parameters
	// SplitN always returns at least one element, so no need to check len(parts) < 1
	parts := strings.SplitN(header, " ", 2)

	challenge := &WWWAuthenticateChallenge{
		Scheme: parts[0],
	}

	// Parse parameters if present
	if len(parts) == 2 {
		params := parseAuthParams(parts[1])
		challenge.ResourceMetadataURL = params["resource_metadata"]
		challenge.Error = params["error"]
		challenge.ErrorDescription = params["error_description"]

		// Parse scope parameter (space-separated list)
		if scopeParam := params["scope"]; scopeParam != "" {
			challenge.Scopes = strings.Fields(scopeParam)
		}
	}

	return challenge, nil
}

// parseAuthParams parses OAuth authentication parameters from the challenge.
// Handles both quoted and unquoted values.
// Format: key1="value1", key2="value2", key3=value3
func parseAuthParams(params string) map[string]string {
	result := make(map[string]string)

	// Split by comma, but respect quotes
	parts := splitPreservingQuotes(params, ',')

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Split by equals sign
		eqIdx := strings.Index(part, "=")
		if eqIdx == -1 {
			continue
		}

		key := strings.TrimSpace(part[:eqIdx])
		value := strings.TrimSpace(part[eqIdx+1:])

		// Remove surrounding quotes from value if present and unescape
		if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			value = value[1 : len(value)-1]
			// Unescape any escaped quotes within the value
			value = strings.ReplaceAll(value, `\"`, `"`)
			// Unescape any escaped backslashes
			value = strings.ReplaceAll(value, `\\`, `\`)
		}

		if key != "" {
			result[key] = value
		}
	}

	return result
}

// splitPreservingQuotes splits a string by delimiter but preserves quoted sections.
// Handles escaped quotes (backslash-escaped) within quoted strings.
func splitPreservingQuotes(s string, delimiter byte) []string {
	var result []string
	var current strings.Builder
	inQuotes := false
	escaped := false

	for i := 0; i < len(s); i++ {
		ch := s[i]

		if escaped {
			// Previous character was backslash, add current char literally
			current.WriteByte(ch)
			escaped = false
		} else if ch == '\\' {
			// Backslash - mark as escape for next character
			current.WriteByte(ch)
			escaped = true
		} else if ch == '"' {
			// Quote - toggle quote mode
			inQuotes = !inQuotes
			current.WriteByte(ch)
		} else if ch == delimiter && !inQuotes {
			// Delimiter outside quotes - split here
			result = append(result, current.String())
			current.Reset()
		} else {
			current.WriteByte(ch)
		}
	}

	// Add last segment
	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}

// isAllowedMetadataURL validates that a metadata URL is safe to fetch
// and does not point to internal/private network resources.
// This prevents SSRF (Server-Side Request Forgery) attacks.
func isAllowedMetadataURL(urlStr string) error {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Require absolute URL with scheme
	if !parsed.IsAbs() {
		return fmt.Errorf("metadata URL must be absolute")
	}

	// Only allow http and https schemes
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("metadata URL must use http or https scheme, got: %s", parsed.Scheme)
	}

	hostname := parsed.Hostname()
	if hostname == "" {
		return fmt.Errorf("metadata URL missing hostname")
	}

	// Deny localhost and loopback addresses
	if hostname == "localhost" || hostname == "127.0.0.1" || strings.HasPrefix(hostname, "127.") ||
		hostname == "::1" || hostname == "[::1]" {
		return fmt.Errorf("localhost URLs not allowed for metadata discovery")
	}

	// Try to parse as IP address
	ip := net.ParseIP(hostname)
	if ip != nil {
		// Deny private IP ranges (RFC 1918, RFC 4193, link-local)
		if ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return fmt.Errorf("private IP addresses not allowed for metadata discovery: %s", hostname)
		}

		// Deny IPv4 special-use addresses
		// 0.0.0.0/8, 169.254.0.0/16, 224.0.0.0/4, 240.0.0.0/4
		if ip4 := ip.To4(); ip4 != nil {
			if ip4[0] == 0 || ip4[0] == 169 && ip4[1] == 254 ||
				ip4[0] >= 224 {
				return fmt.Errorf("special-use IP addresses not allowed: %s", hostname)
			}
		}
	}

	return nil
}

// discoverProtectedResourceMetadata discovers protected resource metadata
// for the given MCP server endpoint per RFC 9728.
//
// Discovery order:
//  1. If challenge provides resource_metadata URL, use it
//  2. Try well-known URI with path: /.well-known/oauth-protected-resource/mcp
//  3. Try well-known URI at root: /.well-known/oauth-protected-resource
func discoverProtectedResourceMetadata(ctx context.Context, endpoint string, challenge *WWWAuthenticateChallenge, logger *Logger) (*ProtectedResourceMetadata, error) {
	// Priority 1: Use resource_metadata URL from WWW-Authenticate header
	if challenge != nil && challenge.ResourceMetadataURL != "" {
		logger.InfoVerbose("Using resource_metadata URL from WWW-Authenticate: %s", challenge.ResourceMetadataURL)

		// Validate URL to prevent SSRF attacks
		if err := isAllowedMetadataURL(challenge.ResourceMetadataURL); err != nil {
			return nil, fmt.Errorf("invalid resource_metadata URL from WWW-Authenticate: %w", err)
		}

		metadata, err := fetchProtectedResourceMetadata(ctx, challenge.ResourceMetadataURL)
		if err != nil {
			return nil, err
		}

		// Warn about insecure HTTP authorization servers
		warnInsecureAuthServers(metadata, logger)

		return metadata, nil
	}

	// Priority 2 & 3: Try well-known URIs
	wellKnownURIs, err := buildWellKnownURIs(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to build well-known URIs: %w", err)
	}

	for i, uri := range wellKnownURIs {
		logger.InfoVerbose("Trying well-known URI (%d/%d): %s", i+1, len(wellKnownURIs), uri)

		metadata, err := fetchProtectedResourceMetadata(ctx, uri)
		if err != nil {
			logger.WarningVerbose("Failed to fetch from %s: %v", uri, err)
			continue
		}

		if logger != nil {
			logger.Info("Successfully discovered protected resource metadata from: %s", uri)
		}

		// Warn about insecure HTTP authorization servers
		warnInsecureAuthServers(metadata, logger)

		return metadata, nil
	}

	return nil, fmt.Errorf("no protected resource metadata found at well-known URIs")
}

// buildWellKnownURIs constructs the well-known URIs for protected resource metadata
// per RFC 9728 Section 3.
func buildWellKnownURIs(endpoint string) ([]string, error) {
	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse endpoint URL: %w", err)
	}

	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, fmt.Errorf("endpoint URL must include scheme and host")
	}

	baseURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)

	// Build well-known URIs in priority order
	var uris []string

	// If endpoint has a path, try path-based discovery first
	if parsedURL.Path != "" && parsedURL.Path != "/" {
		// Path-based: /.well-known/oauth-protected-resource/<path>
		// Remove leading slash from path for construction
		path := strings.TrimPrefix(parsedURL.Path, "/")
		uris = append(uris, fmt.Sprintf("%s/.well-known/oauth-protected-resource/%s", baseURL, path))
	}

	// Always try root-level discovery
	uris = append(uris, fmt.Sprintf("%s/.well-known/oauth-protected-resource", baseURL))

	return uris, nil
}

// fetchProtectedResourceMetadata fetches and parses protected resource metadata
// from the specified URL.
func fetchProtectedResourceMetadata(ctx context.Context, metadataURL string) (*ProtectedResourceMetadata, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: metadataRequestTimeout,
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, metadataURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create metadata request: %w", err)
	}

	// Set appropriate headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "mcp-debug/1.0")

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch metadata: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metadata request failed with status %d", resp.StatusCode)
	}

	// Validate Content-Type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(strings.ToLower(contentType), "application/json") {
		return nil, fmt.Errorf("unexpected Content-Type: %s (expected application/json)", contentType)
	}

	// Read response body with size limit
	limitedReader := io.LimitReader(resp.Body, maxMetadataSize)
	bodyBytes, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata response: %w", err)
	}

	// Check if response was truncated
	if int64(len(bodyBytes)) >= maxMetadataSize {
		return nil, fmt.Errorf("metadata response exceeds maximum size of %d bytes", maxMetadataSize)
	}

	// Parse JSON
	var metadata ProtectedResourceMetadata
	if err := json.Unmarshal(bodyBytes, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata JSON: %w", err)
	}

	// Validate required fields per RFC 9728
	if err := validateProtectedResourceMetadata(&metadata); err != nil {
		return nil, fmt.Errorf("invalid metadata: %w", err)
	}

	return &metadata, nil
}

// validateProtectedResourceMetadata validates that required fields are present
// and that authorization server URLs are valid absolute HTTP(S) URLs per RFC 9728.
func validateProtectedResourceMetadata(metadata *ProtectedResourceMetadata) error {
	if metadata.Resource == "" {
		return fmt.Errorf("missing required field: resource")
	}

	if len(metadata.AuthorizationServers) == 0 {
		return fmt.Errorf("missing required field: authorization_servers (at least one required)")
	}

	// Validate authorization server URLs
	// Per RFC 9728, authorization servers must be absolute URLs with http or https scheme
	for i, asURL := range metadata.AuthorizationServers {
		parsed, err := url.Parse(asURL)
		if err != nil {
			return fmt.Errorf("invalid authorization server URL at index %d: %w", i, err)
		}

		if !parsed.IsAbs() {
			return fmt.Errorf("authorization server URL at index %d must be absolute: %s", i, asURL)
		}

		if parsed.Scheme != "https" && parsed.Scheme != "http" {
			return fmt.Errorf("authorization server URL at index %d must use http or https scheme: %s", i, asURL)
		}

		if parsed.Host == "" {
			return fmt.Errorf("authorization server URL at index %d missing host: %s", i, asURL)
		}
	}

	return nil
}

// warnInsecureAuthServers logs warnings for authorization servers using HTTP instead of HTTPS
func warnInsecureAuthServers(metadata *ProtectedResourceMetadata, logger *Logger) {
	if logger == nil {
		return
	}

	for _, asURL := range metadata.AuthorizationServers {
		parsed, err := url.Parse(asURL)
		if err != nil {
			continue
		}

		if parsed.Scheme == "http" {
			logger.Warning("Authorization server using HTTP (not HTTPS): %s - credentials may be exposed to network attacks", asURL)
		}
	}
}

// selectAuthorizationServer selects an authorization server from the metadata
// based on configuration preferences.
//
// If preferredServer is specified and found in metadata.AuthorizationServers,
// it is returned. Otherwise, the first server in the list is returned per
// RFC 9728 Section 3 recommendation.
//
// Returns an error if no authorization servers are available or if the
// preferred server is not found in the list.
func selectAuthorizationServer(metadata *ProtectedResourceMetadata, preferredServer string) (string, error) {
	if len(metadata.AuthorizationServers) == 0 {
		return "", fmt.Errorf("no authorization servers available")
	}

	// If preferred server is specified, try to find it
	if preferredServer != "" {
		for _, server := range metadata.AuthorizationServers {
			if server == preferredServer {
				return server, nil
			}
		}
		return "", fmt.Errorf("preferred authorization server not found: %s", preferredServer)
	}

	// Default: use first server in the list
	return metadata.AuthorizationServers[0], nil
}
