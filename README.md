# MCP-Debug

A debugging tool for MCP (Model Context Protocol) servers that provides an interactive client to inspect tools, resources, and prompts.

## Features

- Connect to MCP servers via SSE (Server-Sent Events)
- List available tools, resources, and prompts
- Execute tools interactively with JSON arguments
- View resources and retrieve their contents
- Execute prompts with arguments
- Monitor server notifications in real-time
- Full JSON-RPC message logging for debugging
- Interactive REPL mode for exploration
- MCP server mode for AI assistant integration

## Installation

Clone the repository and build:

```bash
git clone <repository-url>
cd mcp-debug
go build -o mcp-debug
```

## Usage

### Basic Usage

Connect to an MCP server and list available tools:

```bash
mcp-debug --endpoint http://localhost:8090/sse
```

### Interactive REPL Mode

For interactive exploration and tool execution:

```bash
mcp-debug --repl --endpoint http://localhost:8090/sse
```

### Available REPL Commands

In REPL mode, you can use these commands:
- `help` - Show available commands
- `list` - List all available tools, resources, and prompts
- `tools` - List available tools
- `tool <name>` - Get detailed information about a specific tool
- `call <name> [args]` - Execute a tool with optional JSON arguments
- `resources` - List available resources
- `resource <uri>` - Get detailed information about a specific resource
- `read <uri>` - Read the contents of a resource
- `prompts` - List available prompts
- `prompt <name>` - Get detailed information about a specific prompt
- `complete <name> [args]` - Execute a prompt with optional JSON arguments
- `notifications` - Toggle notification display
- `verbose` - Toggle verbose logging
- `exit` - Exit the REPL

### Command Line Options

- `--endpoint` - SSE endpoint URL (default: http://localhost:8090/sse)
- `--timeout` - Timeout for waiting for notifications (default: 5m)
- `--verbose` - Enable verbose logging (show keepalive messages)
- `--no-color` - Disable colored output
- `--json-rpc` - Enable full JSON-RPC message logging
- `--repl` - Start interactive REPL mode
- `--mcp-server` - Run as MCP server (stdio transport)

### MCP Server Mode

To use mcp-debug as an MCP server (for integration with AI assistants):

```bash
mcp-debug --mcp-server --endpoint http://localhost:8090/sse
```

This mode allows AI assistants to use mcp-debug as an MCP server that provides debugging capabilities for other MCP servers.

#### Cursor Integration

Add this to your `.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "mcp-debug": {
      "command": "mcp-debug",
      "args": ["--mcp-server", "--endpoint", "http://localhost:8090/sse"]
    }
  }
}
```

### Examples

1. **Basic debugging session:**
   ```bash
   mcp-debug --endpoint http://localhost:8090/sse --verbose
   ```

2. **Interactive exploration:**
   ```bash
   mcp-debug --repl --endpoint http://localhost:8090/sse
   ```

3. **Full JSON-RPC logging:**
   ```bash
   mcp-debug --json-rpc --endpoint http://localhost:8090/sse
   ```

## Architecture

The tool consists of several components:

- **Client**: Connects to MCP servers via SSE and handles JSON-RPC communication
- **REPL**: Interactive interface for exploring and executing MCP operations
- **Logger**: Handles formatted output with color support and message filtering
- **MCP Server**: Exposes REPL functionality as an MCP server for AI assistant integration

## Development

Run tests:
```bash
make test
```

Build:
```bash
make build
```

## License

[Add your license here] 