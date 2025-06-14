# Using mcp-debug

`mcp-debug` is a versatile command-line tool for interacting with and debugging MCP (Model Context Protocol) servers. It can act as a client to inspect server capabilities, an interactive REPL for hands-on debugging, and even as an MCP server itself for integration with AI assistants.

This guide covers the main functionalities and how to use them.

## Table of Contents

- [Installation](#installation)
- [Modes of Operation](#modes-of-operation)
  - [1. Normal Mode (Passive Listening)](#1-normal-mode-passive-listening)
  - [2. REPL Mode (Interactive Debugging)](#2-repl-mode-interactive-debugging)
  - [3. MCP Server Mode (AI Assistant Integration)](#3-mcp-server-mode-ai-assistant-integration)
- [Transport Protocols](#transport-protocols)
- [Command-Line Flags](#command-line-flags)
- [Usage Examples](#usage-examples)
  - [Connecting to a Server](#connecting-to-a-server)
  - [Using the REPL](#using-the-repl)
  - [Running as an MCP Server](#running-as-an-mcp-server)

---

## Installation

You can install `mcp-debug` by downloading a pre-built binary or by building it from source.

### 1. Pre-built Binaries (Recommended)

The easiest way to get `mcp-debug` is by downloading the latest pre-built binary for your operating system and architecture.

**Quick Install with cURL (Linux/macOS)**

You can run the following command in your terminal to automatically download the latest version for your system and make it executable in your current directory. This script requires `curl`, `grep`, `sed`, and `tar` to be installed.

```bash
curl -sL https://raw.githubusercontent.com/giantswarm/mcp-debug/main/install.sh | bash
```

**Manual Download**

1.  Navigate to the [latest release page](https://github.com/giantswarm/mcp-debug/releases/latest).
2.  Under the **Assets** section, download the `.tar.gz` archive that matches your system (e.g., `mcp-debug_0.0.4_linux_amd64.tar.gz`).
3.  Un-tar the archive to extract the `mcp-debug` binary.
4.  Make the binary executable.

```bash
# Example for Linux
tar -xzf mcp-debug_*.tar.gz
chmod +x ./mcp-debug
```

You can then run `./mcp-debug` from the download directory. For system-wide access, you can move the binary to a directory in your system's `PATH`.

```bash
# Optional: Move to make it accessible system-wide
sudo mv ./mcp-debug /usr/local/bin/mcp-debug
```

### 2. Build from Source

If you prefer to build from source, ensure you have Go installed and run:

```bash
make build
```

This will create the `mcp-debug` binary in the project's root directory.

---

## Modes of Operation

`mcp-debug` offers three primary modes to suit different debugging needs.

### 1. Normal Mode (Passive Listening)

This is the default mode. The tool connects to an MCP server, logs any notifications (like tool updates), and then waits for a specified duration. It's useful for quickly checking if a server is running and responsive.

**How to Run:**
```bash
./mcp-debug --endpoint <server-url>
```
By default, it connects to `http://localhost:8899/mcp` and waits for 5 minutes.

### 2. REPL Mode (Interactive Debugging)

The REPL (Read-Eval-Print Loop) provides an interactive shell for exploring an MCP server's capabilities. This is the most powerful mode for hands-on debugging.

**How to Run:**
```bash
./mcp-debug --repl --endpoint <server-url>
```

**REPL Commands:**
- `tools`: List all available tools.
- `tool <name>`: Get detailed information about a specific tool.
- `exec <tool_name> '{"arg1": "value1"}'`: Execute a tool with JSON arguments.
- `resources`: List available resources.
- `resource <name>`: View the content of a resource.
- `prompts`: List available prompts.
- `prompt <name> '{"arg": "value"}'`: Execute a prompt with arguments.
- `notifications [on|off]`: Control the display of server notifications.
- `help`: Show available commands.
- `exit`: Quit the REPL.

### 3. MCP Server Mode (AI Assistant Integration)

In this mode, `mcp-debug` acts as an MCP server itself. It exposes all its REPL commands as MCP tools. This is designed for integration with AI assistants that can communicate via MCP, such as Cursor. You can configure your assistant to connect to `mcp-debug`, allowing it to perform debugging tasks on your behalf.

**Running the Server**

The simplest way to run the server is to use the default `stdio` transport, which requires no network configuration. The assistant will communicate with the `mcp-debug` process over its standard input and output.

```bash
./mcp-debug --mcp-server
```

**Example Configuration for Cursor (`mcp.json`)**

To integrate `mcp-debug` with an assistant like Cursor, you can add the following configuration to your project's `.cursor/mcp.json` file. This tells Cursor how to launch the `mcp-debug` server.

```json
{
  "mcpServers": {
    "mcp-debug": {
      "command": "mcp-debug",
      "args": [
        "--mcp-server"
      ]
    }
  }
}
```
> **Note:** The `"command"` value should be `mcp-debug` if you moved the binary to your system's `PATH`. Otherwise, you should provide the full path to the executable (e.g., `/path/to/your/project/mcp-debug`).

For assistants that need to connect over the network, you can run the server in `streamable-http` mode:
```bash
# Start in MCP server mode and listen for HTTP connections
./mcp-debug --mcp-server --server-transport streamable-http --listen-addr :9000
```
You would then configure your AI assistant to connect to `http://localhost:9000/mcp`.

---

## Transport Protocols

`mcp-debug` supports multiple transport protocols for communication:

- **`streamable-http`** (Default): A modern, efficient protocol for MCP communication.
- **`sse`**: Server-Sent Events, a common protocol for streaming updates.
- **`stdio`**: Uses standard input/output for communication, primarily for the MCP Server mode.

You can specify the transport using the `--transport` and `--server-transport` flags.

---

## Command-Line Flags

Here are the most important flags to configure `mcp-debug`:

| Flag                | Description                                                                          | Default                        |
| ------------------- | ------------------------------------------------------------------------------------ | ------------------------------ |
| `--repl`            | Start the interactive REPL mode.                                                     | `false`                        |
| `--mcp-server`      | Run as an MCP server.                                                                | `false`                        |
| `--endpoint`        | The URL of the target MCP server.                                                    | `http://localhost:8899/mcp`    |
| `--transport`       | Client transport protocol (`streamable-http`, `sse`).                                | `streamable-http`              |
| `--server-transport`| Server transport protocol (`stdio`, `streamable-http`).                                | `stdio`                        |
| `--listen-addr`     | Listen address for the `streamable-http` server.                                     | `:8899`                        |
| `--timeout`         | Timeout for waiting for notifications in normal mode.                                | `5m`                           |
| `--verbose`         | Enable verbose logging (shows keep-alive messages).                                  | `false`                        |
| `--json-rpc`        | Enable full logging of JSON-RPC messages.                                            | `false`                        |
| `--no-color`        | Disable colored output.                                                              | `false`                        |
| `--version`         | Show the application version.                                                        |                                |

---

## Usage Examples

### Connecting to a Server

**Connect using default settings:**
```bash
./mcp-debug
```

**Connect to a different server with a 10-second timeout:**
```bash
./mcp-debug --endpoint http://custom.server:1234/mcp --timeout 10s
```

**Connect to an SSE server and log all JSON-RPC traffic:**
```bash
./mcp-debug --endpoint http://legacy.server/sse --transport sse --json-rpc
```

### Using the REPL

**Start the REPL and connect to a local server:**
```bash
./mcp-debug --repl
```

**Inside the REPL, list tools and execute one:**
```
>> tools
Available tools:
- list_files
- read_file

>> exec list_files '{"path": "./"}'
...
```

### Running as an MCP Server

**Start in MCP server mode using stdio (for local AI assistant integration):**
```bash
./mcp-debug --mcp-server
```

**Start in MCP server mode and listen for HTTP connections:**
```bash
./mcp-debug --mcp-server --server-transport streamable-http --listen-addr :9000
```
Then, configure your AI assistant to connect to `http://localhost:9000/mcp`. 