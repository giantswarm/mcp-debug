# MCP Debug

[![Go Report Card](https://goreportcard.com/badge/github.com/giantswarm/mcp-debug)](https://goreportcard.com/report/github.com/giantswarm/mcp-debug)

`mcp-debug` is a command-line tool for debugging MCP (Model Context Protocol) servers. It helps developers inspect MCP server capabilities, debug tool integrations, and test server notifications.

## Key Features

- **Connect to any MCP Server**: Works with servers using `streamable-http` or `sse` transports.
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

### Full Usage Guide

For detailed instructions on modes, flags, and examples, please see the full usage guide:

**[➡️ mcp-debug Usage Guide](./docs/usage.md)**

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