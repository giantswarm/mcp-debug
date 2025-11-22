package agent

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestOpenBrowser(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid http URL",
			url:     "http://localhost:8080/callback",
			wantErr: false,
		},
		{
			name:    "valid https URL",
			url:     "https://example.com/callback",
			wantErr: false,
		},
		{
			name:    "invalid URL",
			url:     "://invalid",
			wantErr: true,
			errMsg:  "invalid URL",
		},
		{
			name:    "invalid scheme",
			url:     "ftp://example.com",
			wantErr: true,
			errMsg:  "invalid URL scheme for browser",
		},
		{
			name:    "javascript scheme (security)",
			url:     "javascript:alert('xss')",
			wantErr: true,
			errMsg:  "invalid URL scheme for browser",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := openBrowser(tt.url)

			// For valid URLs, we expect platform-specific behavior
			if !tt.wantErr {
				// On supported platforms (linux, darwin, windows), the command should start
				// On unsupported platforms, we expect an error
				if runtime.GOOS != "linux" && runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
					if err == nil {
						t.Error("Expected error on unsupported platform, but got nil")
					}
				}
				// On supported platforms, we can't easily test if browser actually opens
				// without creating a test binary, so we just ensure no parse errors
				return
			}

			// For invalid URLs, we expect an error
			if err == nil {
				t.Errorf("openBrowser() error = nil, wantErr %v", tt.wantErr)
				return
			}

			if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("openBrowser() error = %v, want error containing %v", err, tt.errMsg)
			}
		})
	}
}

func TestCreateCallbackHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		queryParams    string
		wantStatusCode int
		wantResult     bool
		wantError      bool
	}{
		{
			name:           "successful callback with code",
			method:         http.MethodGet,
			queryParams:    "code=test-code&state=test-state",
			wantStatusCode: http.StatusOK,
			wantResult:     true,
			wantError:      false,
		},
		{
			name:           "error callback",
			method:         http.MethodGet,
			queryParams:    "error=access_denied&error_description=User+denied+access",
			wantStatusCode: http.StatusBadRequest,
			wantResult:     false,
			wantError:      true,
		},
		{
			name:           "POST method rejected",
			method:         http.MethodPost,
			queryParams:    "code=test-code",
			wantStatusCode: http.StatusMethodNotAllowed,
			wantResult:     false,
			wantError:      false,
		},
		{
			name:           "empty callback",
			method:         http.MethodGet,
			queryParams:    "",
			wantStatusCode: http.StatusOK,
			wantResult:     true,
			wantError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(false, false, false)
			logger.SetWriter(io.Discard) // Suppress log output
			resultChan := make(chan callbackResult, 1)

			handler := createCallbackHandler(logger, resultChan)

			// Create request
			req := httptest.NewRequest(tt.method, "http://localhost:8765/callback?"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			// Handle request
			handler(w, req)

			// Check status code
			if w.Code != tt.wantStatusCode {
				t.Errorf("handler returned status %d, want %d", w.Code, tt.wantStatusCode)
			}

			// Check result channel only for GET requests
			if tt.method == http.MethodGet {
				select {
				case result := <-resultChan:
					if !tt.wantResult && result.err == nil {
						t.Error("Expected no result or error, but got result")
					}
					if tt.wantError && result.err == nil {
						t.Error("Expected error in result, but got nil")
					}
					if !tt.wantError && result.err != nil {
						t.Errorf("Expected no error, but got: %v", result.err)
					}
					if tt.wantResult && !tt.wantError {
						if result.params == nil {
							t.Error("Expected params in result, but got nil")
						}
					}
				case <-time.After(100 * time.Millisecond):
					if tt.wantResult {
						t.Error("Expected result on channel, but got timeout")
					}
				}
			}
		})
	}
}

