// Package agent implements MCP client functionality including OAuth 2.1
// authentication with RFC 8707 Resource Indicators support.
package agent

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// deriveResourceURI derives a canonical resource URI from an endpoint URL
// per RFC 8707 (Resource Indicators for OAuth 2.0).
//
// Canonicalization rules:
//   - Lowercase scheme and host
//   - Include port if non-standard (not 80/443)
//   - Include path if necessary to identify the MCP server
//   - No trailing slash (unless semantically significant)
//   - No fragment identifiers
//   - No query parameters
//
// Examples:
//   - https://MCP.Example.Com:443/mcp -> https://mcp.example.com/mcp
//   - https://example.com:8443/mcp -> https://example.com:8443/mcp
//   - http://localhost:8090/mcp -> http://localhost:8090/mcp
func deriveResourceURI(endpoint string) (string, error) {
	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("failed to parse endpoint URL: %w", err)
	}

	// Validate required components
	if parsedURL.Scheme == "" {
		return "", fmt.Errorf("endpoint URL missing scheme: %s", endpoint)
	}
	if parsedURL.Host == "" {
		return "", fmt.Errorf("endpoint URL missing host: %s", endpoint)
	}

	// Normalize scheme and host to lowercase
	scheme := strings.ToLower(parsedURL.Scheme)
	host := strings.ToLower(parsedURL.Host)

	// Extract hostname and port using standard library
	hostname, port, err := net.SplitHostPort(host)
	if err != nil {
		// No port specified, use the whole host
		hostname = host
		port = ""
	}

	// Standard ports that should be omitted
	omitPort := (scheme == "https" && port == "443") || (scheme == "http" && port == "80")

	// Reconstruct host with normalized hostname and conditional port
	// net.SplitHostPort strips brackets from IPv6 addresses, so we need to add them back
	if strings.Contains(hostname, ":") {
		// IPv6 address - add brackets
		if omitPort || port == "" {
			host = "[" + hostname + "]"
		} else {
			host = "[" + hostname + "]:" + port
		}
	} else {
		// IPv4 or hostname
		if omitPort || port == "" {
			host = hostname
		} else {
			host = hostname + ":" + port
		}
	}

	// Build canonical URI
	// Include path but remove trailing slash unless it's just "/"
	path := parsedURL.Path
	if path != "/" && strings.HasSuffix(path, "/") {
		path = strings.TrimSuffix(path, "/")
	}

	resourceURI := scheme + "://" + host + path

	return resourceURI, nil
}

// resourceRoundTripper is an HTTP RoundTripper that adds the RFC 8707 resource
// parameter to OAuth authorization and token requests.
type resourceRoundTripper struct {
	base         http.RoundTripper
	resourceURI  string
	skipResource bool
	logger       *Logger
}

// newResourceRoundTripper creates a new resourceRoundTripper.
//
// If skipResource is true, the resource parameter will not be added (for testing).
// If resourceURI is empty, resource parameter will not be added.
func newResourceRoundTripper(resourceURI string, skipResource bool, base http.RoundTripper, logger *Logger) *resourceRoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return &resourceRoundTripper{
		base:         base,
		resourceURI:  resourceURI,
		skipResource: skipResource,
		logger:       logger,
	}
}

// RoundTrip implements http.RoundTripper by adding the resource parameter to OAuth requests
func (t *resourceRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Skip if resource parameter is disabled or empty
	if t.skipResource || t.resourceURI == "" {
		return t.base.RoundTrip(req)
	}

	// Clone the request to avoid modifying the original
	clonedReq := req.Clone(req.Context())

	// Check if this is an OAuth request that needs resource parameter
	if t.isOAuthRequest(clonedReq) {
		if err := t.addResourceParameter(clonedReq); err != nil {
			if t.logger != nil {
				t.logger.Warning("Failed to add resource parameter: %v", err)
			}
			// Continue with original request if adding resource parameter fails
			return t.base.RoundTrip(req)
		}
		if t.logger != nil && t.logger.verbose {
			t.logger.Info("Added resource parameter to OAuth request: %s", t.resourceURI)
		}
	}

	return t.base.RoundTrip(clonedReq)
}

// isOAuthRequest checks if the request is an OAuth authorization or token request
func (t *resourceRoundTripper) isOAuthRequest(req *http.Request) bool {
	// Check for token endpoint (POST to /token or similar)
	if req.Method == http.MethodPost {
		path := strings.ToLower(req.URL.Path)
		// Common OAuth token endpoint paths - use suffix matching to avoid false positives
		if strings.HasSuffix(path, "/token") ||
			strings.HasSuffix(path, "/oauth/token") ||
			strings.HasSuffix(path, "/oauth2/token") {
			return true
		}
	}

	// Check for authorization endpoint (GET with response_type=code)
	if req.Method == http.MethodGet {
		query := req.URL.Query()
		if query.Get("response_type") == "code" && query.Get("client_id") != "" {
			return true
		}
	}

	return false
}

// addResourceParameter adds the resource parameter to the OAuth request
func (t *resourceRoundTripper) addResourceParameter(req *http.Request) error {
	if req.Method == http.MethodGet {
		// Authorization request - add to query parameters
		query := req.URL.Query()
		query.Set("resource", t.resourceURI)
		req.URL.RawQuery = query.Encode()
		return nil
	}

	if req.Method == http.MethodPost {
		// Token request - add to POST body
		// Read the existing body
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			return fmt.Errorf("failed to read request body: %w", err)
		}
		_ = req.Body.Close()

		// Parse form data
		values, err := url.ParseQuery(string(bodyBytes))
		if err != nil {
			return fmt.Errorf("failed to parse form data: %w", err)
		}

		// Add resource parameter
		values.Set("resource", t.resourceURI)

		// Re-encode and set new body
		newBody := values.Encode()
		req.Body = io.NopCloser(strings.NewReader(newBody))
		req.ContentLength = int64(len(newBody))

		return nil
	}

	return nil
}
