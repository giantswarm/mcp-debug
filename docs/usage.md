# Using mcp-debug

`mcp-debug` is a versatile command-line tool for interacting with and debugging MCP (Model Context Protocol) servers. It can act as a client to inspect server capabilities, an interactive REPL for hands-on debugging, and even as an MCP server itself for integration with AI assistants.

This guide covers the main functionalities and how to use them.

## Table of Contents

- [Using mcp-debug](#using-mcp-debug)
  - [Table of Contents](#table-of-contents)
  - [Installation](#installation)
    - [1. Pre-built Binaries (Recommended)](#1-pre-built-binaries-recommended)
    - [2. Build from Source](#2-build-from-source)
  - [Keeping the Tool Updated](#keeping-the-tool-updated)
  - [Modes of Operation](#modes-of-operation)
    - [1. Normal Mode (Passive Listening)](#1-normal-mode-passive-listening)
    - [2. REPL Mode (Interactive Debugging)](#2-repl-mode-interactive-debugging)
    - [3. MCP Server Mode (AI Assistant Integration)](#3-mcp-server-mode-ai-assistant-integration)
  - [Transport Protocols](#transport-protocols)
  - [Command-Line Flags](#command-line-flags)
  - [Shell Autocompletion](#shell-autocompletion)
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

## Keeping the Tool Updated

You can easily update `mcp-debug` to the latest version using the built-in `self-update` command. This will check GitHub for the latest release and replace the current binary if a newer version is available.

```bash
./mcp-debug self-update
```

This ensures you always have the latest features and bug fixes.

---

## Modes of Operation

`mcp-debug` offers three primary modes to suit different debugging needs.

### 1. Normal Mode (Passive Listening)

This is the default mode. The tool connects to an MCP server, logs any notifications (like tool updates), and then waits for a specified duration. It's useful for quickly checking if a server is running and responsive.

**How to Run:**
```bash
./mcp-debug --endpoint <server-url>
```
By default, it connects to `http://localhost:8090/mcp` and waits for 5 minutes.

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

`mcp-debug` supports the `streamable-http` transport protocol for communication with MCP servers. This is a modern, efficient protocol designed for MCP communication.

For the MCP Server mode, you can choose between:
- **`stdio`** (Default): Uses standard input/output for communication, ideal for local AI assistant integration.
- **`streamable-http`**: Network-based transport for remote connections.

You can specify the server transport using the `--server-transport` flag.

---

## OAuth Authentication

`mcp-debug` supports OAuth 2.1 authentication for connecting to protected MCP servers. This allows you to debug servers that require user authorization.

### Basic OAuth Usage

**With Pre-Registered Client Credentials:**

```bash
./mcp-debug --oauth \
  --oauth-client-id your-client-id \
  --oauth-client-secret your-client-secret \
  --endpoint https://protected.server.com/mcp
```

**With Dynamic Client Registration (DCR):**

If the MCP server supports Dynamic Client Registration (RFC 7591), you can connect without pre-registered credentials:

```bash
./mcp-debug --oauth \
  --endpoint https://protected.server.com/mcp
```

The tool will automatically register itself with the authorization server and obtain a client ID.

### OAuth Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--oauth` | Enable OAuth authentication | `false` |
| `--oauth-client-id` | OAuth client ID (optional - uses DCR if not provided) | |
| `--oauth-client-secret` | OAuth client secret (optional) | |
| `--oauth-scopes` | OAuth scopes to request (comma-separated) | `mcp:tools,mcp:resources` |
| `--oauth-redirect-url` | Redirect URL for OAuth callback | `http://localhost:8765/callback` |
| `--oauth-pkce` | Use PKCE for authorization | `true` |

### OAuth Flow

When you run `mcp-debug` with OAuth enabled:

1. **mcp-debug** attempts to connect to the server
2. If OAuth endpoints are not provided, they are auto-discovered via server metadata
3. A local callback server starts on your machine (default: port 8765)
4. Your default browser opens to the authorization page
5. You log in and grant permissions
6. The authorization server redirects back to mcp-debug
7. **mcp-debug** exchanges the authorization code for an access token
8. Tokens are stored securely at `~/.mcp-debug/tokens.json`
9. The connection proceeds with authenticated requests

### Token Management

Tokens are managed automatically by mcp-go:

- **Stored in memory** during the session
- **Automatically refreshed** when expired
- **Not persisted** to disk (re-authorization required per session)

### OAuth with REPL Mode

```bash
./mcp-debug --repl --oauth \
  --oauth-client-id my-client-id \
  --oauth-client-secret my-secret \
  --endpoint https://api.example.com/mcp
```

Once authenticated, you can use all REPL commands normally. The token is automatically included in all requests.

### Connecting to Servers with Google OAuth (or other providers)

If your MCP server uses Google OAuth or another OAuth provider but you don't have client credentials, you have three options:

**Option 1: Dynamic Client Registration (Recommended)**

Try connecting without credentials to see if the server supports DCR:

```bash
./mcp-debug --oauth --endpoint https://your-server.com/mcp
```

If the server supports RFC 7591 Dynamic Client Registration, mcp-debug will automatically register and obtain credentials.

**Option 2: Register Your Own OAuth Application**

For Google OAuth:
1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select an existing one
3. Enable the required APIs
4. Go to "Credentials" → "Create Credentials" → "OAuth 2.0 Client ID"
5. Set application type to "Web application"
6. Add redirect URI: `http://localhost:8765/callback`
7. Copy the Client ID and Client Secret

Then use mcp-debug:

```bash
./mcp-debug --oauth \
  --oauth-client-id YOUR_GOOGLE_CLIENT_ID \
  --oauth-client-secret YOUR_GOOGLE_CLIENT_SECRET \
  --endpoint https://your-server.com/mcp
```

**Option 3: Contact Server Administrator**

If the server doesn't support DCR and you can't register your own application, contact the server administrator to:
- Provide you with client credentials, or
- Register mcp-debug as a client on their authorization server

### Security Best Practices

- **Never commit** OAuth client secrets to version control
- Use **environment variables** for sensitive credentials:
  ```bash
  export OAUTH_CLIENT_SECRET="your-secret"
  ./mcp-debug --oauth --oauth-client-id="$CLIENT_ID" --oauth-client-secret="$OAUTH_CLIENT_SECRET"
  ```
- The tool uses **PKCE** (Proof Key for Code Exchange) by default for enhanced security
- Tokens are stored with **restricted file permissions** (owner read/write only)
- **Dynamic Client Registration** is attempted automatically when no client ID is provided

---

## Command-Line Flags

Here are the most important flags to configure `mcp-debug`:

| Flag                | Description                                                                          | Default                        |
| ------------------- | ------------------------------------------------------------------------------------ | ------------------------------ |
| `--repl`            | Start the interactive REPL mode.                                                     | `false`                        |
| `--mcp-server`      | Run as an MCP server.                                                                | `false`                        |
| `--endpoint`        | The URL of the target MCP server.                                                    | `http://localhost:8090/mcp`    |
| `--transport`       | Client transport protocol (`streamable-http` only).                                  | `streamable-http`              |
| `--server-transport`| Server transport protocol (`stdio`, `streamable-http`).                                | `stdio`                        |
| `--listen-addr`     | Listen address for the `streamable-http` server.                                     | `:8899`                        |
| `--timeout`         | Timeout for waiting for notifications in normal mode.                                | `5m`                           |
| `--verbose`         | Enable verbose logging (shows keep-alive messages).                                  | `false`                        |
| `--json-rpc`        | Enable full logging of JSON-RPC messages.                                            | `false`                        |
| `--no-color`        | Disable colored output.                                                              | `false`                        |
| `--version`         | Show the application version.                                                        |                                |

---

## Shell Autocompletion

`mcp-debug` can generate autocompletion scripts for various shells. This helps you quickly complete commands and flags by pressing the `Tab` key.

To generate the script for your shell, use the `completion` command.

**Example for Bash:**

To load completion for the current session, run:
```bash
source <(./mcp-debug completion bash)
```

To make it permanent for every new session, add it to your `~/.bashrc` file:
```bash
echo "source <(./mcp-debug completion bash)" >> ~/.bashrc
```

**Example for Zsh:**

To load completion for the current session, run:
```bash
source <(./mcp-debug completion zsh)
```

To make it permanent, add to your `~/.zshrc`:
```bash
echo "source <(./mcp-debug completion zsh)" >> ~/.zshrc
```

For other shells like `fish` or `powershell`, you can get specific instructions by running:
```bash
./mcp-debug completion <shell> --help
```

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

**Connect with verbose logging and full JSON-RPC traffic:**
```bash
./mcp-debug --endpoint http://server.example.com/mcp --verbose --json-rpc
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