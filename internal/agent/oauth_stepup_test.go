package agent

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDetectInsufficientScope(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		wwwAuth       string
		wantChallenge bool
		wantError     bool
		wantScopes    []string
		wantErrMsg    string
	}{
		{
			name:          "403 with insufficient_scope",
			statusCode:    http.StatusForbidden,
			wwwAuth:       `Bearer error="insufficient_scope", scope="files:read files:write", error_description="Additional file write permission required"`,
			wantChallenge: true,
			wantScopes:    []string{"files:read", "files:write"},
		},
		{
			name:          "403 without WWW-Authenticate",
			statusCode:    http.StatusForbidden,
			wwwAuth:       "",
			wantChallenge: false,
		},
		{
			name:          "403 with different error",
			statusCode:    http.StatusForbidden,
			wwwAuth:       `Bearer error="invalid_token"`,
			wantChallenge: false,
		},
		{
			name:          "401 with insufficient_scope (wrong status)",
			statusCode:    http.StatusUnauthorized,
			wwwAuth:       `Bearer error="insufficient_scope", scope="files:read"`,
			wantChallenge: false,
		},
		{
			name:          "200 OK (no error)",
			statusCode:    http.StatusOK,
			wwwAuth:       "",
			wantChallenge: false,
		},
		{
			name:          "403 with malformed WWW-Authenticate",
			statusCode:    http.StatusForbidden,
			wwwAuth:       "NotBearer malformed",
			wantChallenge: false,
		},
		{
			name:          "403 with insufficient_scope and no scopes",
			statusCode:    http.StatusForbidden,
			wwwAuth:       `Bearer error="insufficient_scope"`,
			wantChallenge: true,
			wantScopes:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: tt.statusCode,
				Header:     make(http.Header),
			}

			if tt.wwwAuth != "" {
				resp.Header.Set("WWW-Authenticate", tt.wwwAuth)
			}

			challenge, err := detectInsufficientScope(resp)

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error, got nil")
				} else if tt.wantErrMsg != "" && !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("expected error containing %q, got %q", tt.wantErrMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.wantChallenge {
				if challenge == nil {
					t.Errorf("expected challenge, got nil")
					return
				}
				if challenge.Error != "insufficient_scope" {
					t.Errorf("expected error=insufficient_scope, got %q", challenge.Error)
				}
				if tt.wantScopes != nil {
					if len(challenge.Scopes) != len(tt.wantScopes) {
						t.Errorf("expected %d scopes, got %d", len(tt.wantScopes), len(challenge.Scopes))
					}
					for i, scope := range tt.wantScopes {
						if i >= len(challenge.Scopes) || challenge.Scopes[i] != scope {
							t.Errorf("expected scope[%d]=%q, got %q", i, scope, challenge.Scopes[i])
						}
					}
				}
			} else {
				if challenge != nil {
					t.Errorf("expected no challenge, got %+v", challenge)
				}
			}
		})
	}
}

