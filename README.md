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

To build the tool from source, run:
```bash
make build
```
This will create the `mcp-debug` binary in the project directory.

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
- Registers as a client if needed (Dynamic Client Registration)
- Opens your browser for authorization
- Connects with audience-bound tokens (RFC 8707)

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