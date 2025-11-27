# Example 1: Basic OAuth Authentication

This example demonstrates the simplest OAuth flow with automatic discovery and Dynamic Client Registration.

## Scenario

You want to connect to an OAuth-protected MCP server without any pre-configured credentials. The server supports:

- RFC 9728 (Protected Resource Metadata)
- RFC 8414 (Authorization Server Metadata)
- RFC 7591 (Dynamic Client Registration)
- RFC 8707 (Resource Indicators)
- PKCE with S256 method

## Prerequisites

- `mcp-debug` installed
- Network access to MCP server
- Web browser for authorization

## Command

```bash
./mcp-debug --oauth --endpoint https://mcp.example.com/mcp
```

## What Happens

### Step 1: Initial Connection Attempt

```
[INFO] OAuth enabled - will attempt Dynamic Client Registration
[INFO] Attempting to connect to MCP server...
```

`mcp-debug` tries to connect to the MCP server.

### Step 2: Protected Resource Metadata Discovery

```
[INFO] Received 401 Unauthorized with WWW-Authenticate header
[INFO] Protected Resource Metadata discovery enabled
[INFO] Fetching Protected Resource Metadata from https://mcp.example.com/.well-known/oauth-protected-resource
[INFO] ✓ Protected Resource Metadata retrieved successfully
[INFO] Discovered authorization servers: [https://auth.example.com]
[INFO] Discovered required scopes: [mcp:read mcp:write]
```

The server responds with 401, providing metadata URL and required scopes.

### Step 3: Authorization Server Metadata Discovery

```
[INFO] Attempting AS Metadata discovery for issuer: https://auth.example.com
[INFO] Trying: https://auth.example.com/.well-known/oauth-authorization-server
[INFO] ✓ AS Metadata retrieved successfully
[INFO] Authorization endpoint: https://auth.example.com/oauth/authorize
[INFO] Token endpoint: https://auth.example.com/oauth/token
[INFO] Registration endpoint: https://auth.example.com/oauth/register
```

`mcp-debug` discovers OAuth endpoints automatically.

### Step 4: PKCE Validation

```
[INFO] Validating PKCE support...
[INFO] PKCE methods supported: [S256]
[INFO] ✓ PKCE validation passed
```

Ensures the authorization server supports required security features.

### Step 5: Dynamic Client Registration

```
[INFO] No client ID configured, attempting dynamic client registration...
[INFO] Registration endpoint: https://auth.example.com/oauth/register
[INFO] Sending registration request...
[INFO] ✓ Client registered successfully with ID: dcr_abc123xyz
```

`mcp-debug` automatically registers as an OAuth client.

### Step 6: Authorization

```
[INFO] Resource URI: https://mcp.example.com/mcp
[INFO] Selected scopes (Priority 1 - WWW-Authenticate): [mcp:read mcp:write]
[INFO] Opening browser for authorization...
[INFO] Callback server listening on http://localhost:8765
[INFO] Please visit: https://auth.example.com/oauth/authorize?...
[INFO] Waiting for authorization...
```

Your browser opens to the authorization page. You log in and grant permissions.

### Step 7: Authorization Code Exchange

```
[INFO] ✓ Authorization code received
[INFO] Exchanging code for access token...
[INFO] Including resource parameter: https://mcp.example.com/mcp
[INFO] ✓ Access token obtained successfully!
```

`mcp-debug` exchanges the authorization code for an access token.

### Step 8: MCP Connection

```
[INFO] ✓ Session initialized successfully (protocol: 2024-11-05)
[INFO] Connected to MCP server
[INFO] Server capabilities: tools, resources, prompts
```

You're now connected and authenticated!

## Full Output Example

```bash
$ ./mcp-debug --oauth --endpoint https://mcp.example.com/mcp

[INFO] OAuth enabled - will attempt Dynamic Client Registration
[INFO] Attempting to connect to MCP server...
[INFO] Received 401 Unauthorized with WWW-Authenticate header
[INFO] Fetching Protected Resource Metadata...
[INFO] ✓ Discovered authorization servers: [https://auth.example.com]
[INFO] ✓ Discovered scopes: [mcp:read mcp:write]
[INFO] Attempting AS Metadata discovery...
[INFO] ✓ AS Metadata retrieved
[INFO] ✓ PKCE validation passed
[INFO] Attempting Dynamic Client Registration...
[INFO] ✓ Client registered: dcr_abc123xyz
[INFO] Opening browser for authorization...
[INFO] ✓ Authorization code received
[INFO] ✓ Access token obtained
[INFO] ✓ Connected to MCP server

Press Ctrl+C to exit...
```

## Security Features Enabled

- ✅ **PKCE**: Authorization code protected with code challenge
- ✅ **Resource Indicators**: Token bound to `https://mcp.example.com/mcp`
- ✅ **Scope Minimization**: Requested only `[mcp:read mcp:write]`
- ✅ **Automatic Discovery**: No manual configuration needed
- ✅ **Step-Up Authorization**: Ready to handle additional scope requests

## Using in REPL Mode

```bash
./mcp-debug --repl --oauth --endpoint https://mcp.example.com/mcp
```

After authorization, you get an interactive shell:

```
OAuth authorization successful!
Connected to MCP server

MCP Debug REPL. Type 'help' for available commands.

>> tools
Available tools:
- list_files
- read_file
- search_files

>> exec list_files '{"path": "/"}'
{
  "files": [
    "document1.txt",
    "document2.md"
  ]
}
```

## Troubleshooting

### Browser Doesn't Open

If the browser doesn't open automatically, copy the URL from the logs:

```
INFO: Please visit: https://auth.example.com/oauth/authorize?...
```

Paste it into your browser manually.

### DCR Not Supported

If the server doesn't support Dynamic Client Registration:

```
ERROR: No registration endpoint found in AS metadata
```

You need pre-registered credentials:

```bash
./mcp-debug --oauth \
  --oauth-client-id "your-client-id" \
  --oauth-client-secret "your-secret" \
  --endpoint https://mcp.example.com/mcp
```

See [Example 2](02-pre-registered.md) for details.

## Next Steps

- [Example 2: Pre-Registered Client](02-pre-registered.md)
- [Example 3: Manual Scope Configuration](03-manual-scopes.md)
- [Example 4: Testing Legacy Servers](04-legacy-servers.md)

