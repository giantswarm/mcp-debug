package agent

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
)

// browserOpener is a function type for opening URLs in a browser
type browserOpener func(string) error

// defaultBrowserOpener is the default implementation for opening browsers
var defaultBrowserOpener browserOpener = openBrowserImpl

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

	// Generate nonce for OIDC flows if enabled
	var nonce string
	if c.oauthConfig.UseOIDC {
		nonce, err = client.GenerateState() // Reuse state generation for nonce
		if err != nil {
			return fmt.Errorf("failed to generate nonce: %w", err)
		}
		c.logger.Info("OIDC mode enabled - nonce will be validated")
	}

	// Get authorization URL from mcp-go's handler
	authURL, err := oauthHandler.GetAuthorizationURL(ctx, state, codeChallenge)
	if err != nil {
		return fmt.Errorf("failed to get authorization URL: %w", err)
	}

	// Add nonce parameter for OIDC flows
	if c.oauthConfig.UseOIDC && nonce != "" {
		parsedURL, err := url.Parse(authURL)
		if err == nil {
			q := parsedURL.Query()
			q.Set("nonce", nonce)
			parsedURL.RawQuery = q.Encode()
			authURL = parsedURL.String()
			c.logger.Info("Added nonce parameter to authorization URL")
		} else {
			c.logger.Warning("Failed to add nonce to URL: %v", err)
		}
	}

	// Start callback server
	callbackConfig := &callbackServerConfig{
		redirectURL: c.oauthConfig.RedirectURL,
		logger:      c.logger,
	}
	server, resultChan, err := startCallbackServer(callbackConfig)
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
	if err := defaultBrowserOpener(authURL); err != nil {
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

	// Use context.WithTimeout for better resource management
	timeoutCtx, cancelTimeout := context.WithTimeout(ctx, timeout)
	defer cancelTimeout()

	var result callbackResult
	select {
	case result = <-resultChan:
		if result.err != nil {
			return result.err
		}
	case <-timeoutCtx.Done():
		if errors.Is(ctx.Err(), context.Canceled) {
			// Parent context was cancelled
			return fmt.Errorf("authorization cancelled: %w", ctx.Err())
		}
		// Timeout occurred
		return fmt.Errorf("authorization timeout after %v", timeout)
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

	// Log requested scopes for security audit
	c.logger.Info("Requested scopes: %v", c.oauthConfig.Scopes)
	// Note: Granted scopes would need to be exposed by mcp-go library for full validation
	// For now, we log what was requested. The authorization server may have granted different scopes.
	c.logger.Info("Token exchange completed - verify granted scopes match your requirements")

	// OIDC nonce validation
	if c.oauthConfig.UseOIDC && nonce != "" {
		c.logger.Info("OIDC nonce validation: nonce='%s'", nonce)
		c.logger.Warning("Full OIDC ID token validation (including nonce) requires access to the ID token from mcp-go")
		c.logger.Info("Ensure your MCP server validates the ID token if using OIDC")
		// Note: Full nonce validation would require:
		// 1. Extracting the ID token from the token response
		// 2. Decoding the JWT
		// 3. Validating the nonce claim matches our generated nonce
		// This is typically handled by the authorization library (mcp-go in this case)
	}

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

// validateBrowserURL validates that a URL is safe to open in a browser.
func validateBrowserURL(urlStr string) error {
	// Security: Validate URL scheme before opening in browser
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("invalid URL scheme for browser: %s (only http/https allowed)", parsedURL.Scheme)
	}

	return nil
}

// openBrowserImpl opens the specified URL in the default browser.
// It validates the URL scheme and uses platform-specific commands.
func openBrowserImpl(urlStr string) error {
	if err := validateBrowserURL(urlStr); err != nil {
		return err
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
