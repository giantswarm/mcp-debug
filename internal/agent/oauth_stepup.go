// Package agent implements step-up authorization per MCP OAuth 2.1 spec.
//
// Step-up authorization handles runtime 403 Forbidden responses with
// insufficient_scope errors by automatically requesting additional permissions
// and retrying the operation.
package agent

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

// scopeRetryTracker tracks retry attempts per resource/operation to prevent
// infinite authorization loops during step-up authorization.
type scopeRetryTracker struct {
	mu         sync.Mutex
	attempts   map[string]int // key: "resource:operation"
	maxRetries int
}

// newScopeRetryTracker creates a new retry tracker with the specified maximum retries
func newScopeRetryTracker(maxRetries int) *scopeRetryTracker {
	return &scopeRetryTracker{
		attempts:   make(map[string]int),
		maxRetries: maxRetries,
	}
}

// shouldRetry checks if another retry attempt is allowed for the given resource and operation.
// Returns true if retry is allowed, false if max retries exceeded.
func (t *scopeRetryTracker) shouldRetry(resource, operation string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	key := fmt.Sprintf("%s:%s", resource, operation)
	if t.attempts[key] >= t.maxRetries {
		return false
	}
	t.attempts[key]++
	return true
}

// reset clears the retry count for a specific resource/operation combination
// after a successful operation.
func (t *scopeRetryTracker) reset(resource, operation string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	key := fmt.Sprintf("%s:%s", resource, operation)
	delete(t.attempts, key)
}

// getAttempts returns the current number of attempts for a resource/operation
func (t *scopeRetryTracker) getAttempts(resource, operation string) int {
	t.mu.Lock()
	defer t.mu.Unlock()

	key := fmt.Sprintf("%s:%s", resource, operation)
	return t.attempts[key]
}

// detectInsufficientScope checks if an HTTP response indicates an insufficient_scope error
// per RFC 6750 Section 3 and extracts the challenge information.
//
// Returns the parsed challenge if insufficient_scope is detected, nil otherwise.
func detectInsufficientScope(resp *http.Response) (*WWWAuthenticateChallenge, error) {
	// Must be 403 Forbidden
	if resp.StatusCode != http.StatusForbidden {
		return nil, nil
	}

	// Must have WWW-Authenticate header
	wwwAuth := resp.Header.Get("WWW-Authenticate")
	if wwwAuth == "" {
		return nil, nil
	}

	// Parse the challenge
	challenge, err := parseWWWAuthenticate(wwwAuth)
	if err != nil {
		return nil, fmt.Errorf("failed to parse WWW-Authenticate header: %w", err)
	}

	// Must have error="insufficient_scope"
	if challenge.Error != "insufficient_scope" {
		return nil, nil
	}

	return challenge, nil
}

// stepUpRoundTripper is an HTTP RoundTripper that intercepts 403 Forbidden responses
// with insufficient_scope errors and triggers step-up authorization.
type stepUpRoundTripper struct {
	base            http.RoundTripper
	config          *OAuthConfig
	retryTracker    *scopeRetryTracker
	logger          *Logger
	reauthorizeFunc func(ctx context.Context, newScopes []string) error
}

// newStepUpRoundTripper creates a new step-up round tripper.
//
// The reauthorizeFunc should handle the actual re-authorization flow with new scopes.
// It will be called when insufficient_scope is detected and retries are available.
func newStepUpRoundTripper(
	config *OAuthConfig,
	base http.RoundTripper,
	logger *Logger,
	reauthorizeFunc func(ctx context.Context, newScopes []string) error,
) *stepUpRoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}

	maxRetries := config.StepUpMaxRetries
	if maxRetries <= 0 {
		maxRetries = 2 // Safe default
	}

	return &stepUpRoundTripper{
		base:            base,
		config:          config,
		retryTracker:    newScopeRetryTracker(maxRetries),
		logger:          logger,
		reauthorizeFunc: reauthorizeFunc,
	}
}

