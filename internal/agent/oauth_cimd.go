// Package agent implements Client ID Metadata Document support
// per draft-ietf-oauth-client-id-metadata-document-00.
//
// Client ID Metadata Documents allow clients to use HTTPS URLs as client identifiers,
// enabling clients to self-assert metadata without pre-registration.
//
// Security:
// - HTTPS is REQUIRED for client_id URLs (HTTP not allowed)
// - client_id URL MUST contain a path component
// - localhost redirect URIs have specific security implications (see spec Section 6)
package agent

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ClientMetadataDocument represents an OAuth Client ID Metadata Document
// as defined in draft-ietf-oauth-client-id-metadata-document-00.
//
// When a client uses an HTTPS URL as its client_id, the Authorization Server
// can fetch this document to retrieve the client's metadata.
type ClientMetadataDocument struct {
	// ClientID is the HTTPS URL that identifies this client
	// REQUIRED: Must use https scheme and include a path component
	ClientID string `json:"client_id"`

	// ClientName is the human-readable name of the client
	ClientName string `json:"client_name,omitempty"`

	// ClientURI is the URL of the client's home page
	ClientURI string `json:"client_uri,omitempty"`

	// LogoURI is the URL of the client's logo image
	LogoURI string `json:"logo_uri,omitempty"`

	// RedirectURIs are the redirect URIs for OAuth callbacks
	// REQUIRED for authorization code grant
	RedirectURIs []string `json:"redirect_uris"`

	// GrantTypes are the OAuth grant types the client will use
	// Default: ["authorization_code"]
	GrantTypes []string `json:"grant_types,omitempty"`

	// ResponseTypes are the OAuth response types the client will use
	// Default: ["code"]
	ResponseTypes []string `json:"response_types,omitempty"`

	// TokenEndpointAuthMethod is the authentication method for the token endpoint
	// Default: "none" for public clients
	TokenEndpointAuthMethod string `json:"token_endpoint_auth_method,omitempty"`
}

const (
	// Maximum size for client metadata documents (100KB)
	// Smaller than other metadata documents as client metadata should be concise
	maxClientMetadataSize = 100 * 1024

	// HTTP timeout for client metadata requests
	clientMetadataRequestTimeout = 10 * time.Second
)

// GenerateClientMetadata generates a Client ID Metadata Document for mcp-debug.
//
// The document describes the client's OAuth configuration and is meant to be:
//  1. Hosted at the client_id URL for AS discovery
//  2. Generated for user review/manual hosting
//  3. Used as template for custom configurations
func GenerateClientMetadata(config *OAuthConfig) (*ClientMetadataDocument, error) {
	if config.ClientIDMetadataURL == "" {
		return nil, fmt.Errorf("ClientIDMetadataURL is required for CIMD generation")
	}

	// Validate the client_id URL meets CIMD requirements
	if err := ValidateClientIDURL(config.ClientIDMetadataURL); err != nil {
		return nil, fmt.Errorf("invalid client_id URL: %w", err)
	}

	doc := &ClientMetadataDocument{
		ClientID:                config.ClientIDMetadataURL,
		ClientName:              "mcp-debug",
		ClientURI:               "https://github.com/giantswarm/mcp-debug",
		RedirectURIs:            []string{config.RedirectURL},
		GrantTypes:              []string{"authorization_code"},
		ResponseTypes:           []string{"code"},
		TokenEndpointAuthMethod: "none", // Public client
	}

	return doc, nil
}

// ValidateClientIDURL validates that a client_id URL meets CIMD requirements
// per draft-ietf-oauth-client-id-metadata-document-00 Section 2.
//
// Requirements:
//   - MUST use https scheme (HTTP is not allowed)
//   - MUST contain a path component (cannot be just https://example.com)
//   - MUST be a valid absolute URL
func ValidateClientIDURL(clientIDURL string) error {
	if clientIDURL == "" {
		return fmt.Errorf("client_id URL cannot be empty")
	}

	parsed, err := url.Parse(clientIDURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	// Must be absolute URL
	if !parsed.IsAbs() {
		return fmt.Errorf("client_id URL must be absolute")
	}

	// MUST use https scheme (per spec Section 2)
	// Note: HTTP is explicitly not allowed even for localhost
	if parsed.Scheme != schemeHTTPS {
		return fmt.Errorf("client_id URL must use https scheme, got: %s", parsed.Scheme)
	}

	// Host is required
	if parsed.Host == "" {
		return fmt.Errorf("client_id URL missing host")
	}

	// MUST contain path component (per spec Section 2)
	if parsed.Path == "" || parsed.Path == "/" {
		return fmt.Errorf("client_id URL must contain a path component (cannot be just https://%s)", parsed.Host)
	}

	return nil
}

// FetchClientMetadata fetches and parses a Client ID Metadata Document
// from the specified HTTPS URL.
//
// This is typically used by Authorization Servers to discover client metadata,
// but can also be used for validation/testing.
func FetchClientMetadata(ctx context.Context, clientIDURL string) (*ClientMetadataDocument, error) {
	return fetchClientMetadataWithClient(ctx, clientIDURL, nil)
}

// fetchClientMetadataWithClient allows injecting a custom HTTP client for testing
func fetchClientMetadataWithClient(ctx context.Context, clientIDURL string, httpClient *http.Client) (*ClientMetadataDocument, error) {
	// Validate URL before fetching
	if err := ValidateClientIDURL(clientIDURL); err != nil {
		return nil, err
	}

	// Create HTTP client with timeout and secure TLS configuration
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: clientMetadataRequestTimeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS12,
					// Use secure defaults: modern cipher suites only
					// Go's default cipher suite selection is secure for TLS 1.2+
				},
			},
		}
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, clientIDURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set appropriate headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)

	// Execute request
	resp, err := httpClient.Do(req)
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
		return nil, fmt.Errorf("unexpected Content-Type: %s (expected application/json)", contentType)
	}

	// Read response body with size limit
	limitedReader := io.LimitReader(resp.Body, maxClientMetadataSize)
	bodyBytes, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check if response was truncated
	if int64(len(bodyBytes)) >= maxClientMetadataSize {
		return nil, fmt.Errorf("response exceeds maximum size of %d bytes", maxClientMetadataSize)
	}

	// Parse JSON
	var metadata ClientMetadataDocument
	if err := json.Unmarshal(bodyBytes, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Validate the fetched metadata
	if err := ValidateClientMetadata(&metadata); err != nil {
		return nil, fmt.Errorf("invalid client metadata: %w", err)
	}

	return &metadata, nil
}

