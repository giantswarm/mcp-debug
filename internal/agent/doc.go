// Package agent provides the MCP debugging agent implementation.
//
// This package includes client connectivity for MCP servers, OAuth 2.1 authentication support,
// interactive REPL capabilities for exploring MCP servers, and an MCP server implementation
// that exposes debugging functionality as MCP tools.
//
// # OAuth 2.1 Support
//
// The agent package implements the MCP authorization specification (2025-11-25) with comprehensive
// OAuth 2.1 support including:
//
// Discovery:
//   - RFC 9728: Protected Resource Metadata Discovery
//   - RFC 8414: Authorization Server Metadata Discovery
//   - Multi-endpoint probing (OAuth 2.0 and OIDC)
//   - WWW-Authenticate header parsing
//
// Security Features:
//   - RFC 7636: PKCE (Proof Key for Code Exchange) with S256 method validation
//   - RFC 8707: Resource Indicators for token audience binding
//   - In-memory token storage (never persisted to disk)
//   - HTTPS enforcement for sensitive operations
//
// Client Registration:
//   - RFC 7591: Dynamic Client Registration (DCR)
//   - Client ID Metadata Documents (draft-ietf-oauth-client-id-metadata-document-00)
//   - Pre-registered client support
//   - Authenticated DCR with registration tokens
//
// Scope Management:
//   - Automatic scope selection per MCP spec priority
//   - Manual scope override capability
//   - Step-up authorization for runtime permission escalation
//   - Scope validation and security checks
//
// # Key Components
//
//   - Client: Connects to MCP servers and handles communication
//   - OAuthConfig: Configuration for OAuth 2.1 authentication
//   - REPL: Interactive Read-Eval-Print Loop for exploring MCP servers
//   - MCPServer: Exposes debugging functionality as an MCP server
//   - Logger: Formatted logging with color support and JSON-RPC message tracking
//
// # Documentation
//
// For complete OAuth documentation, see:
//   - docs/oauth/ - Complete OAuth 2.1 documentation
//   - docs/oauth/README.md - Overview and quick start
//   - docs/oauth/security.md - Security features and best practices
//   - docs/oauth/examples/ - Practical examples and tutorials
//
// For usage documentation, see:
//   - docs/usage.md - Complete usage guide
//   - README.md - Project overview and quick start
package agent