func TestScopeRetryTracker(t *testing.T) {
	t.Run("basic retry tracking", func(t *testing.T) {
		tracker := newScopeRetryTracker(2)

		// First attempt should succeed
		if !tracker.shouldRetry("example.com", "GET") {
			t.Error("first retry should be allowed")
		}

		// Check attempts
		if attempts := tracker.getAttempts("example.com", "GET"); attempts != 1 {
			t.Errorf("expected 1 attempt, got %d", attempts)
		}

		// Second attempt should succeed
		if !tracker.shouldRetry("example.com", "GET") {
			t.Error("second retry should be allowed")
		}

		// Third attempt should fail (exceeded max)
		if tracker.shouldRetry("example.com", "GET") {
			t.Error("third retry should be denied")
		}
	})

	t.Run("different resource/operation combinations", func(t *testing.T) {
		tracker := newScopeRetryTracker(2)

		// Different resources are tracked separately
		if !tracker.shouldRetry("example.com", "GET") {
			t.Error("first resource retry 1 should be allowed")
		}
		if !tracker.shouldRetry("other.com", "GET") {
			t.Error("second resource retry 1 should be allowed")
		}

		// Different operations are tracked separately
		if !tracker.shouldRetry("example.com", "POST") {
			t.Error("different operation retry should be allowed")
		}

		// Original combination should now be at attempt 2
		if !tracker.shouldRetry("example.com", "GET") {
			t.Error("first resource retry 2 should be allowed")
		}
		if tracker.shouldRetry("example.com", "GET") {
			t.Error("first resource retry 3 should be denied")
		}

		// Other combinations should still work
		if !tracker.shouldRetry("other.com", "GET") {
			t.Error("second resource retry 2 should be allowed")
		}
	})

	t.Run("reset clears attempts", func(t *testing.T) {
		tracker := newScopeRetryTracker(2)

		// Use up one retry
		if !tracker.shouldRetry("example.com", "GET") {
			t.Error("first retry should be allowed")
		}
		if attempts := tracker.getAttempts("example.com", "GET"); attempts != 1 {
			t.Errorf("expected 1 attempt, got %d", attempts)
		}

		// Reset
		tracker.reset("example.com", "GET")

		// Should be back to 0
		if attempts := tracker.getAttempts("example.com", "GET"); attempts != 0 {
			t.Errorf("expected 0 attempts after reset, got %d", attempts)
		}

		// Should have full retries available again
		if !tracker.shouldRetry("example.com", "GET") {
			t.Error("retry should be allowed after reset")
		}
		if !tracker.shouldRetry("example.com", "GET") {
			t.Error("second retry should be allowed after reset")
		}
		if tracker.shouldRetry("example.com", "GET") {
			t.Error("third retry should be denied")
		}
	})

	t.Run("zero max retries", func(t *testing.T) {
		tracker := newScopeRetryTracker(0)

		// Should immediately deny retries
		if tracker.shouldRetry("example.com", "GET") {
			t.Error("retry should be denied with max retries = 0")
		}
	})
}

