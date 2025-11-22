// Package agent provides the MCP debugging agent implementation.
//
// This package includes client connectivity for MCP servers, OAuth 2.1 authentication support,
// interactive REPL capabilities for exploring MCP servers, and an MCP server implementation
// that exposes debugging functionality as MCP tools.
//
// Key components:
//   - Client: Connects to MCP servers and handles communication
//   - OAuthConfig: Configuration for OAuth 2.1 authentication
//   - REPL: Interactive Read-Eval-Print Loop for exploring MCP servers
//   - MCPServer: Exposes debugging functionality as an MCP server
//   - Logger: Formatted logging with color support and JSON-RPC message tracking
package agent
