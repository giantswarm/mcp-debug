# OAuth Troubleshooting

Common issues, error messages, and solutions for OAuth authentication in `mcp-debug`.

## Table of Contents

- [Quick Diagnostics](#quick-diagnostics)
- [Connection Issues](#connection-issues)
- [Discovery Issues](#discovery-issues)
- [Client Registration Issues](#client-registration-issues)
- [Authorization Issues](#authorization-issues)
- [Token Issues](#token-issues)
- [Scope Issues](#scope-issues)
- [Security Errors](#security-errors)
- [Debug Logging](#debug-logging)

## Quick Diagnostics

### Enable Verbose Logging

Always start troubleshooting with verbose mode:

```bash
./mcp-debug --oauth --verbose --endpoint https://mcp.example.com/mcp
```

This shows:
- Discovery attempts and results
- Registration process details
- Authorization flow steps
- Token exchange details
- Error context

### Check Authorization Server Metadata

Manually verify AS metadata:

```bash
# OAuth 2.0
curl https://auth.example.com/.well-known/oauth-authorization-server | jq .

# OIDC
curl https://auth.example.com/.well-known/openid-configuration | jq .
```

### Check Protected Resource Metadata

```bash
# Try path-specific
curl https://mcp.example.com/mcp/.well-known/oauth-protected-resource | jq .

# Try root
curl https://mcp.example.com/.well-known/oauth-protected-resource | jq .
```

## Connection Issues

### "Connection refused"

**Error:**

```
ERROR: Failed to connect to MCP server
ERROR: dial tcp 127.0.0.1:8090: connect: connection refused
```

**Cause:** MCP server not running or wrong endpoint

**Solutions:**

1. Verify server is running
2. Check endpoint URL:
   ```bash
   curl https://mcp.example.com/mcp
   ```
3. Verify port number
4. Check firewall rules

### "Certificate verification failed"

**Error:**

```
ERROR: x509: certificate signed by unknown authority
```

**Cause:** Self-signed certificate or untrusted CA

**Solutions:**

1. Use a valid certificate (production)
2. For testing, add CA to system trust store
3. **Not recommended:** Skip certificate validation (testing only)

### "No route to host"

**Error:**

```
ERROR: no route to host
```

**Cause:** Network connectivity issue

**Solutions:**

1. Check DNS resolution: `nslookup mcp.example.com`
2. Check network connectivity: `ping mcp.example.com`
3. Check VPN/proxy settings
4. Verify firewall rules

## Discovery Issues

### Protected Resource Metadata 404

**Error:**

```
INFO: Fetching Protected Resource Metadata from https://mcp.example.com/.well-known/oauth-protected-resource
ERROR: 404 Not Found
```

**Cause:** Server doesn't implement RFC 9728

**Solution 1:** Skip resource metadata discovery

```bash
./mcp-debug --oauth \
  --oauth-skip-resource-metadata \
  --endpoint https://mcp.example.com/mcp
```

**Solution 2:** Report to server operator

The MCP specification requires RFC 9728 support.

### AS Metadata Discovery Fails

**Error:**

```
ERROR: Failed to discover authorization server metadata
ERROR: All well-known URIs returned 404
```

**Cause:** Server doesn't implement RFC 8414 or uses non-standard location

**Solution 1:** Skip AS metadata discovery

```bash
./mcp-debug --oauth \
  --oauth-skip-auth-server-discovery \
  --endpoint https://mcp.example.com/mcp
```

**Solution 2:** Check metadata manually

```bash
# Try different locations
curl https://auth.example.com/.well-known/oauth-authorization-server
curl https://auth.example.com/.well-known/openid-configuration
curl https://auth.example.com/tenant/.well-known/openid-configuration
```

### Multiple Authorization Servers

**Situation:**

```
INFO: Discovered authorization servers: [https://auth1.example.com, https://auth2.example.com]
INFO: Using first server: https://auth1.example.com
```

**To use different server:**

```bash
./mcp-debug --oauth \
  --oauth-preferred-auth-server https://auth2.example.com \
  --endpoint https://mcp.example.com/mcp
```

## Client Registration Issues

### DCR: "Registration endpoint not found"

**Error:**

```
INFO: No registration endpoint found in AS metadata
INFO: Dynamic Client Registration not available
```

**Cause:** Server doesn't support RFC 7591 DCR

**Solutions:**

1. Use pre-registered client:
   ```bash
   ./mcp-debug --oauth \
     --oauth-client-id "your-client-id" \
     --endpoint https://mcp.example.com/mcp
   ```

2. Register client manually with server operator

### DCR: "Registration access token required"

**Error:**

```
ERROR: Dynamic client registration failed: invalid_token
ERROR: Registration access token required
```

**Cause:** Server requires authenticated DCR

**Solution:**

```bash
./mcp-debug --oauth \
  --oauth-registration-token "your-registration-token" \
  --endpoint https://mcp.example.com/mcp
```

Contact server administrator to obtain registration token.

### DCR: Security errors

**Error:**

```
ERROR: security: registration token can only be sent over HTTPS
```

**Cause:** Attempting to use registration token with HTTP endpoint

**Solution:** Use HTTPS:

```bash
./mcp-debug --oauth \
  --oauth-registration-token "token" \
  --endpoint https://mcp.example.com/mcp  # HTTPS, not HTTP
```

## Authorization Issues

### Browser doesn't open

**Error:**

```
INFO: Opening browser for authorization...
ERROR: Failed to open browser
```

**Cause:** No browser available or `$BROWSER` not set

**Solution 1:** Copy URL manually

Look for log message with authorization URL:

```
INFO: Please visit: https://auth.example.com/oauth/authorize?...
```

**Solution 2:** Set browser:

```bash
export BROWSER=firefox
./mcp-debug --oauth --endpoint https://mcp.example.com/mcp
```

### "Invalid redirect URI"

**Error:**

```
ERROR: OAuth error: invalid_request
ERROR: Invalid redirect_uri parameter
```

**Cause:** Redirect URI not registered with OAuth provider

**Solutions:**

1. Register `http://localhost:8765/callback` with provider
2. Use different port if registered:
   ```bash
   ./mcp-debug --oauth \
     --oauth-redirect-url "http://localhost:9000/callback" \
     --endpoint https://mcp.example.com/mcp
   ```

### "Timeout waiting for authorization"

**Error:**

```
ERROR: Timeout waiting for authorization code
```

**Cause:** User didn't complete authorization within timeout

**Solutions:**

1. Increase timeout:
   ```bash
   ./mcp-debug --oauth \
     --oauth-timeout 10m \
     --endpoint https://mcp.example.com/mcp
   ```

2. Complete authorization faster
3. Check if browser is blocking pop-ups

### "State parameter mismatch"

**Error:**

```
ERROR: OAuth callback error: state mismatch
ERROR: Possible CSRF attack
```

**Cause:** State parameter doesn't match (security error)

**Solutions:**

1. Retry authorization (generate new state)
2. Check for malicious redirect
3. Verify authorization server is legitimate

## Token Issues

### "Invalid client"

**Error:**

```
ERROR: Token request failed: invalid_client
ERROR: Client authentication failed
```

**Causes & Solutions:**

**Cause 1:** Wrong client secret

```bash
# Verify secret
./mcp-debug --oauth \
  --oauth-client-id "correct-id" \
  --oauth-client-secret "$CORRECT_SECRET" \
  --endpoint https://mcp.example.com/mcp
```

**Cause 2:** Client secret required but not provided

```bash
./mcp-debug --oauth \
  --oauth-client-id "your-id" \
  --oauth-client-secret "your-secret" \
  --endpoint https://mcp.example.com/mcp
```

**Cause 3:** Wrong client ID

Verify with server administrator.

### "Invalid grant"

**Error:**

```
ERROR: Token request failed: invalid_grant
ERROR: Authorization code is invalid or expired
```

**Causes:**

- Authorization code already used
- Authorization code expired
- Code used with wrong client ID
- PKCE verifier mismatch

**Solution:** Retry authorization flow from beginning

### "Unknown parameter: resource"

**Error:**

```
ERROR: Token request failed: invalid_request
ERROR: Unknown parameter: resource
```

**Cause:** Server doesn't support RFC 8707

**Solution:**

```bash
./mcp-debug --oauth \
  --oauth-skip-resource-param \
  --endpoint https://mcp.example.com/mcp
```

Report to server operator - RFC 8707 required for MCP compliance.

## Scope Issues

### "Insufficient scope"

**Error:**

```
ERROR: 403 Forbidden - Insufficient scope
ERROR: Required scopes: [mcp:write mcp:delete]
```

**Cause:** Token doesn't have required scopes

**Solutions:**

**Solution 1:** Enable step-up authorization (default):

```bash
./mcp-debug --oauth --endpoint https://mcp.example.com/mcp
```

**Solution 2:** Request scopes upfront:

```bash
./mcp-debug --oauth \
  --oauth-scope-mode manual \
  --oauth-scopes "mcp:read,mcp:write,mcp:delete" \
  --endpoint https://mcp.example.com/mcp
```

### "Step-up max retries exceeded"

**Error:**

```
ERROR: Max step-up authorization retries (2) exceeded
ERROR: Possible causes:
  - Server repeatedly requesting same scopes
  - Authorization server not granting requested scopes
```

**Diagnosis:** Check logs for repeated scope requests

**Solutions:**

1. Increase retries:
   ```bash
   ./mcp-debug --oauth \
     --oauth-step-up-max-retries 5 \
     --endpoint https://mcp.example.com/mcp
   ```

2. Request all scopes upfront (manual mode)
3. Report issue to server operator

### "Scope validation failed"

**Error:**

```
ERROR: scope at index 3 contains invalid control character
ERROR: Excessive number of scopes requested (25 > 20)
```

**Cause:** Invalid or suspicious scope values

**Solution:** Review and fix scope values:

```bash
# Remove invalid scopes
./mcp-debug --oauth \
  --oauth-scope-mode manual \
  --oauth-scopes "valid:scope1,valid:scope2" \
  --endpoint https://mcp.example.com/mcp
```

## Security Errors

### PKCE Validation Failed

**Error:**

```
ERROR: Authorization server does not advertise PKCE support
ERROR: code_challenge_methods_supported: []
ERROR: Per MCP spec, PKCE is required for security
```

**Diagnosis:** Check if server actually supports PKCE

```bash
curl https://auth.example.com/.well-known/oauth-authorization-server | jq .code_challenge_methods_supported
```

**Solution 1:** Server needs to advertise PKCE support

Report to server operator.

**Solution 2:** Bypass validation (testing only):

```bash
./mcp-debug --oauth \
  --oauth-skip-pkce-validation \
  --endpoint https://mcp.example.com/mcp
```

**Warning:** Only for testing. PKCE is a critical security feature.

### "HTTPS redirect URIs not supported"

**Error:**

```
ERROR: HTTPS redirect URIs are not supported
ERROR: callback server only runs on localhost with HTTP
```

**Cause:** Attempted to use HTTPS redirect URL

**Solution:** Use HTTP localhost:

```bash
./mcp-debug --oauth \
  --oauth-redirect-url "http://localhost:8765/callback" \
  --endpoint https://mcp.example.com/mcp
```

### "HTTP redirect URIs only allowed for localhost"

**Error:**

```
ERROR: HTTP redirect URIs are only allowed for localhost/127.0.0.1/[::1]
```

**Cause:** Attempted to use HTTP redirect URL with non-localhost host

**Solution:** Use localhost:

```bash
./mcp-debug --oauth \
  --oauth-redirect-url "http://localhost:8765/callback" \
  --endpoint https://mcp.example.com/mcp
```

## Debug Logging

### Enable All Logging

```bash
./mcp-debug --oauth --verbose --json-rpc \
  --endpoint https://mcp.example.com/mcp
```

Logs include:

- Discovery attempts
- HTTP requests/responses (without tokens)
- Authorization flow steps
- Token exchange (tokens redacted)
- MCP protocol messages

### Log File Output

Redirect logs to file:

```bash
./mcp-debug --oauth --verbose \
  --endpoint https://mcp.example.com/mcp 2>&1 | tee oauth-debug.log
```

### Interpreting Logs

**Successful flow:**

```
[INFO] OAuth enabled
[INFO] Protected Resource Metadata discovery enabled
[INFO] Fetching metadata from ...
[INFO] ✓ Metadata retrieved
[INFO] Discovered authorization servers: [...]
[INFO] Attempting AS Metadata discovery...
[INFO] ✓ AS Metadata retrieved
[INFO] ✓ PKCE validation passed
[INFO] Client registered successfully
[INFO] Opening browser for authorization...
[INFO] ✓ Authorization code received
[INFO] Exchanging code for access token...
[INFO] ✓ Access token obtained
[INFO] ✓ Session initialized
```

**Failed flow - look for ERROR/WARNING:**

```
[INFO] OAuth enabled
[WARNING] Protected Resource Metadata not found - using fallback
[ERROR] AS Metadata discovery failed
[ERROR] Token request failed: invalid_client
```

## Getting Help

### Reporting Issues

When reporting OAuth issues, include:

1. **Command used:**
   ```bash
   ./mcp-debug --oauth --verbose --endpoint https://...
   ```

2. **Error messages** (full logs with `--verbose`)

3. **Authorization server metadata:**
   ```bash
   curl https://auth.example.com/.well-known/oauth-authorization-server | jq .
   ```

4. **Protected resource metadata:**
   ```bash
   curl https://mcp.example.com/.well-known/oauth-protected-resource | jq .
   ```

5. **mcp-debug version:**
   ```bash
   ./mcp-debug --version
   ```

6. **Sanitized logs** (remove tokens/secrets)

### GitHub Issues

Report bugs at: https://github.com/giantswarm/mcp-debug/issues

Include label: `oauth`

## See Also

- [Configuration](configuration.md): Complete flag reference
- [Security](security.md): Security features and implications
- [Discovery](discovery.md): Discovery mechanisms
- [Testing](testing.md): Compatibility flags
- [Examples](examples/): Working examples