func TestStepUpRoundTripper(t *testing.T) {
	t.Run("pass through non-403 responses", func(t *testing.T) {
		config := &OAuthConfig{
			EnableStepUpAuth: true,
			StepUpMaxRetries: 2,
		}

		callCount := 0
		baseTransport := &mockRoundTripper{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				callCount++
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("success")),
					Header:     make(http.Header),
				}, nil
			},
		}

		reauthorized := false
		rt := newStepUpRoundTripper(
			config,
			baseTransport,
			NewLogger(false, false, false),
			func(ctx context.Context, scopes []string) error {
				reauthorized = true
				return nil
			},
		)

		req := httptest.NewRequest(http.MethodGet, "https://example.com/test", nil)
		resp, err := rt.RoundTrip(req)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
		if reauthorized {
			t.Error("should not trigger reauthorization for 200 OK")
		}
		if callCount != 1 {
			t.Errorf("expected 1 call to base transport, got %d", callCount)
		}
	})

	t.Run("handle insufficient_scope with retry", func(t *testing.T) {
		config := &OAuthConfig{
			EnableStepUpAuth: true,
			StepUpMaxRetries: 2,
		}

		callCount := 0
		baseTransport := &mockRoundTripper{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				callCount++
				// First call returns 403, second call returns 200
				if callCount == 1 {
					header := make(http.Header)
					header.Set("WWW-Authenticate", `Bearer error="insufficient_scope", scope="files:read files:write"`)
					return &http.Response{
						StatusCode: http.StatusForbidden,
						Body:       io.NopCloser(strings.NewReader("forbidden")),
						Header:     header,
					}, nil
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("success")),
					Header:     make(http.Header),
				}, nil
			},
		}

		var reauthorizedScopes []string
		rt := newStepUpRoundTripper(
			config,
			baseTransport,
			NewLogger(false, false, false),
			func(ctx context.Context, scopes []string) error {
				reauthorizedScopes = scopes
				return nil
			},
		)

		req := httptest.NewRequest(http.MethodGet, "https://example.com/test", nil)
		resp, err := rt.RoundTrip(req)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200 after retry, got %d", resp.StatusCode)
		}
		if len(reauthorizedScopes) != 2 {
			t.Errorf("expected 2 scopes, got %d", len(reauthorizedScopes))
		}
		if callCount != 2 {
			t.Errorf("expected 2 calls (initial + retry), got %d", callCount)
		}
	})

	t.Run("respect max retries", func(t *testing.T) {
		config := &OAuthConfig{
			EnableStepUpAuth: true,
			StepUpMaxRetries: 2,
		}

		baseTransport := &mockRoundTripper{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				// Always return 403 insufficient_scope
				header := make(http.Header)
				header.Set("WWW-Authenticate", `Bearer error="insufficient_scope", scope="files:read"`)
				return &http.Response{
					StatusCode: http.StatusForbidden,
					Body:       io.NopCloser(strings.NewReader("forbidden")),
					Header:     header,
				}, nil
			},
		}

		reauthorizeCount := 0
		rt := newStepUpRoundTripper(
			config,
			baseTransport,
			NewLogger(false, false, false),
			func(ctx context.Context, scopes []string) error {
				reauthorizeCount++
				return nil
			},
		)

		req := httptest.NewRequest(http.MethodGet, "https://example.com/test", nil)

		// First attempt
		_, err := rt.RoundTrip(req)
		if err != nil {
			t.Fatalf("first attempt failed: %v", err)
		}
		if reauthorizeCount != 1 {
			t.Errorf("expected 1 reauthorization, got %d", reauthorizeCount)
		}

		// Second attempt
		req = httptest.NewRequest(http.MethodGet, "https://example.com/test", nil)
		_, err = rt.RoundTrip(req)
		if err != nil {
			t.Fatalf("second attempt failed: %v", err)
		}
		if reauthorizeCount != 2 {
			t.Errorf("expected 2 reauthorizations, got %d", reauthorizeCount)
		}

		// Third attempt should fail (max retries exceeded)
		req = httptest.NewRequest(http.MethodGet, "https://example.com/test", nil)
		_, err = rt.RoundTrip(req)
		if err == nil {
			t.Error("expected error when max retries exceeded")
		}
		if !strings.Contains(err.Error(), "max step-up authorization retries") {
			t.Errorf("expected max retries error, got: %v", err)
		}
		if reauthorizeCount != 2 {
			t.Errorf("expected reauthorization count to stay at 2, got %d", reauthorizeCount)
		}
	})

	t.Run("skip when step-up disabled", func(t *testing.T) {
		config := &OAuthConfig{
			EnableStepUpAuth: false, // Disabled
			StepUpMaxRetries: 2,
		}

		baseTransport := &mockRoundTripper{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				header := make(http.Header)
				header.Set("WWW-Authenticate", `Bearer error="insufficient_scope", scope="files:read"`)
				return &http.Response{
					StatusCode: http.StatusForbidden,
					Body:       io.NopCloser(strings.NewReader("forbidden")),
					Header:     header,
				}, nil
			},
		}

		reauthorized := false
		rt := newStepUpRoundTripper(
			config,
			baseTransport,
			NewLogger(false, false, false),
			func(ctx context.Context, scopes []string) error {
				reauthorized = true
				return nil
			},
		)

		req := httptest.NewRequest(http.MethodGet, "https://example.com/test", nil)
		resp, err := rt.RoundTrip(req)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("expected original 403 response, got %d", resp.StatusCode)
		}
		if reauthorized {
			t.Error("should not trigger reauthorization when disabled")
		}
	})

	t.Run("reset retry counter on successful request", func(t *testing.T) {
		config := &OAuthConfig{
			EnableStepUpAuth: true,
			StepUpMaxRetries: 2,
		}

		callCount := 0
		baseTransport := &mockRoundTripper{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				callCount++
				// First call: 403, second: 200, third: 403 again
				if callCount == 1 || callCount == 3 {
					header := make(http.Header)
					header.Set("WWW-Authenticate", `Bearer error="insufficient_scope", scope="files:read"`)
					return &http.Response{
						StatusCode: http.StatusForbidden,
						Body:       io.NopCloser(strings.NewReader("forbidden")),
						Header:     header,
					}, nil
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("success")),
					Header:     make(http.Header),
				}, nil
			},
		}

		rt := newStepUpRoundTripper(
			config,
			baseTransport,
			NewLogger(false, false, false),
			func(ctx context.Context, scopes []string) error {
				return nil
			},
		)

		// First attempt - should succeed after step-up
		req := httptest.NewRequest(http.MethodGet, "https://example.com/test", nil)
		resp, err := rt.RoundTrip(req)
		if err != nil {
			t.Fatalf("first attempt failed: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 after first step-up, got %d", resp.StatusCode)
		}

		// Second attempt - should succeed again (counter was reset)
		req = httptest.NewRequest(http.MethodGet, "https://example.com/test", nil)
		resp, err = rt.RoundTrip(req)
		if err != nil {
			t.Fatalf("second attempt failed: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 after second step-up, got %d", resp.StatusCode)
		}
	})
}