// RoundTrip implements http.RoundTripper by intercepting responses and handling
// insufficient_scope errors through step-up authorization.
func (rt *stepUpRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Skip if step-up is disabled
	if !rt.config.EnableStepUpAuth {
		return rt.base.RoundTrip(req)
	}

	// Execute the request
	resp, err := rt.base.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	// Check for insufficient_scope error
	challenge, err := detectInsufficientScope(resp)
	if err != nil {
		rt.logger.Warning("Error detecting insufficient_scope: %v", err)
		return resp, nil // Return original response
	}

	// No insufficient_scope detected
	if challenge == nil {
		// Successful request - reset retry counter
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			rt.retryTracker.reset(req.URL.Host, req.Method)
		}
		return resp, nil
	}

	// Insufficient scope detected
	if rt.logger != nil {
		rt.logger.Warning("Insufficient scope detected for %s %s", req.Method, req.URL.Path)
		if challenge.ErrorDescription != "" {
			rt.logger.Info("Server message: %s", challenge.ErrorDescription)
		}
	}

	// Check retry limit
	resource := req.URL.Host
	operation := req.Method

	if !rt.retryTracker.shouldRetry(resource, operation) {
		attempts := rt.retryTracker.getAttempts(resource, operation)
		if rt.logger != nil {
			rt.logger.Error("Max retries (%d) exceeded for step-up authorization on %s %s",
				rt.config.StepUpMaxRetries, req.Method, req.URL.Path)
		}
		// Close original response body
		resp.Body.Close()
		return nil, fmt.Errorf("max step-up authorization retries (%d) exceeded for %s %s (attempts: %d)",
			rt.config.StepUpMaxRetries, req.Method, req.URL.Path, attempts)
	}

	attempts := rt.retryTracker.getAttempts(resource, operation)
	if rt.logger != nil {
		rt.logger.Info("Step-up authorization attempt %d/%d",
			attempts, rt.config.StepUpMaxRetries)
	}

	// Extract required scopes
	if len(challenge.Scopes) == 0 {
		if rt.logger != nil {
			rt.logger.Warning("Insufficient_scope error without scope parameter - cannot determine required scopes")
		}
		resp.Body.Close()
		return nil, fmt.Errorf("insufficient_scope error without scope parameter")
	}

	if rt.logger != nil {
		rt.logger.Info("Required scopes: %v", challenge.Scopes)
	}

	// User prompt if enabled
	if rt.config.StepUpUserPrompt {
		if !rt.promptUserForStepUp(challenge.Scopes) {
			if rt.logger != nil {
				rt.logger.Info("User declined step-up authorization")
			}
			resp.Body.Close()
			return nil, fmt.Errorf("user declined step-up authorization")
		}
	}

	// Close original response body before re-authorization
	resp.Body.Close()

	// Trigger re-authorization with new scopes
	if rt.logger != nil {
		rt.logger.Info("Requesting additional permissions...")
	}
	if err := rt.reauthorizeFunc(req.Context(), challenge.Scopes); err != nil {
		return nil, fmt.Errorf("step-up re-authorization failed: %w", err)
	}

	if rt.logger != nil {
		rt.logger.Success("Additional permissions granted")
	}

	// Clone and retry the original request
	// We need to reconstruct the request body if it was consumed
	clonedReq := req.Clone(req.Context())

	// If the original request had a body, we need to handle it carefully
	// For most OAuth/MCP operations, bodies are small enough to buffer
	if req.Body != nil && req.GetBody != nil {
		newBody, err := req.GetBody()
		if err != nil {
			return nil, fmt.Errorf("failed to get request body for retry: %w", err)
		}
		clonedReq.Body = newBody
	}

	if rt.logger != nil {
		rt.logger.Info("Retrying request with new token...")
	}
	retryResp, retryErr := rt.base.RoundTrip(clonedReq)
	if retryErr != nil {
		return nil, fmt.Errorf("retry after step-up authorization failed: %w", retryErr)
	}

	// Check if retry was successful
	if retryResp.StatusCode >= 200 && retryResp.StatusCode < 300 {
		if rt.logger != nil {
			rt.logger.Success("Request successful after step-up authorization")
		}
		rt.retryTracker.reset(resource, operation)
	}

	return retryResp, nil
}

// promptUserForStepUp asks the user whether to proceed with step-up authorization.
// Returns true if user approves, false otherwise.
func (rt *stepUpRoundTripper) promptUserForStepUp(newScopes []string) bool {
	// TODO: Implement interactive prompt
	// For now, this is a placeholder that always returns true
	// In a future PR, we can add interactive prompting via stdin
	if rt.logger != nil {
		rt.logger.Info("Additional permissions required: %v", newScopes)
		rt.logger.Info("Proceeding with re-authorization (automatic mode)")
	}
	return true
}

// cloneRequest creates a deep copy of an HTTP request including the body
func cloneRequest(req *http.Request) (*http.Request, error) {
	cloned := req.Clone(req.Context())

	if req.Body != nil {
		// Read the body
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}

		// Restore original body
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// Set cloned body
		cloned.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		cloned.ContentLength = int64(len(bodyBytes))
	}

	return cloned, nil
}

// mergeScopes merges existing and new scopes, removing duplicates
func mergeScopes(existing, new []string) []string {
	scopeMap := make(map[string]bool)

	// Add existing scopes
	for _, scope := range existing {
		scopeMap[scope] = true
	}

	// Add new scopes
	for _, scope := range new {
		scopeMap[scope] = true
	}

	// Convert back to slice
	var result []string
	for scope := range scopeMap {
		result = append(result, scope)
	}

	return result
}

// scopeInclusionNeeded determines if we need to include existing scopes
// along with new ones during step-up authorization.
//
// Per MCP spec, the authorization server should handle scope merging,
// but some servers may require clients to send the full desired scope set.
func scopeInclusionNeeded(existingScopes, newScopes []string) []string {
	// For now, use the new scopes as provided by the server
	// The server's WWW-Authenticate should contain the complete required scope set
	// If we find servers that don't follow this pattern, we can revisit
	return newScopes
}

// formatScopeList formats a scope list for display
func formatScopeList(scopes []string) string {
	if len(scopes) == 0 {
		return "(none)"
	}
	return strings.Join(scopes, ", ")
}
