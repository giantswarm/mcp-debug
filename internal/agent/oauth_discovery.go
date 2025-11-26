// Package agent implements Protected Resource Metadata Discovery per RFC 9728.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	parts := strings.SplitN(header, " ", 2)
	if len(parts) < 1 {
		return nil, fmt.Errorf("invalid WWW-Authenticate header format")
	}

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
func parseAuthParams(params string) map[string]string {
	result := make(map[string]string)

	// Split by comma, handling quoted strings
	var current strings.Builder
	var inQuotes bool
	var key string
	var afterEquals bool

	for i := 0; i < len(params); i++ {
		ch := params[i]

		switch ch {
		case '=':
			if !inQuotes {
				key = strings.TrimSpace(current.String())
				current.Reset()
				afterEquals = true
				continue
			}
		case '"':
			if afterEquals {
				inQuotes = !inQuotes
				continue
			}
		case ',':
			if !inQuotes && afterEquals {
				value := strings.TrimSpace(current.String())
				if key != "" {
					result[key] = value
				}
				current.Reset()
				key = ""
				afterEquals = false
				continue
			}
		}

		current.WriteByte(ch)
	}

	// Handle last parameter
	if afterEquals && key != "" {
		value := strings.TrimSpace(current.String())
		result[key] = value
	}

	return result
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
		if logger != nil && logger.verbose {
			logger.Info("Using resource_metadata URL from WWW-Authenticate: %s", challenge.ResourceMetadataURL)
		}
		return fetchProtectedResourceMetadata(ctx, challenge.ResourceMetadataURL)
	}

	// Priority 2 & 3: Try well-known URIs
	wellKnownURIs, err := buildWellKnownURIs(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to build well-known URIs: %w", err)
	}

	for i, uri := range wellKnownURIs {
		if logger != nil && logger.verbose {
			logger.Info("Trying well-known URI (%d/%d): %s", i+1, len(wellKnownURIs), uri)
		}

		metadata, err := fetchProtectedResourceMetadata(ctx, uri)
		if err != nil {
			if logger != nil && logger.verbose {
				logger.Warning("Failed to fetch from %s: %v", uri, err)
			}
			continue
		}

		if logger != nil {
			logger.Info("Successfully discovered protected resource metadata from: %s", uri)
		}
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
func validateProtectedResourceMetadata(metadata *ProtectedResourceMetadata) error {
	if metadata.Resource == "" {
		return fmt.Errorf("missing required field: resource")
	}

	if len(metadata.AuthorizationServers) == 0 {
		return fmt.Errorf("missing required field: authorization_servers (at least one required)")
	}

	// Validate authorization server URLs
	for i, asURL := range metadata.AuthorizationServers {
		if _, err := url.Parse(asURL); err != nil {
			return fmt.Errorf("invalid authorization server URL at index %d: %w", i, err)
		}
	}

	return nil
}

// selectAuthorizationServer selects an authorization server from the metadata
// based on configuration preferences.
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