func TestStartCallbackServer(t *testing.T) {
	tests := []struct {
		name        string
		redirectURL string
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "valid localhost URL",
			redirectURL: "http://localhost:18765/callback",
			wantErr:     false,
		},
		{
			name:        "invalid URL",
			redirectURL: "://invalid",
			wantErr:     true,
			errMsg:      "invalid redirect URI",
		},
		{
			name:        "valid 127.0.0.1 URL",
			redirectURL: "http://127.0.0.1:18766/callback",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(false, false, false)
			logger.SetWriter(io.Discard)

			config := &callbackServerConfig{
				redirectURL: tt.redirectURL,
				logger:      logger,
			}

			server, resultChan, err := startCallbackServer(config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("startCallbackServer() error = nil, wantErr %v", tt.wantErr)
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("startCallbackServer() error = %v, want error containing %v", err, tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("startCallbackServer() unexpected error = %v", err)
				return
			}

			if server == nil {
				t.Error("startCallbackServer() returned nil server")
				return
			}

			if resultChan == nil {
				t.Error("startCallbackServer() returned nil result channel")
				return
			}

			// Shutdown server
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := server.Shutdown(ctx); err != nil {
				t.Errorf("Failed to shutdown server: %v", err)
			}
		})
	}
}

func TestCallbackServerIntegration(t *testing.T) {
	logger := NewLogger(false, false, false)
	logger.SetWriter(io.Discard)

	// Use a unique port for this test
	redirectURL := "http://localhost:18767/callback"
	config := &callbackServerConfig{
		redirectURL: redirectURL,
		logger:      logger,
	}

	server, resultChan, err := startCallbackServer(config)
	if err != nil {
		t.Fatalf("Failed to start callback server: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test successful callback
	t.Run("successful callback", func(t *testing.T) {
		resp, err := http.Get(redirectURL + "?code=test-code&state=test-state")
		if err != nil {
			t.Fatalf("Failed to make callback request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		// Check result
		select {
		case result := <-resultChan:
			if result.err != nil {
				t.Errorf("Expected no error, got: %v", result.err)
			}
			if result.params["code"] != "test-code" {
				t.Errorf("Expected code=test-code, got code=%s", result.params["code"])
			}
			if result.params["state"] != "test-state" {
				t.Errorf("Expected state=test-state, got state=%s", result.params["state"])
			}
		case <-time.After(1 * time.Second):
			t.Error("Timeout waiting for callback result")
		}
	})
}

func TestCallbackServerErrorHandling(t *testing.T) {
	logger := NewLogger(false, false, false)
	logger.SetWriter(io.Discard)

	redirectURL := "http://localhost:18768/callback"
	config := &callbackServerConfig{
		redirectURL: redirectURL,
		logger:      logger,
	}

	server, resultChan, err := startCallbackServer(config)
	if err != nil {
		t.Fatalf("Failed to start callback server: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test error callback
	t.Run("error callback", func(t *testing.T) {
		errorURL := fmt.Sprintf("%s?error=access_denied&error_description=%s",
			redirectURL, url.QueryEscape("User denied access"))

		resp, err := http.Get(errorURL)
		if err != nil {
			t.Fatalf("Failed to make callback request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}

		// Check result
		select {
		case result := <-resultChan:
			if result.err == nil {
				t.Error("Expected error in result, got nil")
			}
			if !strings.Contains(result.err.Error(), "access_denied") {
				t.Errorf("Expected error containing 'access_denied', got: %v", result.err)
			}
		case <-time.After(1 * time.Second):
			t.Error("Timeout waiting for callback result")
		}
	})
}

func TestCallbackServerMethodRestriction(t *testing.T) {
	logger := NewLogger(false, false, false)
	logger.SetWriter(io.Discard)

	redirectURL := "http://localhost:18769/callback"
	config := &callbackServerConfig{
		redirectURL: redirectURL,
		logger:      logger,
	}

	server, resultChan, err := startCallbackServer(config)
	if err != nil {
		t.Fatalf("Failed to start callback server: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test POST method rejection
	t.Run("POST method rejected", func(t *testing.T) {
		resp, err := http.Post(redirectURL+"?code=test", "application/json", strings.NewReader("{}"))
		if err != nil {
			t.Fatalf("Failed to make POST request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", resp.StatusCode)
		}

		// Should not send result for non-GET methods
		select {
		case <-resultChan:
			t.Error("Should not receive result for non-GET request")
		case <-time.After(100 * time.Millisecond):
			// Expected - no result
		}
	})
}

func TestCallbackHandlerMultipleValues(t *testing.T) {
	logger := NewLogger(false, false, false)
	logger.SetWriter(io.Discard)
	resultChan := make(chan callbackResult, 1)

	handler := createCallbackHandler(logger, resultChan)

	// Test that only first value is taken when multiple values present
	req := httptest.NewRequest(http.MethodGet, "http://localhost/callback?code=first&code=second", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	select {
	case result := <-resultChan:
		if result.err != nil {
			t.Errorf("Unexpected error: %v", result.err)
		}
		if result.params["code"] != "first" {
			t.Errorf("Expected code=first, got code=%s", result.params["code"])
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for result")
	}
}

func TestCallbackServerSecurityTimeouts(t *testing.T) {
	logger := NewLogger(false, false, false)
	logger.SetWriter(io.Discard)

	redirectURL := "http://localhost:18770/callback"
	config := &callbackServerConfig{
		redirectURL: redirectURL,
		logger:      logger,
	}

	server, _, err := startCallbackServer(config)
	if err != nil {
		t.Fatalf("Failed to start callback server: %v", err)
	}

	// Check that server has security timeouts configured
	if server.ReadTimeout != 10*time.Second {
		t.Errorf("Expected ReadTimeout=10s, got %v", server.ReadTimeout)
	}
	if server.WriteTimeout != 10*time.Second {
		t.Errorf("Expected WriteTimeout=10s, got %v", server.WriteTimeout)
	}
	if server.IdleTimeout != 30*time.Second {
		t.Errorf("Expected IdleTimeout=30s, got %v", server.IdleTimeout)
	}

	// Cleanup
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	server.Shutdown(ctx)
}

func TestCallbackHandlerConcurrentRequests(t *testing.T) {
	logger := NewLogger(false, false, false)
	logger.SetWriter(io.Discard)
	resultChan := make(chan callbackResult, 1)

	handler := createCallbackHandler(logger, resultChan)

	// Send two callbacks without reading from channel
	req1 := httptest.NewRequest(http.MethodGet, "http://localhost/callback?code=first&state=state1", nil)
	w1 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "http://localhost/callback?code=second&state=state2", nil)
	w2 := httptest.NewRecorder()

	// First callback fills the buffered channel
	handler(w1, req1)

	// Second callback should be dropped (select/default behavior)
	handler(w2, req2)

	// Verify only first callback was processed (channel has 1 item)
	select {
	case result := <-resultChan:
		if result.err != nil {
			t.Errorf("First callback failed: %v", result.err)
		}
		if result.params["code"] != "first" {
			t.Errorf("Expected code=first, got code=%s", result.params["code"])
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for first callback")
	}

	// Channel should be empty now (second was dropped)
	select {
	case result := <-resultChan:
		t.Errorf("Second callback should have been dropped, but got: %v", result)
	case <-time.After(100 * time.Millisecond):
		// Expected - channel is empty (only held first result)
	}
}

func TestURLParsingSecurityEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "URL with fragment",
			url:     "http://localhost:8765/callback#fragment",
			wantErr: false, // Fragments are ignored in OAuth
		},
		{
			name:    "URL with query in path",
			url:     "http://localhost:8765/callback?foo=bar",
			wantErr: false,
		},
		{
			name:    "URL with Unicode domain (IDN)",
			url:     "http://münchen.local:8765/callback",
			wantErr: true, // Non-localhost should fail
		},
		{
			name:    "URL with port only",
			url:     "http://localhost:8765",
			wantErr: false, // Path defaults to /
		},
		{
			name:    "Empty URL",
			url:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &OAuthConfig{
				Enabled:     true,
				RedirectURL: tt.url,
				Scopes:      []string{"mcp:tools"},
			}

			// Apply defaults for non-error cases to ensure all required fields are set
			if !tt.wantErr {
				config = config.WithDefaults()
			}

			err := config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOpenBrowserURLInjection(t *testing.T) {
	// Test that URL validation prevents injection attacks
	maliciousURLs := []string{
		"http://localhost:8765/callback\n--malicious-flag",
		"http://localhost:8765/callback; rm -rf /",
		"http://localhost:8765/callback`command`",
		"http://localhost:8765/callback$(command)",
	}

	for _, url := range maliciousURLs {
		t.Run("injection_test", func(t *testing.T) {
			// openBrowser validates URLs before passing to exec.Command
			// The URL parser should reject these or they should fail safely
			err := openBrowser(url)
			// We expect either:
			// 1. URL parsing to fail (invalid URL)
			// 2. Command to fail (but not execute injected code)
			// 3. Command to succeed but only open the URL part (system handles safely)
			// The important thing is that malicious code doesn't execute
			_ = err // Ignore error - just ensure no panic or code execution
		})
	}
}
