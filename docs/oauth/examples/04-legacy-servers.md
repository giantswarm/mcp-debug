# Example 4: Testing with Legacy OAuth Servers

This example shows how to connect to older authorization servers that don't fully support the MCP OAuth specification.

## Scenario

You need to connect to an MCP server with an older authorization server that:

- Doesn't support RFC 8707 (Resource Indicators)
- Doesn't advertise PKCE support (but may still support it)
- Doesn't implement RFC 9728 (Protected Resource Metadata)
- Uses non-standard endpoint locations

## WARNING: Security Implications

Compatibility flags **disable security features**. Use only for:

- Testing with legacy systems
- Development/staging environments
- Temporary workarounds while server is upgraded

**NEVER** use in production without understanding the risks.

## Common Legacy Server Scenarios

### Scenario 1: Pre-RFC 8707 Server

**Problem:** Server rejects `resource` parameter

**Error:**

```
ERROR: Token request failed: invalid_request
ERROR: Unknown parameter: resource
```

**Solution:**

```bash
./mcp-debug --oauth \
  --oauth-skip-resource-param \
  --endpoint https://legacy-server.com/mcp
```

**Security impact:** **HIGH** - Tokens not audience-bound

### Scenario 2: No Protected Resource Metadata

**Problem:** Server doesn't implement RFC 9728

**Error:**

```
ERROR: 404 Not Found
ERROR: Protected Resource Metadata not available
```

**Solution:**

```bash
./mcp-debug --oauth \
  --oauth-skip-resource-metadata \
  --oauth-preferred-auth-server https://auth.example.com \
  --endpoint https://legacy-server.com/mcp
```

**Security impact:** **MEDIUM** - Manual AS configuration needed

### Scenario 3: PKCE Not Advertised

**Problem:** Server doesn't advertise PKCE support

**Error:**

```
ERROR: Authorization server does not advertise PKCE support
ERROR: code_challenge_methods_supported: []
ERROR: Per MCP spec, PKCE is required for security
```

**Important:** PKCE is **mandatory** per MCP specification. There is no bypass flag.

**Actions:**

1. **Report to server operator** - PKCE with S256 method is required
2. **Check server documentation** - Verify if server actually supports PKCE
3. **Request metadata update** - Server should advertise `code_challenge_methods_supported: ["S256"]`

**Security Note:** If the authorization server doesn't support PKCE, you cannot connect to MCP servers through it. This is a security requirement, not an optional feature.

### Scenario 4: Multiple Features Missing

**Problem:** Old OAuth server missing multiple modern features

**Solution:**

```bash
./mcp-debug --oauth \
  --oauth-skip-resource-param \
  --oauth-skip-resource-metadata \
  --oauth-client-id "pre-registered-client" \
  --oauth-client-secret "$CLIENT_SECRET" \
  --endpoint https://legacy-server.com/mcp
```

**Security impact:** **HIGH** - Reduced OAuth security

**Note:** PKCE support is still required. If the authorization server doesn't support PKCE, you cannot proceed - see Scenario 3.

## Step-by-Step Testing Process

### Step 1: Try Full Compliance

Always start with no compatibility flags:

```bash
./mcp-debug --oauth --verbose --endpoint https://server.example.com/mcp 2>&1 | tee test.log
```

Note which features fail.

### Step 2: Add Flags Incrementally

Add compatibility flags one at a time:

**Test 1: Skip resource parameter**

```bash
./mcp-debug --oauth --verbose \
  --oauth-skip-resource-param \
  --endpoint https://server.example.com/mcp
```

**Test 2: Skip resource metadata**

```bash
./mcp-debug --oauth --verbose \
  --oauth-skip-resource-param \
  --oauth-skip-resource-metadata \
  --endpoint https://server.example.com/mcp
```

### Step 3: Document Findings

Create a compatibility report:

```markdown
## OAuth Compatibility Test Results

**Server:** https://server.example.com/mcp
**Date:** 2025-11-26
**Tester:** Your Name

### Features Tested

| Feature | Supported | Notes |
|---------|-----------|-------|
| RFC 9728 (Resource Metadata) | ❌ | 404 on well-known URI |
| RFC 8414 (AS Metadata) | ✅ | Works |
| RFC 8707 (Resource Indicators) | ❌ | "Unknown parameter" error |
| PKCE | ✅ | Works but not advertised |
| DCR | ❌ | No registration endpoint |

### Required Compatibility Flags

\`\`\`bash
--oauth-skip-resource-param
--oauth-skip-resource-metadata
\`\`\`

### Recommendations

1. Update server to advertise PKCE support (required by MCP spec)
2. Implement RFC 8707 resource indicators
3. Implement RFC 9728 protected resource metadata
4. Enable RFC 7591 dynamic client registration

### Security Warnings

- Tokens not audience-bound (high risk)
- Manual authorization server configuration required
- PKCE validation bypassed (critical risk)
```

## Testing with Pre-Registered Client

Legacy servers often don't support DCR:

```bash
./mcp-debug --oauth \
  --oauth-client-id "legacy-client-id" \
  --oauth-client-secret "$LEGACY_CLIENT_SECRET" \
  --oauth-skip-resource-param \
  --oauth-skip-resource-metadata \
  --endpoint https://legacy-server.com/mcp
```

## Example: Google OAuth (Pre-MCP Spec)

Google's OAuth predates the MCP specification:

```bash
# Google OAuth with legacy compatibility
./mcp-debug --oauth \
  --oauth-client-id "$GOOGLE_CLIENT_ID" \
  --oauth-client-secret "$GOOGLE_CLIENT_SECRET" \
  --oauth-skip-resource-param \
  --oauth-skip-resource-metadata \
  --endpoint https://mcp-server-using-google.com/mcp
```

**Note:** The MCP server still handles MCP protocol; it just uses Google for authentication.

## Migration Planning

### For MCP Server Operators

**Phase 1: Baseline (Current State)**

Document current compatibility flags needed:

```bash
./mcp-debug --oauth --verbose \
  --oauth-skip-resource-param \
  --oauth-skip-resource-metadata \
  --endpoint https://your-server.com/mcp
```

**Phase 2: Incremental Upgrades**

1. **Week 1:** Implement RFC 9728 (Resource Metadata)
   ```bash
   # Test: Remove --oauth-skip-resource-metadata
   ./mcp-debug --oauth --oauth-skip-resource-param \
     --endpoint https://your-server.com/mcp
   ```

2. **Week 2:** Implement RFC 8707 (Resource Indicators)
   ```bash
   # Test: Remove --oauth-skip-resource-param
   ./mcp-debug --oauth --endpoint https://your-server.com/mcp
   ```

3. **Week 3:** Advertise PKCE support
   ```bash
   # Test: Full compliance
   ./mcp-debug --oauth --endpoint https://your-server.com/mcp
   ```

**Phase 3: Deprecation**

Announce deprecated compatibility:

- "OAuth resource parameter support will be required starting 2026-01-01"
- "Update your authorization server to support RFC 8707"
- Provide migration guide

## Troubleshooting

### "Invalid client" with Legacy Server

**Error:**

```
ERROR: Token request failed: invalid_client
```

**Common causes:**

1. Wrong client credentials
2. Client authentication method mismatch
3. Client not registered for this authorization server

**Solution:** Verify registration:

```bash
# Check registered redirect URIs
# Check client authentication method
# Verify client is active
```

### Multiple Compatibility Flags Still Fail

If the server fails even with all available compatibility flags:

```bash
./mcp-debug --oauth --verbose \
  --oauth-skip-resource-param \
  --oauth-skip-resource-metadata \
  --oauth-preferred-auth-server https://auth.example.com \
  --oauth-client-id "$CLIENT_ID" \
  --oauth-client-secret "$CLIENT_SECRET" \
  --endpoint https://server.com/mcp
```

**Note:** PKCE support is mandatory - if the authorization server doesn't support PKCE with S256 method, you cannot proceed.

The server may have custom OAuth implementation. Contact server operator.

## Best Practices

### DO

✅ Document which flags are needed and why
✅ Test flags one at a time
✅ Report missing features to server operators
✅ Create a migration timeline
✅ Monitor for security warnings in logs

### DON'T

❌ Use compatibility flags in production
❌ Ignore security warnings
❌ Use flags without understanding impact
❌ Assume flags are permanent solutions
❌ Skip reporting issues to server operators

## See Also

- [Testing Documentation](../testing.md): Complete compatibility flags guide
- [Security](../security.md): Understanding security risks
- [Troubleshooting](../troubleshooting.md): Common error messages
- [Configuration](../configuration.md): All flag reference

