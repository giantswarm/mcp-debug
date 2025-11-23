package agent

import (
	"fmt"
	"net/http"
	"strings"
)

// registrationTokenRoundTripper is an HTTP RoundTripper that adds a registration access token
// to Dynamic Client Registration requests
type registrationTokenRoundTripper struct {
	transport         http.RoundTripper
	registrationToken string
	logger            *Logger
}

// newRegistrationTokenRoundTripper creates a new RoundTripper that injects the registration token
// into DCR requests (identified by POST requests to registration endpoints)
func newRegistrationTokenRoundTripper(registrationToken string, base http.RoundTripper, logger *Logger) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return &registrationTokenRoundTripper{
		transport:         base,
		registrationToken: registrationToken,
		logger:            logger,
	}
}

// isRegistrationEndpoint checks if a URL path matches a known DCR endpoint pattern
// per RFC 7591. This uses precise matching to prevent token leakage to unintended endpoints.
func isRegistrationEndpoint(path string) bool {
	// Normalize the path: lowercase and remove trailing slash
	path = strings.ToLower(strings.TrimSuffix(path, "/"))

	// Match common DCR endpoint patterns per RFC 7591
	// These patterns cover standard OAuth 2.0 and OpenID Connect registration endpoints
	registrationPatterns := []string{
		"/register",
		"/registration",
		"/oauth/register",
		"/oauth2/register",
		"/connect/register",
		"/oauth/registration",
		"/oauth2/registration",
		"/connect/registration",
		"/.well-known/openid-registration",
	}

	for _, pattern := range registrationPatterns {
		// Match exact path or path ending with the pattern
		if path == pattern || strings.HasSuffix(path, pattern) {
			return true
		}
	}

	return false
}

// RoundTrip implements the http.RoundTripper interface
func (rt *registrationTokenRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid modifying the original
	clonedReq := req.Clone(req.Context())

	// Check if this is a DCR request (POST to registration endpoint)
	if clonedReq.Method == http.MethodPost && isRegistrationEndpoint(clonedReq.URL.Path) {
		// Security: Only inject token over HTTPS to prevent credential exposure
		// Per OAuth 2.0 Security Best Current Practice (BCP), sensitive credentials
		// MUST only be transmitted over secure channels
		if clonedReq.URL.Scheme != "https" {
			if rt.logger != nil {
				rt.logger.Error("Security: Registration token can only be sent over HTTPS, got %s", clonedReq.URL.Scheme)
			}
			return nil, fmt.Errorf("security: registration token can only be sent over HTTPS, refusing to send over %s", clonedReq.URL.Scheme)
		}

		// Security: Check for existing Authorization header to prevent accidental overwrite
		if existingAuth := clonedReq.Header.Get("Authorization"); existingAuth != "" {
			if rt.logger != nil {
				rt.logger.Warning("Authorization header already present on registration request, refusing to overwrite")
			}
			return nil, fmt.Errorf("security: authorization header already present, refusing to overwrite (potential credential conflict)")
		}

		// Add the registration access token as a Bearer token
		// Per RFC 7591 Section 3.2, this is required for authenticated DCR
		if rt.registrationToken != "" {
			clonedReq.Header.Set("Authorization", "Bearer "+rt.registrationToken)
			if rt.logger != nil {
				rt.logger.Info("Injecting registration access token for DCR request to %s", clonedReq.URL.Path)
			}
		}
	}

	// Use the underlying transport to execute the request
	return rt.transport.RoundTrip(clonedReq)
}
