package cmd

import (
	"context"
	"errors"
	"fmt"
	"mcp-debug/internal/agent"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

const (
	// transportStreamableHTTP is the only supported transport protocol
	transportStreamableHTTP = "streamable-http"
)

var (
	version         string
	endpoint        string
	timeout         time.Duration
	verbose         bool
	noColor         bool
	jsonRPC         bool
	repl            bool
	mcpServer       bool
	transport       string
	serverTransport string
	listenAddr      string

	// OAuth flags
	oauthEnabled           bool
	oauthClientID          string
	oauthClientSecret      string
	oauthScopes            []string
	oauthScopeMode         string
	oauthRedirectURL       string
	oauthUsePKCE           bool
	oauthTimeout           time.Duration
	oauthUseOIDC           bool
	oauthRegistrationToken string
	oauthResourceURI       string
	oauthSkipResource      bool
	oauthSkipResourceMeta  bool
	oauthPreferredAuthSrv  string
	oauthDisableStepUp     bool
	oauthStepUpMaxRetries  int
	oauthStepUpPrompt      bool
	oauthClientIDMetaURL   string
	oauthDisableCIMD       bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "mcp-debug",
	Short: "MCP debugging tool",
	Long: `mcp-debug is a tool for debugging MCP (Model Context Protocol) servers.

It provides an agent that can connect to MCP servers via streamable-http transport,
inspect available tools, resources, and prompts, and execute them interactively.

The tool supports multiple modes:
- Normal mode (default): Connect and wait for notifications
- REPL mode (--repl): Interactive exploration and execution
- MCP Server mode (--mcp-server): Act as an MCP server for integration with AI assistants

The agent connects to an MCP server as a client agent, 
logs all JSON-RPC communication, and demonstrates dynamic tool updates.

This is useful for debugging MCP server behavior, verifying that
tools are properly exposed, and ensuring that notifications work correctly
when tools are added or removed.

In REPL mode, you can:
- List available tools, resources, and prompts
- Get detailed information about specific items
- Execute tools interactively with JSON arguments
- View resources and retrieve their contents
- Execute prompts with arguments
- Toggle notification display

In MCP Server mode:
- The agent acts as an MCP server using stdio transport
- It exposes all REPL functionality as MCP tools
- It's designed for integration with AI assistants like Claude or Cursor
- Configure it in your AI assistant's MCP settings

By default, it connects to http://localhost:8090/mcp. You can override this with the --endpoint flag.`,
	RunE: runMCPDebug,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// SetVersion sets the version for the application
func SetVersion(v string) {
	version = v
	rootCmd.Version = v
}

func init() {
	// Add flags
	rootCmd.Flags().StringVar(&endpoint, "endpoint", "http://localhost:8090/mcp", "MCP endpoint URL (must end with /mcp)")
	rootCmd.Flags().StringVar(&transport, "transport", transportStreamableHTTP, "Transport protocol to use for client connections (streamable-http only)")
	rootCmd.Flags().StringVar(&serverTransport, "server-transport", "stdio", "Transport protocol for the MCP server itself (stdio, streamable-http)")
	rootCmd.Flags().StringVar(&listenAddr, "listen-addr", ":8899", "Listen address for streamable-http server (path is fixed to /mcp)")
	rootCmd.Flags().DurationVar(&timeout, "timeout", 5*time.Minute, "Timeout for waiting for notifications")
	rootCmd.Flags().BoolVar(&verbose, "verbose", false, "Enable verbose logging (show keepalive messages)")
	rootCmd.Flags().BoolVar(&noColor, "no-color", false, "Disable colored output")
	rootCmd.Flags().BoolVar(&jsonRPC, "json-rpc", false, "Enable full JSON-RPC message logging")
	rootCmd.Flags().BoolVar(&repl, "repl", false, "Start interactive REPL mode")
	rootCmd.Flags().BoolVar(&mcpServer, "mcp-server", false, "Run as MCP server (stdio transport)")

	// OAuth flags
	rootCmd.Flags().BoolVar(&oauthEnabled, "oauth", false, "Enable OAuth authentication for connecting to protected MCP servers")
	rootCmd.Flags().StringVar(&oauthClientID, "oauth-client-id", "", "OAuth client ID (optional - will use Dynamic Client Registration if not provided)")
	rootCmd.Flags().StringVar(&oauthClientSecret, "oauth-client-secret", "", "OAuth client secret (optional)")
	rootCmd.Flags().StringSliceVar(&oauthScopes, "oauth-scopes", []string{}, "OAuth scopes to request (optional, used with --oauth-scope-mode=manual)")
	rootCmd.Flags().StringVar(&oauthScopeMode, "oauth-scope-mode", "auto", "Scope selection mode: 'auto' (MCP spec priority, default) or 'manual' (use --oauth-scopes only)")
	rootCmd.Flags().StringVar(&oauthRedirectURL, "oauth-redirect-url", "http://localhost:8765/callback", "OAuth redirect URL for callback")
	rootCmd.Flags().BoolVar(&oauthUsePKCE, "oauth-pkce", true, "Use PKCE (Proof Key for Code Exchange) for OAuth flow")
	rootCmd.Flags().DurationVar(&oauthTimeout, "oauth-timeout", 5*time.Minute, "Maximum time to wait for OAuth authorization")
	rootCmd.Flags().BoolVar(&oauthUseOIDC, "oauth-oidc", false, "Enable OpenID Connect features including nonce validation")
	rootCmd.Flags().StringVar(&oauthRegistrationToken, "oauth-registration-token", "", "OAuth registration access token for Dynamic Client Registration (required if server has DCR authentication enabled)")
	rootCmd.Flags().StringVar(&oauthResourceURI, "oauth-resource-uri", "", "Target resource URI for RFC 8707 (auto-derived from endpoint if not specified)")
	rootCmd.Flags().BoolVar(&oauthSkipResource, "oauth-skip-resource-param", false, "Skip RFC 8707 resource parameter (for testing with older servers)")
	rootCmd.Flags().BoolVar(&oauthSkipResourceMeta, "oauth-skip-resource-metadata", false, "Skip RFC 9728 Protected Resource Metadata discovery (for testing with older servers)")
	rootCmd.Flags().StringVar(&oauthPreferredAuthSrv, "oauth-preferred-auth-server", "", "Preferred authorization server URL when multiple are available")
	rootCmd.Flags().BoolVar(&oauthDisableStepUp, "oauth-disable-step-up", false, "Disable automatic step-up authorization for insufficient_scope errors")
	rootCmd.Flags().IntVar(&oauthStepUpMaxRetries, "oauth-step-up-max-retries", 2, "Maximum number of step-up authorization retry attempts")
	rootCmd.Flags().BoolVar(&oauthStepUpPrompt, "oauth-step-up-prompt", false, "Prompt user before requesting additional scopes during step-up authorization")
	rootCmd.Flags().StringVar(&oauthClientIDMetaURL, "oauth-client-id-metadata-url", "", "HTTPS URL hosting Client ID Metadata Document (enables CIMD support)")
	rootCmd.Flags().BoolVar(&oauthDisableCIMD, "oauth-disable-cimd", false, "Disable Client ID Metadata Documents (falls back to DCR or manual registration)")

	// Add subcommands
	rootCmd.AddCommand(newSelfUpdateCmd())

	// Mark flags as mutually exclusive
	rootCmd.MarkFlagsMutuallyExclusive("repl", "mcp-server")
}

// validateTransport validates the transport configuration
func validateTransport() error {
	if transport == transportStreamableHTTP && !strings.HasSuffix(endpoint, "/mcp") {
		return fmt.Errorf("endpoint '%s' must end with /mcp for streamable-http transport", endpoint)
	}
	if transport != transportStreamableHTTP {
		return fmt.Errorf("unsupported transport '%s' (only streamable-http is supported)", transport)
	}
	return nil
}

// setupSignalHandler sets up graceful shutdown on interrupt signals
func setupSignalHandler(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		if !mcpServer {
			fmt.Println("\nReceived interrupt signal, shutting down gracefully...")
		}
		cancel()
	}()
}

// buildOAuthConfig creates an OAuth configuration from CLI flags
func buildOAuthConfig(cmd *cobra.Command, logger *agent.Logger) (*agent.OAuthConfig, error) {
	if !oauthEnabled {
		return nil, nil
	}

	// Security warning: Check if client secret was passed via CLI flag
	if oauthClientSecret != "" && cmd.Flags().Changed("oauth-client-secret") {
		logger.Warning("Security Warning: Client secret passed via CLI flag is visible in process listings")
		logger.Info("Consider using environment variables instead: export OAUTH_CLIENT_SECRET=\"...\"")
	}

	config := &agent.OAuthConfig{
		Enabled:              true,
		ClientID:             oauthClientID,
		ClientSecret:         oauthClientSecret,
		Scopes:               oauthScopes,
		ScopeSelectionMode:   oauthScopeMode,
		RedirectURL:          oauthRedirectURL,
		UsePKCE:              oauthUsePKCE,
		AuthorizationTimeout: oauthTimeout,
		UseOIDC:              oauthUseOIDC,
		RegistrationToken:    oauthRegistrationToken,
		ResourceURI:          oauthResourceURI,
		SkipResourceParam:    oauthSkipResource,
		SkipResourceMetadata: oauthSkipResourceMeta,
		PreferredAuthServer:  oauthPreferredAuthSrv,
		EnableStepUpAuth:     !oauthDisableStepUp,
		StepUpMaxRetries:     oauthStepUpMaxRetries,
		StepUpUserPrompt:     oauthStepUpPrompt,
		ClientIDMetadataURL:  oauthClientIDMetaURL,
		DisableCIMD:          oauthDisableCIMD,
	}

	config = config.WithDefaults()

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid OAuth configuration: %w", err)
	}

	if oauthClientID == "" {
		logger.Info("OAuth enabled - will attempt Dynamic Client Registration")
	} else {
		logger.Info("OAuth enabled with client ID: %s", oauthClientID)
	}

	return config, nil
}

