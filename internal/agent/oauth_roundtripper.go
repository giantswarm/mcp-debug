package agent

import (
	"net/http"
	"strings"
)

// registrationTokenRoundTripper is an HTTP RoundTripper that adds a registration access token
// to Dynamic Client Registration requests
type registrationTokenRoundTripper struct {
	transport         http.RoundTripper
	registrationToken string
}

// newRegistrationTokenRoundTripper creates a new RoundTripper that injects the registration token
// into DCR requests (identified by POST requests to endpoints containing "register" or "registration")
func newRegistrationTokenRoundTripper(registrationToken string, base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return &registrationTokenRoundTripper{
		transport:         base,
		registrationToken: registrationToken,
	}
}

// RoundTrip implements the http.RoundTripper interface
func (rt *registrationTokenRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid modifying the original
	clonedReq := req.Clone(req.Context())

	// Check if this is a DCR request (POST to registration endpoint)
	// Per RFC 7591, the registration endpoint URL typically contains "register" or "registration"
	if clonedReq.Method == http.MethodPost &&
		(strings.Contains(strings.ToLower(clonedReq.URL.Path), "register") ||
			strings.Contains(strings.ToLower(clonedReq.URL.Path), "registration")) {
		// Add the registration access token as a Bearer token
		// Per RFC 7591 Section 3.2, this is required for authenticated DCR
		if rt.registrationToken != "" {
			clonedReq.Header.Set("Authorization", "Bearer "+rt.registrationToken)
		}
	}

	// Use the underlying transport to execute the request
	return rt.transport.RoundTrip(clonedReq)
}