// ValidateClientMetadata validates a Client ID Metadata Document structure
// per draft-ietf-oauth-client-id-metadata-document-00.
func ValidateClientMetadata(metadata *ClientMetadataDocument) error {
	// client_id is required and must be valid
	if err := ValidateClientIDURL(metadata.ClientID); err != nil {
		return fmt.Errorf("invalid client_id: %w", err)
	}

	// redirect_uris is required for authorization code grant
	if len(metadata.RedirectURIs) == 0 {
		return fmt.Errorf("redirect_uris is required (at least one)")
	}

	// Validate each redirect URI
	for i, uri := range metadata.RedirectURIs {
		parsed, err := url.Parse(uri)
		if err != nil {
			return fmt.Errorf("invalid redirect_uri at index %d: %w", i, err)
		}

		if !parsed.IsAbs() {
			return fmt.Errorf("redirect_uri at index %d must be absolute: %s", i, uri)
		}

		// Security: Only allow HTTP for localhost/loopback addresses
		// HTTPS is allowed for any host (per spec Section 6: Security Considerations)
		if parsed.Scheme == schemeHTTP {
			hostname := parsed.Hostname()
			// Note: Hostname() strips brackets from IPv6 addresses, so [::1] becomes ::1
			// Accept various forms of localhost: localhost, 127.0.0.1, ::1, and expanded IPv6 0:0:0:0:0:0:0:1
			isLocalhost := hostname == hostLocal ||
				hostname == hostLoopback ||
				hostname == "::1" ||
				hostname == "0:0:0:0:0:0:0:1"
			if !isLocalhost {
				return fmt.Errorf("redirect_uri at index %d: HTTP scheme only allowed for localhost/127.0.0.1/[::1], got %s", i, hostname)
			}
		} else if parsed.Scheme != schemeHTTPS {
			return fmt.Errorf("redirect_uri at index %d must use http or https scheme: %s", i, uri)
		}
	}

	return nil
}

// SupportsClientIDMetadata checks if the Authorization Server supports
// Client ID Metadata Documents based on its metadata.
func SupportsClientIDMetadata(asMetadata *AuthorizationServerMetadata) bool {
	return asMetadata != nil && asMetadata.ClientIDMetadataDocumentSupported
}

// ValidateCIMDConsistency fetches a Client ID Metadata Document and validates
// that the client_id field in the document matches the URL it was fetched from.
//
// This is a security check to catch configuration errors early, especially when
// users provide custom CIMD URLs. The CIMD spec requires that the client_id in
// the document matches the URL where it is hosted.
//
// Returns nil if validation passes, or an error describing the mismatch.
func ValidateCIMDConsistency(ctx context.Context, cimdURL string) error {
	return validateCIMDConsistencyWithClient(ctx, cimdURL, nil)
}

// validateCIMDConsistencyWithClient allows injecting a custom HTTP client for testing
func validateCIMDConsistencyWithClient(ctx context.Context, cimdURL string, httpClient *http.Client) error {
	// Fetch the metadata document
	metadata, err := fetchClientMetadataWithClient(ctx, cimdURL, httpClient)
	if err != nil {
		return fmt.Errorf("failed to fetch CIMD from %s: %w", cimdURL, err)
	}

	// Verify client_id matches the URL
	// Per CIMD spec, the client_id in the document MUST match the URL where it's hosted
	if metadata.ClientID != cimdURL {
		return fmt.Errorf("CIMD consistency check failed: client_id in document (%s) does not match URL (%s)",
			metadata.ClientID, cimdURL)
	}

	return nil
}
