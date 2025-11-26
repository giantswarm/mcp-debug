// Package agent provides the MCP debugging agent implementation.
//
// This package includes client connectivity for MCP servers, OAuth 2.1 authentication support,
// interactive REPL capabilities for exploring MCP servers, and an MCP server implementation
// that exposes debugging functionality as MCP tools.
//
// # OAuth 2.1 Support
//
// The agent package implements the MCP authorization specification (2025-11-25) with support for:
//   - RFC 8707: Resource Indicators for OAuth 2.0
//   - RFC 9728: Protected Resource Metadata Discovery
//   - RFC 8414: Authorization Server Metadata Discovery
//   - PKCE validation (code_challenge_methods_supported enforcement)
//   - Dynamic Client Registration (RFC 7591)
//   - Multi-endpoint discovery probing (OAuth 2.0 and OIDC)
//
// # Key Components
//
//   - Client: Connects to MCP servers and handles communication
//   - OAuthConfig: Configuration for OAuth 2.1 authentication
//   - REPL: Interactive Read-Eval-Print Loop for exploring MCP servers
//   - MCPServer: Exposes debugging functionality as an MCP server
//   - Logger: Formatted logging with color support and JSON-RPC message tracking
//
// See docs/oauth/ for detailed OAuth documentation.
package agent
