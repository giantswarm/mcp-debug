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

**Problem:** Server supports PKCE but doesn't advertise it

**Error:**

```
ERROR: Authorization server does not advertise PKCE support
ERROR: code_challenge_methods_supported: []
```

**Solution:**

```bash
./mcp-debug --oauth \
  --oauth-skip-pkce-validation \
  --endpoint https://legacy-server.com/mcp
```

**Security impact:** **CRITICAL** - May connect without PKCE

**Verify:** Check if server actually supports PKCE by examining logs:

```
[INFO] Authorization request includes code_challenge
[INFO] ✓ Access token obtained (PKCE succeeded)
```

If PKCE works, report to server operator to update metadata.

### Scenario 4: All Features Missing

**Problem:** Very old OAuth server

**Solution:**

```bash
./mcp-debug --oauth \
  --oauth-skip-resource-param \
  --oauth-skip-resource-metadata \
  --oauth-skip-pkce-validation \
  --oauth-client-id "pre-registered-client" \
  --oauth-client-secret "$CLIENT_SECRET" \
  --endpoint https://very-legacy-server.com/mcp
```

**Security impact:** **CRITICAL** - Minimal OAuth security

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

**Test 3: Skip PKCE validation**

```bash
./mcp-debug --oauth --verbose \
  --oauth-skip-resource-param \
  --oauth-skip-resource-metadata \
  --oauth-skip-pkce-validation \
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
--oauth-skip-pkce-validation
\`\`\`

### Recommendations

1. Update server to advertise PKCE support
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

If the server fails even with all compatibility flags:

```bash
./mcp-debug --oauth --verbose \
  --oauth-skip-resource-param \
  --oauth-skip-resource-metadata \
  --oauth-skip-pkce-validation \
  --oauth-skip-auth-server-discovery \
  --oauth-client-id "$CLIENT_ID" \
  --oauth-client-secret "$CLIENT_SECRET" \
  --endpoint https://server.com/mcp
```

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

