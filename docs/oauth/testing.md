# Testing and Compatibility

`mcp-debug` provides compatibility flags to test with older or non-compliant authorization servers. These flags disable security features and should **only be used for testing**.

## Table of Contents

- [Overview](#overview)
- [Compatibility Flags](#compatibility-flags)
- [Security Implications](#security-implications)
- [Testing Workflows](#testing-workflows)
- [Common Testing Scenarios](#common-testing-scenarios)
- [Migration Strategy](#migration-strategy)
- [Best Practices](#best-practices)

## Overview

The MCP authorization specification (2025-11-25) requires modern OAuth 2.1 features. Some existing authorization servers may not fully support these requirements. Compatibility flags allow testing with such servers while clearly indicating reduced security.

**Production vs Testing:**

```bash
# Production: All security features enabled
./mcp-debug --oauth --endpoint https://mcp.example.com/mcp

# Testing: Compatibility flags for legacy server
./mcp-debug --oauth \
  --oauth-skip-resource-param \
  --oauth-skip-pkce-validation \
  --endpoint https://legacy-server.com/mcp
```

## Compatibility Flags

### Complete Reference

| Flag | Feature Disabled | Security Impact | Use Case |
|------|------------------|-----------------|----------|
| `--oauth-skip-resource-param` | RFC 8707 Resource Indicators | **HIGH** - No token audience binding | Test pre-RFC 8707 servers |
| `--oauth-skip-resource-metadata` | RFC 9728 Protected Resource Metadata | **MEDIUM** - Manual AS config needed | Test pre-RFC 9728 servers |
| `--oauth-skip-pkce-validation` | PKCE support validation | **CRITICAL** - May allow insecure connections | Test servers with PKCE but no advertisement |
| `--oauth-skip-auth-server-discovery` | RFC 8414 AS Metadata discovery | **LOW** - Falls back to internal mechanisms | Test non-standard endpoints |
| `--oauth-disable-step-up` | Step-up authorization | **LOW** - Manual scope management needed | Test scope handling |

### --oauth-skip-resource-param

**What it does:** Disables RFC 8707 resource parameter inclusion

**Security impact:** **HIGH**

- Tokens not bound to specific resources
- Stolen tokens can be used against any resource
- Weakens token audience validation
- Violates MCP specification requirements

**When to use:**

- Authorization server predates RFC 8707 (2019)
- Server returns errors with `resource` parameter
- Testing token usage without audience binding

**Example:**

```bash
./mcp-debug --oauth \
  --oauth-skip-resource-param \
  --endpoint https://legacy-oauth.example.com/mcp
```

**Warning logged:**

```
[WARNING] RFC 8707 resource parameter disabled
[WARNING] Tokens will not be audience-bound - reduced security
[WARNING] Use only for testing with legacy servers
```

### --oauth-skip-resource-metadata

**What it does:** Disables RFC 9728 Protected Resource Metadata discovery

**Security impact:** **MEDIUM**

- No automatic authorization server discovery
- No automatic scope discovery
- May need manual AS configuration
- Falls back to alternative discovery methods

**When to use:**

- MCP server doesn't implement RFC 9728
- Server doesn't provide WWW-Authenticate header
- Testing manual authorization server configuration

**Example:**

```bash
./mcp-debug --oauth \
  --oauth-skip-resource-metadata \
  --oauth-preferred-auth-server https://auth.example.com \
  --endpoint https://legacy-mcp.example.com/mcp
```

**Note:** You may need to manually specify authorization server URL.

### --oauth-skip-pkce-validation

**What it does:** Skips checking for PKCE support in AS metadata

**Security impact:** **CRITICAL**

- May connect to servers without PKCE
- Authorization code interception attacks possible
- Man-in-the-middle vulnerabilities
- **Violates MCP specification**

**When to use:**

- Server supports PKCE but doesn't advertise it
- Testing legacy servers during migration
- **NEVER** in production

**Example:**

```bash
./mcp-debug --oauth \
  --oauth-skip-pkce-validation \
  --endpoint https://legacy-auth.example.com/mcp
```

**Warnings logged:**

```
[WARNING] PKCE validation disabled - DANGEROUS
[WARNING] Connecting to servers without PKCE support is insecure
[WARNING] Authorization code interception attacks are possible
[WARNING] Use only for testing legacy servers
```

### --oauth-skip-auth-server-discovery

**What it does:** Disables RFC 8414 AS Metadata discovery

**Security impact:** **LOW**

- Falls back to mcp-go internal discovery
- May not detect all AS capabilities
- Endpoint configuration may be incomplete

**When to use:**

- Non-standard AS metadata locations
- Testing custom authorization server setups
- Debugging endpoint configuration issues

**Example:**

```bash
./mcp-debug --oauth \
  --oauth-skip-auth-server-discovery \
  --endpoint https://custom-server.com/mcp
```

### --oauth-disable-step-up

**What it does:** Disables automatic step-up authorization

**Security impact:** **LOW**

- Insufficient_scope errors returned to user
- Manual scope management required
- No automatic permission escalation

**When to use:**

- Testing scope validation
- Debugging insufficient_scope errors
- Verifying required scopes upfront
- Preventing automatic authorization prompts

**Example:**

```bash
./mcp-debug --oauth \
  --oauth-disable-step-up \
  --endpoint https://mcp.example.com/mcp
```

**Behavior:**

```
$ ./mcp-debug --repl --oauth --oauth-disable-step-up --endpoint https://mcp.example.com/mcp
[INFO] Step-up authorization disabled

>> exec delete_resource '{"id": "123"}'
ERROR: 403 Forbidden - Insufficient scope
ERROR: Required scopes: [mcp:write mcp:delete]
ERROR: Please re-run with additional scopes
```

## Security Implications

### Risk Matrix

| Flags | Combined Risk | Recommendation |
|-------|---------------|----------------|
| None | ✅ **Secure** | Production use |
| `--oauth-skip-resource-metadata` | ⚠️ **Low** | Acceptable for legacy testing |
| `--oauth-skip-auth-server-discovery` | ⚠️ **Low** | Acceptable for custom setups |
| `--oauth-skip-resource-param` | ❌ **High** | Test only, document reason |
| `--oauth-skip-pkce-validation` | 🔴 **Critical** | **Never** in production |
| Multiple flags | 🔴 **Critical** | Extreme caution required |

### Cumulative Risk

Using multiple compatibility flags compounds security risks:

```bash
# DANGEROUS: Multiple security features disabled
./mcp-debug --oauth \
  --oauth-skip-resource-param \
  --oauth-skip-pkce-validation \
  --oauth-skip-resource-metadata \
  --endpoint https://very-legacy-server.com/mcp
```

**Disabled protections:**

- ❌ Token audience binding
- ❌ PKCE enforcement
- ❌ Automatic discovery

**Result:** Minimal OAuth security, suitable only for isolated testing.

## Testing Workflows

### Workflow 1: Testing Legacy MCP Server

**Goal:** Connect to MCP server with pre-RFC 9728 authorization

**Steps:**

1. Try connecting normally:
   ```bash
   ./mcp-debug --oauth --endpoint https://legacy-mcp.example.com/mcp
   ```

2. If resource metadata discovery fails:
   ```bash
   ./mcp-debug --oauth \
     --oauth-skip-resource-metadata \
     --endpoint https://legacy-mcp.example.com/mcp
   ```

3. If automatic AS discovery fails:
   ```bash
   ./mcp-debug --oauth \
     --oauth-skip-resource-metadata \
     --oauth-skip-auth-server-discovery \
     --endpoint https://legacy-mcp.example.com/mcp
   ```

4. Document which flags were needed and why.

### Workflow 2: Testing Authorization Server Compliance

**Goal:** Verify which MCP specification features an authorization server supports

**Test 1: PKCE Support**

```bash
# Should work if server supports PKCE
./mcp-debug --oauth --endpoint https://test-server.com/mcp

# If fails, check AS metadata manually
curl https://auth.example.com/.well-known/oauth-authorization-server | jq .code_challenge_methods_supported
```

**Test 2: Resource Indicators (RFC 8707)**

```bash
# Should work if server supports resource parameter
./mcp-debug --oauth --endpoint https://test-server.com/mcp

# If fails with "unknown parameter: resource"
./mcp-debug --oauth \
  --oauth-skip-resource-param \
  --endpoint https://test-server.com/mcp
```

**Test 3: Protected Resource Metadata (RFC 9728)**

```bash
# Check if MCP server provides metadata
curl -i https://test-server.com/mcp
# Look for WWW-Authenticate header with resource_metadata parameter

# Check well-known URIs
curl https://test-server.com/.well-known/oauth-protected-resource
curl https://test-server.com/mcp/.well-known/oauth-protected-resource
```

### Workflow 3: Gradual Migration Testing

**Goal:** Test migration from legacy OAuth to MCP-compliant OAuth

**Phase 1: Baseline (all features disabled)**

```bash
./mcp-debug --oauth \
  --oauth-skip-resource-param \
  --oauth-skip-resource-metadata \
  --oauth-skip-pkce-validation \
  --endpoint https://server.example.com/mcp
```

**Phase 2: Enable PKCE**

```bash
# Remove --oauth-skip-pkce-validation
./mcp-debug --oauth \
  --oauth-skip-resource-param \
  --oauth-skip-resource-metadata \
  --endpoint https://server.example.com/mcp
```

**Phase 3: Enable Resource Indicators**

```bash
# Remove --oauth-skip-resource-param
./mcp-debug --oauth \
  --oauth-skip-resource-metadata \
  --endpoint https://server.example.com/mcp
```

**Phase 4: Enable Protected Resource Metadata**

```bash
# Remove all compatibility flags
./mcp-debug --oauth --endpoint https://server.example.com/mcp
```

**Phase 5: Full Compliance**

Verify all features work without compatibility flags.

## Common Testing Scenarios

### Scenario 1: "Unknown parameter: resource" Error

**Problem:**

```
ERROR: Token request failed: invalid_request
ERROR: Unknown parameter: resource
```

**Diagnosis:** Server doesn't support RFC 8707 Resource Indicators

**Solution:**

```bash
./mcp-debug --oauth \
  --oauth-skip-resource-param \
  --endpoint https://server.example.com/mcp
```

**Document:** Server needs RFC 8707 support for MCP compliance.

### Scenario 2: AS Metadata Discovery Fails

**Problem:**

```
ERROR: Failed to discover authorization server metadata
ERROR: All well-known URIs returned 404
```

**Diagnosis:** Server doesn't implement RFC 8414 or uses non-standard locations

**Solution:**

```bash
./mcp-debug --oauth \
  --oauth-skip-auth-server-discovery \
  --endpoint https://server.example.com/mcp
```

**Alternative:** Manually check AS metadata location and report to server operator.

### Scenario 3: PKCE Validation Fails

**Problem:**

```
ERROR: Authorization server does not advertise PKCE support
ERROR: code_challenge_methods_supported: []
```

**Diagnosis:** Server may support PKCE but doesn't advertise it

**Verify:** Check if server actually supports PKCE

```bash
# Try with PKCE anyway (skip validation)
./mcp-debug --oauth \
  --oauth-skip-pkce-validation \
  --verbose \
  --endpoint https://server.example.com/mcp
```

**Check logs:** If PKCE succeeds, server needs to update metadata.

### Scenario 4: Resource Metadata 404

**Problem:**

```
INFO: Fetching Protected Resource Metadata from https://server.com/.well-known/oauth-protected-resource
ERROR: 404 Not Found
```

**Solution:**

```bash
./mcp-debug --oauth \
  --oauth-skip-resource-metadata \
  --oauth-preferred-auth-server https://auth.example.com \
  --endpoint https://server.example.com/mcp
```

## Migration Strategy

### For MCP Server Operators

**Step 1: Assess Current State**

Test with full compliance mode:

```bash
./mcp-debug --oauth --verbose --endpoint https://your-mcp-server.com/mcp
```

Note which features fail.

**Step 2: Implement Missing Features**

Priority order:

1. **PKCE** (Critical)
2. **Protected Resource Metadata** (RFC 9728)
3. **Resource Indicators** (RFC 8707)
4. **AS Metadata Discovery** (RFC 8414)

**Step 3: Gradual Rollout**

1. Enable in development environment
2. Test with `mcp-debug` compatibility flags
3. Gradually remove flags as features are added
4. Deploy to staging
5. Deploy to production

**Step 4: Deprecate Compatibility**

Once compliant:

1. Announce deprecation of legacy OAuth
2. Provide migration timeline
3. Monitor usage of compatibility endpoints
4. Disable legacy support after migration period

### For Authorization Server Operators

**Upgrade Path:**

1. Implement PKCE (RFC 7636) - **Required**
2. Implement AS Metadata (RFC 8414) - **Recommended**
3. Implement Resource Indicators (RFC 8707) - **Recommended**
4. Update metadata to advertise all features
5. Test with `mcp-debug` without compatibility flags
6. Document compliance status

## Best Practices

### For Testing

1. **Document Why**: Document reason for each compatibility flag
2. **Minimize Scope**: Use fewest flags necessary
3. **Temporary Use**: Remove flags once server is fixed
4. **Verbose Logging**: Use `--verbose` to understand behavior
5. **Test in Isolation**: Test compatibility flags one at a time
6. **Report Issues**: Report missing features to server operators

### For Development

1. **Start Strict**: Begin with no compatibility flags
2. **Add Incrementally**: Add flags only when necessary
3. **Understand Impact**: Read security implications before use
4. **Plan Migration**: Create timeline to remove flags
5. **Monitor Security**: Watch for security warnings in logs

### For Production

1. **Avoid Compatibility Flags**: Upgrade servers instead
2. **Prefer Pre-Registration**: Use pre-registered clients
3. **Document Exceptions**: If flags needed, document why
4. **Security Review**: Review all compatibility flag use
5. **Regular Audits**: Audit OAuth configuration regularly

### For Reporting

When reporting issues to server operators:

```markdown
## OAuth Compatibility Issue

**Server:** https://example.com/mcp
**Issue:** Missing PKCE support advertisement
**Tested:** 2025-11-26
**Tool:** mcp-debug v1.2.3

**Current Behavior:**
- Authorization server does not advertise PKCE support
- `code_challenge_methods_supported` missing from AS metadata

**Expected Behavior:**
- Per MCP spec and OAuth 2.1, PKCE with S256 method required
- AS metadata should include: `"code_challenge_methods_supported": ["S256"]`

**Workaround:**
```bash
./mcp-debug --oauth --oauth-skip-pkce-validation --endpoint https://example.com/mcp
```

**Security Impact:**
- PKCE validation bypass weakens security
- Allows authorization code interception attacks

**Request:**
Please update authorization server metadata to advertise PKCE support.
```

## See Also

- [Security](security.md): Understand security implications
- [Discovery](discovery.md): What discovery provides
- [Configuration](configuration.md): Complete flag reference
- [Troubleshooting](troubleshooting.md): Common issues
- [MCP Authorization Spec](https://spec.modelcontextprotocol.io/specification/2025-11-25/basic/authorization/): Compliance requirements