func TestMergeScopes(t *testing.T) {
	tests := []struct {
		name     string
		existing []string
		new      []string
		want     int // Expected number of unique scopes
	}{
		{
			name:     "no overlap",
			existing: []string{"read", "write"},
			new:      []string{"delete", "admin"},
			want:     4,
		},
		{
			name:     "complete overlap",
			existing: []string{"read", "write"},
			new:      []string{"read", "write"},
			want:     2,
		},
		{
			name:     "partial overlap",
			existing: []string{"read", "write", "delete"},
			new:      []string{"write", "admin"},
			want:     4,
		},
		{
			name:     "empty existing",
			existing: []string{},
			new:      []string{"read", "write"},
			want:     2,
		},
		{
			name:     "empty new",
			existing: []string{"read", "write"},
			new:      []string{},
			want:     2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeScopes(tt.existing, tt.new)
			if len(result) != tt.want {
				t.Errorf("expected %d scopes, got %d: %v", tt.want, len(result), result)
			}

			// Verify all original scopes are present
			scopeMap := make(map[string]bool)
			for _, scope := range result {
				scopeMap[scope] = true
			}

			for _, scope := range tt.existing {
				if !scopeMap[scope] {
					t.Errorf("missing existing scope: %s", scope)
				}
			}

			for _, scope := range tt.new {
				if !scopeMap[scope] {
					t.Errorf("missing new scope: %s", scope)
				}
			}
		})
	}
}

// mockRoundTripper is a test helper for mocking HTTP round trips
type mockRoundTripper struct {
	roundTripFunc func(*http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTripFunc(req)
}

func TestCloneRequest(t *testing.T) {
	t.Run("clone request without body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "https://example.com/test", nil)
		cloned, err := cloneRequest(req)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cloned.Method != req.Method {
			t.Errorf("method mismatch: expected %s, got %s", req.Method, cloned.Method)
		}
		if cloned.URL.String() != req.URL.String() {
			t.Errorf("URL mismatch: expected %s, got %s", req.URL.String(), cloned.URL.String())
		}
	})

	t.Run("clone request with body", func(t *testing.T) {
		body := "test request body"
		req := httptest.NewRequest(http.MethodPost, "https://example.com/test", bytes.NewBufferString(body))

		cloned, err := cloneRequest(req)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Read cloned body
		clonedBody, err := io.ReadAll(cloned.Body)
		if err != nil {
			t.Fatalf("failed to read cloned body: %v", err)
		}
		if string(clonedBody) != body {
			t.Errorf("body mismatch: expected %s, got %s", body, string(clonedBody))
		}

		// Original body should also be readable (restored)
		originalBody, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("failed to read original body: %v", err)
		}
		if string(originalBody) != body {
			t.Errorf("original body changed: expected %s, got %s", body, string(originalBody))
		}
	})
}

func TestFormatScopeList(t *testing.T) {
	tests := []struct {
		name   string
		scopes []string
		want   string
	}{
		{
			name:   "empty list",
			scopes: []string{},
			want:   "(none)",
		},
		{
			name:   "nil list",
			scopes: nil,
			want:   "(none)",
		},
		{
			name:   "single scope",
			scopes: []string{"read"},
			want:   "read",
		},
		{
			name:   "multiple scopes",
			scopes: []string{"read", "write", "delete"},
			want:   "read, write, delete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatScopeList(tt.scopes)
			if got != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
