# Example 3: Manual Scope Configuration

This example demonstrates how to explicitly specify OAuth scopes instead of relying on automatic discovery.

## Scenario

You want to:

- Request specific scopes upfront
- Override automatic scope discovery
- Test authorization with different scope combinations
- Ensure certain permissions are granted from the start

## When to Use Manual Scopes

**Good reasons:**

- You know exactly which scopes are needed
- Server doesn't advertise scopes correctly
- Testing specific scope combinations
- Need broader permissions than auto mode requests

**Better to use auto mode when:**

- You want minimal permissions (secure by default)
- Server correctly advertises required scopes
- You're not sure which scopes are needed

## Command

### Basic Manual Scope Selection

```bash
./mcp-debug --oauth \
  --oauth-scope-mode manual \
  --oauth-scopes "mcp:read,mcp:write,user:profile" \
  --endpoint https://mcp.example.com/mcp
```

### With Pre-Registered Client

```bash
./mcp-debug --oauth \
  --oauth-client-id "$CLIENT_ID" \
  --oauth-client-secret "$CLIENT_SECRET" \
  --oauth-scope-mode manual \
  --oauth-scopes "mcp:admin,mcp:debug,system:logs" \
  --endpoint https://mcp.example.com/mcp
```

## What Happens

### Auto Mode (for comparison)

```
[INFO] Scope selection mode: auto (MCP spec priority)
[INFO] Selected scopes from WWW-Authenticate: [mcp:read]
```

Auto mode requests only what the server requires.

### Manual Mode

```
[INFO] Scope selection mode: manual
[INFO] Requested scopes (manual): [mcp:read mcp:write user:profile]
[WARNING] Manual scope mode: requested scopes differ from server-discovered scopes
[WARNING] This may lead to authorization failures or over-privileged tokens
[INFO] Proceeding with manual scopes...
[INFO] Opening browser for authorization...
```

Manual mode uses your specified scopes and warns about differences.

## Examples by Use Case

### 1. Request All Available Scopes

```bash
./mcp-debug --oauth \
  --oauth-scope-mode manual \
  --oauth-scopes "mcp:read,mcp:write,mcp:admin,mcp:delete,user:profile,user:email" \
  --endpoint https://mcp.example.com/mcp
```

**Use case:** Administrative access, testing

**Risk:** Over-privileged token

### 2. Request No Scopes (Server Defaults)

```bash
./mcp-debug --oauth \
  --oauth-scope-mode manual \
  --oauth-scopes "" \
  --endpoint https://mcp.example.com/mcp
```

**Use case:** Let server assign default scopes

**Note:** Omits the `scope` parameter entirely.

### 3. Request Minimal Read-Only Access

```bash
./mcp-debug --oauth \
  --oauth-scope-mode manual \
  --oauth-scopes "mcp:read" \
  --endpoint https://mcp.example.com/mcp
```

**Use case:** Read-only operations, security-conscious access

**Risk:** May trigger insufficient_scope errors

### 4. Request Read + Write

```bash
./mcp-debug --oauth \
  --oauth-scope-mode manual \
  --oauth-scopes "mcp:read,mcp:write" \
  --endpoint https://mcp.example.com/mcp
```

**Use case:** General-purpose access

**Balance:** Good balance of functionality and security

## Combining with Step-Up Authorization

Manual scopes work with step-up authorization:

```bash
./mcp-debug --repl --oauth \
  --oauth-scope-mode manual \
  --oauth-scopes "mcp:read" \
  --endpoint https://mcp.example.com/mcp
```

**What happens:**

1. Initial authorization with `mcp:read`
2. You try to delete a resource
3. Server responds with `403 Forbidden` + `insufficient_scope`
4. Step-up authorization automatically requests additional scopes
5. You authorize again in browser
6. Operation succeeds with new token

**Output:**

