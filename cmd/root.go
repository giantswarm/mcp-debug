package cmd

import (
	"context"
	"fmt"
	"mcp-debug/internal/agent"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
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
	oauthEnabled      bool
	oauthClientID     string
	oauthClientSecret string
	oauthScopes       []string
	oauthRedirectURL  string
	oauthUsePKCE      bool
	oauthTimeout      time.Duration
	oauthUseOIDC      bool
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
	rootCmd.Flags().StringVar(&transport, "transport", "streamable-http", "Transport protocol to use for client connections (streamable-http only)")
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
	rootCmd.Flags().StringSliceVar(&oauthScopes, "oauth-scopes", []string{"mcp:tools", "mcp:resources"}, "OAuth scopes to request")
	rootCmd.Flags().StringVar(&oauthRedirectURL, "oauth-redirect-url", "http://localhost:8765/callback", "OAuth redirect URL for callback")
	rootCmd.Flags().BoolVar(&oauthUsePKCE, "oauth-pkce", true, "Use PKCE (Proof Key for Code Exchange) for OAuth flow")
	rootCmd.Flags().DurationVar(&oauthTimeout, "oauth-timeout", 5*time.Minute, "Maximum time to wait for OAuth authorization")
	rootCmd.Flags().BoolVar(&oauthUseOIDC, "oauth-oidc", false, "Enable OpenID Connect features including nonce validation")

	// Add subcommands
	rootCmd.AddCommand(newSelfUpdateCmd())

	// Mark flags as mutually exclusive
	rootCmd.MarkFlagsMutuallyExclusive("repl", "mcp-server")
}

func runMCPDebug(cmd *cobra.Command, args []string) error {
	// Validate endpoint for streamable-http
	if transport == "streamable-http" && !strings.HasSuffix(endpoint, "/mcp") {
		return fmt.Errorf("endpoint '%s' must end with /mcp for streamable-http transport", endpoint)
	}

	if transport != "streamable-http" {
		return fmt.Errorf("unsupported transport '%s' (only streamable-http is supported)", transport)
	}

	// Create context with signal handling
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	// Handle interrupts gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		if !mcpServer {
			fmt.Println("\nReceived interrupt signal, shutting down gracefully...")
		}
		cancel()
	}()

	// Create logger
	logger := agent.NewLogger(verbose, !noColor, jsonRPC)

	// Create OAuth config if enabled
	var oauthConfig *agent.OAuthConfig
	if oauthEnabled {
		// Security warning: Check if client secret was passed via CLI flag
		if oauthClientSecret != "" && cmd.Flags().Changed("oauth-client-secret") {
			logger.Warning("⚠️  Security Warning: Client secret passed via CLI flag is visible in process listings")
			logger.Info("Consider using environment variables instead: export OAUTH_CLIENT_SECRET=\"...\"")
		}

		oauthConfig = &agent.OAuthConfig{
			Enabled:              true,
			ClientID:             oauthClientID,
			ClientSecret:         oauthClientSecret,
			Scopes:               oauthScopes,
			RedirectURL:          oauthRedirectURL,
			UsePKCE:              oauthUsePKCE,
			AuthorizationTimeout: oauthTimeout,
			UseOIDC:              oauthUseOIDC,
		}

		// Validate OAuth configuration
		if err := oauthConfig.Validate(); err != nil {
			return fmt.Errorf("invalid OAuth configuration: %w", err)
		}

		if oauthClientID == "" {
			logger.Info("OAuth enabled - will attempt Dynamic Client Registration")
		} else {
			logger.Info("OAuth enabled with client ID: %s", oauthClientID)
		}
	}

	// Create and run agent client
	client := agent.NewClient(endpoint, transport, logger, oauthConfig, version)
	if err := client.Run(ctx); err != nil {
		return fmt.Errorf("failed to connect client: %w", err)
	}

	// Run in MCP Server mode if requested
	if mcpServer {
		server, err := agent.NewMCPServer(client, serverTransport, logger, false)
		if err != nil {
			return fmt.Errorf("failed to create MCP server: %w", err)
		}

		logger.Info("Starting mcp-debug MCP server (transport: %s)...", serverTransport)
		if serverTransport == "streamable-http" {
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

	// Run in REPL mode if requested
	if repl {
		// REPL mode doesn't use timeout
		replHandler := agent.NewREPL(client, logger)
		if err := replHandler.Run(ctx); err != nil {
			return fmt.Errorf("REPL error: %w", err)
		}
		return nil
	}

	// Create timeout context for non-REPL mode
	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, timeout)
	defer timeoutCancel()

	// Run the agent in normal mode
	if err := client.Listen(timeoutCtx); err != nil {
		if err == context.DeadlineExceeded {
			logger.Info("Timeout reached after %v", timeout)
			return nil
		}
		return fmt.Errorf("agent error: %w", err)
	}

	return nil
}
