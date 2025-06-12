package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var version string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "mcp-debug",
	Short: "MCP debugging tool",
	Long: `mcp-debug is a tool for debugging MCP (Model Context Protocol) servers.

It provides an agent that can connect to MCP servers via SSE (Server-Sent Events),
inspect available tools, resources, and prompts, and execute them interactively.

The tool supports multiple modes:
- Normal mode: Connect and wait for notifications
- REPL mode: Interactive exploration and execution
- MCP Server mode: Act as an MCP server for integration with AI assistants`,
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
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
} 