```
>> exec delete_resource '{"id": "123"}'
[WARNING] Insufficient scope detected for DELETE /mcp/resource
[INFO] Required scopes: [mcp:write mcp:delete]
[INFO] Requesting additional permissions...
[INFO] Opening browser for authorization...
[INFO] ✓ Additional permissions granted
[INFO] Retrying request with new token...
[INFO] ✓ Request successful
Resource deleted successfully
```

## Scope Mismatch Warnings

When manual scopes differ from discovered scopes:

```
[INFO] Scope selection mode: manual
[INFO] Requested scopes (manual): [mcp:admin mcp:debug]
[INFO] Discovered required scopes from WWW-Authenticate: [mcp:read]
[WARNING] Manual scope mode: requested scopes [mcp:admin mcp:debug] differ from server-discovered scopes [mcp:read]
[WARNING] This may lead to authorization failures or over-privileged tokens
```

**What this means:**

- **Over-privileged:** Requesting more than needed (`mcp:admin` vs `mcp:read`)
- **Security risk:** Token has broader permissions than necessary
- **Privacy concern:** User sees larger consent prompt

**Alternative:** If server's scope discovery is correct, use auto mode instead.

## Testing Scope Validation

### Test Authorization Server Scope Validation

```bash
# Request invalid scope
./mcp-debug --oauth \
  --oauth-scope-mode manual \
  --oauth-scopes "invalid-scope-name" \
  --endpoint https://mcp.example.com/mcp
```

**Expected:** Authorization server rejects invalid scope

### Test Insufficient Scopes

```bash
# Request only read, then try write operation
./mcp-debug --repl --oauth \
  --oauth-scope-mode manual \
  --oauth-scopes "mcp:read" \
  --endpoint https://mcp.example.com/mcp

>> exec write_file '{"path": "/test.txt", "content": "data"}'
ERROR: 403 Forbidden - Insufficient scope
```

## Debugging Scope Issues

### Enable Verbose Logging

```bash
./mcp-debug --oauth --verbose \
  --oauth-scope-mode manual \
  --oauth-scopes "mcp:read,mcp:write" \
  --endpoint https://mcp.example.com/mcp
```

**Logs show:**

- Discovered scopes from server
- Your manual scopes
- Comparison and warnings
- Authorization request with scopes
- Token response (scopes granted)

### Compare Auto vs Manual

**Auto mode:**

```bash
./mcp-debug --oauth --verbose --endpoint https://mcp.example.com/mcp 2>&1 | grep "scope"
```

**Manual mode:**

```bash
./mcp-debug --oauth --verbose \
  --oauth-scope-mode manual \
  --oauth-scopes "mcp:read,mcp:write" \
  --endpoint https://mcp.example.com/mcp 2>&1 | grep "scope"
```

Compare the scopes requested and granted.

## Best Practices

### DO

✅ Use auto mode as default (secure by default)
✅ Document why manual scopes are needed
✅ Request minimal scopes necessary
✅ Review consent screen before approving
✅ Use manual mode for testing specific scenarios

### DON'T

❌ Blindly request all available scopes
❌ Use manual mode without understanding risks
❌ Ignore scope mismatch warnings
❌ Request scopes you don't need

## Troubleshooting

### Authorization Fails with Manual Scopes

**Error:**

```
ERROR: OAuth error: invalid_scope
ERROR: Scope 'mcp:admin' not available
```

**Solution:** Check available scopes:

```bash
curl https://mcp.example.com/.well-known/oauth-protected-resource | jq .scopes_supported
```

Request only available scopes.

### Insufficient Scope Despite Manual Scopes

**Error:**

```
ERROR: 403 Forbidden - Insufficient scope
ERROR: Required scopes: [mcp:write]
ERROR: Granted scopes: [mcp:read]
```

**Cause:** Authorization server didn't grant requested scopes

**Solution:** Check why server rejected scopes (authorization server logs/policy).

## See Also

- [Scopes Documentation](../scopes.md): Complete scope management guide
- [Example 1: Basic OAuth](01-basic-auth.md): Auto mode example
- [Example 4: Testing Legacy Servers](04-legacy-servers.md)
- [Configuration Reference](../configuration.md): All scope-related flags