// runMCPServer runs the agent in MCP server mode
func runMCPServer(ctx context.Context, client *agent.Client, logger *agent.Logger) error {
	server, err := agent.NewMCPServer(client, serverTransport, logger, false)
	if err != nil {
		return fmt.Errorf("failed to create MCP server: %w", err)
	}

	logger.Info("Starting mcp-debug MCP server (transport: %s)...", serverTransport)
	if serverTransport == transportStreamableHTTP {
		addr := listenAddr
		if !strings.Contains(addr, ":") {
			addr = ":" + addr
		}
		logger.Info("Listening on %s%s", addr, "/mcp")
	}

	if err := server.Start(ctx, listenAddr); err != nil {
		return fmt.Errorf("MCP server error: %w", err)
	}
	return nil
}

// runNormalMode runs the agent in normal (listen) mode
func runNormalMode(ctx context.Context, client *agent.Client, logger *agent.Logger) error {
	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, timeout)
	defer timeoutCancel()

	if err := client.Listen(timeoutCtx); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			logger.Info("Timeout reached after %v", timeout)
			return nil
		}
		return fmt.Errorf("agent error: %w", err)
	}
	return nil
}

func runMCPDebug(cmd *cobra.Command, args []string) error {
	if err := validateTransport(); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	setupSignalHandler(cancel)

	logger := agent.NewLogger(verbose, !noColor, jsonRPC)

	oauthConfig, err := buildOAuthConfig(cmd, logger)
	if err != nil {
		return err
	}

	client := agent.NewClient(agent.ClientConfig{
		Endpoint:    endpoint,
		Transport:   transport,
		Logger:      logger,
		OAuthConfig: oauthConfig,
		Version:     version,
	})
	if err := client.Run(ctx); err != nil {
		return fmt.Errorf("failed to connect client: %w", err)
	}

	if mcpServer {
		return runMCPServer(ctx, client, logger)
	}

	if repl {
		replHandler := agent.NewREPL(client, logger)
		if err := replHandler.Run(ctx); err != nil {
			return fmt.Errorf("REPL error: %w", err)
		}
		return nil
	}

	return runNormalMode(ctx, client, logger)
}
