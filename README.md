# MCP Debug

[![Go Report Card](https://goreportcard.com/badge/github.com/giantswarm/mcp-debug)](https://goreportcard.com/report/github.com/giantswarm/mcp-debug)

`mcp-debug` is a command-line tool for debugging MCP (Model Context Protocol) servers. It helps developers inspect MCP server capabilities, debug tool integrations, and test server notifications.

## Key Features

- **Connect to any MCP Server**: Works with servers using `streamable-http` transport.
- **OAuth 2.1 Authentication**: Full MCP authorization specification (2025-11-25) compliance with security-first defaults
  - Automatic discovery (RFC 9728 Protected Resource Metadata, RFC 8414 AS Metadata)
  - Resource Indicators (RFC 8707) for token audience binding
  - PKCE validation (required by MCP spec)
  - Dynamic Client Registration (RFC 7591) and Client ID Metadata Documents
  - Step-up authorization for runtime permission escalation
  - Intelligent scope selection (auto and manual modes)
- **Interactive REPL**: Explore available tools, resources, and prompts interactively.
- **MCP Server Mode**: Acts as an MCP server itself, allowing integration with AI assistants like Cursor.
- **Verbose Logging**: Detailed logging of JSON-RPC messages for in-depth debugging.
- **Self-Update**: Keep the tool up-to-date with a single command.
- **Shell Autocompletion**: Generates autocompletion scripts for Bash, Zsh, Fish, and PowerShell.

## Getting Started

### Installation

To build the tool from source and install it in your PATH:

```bash
go install github.com/giantswarm/mcp-debug@latest
```

### Documentation

For detailed instructions on all features:

- **[➡️ mcp-debug Usage Guide](./docs/usage.md)** - Complete feature guide
- **[➡️ OAuth 2.1 Documentation](./docs/oauth/)** - Comprehensive OAuth authentication guide

### OAuth 2.1 Quick Start

Connect to a protected MCP server with automatic discovery:

```bash
./mcp-debug --oauth --endpoint https://mcp.example.com/mcp
```

This automatically:
- Discovers the authorization server (RFC 9728)
- Validates PKCE support (required by MCP spec)
- Uses the official [Client ID Metadata Document](https://giantswarm.github.io/mcp-debug/client.json) (CIMD)
- Falls back to Dynamic Client Registration if CIMD is not supported
- Opens your browser for authorization
- Connects with audience-bound tokens (RFC 8707)

#### Client ID Metadata Document (CIMD)

mcp-debug uses CIMD by default for OAuth authentication. The official client metadata is hosted at:

**https://giantswarm.github.io/mcp-debug/client.json**

This allows OAuth-protected MCP servers to identify mcp-debug without requiring pre-registration. The authorization server fetches the metadata from this URL and displays "MCP Debugger CLI" on the consent screen.

If you need to use a custom CIMD or disable it:

```bash
# Use a custom CIMD URL
./mcp-debug --oauth --oauth-client-id-metadata-url https://example.com/my-client.json --endpoint https://mcp.example.com/mcp

# Disable CIMD and force Dynamic Client Registration
./mcp-debug --oauth --oauth-disable-cimd --endpoint https://mcp.example.com/mcp
```

See the **[OAuth Documentation](./docs/oauth/)** for detailed guides, examples, and troubleshooting.

## Basic Examples

**Connect to a server and listen for notifications:**
```bash
./mcp-debug --endpoint http://localhost:8090/mcp
```

**Start the interactive REPL:**
```bash
./mcp-debug --repl
```

**Run as an MCP server for AI assistant integration:**
```bash
./mcp-debug --mcp-server
```

Refer to the [usage guide](./docs/usage.md) for more advanced examples.

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

## License

`mcp-debug` is licensed under the Apache 2.0 License. 
