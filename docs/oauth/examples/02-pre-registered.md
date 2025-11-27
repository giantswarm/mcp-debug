# Example 2: Pre-Registered OAuth Client

This example shows how to use pre-configured OAuth client credentials instead of Dynamic Client Registration.

## Scenario

You have already registered an OAuth application with the authorization server and have:

- Client ID
- Client secret (optional, depending on client type)

This is common in production deployments where clients are vetted and pre-approved.

## Prerequisites

- OAuth client credentials registered with authorization server
- Redirect URI `http://localhost:8765/callback` registered with your client

## Getting Client Credentials

### For Google OAuth

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create or select a project
3. Navigate to "APIs & Services" → "Credentials"
4. Click "Create Credentials" → "OAuth 2.0 Client ID"
5. Select application type: "Desktop app" or "Web application"
6. Add authorized redirect URI: `http://localhost:8765/callback`
7. Save and copy the Client ID and Client Secret

### For Other Providers

Consult your authorization server documentation or contact your administrator.

## Command

### With Client Secret (Confidential Client)

```bash
./mcp-debug --oauth \
  --oauth-client-id "your-client-id-abc123" \
  --oauth-client-secret "your-client-secret-xyz789" \
  --endpoint https://mcp.example.com/mcp
```

### Public Client (No Secret)

For native/desktop applications (recommended per OAuth 2.1):

```bash
./mcp-debug --oauth \
  --oauth-client-id "your-public-client-id" \
  --endpoint https://mcp.example.com/mcp
```

Public clients rely on PKCE for security (no client secret).

## Using Environment Variables

**Recommended** for security:

```bash
# Set environment variables
export OAUTH_CLIENT_ID="your-client-id-abc123"
export OAUTH_CLIENT_SECRET="your-client-secret-xyz789"

# Use in command
./mcp-debug --oauth \
  --oauth-client-id "$OAUTH_CLIENT_ID" \
  --oauth-client-secret "$OAUTH_CLIENT_SECRET" \
  --endpoint https://mcp.example.com/mcp
```

**Why environment variables?**
- Secrets not visible in command history
- Secrets not visible in process list (`ps`, `htop`)
- Easier credential rotation

## What Happens

The flow is similar to Example 1, but skips Dynamic Client Registration:

```
[INFO] OAuth enabled
[INFO] Client ID configured: your-client-id-abc123
[INFO] Protected Resource Metadata discovery...
[INFO] ✓ Discovered authorization servers
[INFO] AS Metadata discovery...
[INFO] ✓ PKCE validation passed
[INFO] Opening browser for authorization...
[INFO] ✓ Authorization code received
[INFO] ✓ Access token obtained
[INFO] ✓ Connected to MCP server
```

## Custom Redirect Port

If you registered a different redirect URI:

```bash
./mcp-debug --oauth \
  --oauth-client-id "$OAUTH_CLIENT_ID" \
  --oauth-client-secret "$OAUTH_CLIENT_SECRET" \
  --oauth-redirect-url "http://localhost:9000/callback" \
  --endpoint https://mcp.example.com/mcp
```

**Important:** The redirect URL must match exactly what you registered.

## Security Best Practices

### DO

✅ Store secrets in environment variables
✅ Use public clients when possible (PKCE without secret)
✅ Rotate client secrets regularly
✅ Use different credentials for dev/staging/prod

### DON'T

❌ Hard-code secrets in scripts
❌ Commit secrets to version control
❌ Share client secrets via email/chat
❌ Use the same client ID across environments
❌ Put secrets in command-line arguments

### Example: Secure Credential Management

```bash
# .env file (add to .gitignore)
OAUTH_CLIENT_ID=your-client-id
OAUTH_CLIENT_SECRET=your-secret

# Load and use
source .env
./mcp-debug --oauth \
  --oauth-client-id "$OAUTH_CLIENT_ID" \
  --oauth-client-secret "$OAUTH_CLIENT_SECRET" \
  --endpoint https://mcp.example.com/mcp
```

## Troubleshooting

### "Invalid client"

```
ERROR: Token request failed: invalid_client
```

**Causes:**

1. Wrong client ID
2. Wrong client secret
3. Client secret required but not provided

**Solution:** Verify credentials with authorization server administrator.

### "Unauthorized redirect URI"

```
ERROR: OAuth error: invalid_request
ERROR: Invalid redirect_uri parameter
```

**Cause:** Redirect URI not registered for this client

**Solution:** Register `http://localhost:8765/callback` or use custom port:

```bash
./mcp-debug --oauth \
  --oauth-client-id "$OAUTH_CLIENT_ID" \
  --oauth-redirect-url "http://localhost:REGISTERED_PORT/callback" \
  --endpoint https://mcp.example.com/mcp
```

## Comparison: DCR vs Pre-Registration

| Feature | DCR (Example 1) | Pre-Registration (Example 2) |
|---------|-----------------|------------------------------|
| Setup | Automatic | Manual (register with AS) |
| Credentials | Generated at runtime | Fixed, known in advance |
| Use Case | Testing, development | Production, enterprise |
| Security | Depends on AS policy | Controlled by organization |
| Client Secret | Usually not provided | Optional (confidential clients) |
| Flexibility | High (works with any server) | Lower (tied to specific AS) |

## Next Steps

- [Example 3: Manual Scope Configuration](03-manual-scopes.md)
- [Example 4: Testing Legacy Servers](04-legacy-servers.md)
- [Security Documentation](../security.md)

