package agent

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
)

// callbackServerConfig holds configuration for the OAuth callback server
type callbackServerConfig struct {
	redirectURL string
	logger      *Logger
}

// callbackResult contains the result from the OAuth callback
type callbackResult struct {
	params map[string]string
	err    error
}

// handleMCPOAuthFlow handles OAuth authorization using mcp-go's built-in OAuth handler
func (c *Client) handleMCPOAuthFlow(ctx context.Context, oauthHandler *transport.OAuthHandler) error {
	c.logger.Info("OAuth authorization required")

	// Check if client needs to be registered (Dynamic Client Registration)
	if oauthHandler.GetClientID() == "" {
		c.logger.Info("No client ID configured, attempting dynamic client registration...")

		// Use semantic version in client name
		clientName := "mcp-debug"
		if c.version != "" && c.version != "dev" {
			clientName = fmt.Sprintf("mcp-debug/%s", c.version)
		}

		err := oauthHandler.RegisterClient(ctx, clientName)
		if err != nil {
			c.logger.Warning("Dynamic client registration failed: %v", err)
			c.logger.Info("You may need to manually register a client and provide --oauth-client-id")
			return fmt.Errorf("client registration failed: %w", err)
		}
		c.logger.Success("Client registered successfully with ID: %s", oauthHandler.GetClientID())
	}

	// Generate PKCE parameters
	codeVerifier, err := client.GenerateCodeVerifier()
	if err != nil {
		return fmt.Errorf("failed to generate code verifier: %w", err)
	}
	codeChallenge := client.GenerateCodeChallenge(codeVerifier)

	// Generate state parameter
	state, err := client.GenerateState()
	if err != nil {
		return fmt.Errorf("failed to generate state: %w", err)
	}

	// Get authorization URL from mcp-go's handler
	authURL, err := oauthHandler.GetAuthorizationURL(ctx, state, codeChallenge)
	if err != nil {
		return fmt.Errorf("failed to get authorization URL: %w", err)
	}

	// Start callback server
	config := &callbackServerConfig{
		redirectURL: c.oauthConfig.RedirectURL,
		logger:      c.logger,
	}
	server, resultChan, err := startCallbackServer(config)
	if err != nil {
		return fmt.Errorf("failed to start callback server: %w", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			c.logger.Warning("Failed to shutdown callback server: %v", err)
		}
	}()

	// Open browser
	c.logger.Info("Opening browser for authorization...")
	c.logger.Info("Authorization URL: %s", authURL)
	if err := openBrowser(authURL); err != nil {
		c.logger.Warning("Could not open browser automatically: %v", err)
		c.logger.Info("Please open this URL in your browser:")
		c.logger.Info("%s", authURL)
	}

	// Wait for callback
	c.logger.Info("Waiting for authorization...")

	timeout := c.oauthConfig.AuthorizationTimeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	var result callbackResult
	select {
	case result = <-resultChan:
		if result.err != nil {
			return result.err
		}
	case <-time.After(timeout):
		return fmt.Errorf("authorization timeout after %v", timeout)
	case <-ctx.Done():
		return ctx.Err()
	}

	params := result.params

	// Verify state
	if params["state"] != state {
		return fmt.Errorf("state mismatch (CSRF protection)")
	}

	code := params["code"]
	if code == "" {
		return fmt.Errorf("no authorization code received")
	}

	c.logger.Success("Authorization code received")
	c.logger.Info("Exchanging code for access token...")

	// Use mcp-go's handler to process the authorization response
	err = oauthHandler.ProcessAuthorizationResponse(ctx, code, state, codeVerifier)
	if err != nil {
		return fmt.Errorf("failed to exchange authorization code: %w", err)
	}

	c.logger.Success("Access token obtained successfully!")
	return nil
}

// startCallbackServer starts an HTTP server to receive OAuth callbacks
func startCallbackServer(config *callbackServerConfig) (*http.Server, <-chan callbackResult, error) {
	parsedURL, err := url.Parse(config.redirectURL)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid redirect URI: %w", err)
	}

	resultChan := make(chan callbackResult, 1)

	// Create isolated ServeMux to avoid conflicts with global http.DefaultServeMux
	mux := http.NewServeMux()
	mux.HandleFunc(parsedURL.Path, createCallbackHandler(config.logger, resultChan))

	// Create server with security timeouts
	server := &http.Server{
		Addr:         parsedURL.Host,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	// Start server in background
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			select {
			case resultChan <- callbackResult{err: fmt.Errorf("callback server error: %w", err)}:
			default:
				config.logger.Warning("Server error occurred but callback already processed")
			}
		}
	}()

	return server, resultChan, nil
}

// createCallbackHandler creates an HTTP handler for OAuth callbacks
func createCallbackHandler(logger *Logger, resultChan chan<- callbackResult) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Security: Only accept GET requests (standard for OAuth callbacks)
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		params := make(map[string]string)
		for key, values := range r.URL.Query() {
			if len(values) > 0 {
				params[key] = values[0]
			}
		}

		// Check for error response
		if params["error"] != "" {
			select {
			case resultChan <- callbackResult{
				err: fmt.Errorf("authorization error: %s - %s", params["error"], params["error_description"]),
			}:
			default:
				logger.Warning("Error occurred but callback already processed")
			}
			http.Error(w, "Authorization failed", http.StatusBadRequest)
			return
		}

		// Success - send params
		select {
		case resultChan <- callbackResult{params: params}:
		default:
			logger.Warning("Callback received but already processed")
		}

		w.Header().Set("Content-Type", "text/html")
		if _, err := w.Write([]byte(`<html><body><h1>✅ Authorization Successful!</h1><p>You can close this window.</p></body></html>`)); err != nil {
			logger.Warning("Failed to write response: %v", err)
		}
	}
}

// openBrowser opens the specified URL in the default browser.
// It validates the URL scheme and uses platform-specific commands.
func openBrowser(urlStr string) error {
	// Security: Validate URL scheme before opening in browser
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("invalid URL scheme for browser: %s (only http/https allowed)", parsedURL.Scheme)
	}

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", urlStr)
	case "darwin":
		cmd = exec.Command("open", urlStr)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", urlStr)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}
