# MCP-Debug

A debugging tool for MCP (Model Context Protocol) servers. This tool acts as an MCP client that can connect to MCP servers via SSE (Server-Sent Events), inspect available tools, resources, and prompts, and execute them interactively.

## Features

- **Normal Mode**: Connect to an MCP server and monitor for changes
- **REPL Mode**: Interactive interface to explore and execute MCP capabilities
- **MCP Server Mode**: Run as an MCP server that exposes debugging functionality

## Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/mcp-debug.git
cd mcp-debug

# Build the binary
make build

# Install to $GOPATH/bin
make install
```

## Usage

### Normal Mode (Default)
Connect to an MCP server and wait for notifications:
```bash
mcp-debug agent --endpoint http://localhost:8080/sse
```

### REPL Mode
Interactive mode for exploring MCP capabilities:
```bash
mcp-debug agent --repl
```

### MCP Server Mode
Run as an MCP server (for integration with AI assistants):
```bash
mcp-debug agent --mcp-server
```

## REPL Commands

- `list tools` - List all available tools
- `list resources` - List all available resources
- `list prompts` - List all available prompts
- `describe tool <name>` - Show detailed information about a tool
- `describe resource <uri>` - Show detailed information about a resource
- `describe prompt <name>` - Show detailed information about a prompt
- `call <tool> {json}` - Execute a tool with JSON arguments
- `get <resource-uri>` - Retrieve a resource
- `prompt <name> {json}` - Get a prompt with JSON arguments
- `notifications <on|off>` - Enable/disable notification display
- `help` - Show help message
- `exit` - Exit the REPL

## Command Line Flags

- `--endpoint` - SSE endpoint URL (default: http://localhost:8080/sse)
- `--timeout` - Timeout for waiting for notifications (default: 5m)
- `--verbose` - Enable verbose logging
- `--no-color` - Disable colored output
- `--json-rpc` - Enable full JSON-RPC message logging
- `--repl` - Start interactive REPL mode
- `--mcp-server` - Run as MCP server (stdio transport)

## Examples

### Testing a Tool
```bash
$ mcp-debug agent --repl
MCP> list tools
Available tools (3):
  1. calculate              - Perform mathematical calculations
  2. get_weather           - Get weather information
  3. search               - Search for information

MCP> call calculate {"operation": "add", "x": 5, "y": 3}
Executing tool: calculate...
Result:
{
  "result": 8
}
```

### Integration with Cursor

Add to your `.cursor/mcp.json`:
```json
{
  "mcpServers": {
    "mcp-debug": {
      "command": "mcp-debug",
      "args": ["agent", "--mcp-server", "--endpoint", "http://localhost:8080/sse"]
    }
  }
}
```

## Development

```bash
# Run tests
make test

# Run in REPL mode
make run

# Run as MCP server
make run-mcp

# Run with verbose logging
make run-verbose
```

## License

MIT 