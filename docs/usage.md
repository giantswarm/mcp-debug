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
  - [OAuth Authentication](#oauth-authentication)
    - [Basic OAuth Usage](#basic-oauth-usage)
    - [OAuth Flags](#oauth-flags)
    - [OAuth Flow](#oauth-flow)
    - [Token Management](#token-management)
    - [OpenID Connect (OIDC) Support](#openid-connect-oidc-support)
    - [OAuth with REPL Mode](#oauth-with-repl-mode)
    - [Connecting to Servers with Google OAuth (or other providers)](#connecting-to-servers-with-google-oauth-or-other-providers)
    - [Understanding OAuth Scopes](#understanding-oauth-scopes)
    - [Concurrent Authorization Attempts](#concurrent-authorization-attempts)
    - [Security Best Practices](#security-best-practices)
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

**With Authenticated Dynamic Client Registration:**

Some authorization servers require a registration access token for DCR (per RFC 7591 Section 3.2). If you encounter an error like "Registration access token required", you need to provide a registration token:

```bash
./mcp-debug --oauth \
  --oauth-registration-token your-registration-token \
  --endpoint https://protected.server.com/mcp
```

**Security Considerations for Registration Tokens:**

The registration access token is a sensitive credential that must be protected:

1. **HTTPS Required**: The token will ONLY be transmitted over HTTPS connections. Attempts to use HTTP will fail with a security error.
2. **Endpoint Validation**: The token is only injected into recognized OAuth registration endpoints (e.g., `/oauth/register`, `/oauth2/registration`) to prevent token leakage.
3. **Storage**: Store registration tokens in environment variables or secure vaults, never commit them to version control.
4. **Token Lifecycle**: 
   - Obtain tokens from your authorization server administrator
   - Tokens may have expiration times - check with your administrator
   - Rotate tokens regularly as part of security best practices
5. **Monitoring**: DCR requests are logged for audit purposes

The registration token is provided as a Bearer token in the Authorization header during the client registration request. Contact your authorization server administrator to obtain a registration access token if DCR authentication is enabled.

**Token Rotation:**

If you need to rotate your registration token:

1. Obtain a new token from your authorization server administrator
2. Update your environment variable or command-line flag
3. The old token will no longer be used on the next connection
4. Invalidate the old token with your authorization server if possible

### OAuth Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--oauth` | Enable OAuth authentication | `false` |
| `--oauth-client-id` | OAuth client ID (optional - uses DCR if not provided) | |
| `--oauth-client-secret` | OAuth client secret (optional) | |
| `--oauth-scopes` | OAuth scopes to request (used with manual mode) | (none) |
| `--oauth-scope-mode` | Scope selection mode: `auto` (MCP spec priority) or `manual` (use --oauth-scopes only) | `auto` |
| `--oauth-redirect-url` | Redirect URL for OAuth callback | `http://localhost:8765/callback` |
| `--oauth-pkce` | Use PKCE for authorization | `true` |
| `--oauth-timeout` | Maximum time to wait for OAuth authorization | `5m` |
| `--oauth-oidc` | Enable OpenID Connect features (nonce validation) | `false` |
| `--oauth-registration-token` | OAuth registration access token for authenticated DCR | |
| `--oauth-resource-uri` | Target resource URI for RFC 8707 (auto-derived if not specified) | (auto-derived) |
| `--oauth-skip-resource-param` | Skip RFC 8707 resource parameter (for testing older servers) | `false` |
| `--oauth-skip-resource-metadata` | Skip RFC 9728 Protected Resource Metadata discovery (for testing) | `false` |
| `--oauth-preferred-auth-server` | Preferred authorization server URL when multiple are available | |
| `--oauth-skip-pkce-validation` | Skip PKCE support validation in AS metadata (DANGEROUS - testing only) | `false` |
| `--oauth-skip-auth-server-discovery` | Skip RFC 8414 AS Metadata discovery (for testing) | `false` |

### RFC 8707 Resource Indicators

`mcp-debug` implements [RFC 8707: Resource Indicators for OAuth 2.0](https://www.rfc-editor.org/rfc/rfc8707.html) to improve token security through audience binding. This ensures that access tokens are bound to the specific MCP server you're connecting to, preventing token misuse if they're intercepted.

**How It Works:**

When OAuth is enabled, `mcp-debug` automatically:
1. Derives a canonical resource URI from your endpoint (e.g., `https://mcp.example.com/mcp`)
2. Includes this URI as the `resource` parameter in:
   - Authorization requests (when requesting access)
   - Token exchange requests (when obtaining tokens)
   - Refresh token requests (when renewing tokens)

**Resource URI Canonicalization:**

The resource URI follows RFC 8707 rules:
- Lowercase scheme and host: `HTTPS://Example.COM/mcp` → `https://example.com/mcp`
- Standard ports omitted: `https://example.com:443/mcp` → `https://example.com/mcp`
- Non-standard ports included: `https://example.com:8443/mcp` → `https://example.com:8443/mcp`
- Path preserved: `https://example.com/api/v1/mcp` → `https://example.com/api/v1/mcp`
- Trailing slashes removed: `https://example.com/mcp/` → `https://example.com/mcp`

**Manual Resource URI:**

You can explicitly specify the resource URI if needed:

```bash
./mcp-debug --oauth \
  --oauth-resource-uri "https://mcp.example.com/mcp" \
  --endpoint https://mcp.example.com:8443/mcp
```

**Testing with Older Servers:**

If you're connecting to an older authorization server that doesn't support RFC 8707, you can disable the resource parameter:

```bash
./mcp-debug --oauth \
  --oauth-skip-resource-param \
  --endpoint https://legacy-server.com/mcp
```

**Security Note:** Disabling the resource parameter weakens token audience binding. Only use `--oauth-skip-resource-param` for testing compatibility with legacy servers.

### RFC 9728 Protected Resource Metadata Discovery

`mcp-debug` implements [RFC 9728: OAuth 2.0 Protected Resource Metadata](https://datatracker.ietf.org/doc/html/rfc9728) to automatically discover authorization server locations and required scopes from the MCP server.

**How It Works:**

When OAuth is enabled, `mcp-debug` automatically:
1. Makes an initial request to the MCP server (expects 401 Unauthorized)
2. Parses the `WWW-Authenticate` header to extract:
   - `resource_metadata` URL (if provided)
   - `scope` parameter (required scopes for this resource)
3. Fetches Protected Resource Metadata from:
   - The `resource_metadata` URL if provided, OR
   - Well-known URIs in priority order:
     - `https://mcp.example.com/.well-known/oauth-protected-resource/mcp` (path-based)
     - `https://mcp.example.com/.well-known/oauth-protected-resource` (root)
4. Extracts authorization server URLs and supported scopes
5. Proceeds with OAuth authorization using the discovered authorization server

**Example WWW-Authenticate Header:**

```
WWW-Authenticate: Bearer resource_metadata="https://mcp.example.com/.well-known/oauth-protected-resource",
                         scope="files:read user:profile"
```

**Example Protected Resource Metadata:**

```json
{
  "resource": "https://mcp.example.com",
  "authorization_servers": [
    "https://auth.example.com",
    "https://auth-backup.example.com"
  ],
  "scopes_supported": ["files:read", "files:write", "user:profile"],
  "bearer_methods_supported": ["header"]
}
```

**Multiple Authorization Servers:**

If the protected resource metadata provides multiple authorization servers, `mcp-debug` uses the first one by default. You can specify a preferred server:

```bash
./mcp-debug --oauth \
  --oauth-preferred-auth-server https://auth-backup.example.com \
  --endpoint https://mcp.example.com/mcp
```

**Testing with Older Servers:**

If you're connecting to an older MCP server that doesn't support RFC 9728, you can disable Protected Resource Metadata discovery:

```bash
./mcp-debug --oauth \
  --oauth-skip-resource-metadata \
  --endpoint https://legacy-server.com/mcp
```

When disabled, `mcp-debug` falls back to existing OAuth discovery mechanisms (via mcp-go library).

**Security Note:** RFC 9728 discovery is enabled by default as it's required by the MCP specification for proper authorization server discovery and scope selection.

#### Authorization Server Metadata Discovery (RFC 8414)

Once the authorization server URL is discovered via RFC 9728, `mcp-debug` automatically discovers authorization server metadata per RFC 8414 (OAuth 2.0 Authorization Server Metadata) and OpenID Connect Discovery 1.0.

**Multi-Endpoint Probing:**

For issuer URLs **with path components** (e.g., `https://auth.example.com/tenant1`), `mcp-debug` probes endpoints in this order:

1. OAuth 2.0 with path insertion: `https://auth.example.com/.well-known/oauth-authorization-server/tenant1`
2. OIDC with path insertion: `https://auth.example.com/.well-known/openid-configuration/tenant1`
3. OIDC path appending: `https://auth.example.com/tenant1/.well-known/openid-configuration`

For issuer URLs **without path components** (e.g., `https://auth.example.com`), it probes:

1. OAuth 2.0: `https://auth.example.com/.well-known/oauth-authorization-server`
2. OIDC: `https://auth.example.com/.well-known/openid-configuration`

The first successfully retrieved metadata document is used for the OAuth flow.

**PKCE Support Validation:**

Per the MCP specification (2025-11-25), authorization servers **MUST** support PKCE (Proof Key for Code Exchange) with the S256 method. `mcp-debug` enforces this requirement by:

- Checking for `code_challenge_methods_supported` in the AS metadata
- Verifying that `S256` is listed as a supported method
- **Refusing to proceed** if PKCE support is not advertised (fail closed for security)

If you need to test with an older authorization server that supports PKCE but doesn't advertise it:

```bash
./mcp-debug --oauth \
  --oauth-skip-pkce-validation \
  --endpoint https://legacy-auth-server.com/mcp
```

**Warning:** The `--oauth-skip-pkce-validation` flag weakens security and should only be used for testing. PKCE is a critical security feature that prevents authorization code interception attacks.

**Disabling AS Metadata Discovery:**

For testing with older servers or pre-configured endpoints:

```bash
./mcp-debug --oauth \
  --oauth-skip-auth-server-discovery \
  --endpoint https://legacy-server.com/mcp
```

When AS metadata discovery is disabled, `mcp-debug` relies on mcp-go's internal discovery mechanisms.

### OAuth Flow

When you run `mcp-debug` with OAuth enabled:

1. **mcp-debug** attempts to connect to the server
2. If OAuth endpoints are not provided, they are auto-discovered via server metadata
3. The resource URI is derived from the endpoint for RFC 8707
4. A local callback server starts on your machine (default: port 8765)
5. Your default browser opens to the authorization page (with resource parameter)
6. You log in and grant permissions
7. The authorization server redirects back to mcp-debug
8. **mcp-debug** exchanges the authorization code for an access token (with resource parameter)
9. The connection proceeds with authenticated requests

### Token Management

Tokens are managed automatically by mcp-go:

- **Stored in memory only** during the session (for security)
- **Automatically refreshed** when expired (if refresh tokens are supported)
- **Not persisted** to disk - you'll need to re-authenticate each time you run mcp-debug
- This provides better security by preventing token theft from disk storage
- Token refresh events are logged for security auditing

### OpenID Connect (OIDC) Support

For MCP servers using OpenID Connect, enable OIDC features:

```bash
./mcp-debug --oauth --oauth-oidc \
  --endpoint https://oidc-server.com/mcp
```

**What OIDC mode enables:**
- **Nonce parameter** generation and validation for additional security
- **ID token awareness** (validation is delegated to mcp-go library)
- **Enhanced logging** for OIDC-specific security features

**When to use OIDC mode:**
- Your MCP server explicitly uses OpenID Connect (not just OAuth 2.0)
- You need additional security guarantees beyond PKCE
- You're debugging OIDC-specific issues

**Note:** Most MCP servers use OAuth 2.0 (not OIDC), so this flag is typically not needed.

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

**What happens during DCR:**
1. mcp-debug discovers the OAuth authorization server metadata
2. It checks if the server supports dynamic client registration
3. If supported, it sends a registration request with:
   - Client name: `mcp-debug` (or `mcp-debug/v1.2.3` with version)
   - Redirect URI: `http://localhost:8765/callback`
   - Grant types: `authorization_code`, `refresh_token`
4. The server responds with a `client_id` (and optionally `client_secret`)
5. mcp-debug uses these credentials for the OAuth flow
6. You complete the authorization in your browser
7. mcp-debug exchanges the code for tokens and connects

**Example DCR session:**
```bash
$ ./mcp-debug --oauth --endpoint https://api.example.com/mcp
[2025-11-22 10:15:30] OAuth enabled - will attempt Dynamic Client Registration
[2025-11-22 10:15:31] OAuth authorization required
[2025-11-22 10:15:31] No client ID configured, attempting dynamic client registration...
[2025-11-22 10:15:32] ✓ Client registered successfully with ID: dcr_client_abc123xyz
[2025-11-22 10:15:32] Opening browser for authorization...
[2025-11-22 10:15:33] Waiting for authorization...
[2025-11-22 10:15:45] ✓ Authorization code received
[2025-11-22 10:15:45] Exchanging code for access token...
[2025-11-22 10:15:46] ✓ Access token obtained successfully!
[2025-11-22 10:15:46] ✓ Session initialized successfully (protocol: 2024-11-05)
```

**Custom timeout for authorization:**
```bash
# Wait up to 10 minutes for user to complete authorization
./mcp-debug --oauth --oauth-timeout 10m --endpoint https://api.example.com/mcp
```

**If DCR fails with "Registration access token required":**

Some servers require authenticated DCR. If you see this error:

```
Dynamic client registration failed: registration request failed: OAuth error: invalid_token - Registration access token required
```

You need to provide a registration access token:

```bash
./mcp-debug --oauth \
  --oauth-registration-token YOUR_REGISTRATION_TOKEN \
  --endpoint https://api.example.com/mcp
```

Contact your server administrator to obtain a registration access token.

**Security Errors:**

If you encounter security-related errors:

1. **"security: registration token can only be sent over HTTPS"**
   - The endpoint URL must use `https://` scheme
   - Registration tokens are sensitive credentials and will not be transmitted over unencrypted HTTP
   - Solution: Ensure your endpoint URL starts with `https://`

2. **"security: authorization header already present"**
   - There is a conflict with an existing Authorization header
   - This prevents accidental credential overwrites
   - Solution: Check your HTTP client configuration for conflicting authentication settings

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

### Understanding OAuth Scopes

**Important:** OAuth scopes are **optional**. By default, no scopes are sent. The scopes you specify with `--oauth-scopes` are for the **MCP server**, not for the underlying service provider (like Google).

**Scope Usage:**

```bash
# ✅ Correct: No scopes (default)
./mcp-debug --oauth \
  --endpoint https://mcp-server-for-google.com/mcp

# ✅ Correct: Requesting specific MCP server scopes
./mcp-debug --oauth \
  --oauth-scopes "mcp:tools,mcp:resources" \
  --endpoint https://mcp-server-for-google.com/mcp

# ❌ Incorrect: Don't specify Google API scopes here
./mcp-debug --oauth \
  --oauth-scopes "https://www.googleapis.com/auth/gmail.readonly" \
  --endpoint https://mcp-server-for-google.com/mcp
```

**Why?** In a federated authentication setup:
1. **mcp-debug** authenticates with the **MCP server** (using MCP scopes)
2. The **MCP server** authenticates with **Google** (using Google API scopes)
3. The Google API scopes are configured in the MCP server, not in mcp-debug

When you authorize, you'll see Google's consent screen with the Google API scopes that the MCP server has requested, but you configure only MCP scopes in mcp-debug.

### OAuth Scope Selection Modes

`mcp-debug` implements intelligent scope selection according to the MCP specification, with two modes: **auto** (default, secure) and **manual** (advanced users).

#### Auto Mode (Recommended)

Auto mode follows the MCP specification's scope selection priority to ensure least-privilege access:

```bash
# Auto mode is the default - no flag needed
./mcp-debug --oauth --endpoint https://mcp.example.com/mcp
```

**Priority Order:**
1. **WWW-Authenticate scope** (most specific): If the server returns a 401 with required scopes, use those
2. **Protected Resource Metadata scopes**: Use scopes discovered via RFC 9728
3. **Omit scope parameter**: If no scopes are discovered, omit the scope parameter (least privilege)

**Benefits:**
- Follows MCP specification for security
- Requests only the scopes the server needs
- Prevents over-privileged tokens
- Automatically adapts to server requirements

**Example:**
```bash
$ ./mcp-debug --oauth --endpoint https://mcp.example.com/mcp
[INFO] Scope selection mode: auto (MCP spec priority)
[INFO] Selected scopes from Protected Resource Metadata: [mcp:read mcp:write]
```

#### Manual Mode (Advanced)

Manual mode allows you to explicitly specify scopes, overriding server recommendations. This is useful for testing or when you know the exact scopes required.

```bash
# Manually specify scopes
./mcp-debug --oauth \
  --oauth-scope-mode manual \
  --oauth-scopes "mcp:admin,mcp:debug" \
  --endpoint https://mcp.example.com/mcp
```

**Security Implications:**
- Manual scopes may differ from what the server recommends
- Can lead to authorization failures if scopes are insufficient
- Can result in over-privileged tokens if scopes are excessive
- **Warning logged** when manual scopes differ from discovered scopes

**When to use manual mode:**
- Testing specific scope configurations
- Debugging authorization issues
- Working with custom server implementations
- You know the exact scopes required and want to override discovery

**Example with warning:**
```bash
$ ./mcp-debug --oauth --oauth-scope-mode manual \
    --oauth-scopes "mcp:admin" --endpoint https://mcp.example.com/mcp
[INFO] Scope selection mode: manual
[INFO] Requested scopes (manual): [mcp:admin]
[WARNING] Manual scope mode: requested scopes [mcp:admin] differ from server-discovered scopes [mcp:read]
[WARNING] This may lead to authorization failures or over-privileged tokens
```

#### Scope Selection Best Practices

1. **Use auto mode by default** - It implements the principle of least privilege
2. **Only use manual mode** when you have a specific reason
3. **Monitor warnings** - They indicate potential scope mismatches
4. **Test authorization** - Verify manual scopes work before production use
5. **Document custom scopes** - If using manual mode, document why specific scopes are needed

### Concurrent Authorization Attempts

**Important:** Do not run multiple instances of mcp-debug with OAuth enabled simultaneously using the same redirect URL. This will cause conflicts:

- Only one callback server can listen on `http://localhost:8765/callback` at a time
- The second instance will fail to start the callback server
- Authorization responses may go to the wrong instance

**If you need multiple concurrent sessions:**

```bash
# First instance (default port 8765)
./mcp-debug --oauth --endpoint https://server1.com/mcp

# Second instance (different port)
./mcp-debug --oauth \
  --oauth-redirect-url "http://localhost:8766/callback" \
  --endpoint https://server2.com/mcp
```

**Note:** You'll need to register each redirect URL with your OAuth provider/MCP server.

### Security Best Practices

- **Never commit** OAuth client secrets to version control
- Use **environment variables** for sensitive credentials:
  ```bash
  export OAUTH_CLIENT_SECRET="your-secret"
  ./mcp-debug --oauth --oauth-client-id="$CLIENT_ID" --oauth-client-secret="$OAUTH_CLIENT_SECRET"
  ```
- The tool uses **PKCE** (Proof Key for Code Exchange) by default for enhanced security
- Tokens are stored **in-memory only** during the session (not persisted to disk)
- **Token refresh** is handled automatically when tokens expire
- **Dynamic Client Registration** is attempted automatically when no client ID is provided
- **HTTPS callbacks** are not supported - only `http://localhost:PORT/callback` is allowed for security reasons

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