package cmd

import (
	"context"
	"fmt"
	"mcp-debug/internal/agent"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var (
	agentEndpoint  string
	agentTimeout   time.Duration
	agentVerbose   bool
	agentNoColor   bool
	agentJSONRPC   bool
	agentREPL      bool
	agentMCPServer bool
)

// agentCmd represents the agent command
var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Act as an MCP client to debug MCP servers",
	Long: `The agent command connects to an MCP server as a client agent, 
logs all JSON-RPC communication, and demonstrates dynamic tool updates.

This is useful for debugging MCP server behavior, verifying that
tools are properly exposed, and ensuring that notifications work correctly
when tools are added or removed.

The agent can run in three modes:
1. Normal mode (default): Connects, lists tools, and waits for notifications
2. REPL mode (--repl): Provides an interactive interface to explore and execute tools
3. MCP Server mode (--mcp-server): Runs an MCP server that exposes REPL functionality via stdio

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

By default, it connects to http://localhost:8080/sse. You can override this with the --endpoint flag.`,
	RunE: runAgent,
}

func init() {
	rootCmd.AddCommand(agentCmd)

	// Add flags
	agentCmd.Flags().StringVar(&agentEndpoint, "endpoint", "http://localhost:8080/sse", "SSE endpoint URL")
	agentCmd.Flags().DurationVar(&agentTimeout, "timeout", 5*time.Minute, "Timeout for waiting for notifications")
	agentCmd.Flags().BoolVar(&agentVerbose, "verbose", false, "Enable verbose logging (show keepalive messages)")
	agentCmd.Flags().BoolVar(&agentNoColor, "no-color", false, "Disable colored output")
	agentCmd.Flags().BoolVar(&agentJSONRPC, "json-rpc", false, "Enable full JSON-RPC message logging")
	agentCmd.Flags().BoolVar(&agentREPL, "repl", false, "Start interactive REPL mode")
	agentCmd.Flags().BoolVar(&agentMCPServer, "mcp-server", false, "Run as MCP server (stdio transport)")

	// Mark flags as mutually exclusive
	agentCmd.MarkFlagsMutuallyExclusive("repl", "mcp-server")
}

func runAgent(cmd *cobra.Command, args []string) error {
	// Create context with signal handling
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	// Handle interrupts gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		if !agentMCPServer {
			fmt.Println("\nReceived interrupt signal, shutting down gracefully...")
		}
		cancel()
	}()

	// Create logger
	logger := agent.NewLogger(agentVerbose, !agentNoColor, agentJSONRPC)

	// Run in MCP Server mode if requested
	if agentMCPServer {
		server, err := agent.NewMCPServer(agentEndpoint, logger, false)
		if err != nil {
			return fmt.Errorf("failed to create MCP server: %w", err)
		}

		logger.Info("Starting mcp-debug MCP server (stdio transport)...")
		logger.Info("Connecting to MCP server at: %s", agentEndpoint)

		if err := server.Start(ctx); err != nil {
			return fmt.Errorf("MCP server error: %w", err)
		}
		return nil
	}

	// Create and run agent client
	client := agent.NewClient(agentEndpoint, logger)

	// Run in REPL mode if requested
	if agentREPL {
		// REPL mode doesn't use timeout
		repl := agent.NewREPL(client, logger)
		if err := repl.Run(ctx); err != nil {
			return fmt.Errorf("REPL error: %w", err)
		}
		return nil
	}

	// Create timeout context for non-REPL mode
	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, agentTimeout)
	defer timeoutCancel()

	// Run the agent in normal mode
	if err := client.Run(timeoutCtx); err != nil {
		if err == context.DeadlineExceeded {
			logger.Info("Timeout reached after %v", agentTimeout)
			return nil
		}
		return fmt.Errorf("agent error: %w", err)
	}

	return nil
} 