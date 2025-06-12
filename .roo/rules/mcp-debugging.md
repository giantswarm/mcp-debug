---
description: 
globs: 
alwaysApply: false
---
## Debugging MCP Servers via MCP-Debug Agent

The mcp-debug agent provides essential tools to test and explore MCP (Model Context Protocol) servers. This guide focuses on using the agent to debug MCP server behavior and functionality.

## Core Agent Modes

The agent operates in three distinct modes:

### 1. Normal Mode (Default)
Connects to an MCP server, lists available tools/resources/prompts, and waits for notifications:
```bash
mcp-debug agent --endpoint http://localhost:8080/sse
```

### 2. REPL Mode
Interactive mode for exploring and executing MCP capabilities:
```bash
mcp-debug agent --repl
```

In REPL mode, you can:
- List available tools, resources, and prompts
- Get detailed information about specific items
- Execute tools interactively with JSON arguments
- View resources and retrieve their contents
- Execute prompts with arguments
- Toggle notification display

### 3. MCP Server Mode
Run the agent as an MCP server exposing debugging tools via stdio:
```bash
mcp-debug agent --mcp-server
```

This mode is designed for integration with AI assistants like Claude or Roo Code. Configure it in your AI assistant's MCP settings.

## REPL Commands

### Listing Commands
```
list tools                   # List all available tools
list resources               # List all available resources  
list prompts                 # List all available prompts
```

### Describe Commands
```
describe tool <name>         # Show detailed information about a tool
describe resource <uri>      # Show detailed information about a resource
describe prompt <name>       # Show detailed information about a prompt
```

### Execution Commands
```
call <tool> {json}           # Execute a tool with JSON arguments
get <resource-uri>           # Retrieve a resource
prompt <name> {json}         # Get a prompt with JSON arguments
```

### Control Commands
```
notifications <on|off>       # Enable/disable notification display
help, ?                      # Show help message
exit, quit                   # Exit the REPL
```

## Examples

### Calling a Tool
```
MCP> call calculate {"operation": "add", "x": 5, "y": 3}
Executing tool: calculate...
Result:
{
  "result": 8
}
```

### Getting a Resource
```
MCP> get docs://readme
Retrieving resource: docs://readme...
Contents:
# Project README
This is the project documentation...
```

### Using a Prompt
```
MCP> prompt greeting {"name": "Alice"}
Getting prompt: greeting...
Messages:

[1] Role: user
Content: Hello Alice! How can I help you today?
```

## Debugging Workflow

### 1. Connect and Explore
Start with REPL mode to explore what the MCP server offers:
```bash
mcp-debug agent --repl --endpoint http://your-server:port/sse
```

### 2. Enable Verbose Logging
For detailed protocol inspection:
```bash
mcp-debug agent --verbose --json-rpc
```

### 3. Test Tool Execution
Use the REPL to test individual tools with different arguments to ensure they work correctly.

### 4. Monitor Notifications
Keep the agent running to monitor for dynamic updates when tools/resources/prompts change.

## Common Issues and Solutions

**Connection Failed**
- Verify the endpoint URL is correct
- Check if the MCP server is running
- Ensure the server supports SSE transport

**Tool Not Found**
- Use `list tools` to see available tools
- Tool names are case-sensitive
- Check if the tool was recently added (may need to reconnect)

**Invalid Arguments**
- Arguments must be valid JSON
- Use `describe tool <name>` to see required parameters
- Check the schema for correct types

**No Notifications**
- Some servers don't send notifications
- Enable verbose mode to see keepalive messages
- Check if the server implements notification capabilities

## Integration with AI Assistants

To use mcp-debug as an MCP server in Roo Code:

1. Add to your `.roo/mcp.json`:
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

2. The agent exposes these tools:
- `list_tools` - List all available tools
- `list_resources` - List all available resources  
- `list_prompts` - List all available prompts
- `describe_tool` - Get tool details
- `describe_resource` - Get resource details
- `describe_prompt` - Get prompt details
- `call_tool` - Execute a tool
- `get_resource` - Retrieve resource contents
- `get_prompt` - Execute a prompt

## Tips for Effective Debugging

1. **Start Simple**: Begin with `list` commands to understand what's available
2. **Use Descriptions**: Always check tool/resource/prompt descriptions before using them
3. **Test Incrementally**: Test one tool at a time with minimal arguments first
4. **Monitor Changes**: Keep an agent running to catch dynamic updates
5. **Log Everything**: Use verbose mode when troubleshooting issues
6. **Validate JSON**: Ensure your arguments are valid JSON before execution
