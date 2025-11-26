# Resource Indicators (RFC 8707)

`mcp-debug` implements [RFC 8707: Resource Indicators for OAuth 2.0](https://www.rfc-editor.org/rfc/rfc8707.html) to enhance token security through audience binding. This ensures access tokens are explicitly bound to the specific MCP server you're connecting to.

## Table of Contents

- [Overview](#overview)
- [Why Resource Indicators Matter](#why-resource-indicators-matter)
- [How It Works](#how-it-works)
- [Resource URI Canonicalization](#resource-uri-canonicalization)
- [Configuration](#configuration)
- [Examples](#examples)
- [Security Considerations](#security-considerations)
- [Testing and Compatibility](#testing-and-compatibility)

## Overview

Resource indicators bind OAuth tokens to specific resources (MCP servers). When `mcp-debug` requests an access token, it includes a `resource` parameter identifying the target MCP server.

```bash
# Automatic resource parameter inclusion
./mcp-debug --oauth --endpoint https://mcp.example.com/mcp
```

The `resource` parameter is automatically derived from your endpoint URL and included in:

- Authorization requests (when requesting user consent)
- Token exchange requests (when obtaining access tokens)
- Refresh token requests (when renewing tokens)

## Why Resource Indicators Matter

### Security Without Resource Indicators

Without resource indicators, access tokens are often not bound to specific services:

```
User → Authorization Server → Access Token (valid for "everything")
```

**Risks:**

- Token stolen from one service can be used against others
- No audience restriction on tokens
- Difficult to detect token misuse
- Over-privileged tokens violate least privilege

### Security With Resource Indicators

With RFC 8707, tokens are explicitly bound to resources:

```
User → Authorization Server → Access Token (valid only for "https://mcp.example.com/mcp")
```

**Benefits:**

- Tokens only work with the intended MCP server
- Authorization server can enforce resource-specific policies
- MCP server can validate token audience (aud claim)
- Token theft has limited impact (useless for other resources)
- Enables fine-grained access control

### Real-World Attack Scenario

**Without Resource Indicators:**

1. Attacker compromises token from `https://mcp-dev.example.com`
2. Uses stolen token against `https://mcp-prod.example.com`
3. Gains unauthorized production access

**With Resource Indicators:**

1. Attacker compromises token for `https://mcp-dev.example.com`
2. Tries to use it against `https://mcp-prod.example.com`
3. Production server rejects token (wrong audience)
4. Attack fails

## How It Works

### Automatic Resource URI Derivation

When you specify an endpoint, `mcp-debug` automatically derives the resource URI:

```bash
./mcp-debug --oauth --endpoint https://mcp.example.com:8443/api/v1/mcp
```

Derived resource URI: `https://mcp.example.com:8443/api/v1/mcp`

### Authorization Request

The resource parameter is included in the authorization request:

```
https://auth.example.com/oauth/authorize
  ?client_id=your-client-id
  &response_type=code
  &redirect_uri=http://localhost:8765/callback
  &scope=mcp:read+mcp:write
  &resource=https://mcp.example.com/mcp     ← Resource parameter
  &code_challenge=...
  &code_challenge_method=S256
  &state=...
```

### Token Exchange Request

The resource parameter is also included when exchanging the authorization code:

```
POST https://auth.example.com/oauth/token

grant_type=authorization_code
&code=ABC123
&redirect_uri=http://localhost:8765/callback
&resource=https://mcp.example.com/mcp        ← Resource parameter
&code_verifier=...
&client_id=your-client-id
```

### Token Response

The authorization server returns a token bound to the resource:

```json
{
  "access_token": "eyJ...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "mcp:read mcp:write"
}
```

The access token contains an `aud` (audience) claim:

```json
{
  "iss": "https://auth.example.com",
  "sub": "user@example.com",
  "aud": "https://mcp.example.com/mcp",      ← Audience bound to resource
  "scope": "mcp:read mcp:write",
  "exp": 1735689600
}
```

### MCP Server Validation

The MCP server validates the token's audience:

```go
// MCP server checks token audience
if token.Audience != "https://mcp.example.com/mcp" {
    return errors.New("invalid token audience")
}
```

If the token was obtained for a different resource, validation fails.

## Resource URI Canonicalization

RFC 8707 requires resource URIs to be in canonical form for consistent matching.

### Canonicalization Rules

`mcp-debug` applies these transformations:

1. **Lowercase scheme and host**: `HTTPS://Example.COM` → `https://example.com`
2. **Remove default ports**: `https://example.com:443` → `https://example.com`
3. **Keep non-standard ports**: `https://example.com:8443` → `https://example.com:8443`
4. **Preserve path**: `/api/v1/mcp` → `/api/v1/mcp` (exact case)
5. **Remove trailing slash**: `https://example.com/mcp/` → `https://example.com/mcp`
6. **Remove fragment**: `https://example.com/mcp#section` → `https://example.com/mcp`
7. **Keep query parameters**: `https://example.com/mcp?v=1` → `https://example.com/mcp?v=1`

### Canonicalization Examples

| Original Endpoint | Canonical Resource URI |
|-------------------|------------------------|
| `https://MCP.Example.COM/mcp` | `https://mcp.example.com/mcp` |
| `https://example.com:443/mcp` | `https://example.com/mcp` |
| `https://example.com:8443/mcp` | `https://example.com:8443/mcp` |
| `https://example.com/api/v1/mcp/` | `https://example.com/api/v1/mcp` |
| `https://example.com/MCP` | `https://example.com/MCP` (path preserves case) |
| `http://example.com/mcp` | `http://example.com/mcp` |

### Why Canonicalization Matters

Without canonicalization:

```
Request 1: resource=https://example.com:443/mcp
Request 2: resource=https://example.com/mcp

Authorization Server sees: Two different resources!
```

With canonicalization:

```
Both requests: resource=https://example.com/mcp

Authorization Server sees: Same resource
```

## Configuration

### Automatic Resource URI (Recommended)

By default, `mcp-debug` derives the resource URI from your endpoint:

```bash
./mcp-debug --oauth --endpoint https://mcp.example.com/mcp
```

No additional configuration needed.

### Manual Resource URI

Override the automatic derivation if needed:

```bash
./mcp-debug --oauth \
  --oauth-resource-uri "https://mcp.example.com/mcp" \
  --endpoint https://mcp.example.com:8443/api/mcp
```

**When to use:**

- Resource URI differs from endpoint URL
- Testing specific audience binding scenarios
- Working with complex proxy setups

### Disabling Resource Parameter

For testing with older servers that don't support RFC 8707:

```bash
./mcp-debug --oauth \
  --oauth-skip-resource-param \
  --endpoint https://legacy-server.com/mcp
```

**Warning**: This significantly weakens token security. Only use for testing legacy servers.

## Examples

### Basic Usage

Automatic resource parameter inclusion:

```bash
./mcp-debug --oauth \
  --oauth-client-id my-client-id \
  --endpoint https://mcp.example.com/mcp
```

`mcp-debug` includes `resource=https://mcp.example.com/mcp` in all OAuth requests.

### Non-Standard Port

```bash
./mcp-debug --oauth --endpoint https://mcp.example.com:8443/mcp
```

Resource URI: `https://mcp.example.com:8443/mcp` (port preserved)

### Complex Path

```bash
./mcp-debug --oauth --endpoint https://api.example.com/services/mcp/v2
```

Resource URI: `https://api.example.com/services/mcp/v2`

### Override Resource URI

```bash
./mcp-debug --oauth \
  --oauth-resource-uri "https://mcp-cluster.example.com" \
  --endpoint https://mcp-node1.example.com/mcp
```

Uses `https://mcp-cluster.example.com` as the resource URI despite connecting to a different host.

### Debug Resource Derivation

Use verbose logging to see the derived resource URI:

```bash
./mcp-debug --oauth --verbose --endpoint https://mcp.example.com/mcp
```

Output:

```
[INFO] Resource URI derived from endpoint: https://mcp.example.com/mcp
[INFO] Resource parameter will be included in OAuth requests
```

## Security Considerations

### Token Audience Validation

Authorization servers **SHOULD** validate that clients are authorized to request tokens for specific resources:

```
Client requests: resource=https://sensitive-mcp.example.com/mcp
AS checks: Is this client allowed to access that resource?
```

This prevents clients from obtaining tokens for unauthorized resources.

### Resource Granularity

More specific resource URIs provide better security:

**Less Specific (Weaker)**:

```
resource=https://example.com
```

Token valid for entire domain - higher risk if compromised.

**More Specific (Stronger)**:

```
resource=https://mcp.example.com/api/v1/production/mcp
```

Token valid only for specific MCP endpoint - limited impact if compromised.

### HTTPS Required

Resource URIs **SHOULD** use HTTPS in production:

```bash
# Production
./mcp-debug --oauth --endpoint https://mcp.example.com/mcp  ✓

# Development only
./mcp-debug --oauth --endpoint http://localhost:8090/mcp   ⚠ 
```

HTTP endpoints are acceptable for localhost development but not production.

### Token Reuse Prevention

Resource binding prevents cross-resource token reuse:

```
Token A: resource=https://mcp-dev.example.com/mcp
Token B: resource=https://mcp-prod.example.com/mcp

Token A cannot be used for prod (different audience)
Token B cannot be used for dev (different audience)
```

### Multiple Resources

To access multiple MCP servers, obtain separate tokens:

```bash
# Token for server 1
./mcp-debug --oauth --endpoint https://mcp1.example.com/mcp

# Token for server 2
./mcp-debug --oauth --endpoint https://mcp2.example.com/mcp
```

Each token is bound to its respective resource.

## Testing and Compatibility

### Testing with RFC 8707 Support

Most modern authorization servers support RFC 8707:

```bash
./mcp-debug --oauth --endpoint https://mcp.example.com/mcp
```

Check logs for successful resource parameter inclusion:

```
[INFO] Including resource parameter: https://mcp.example.com/mcp
```

### Testing Without RFC 8707 Support

Older authorization servers may not support the `resource` parameter:

```bash
./mcp-debug --oauth \
  --oauth-skip-resource-param \
  --endpoint https://legacy-server.com/mcp
```

**Warning**: Tokens will not be audience-bound. Only use for testing.

### Detecting RFC 8707 Support

Check authorization server metadata:

```bash
curl https://auth.example.com/.well-known/oauth-authorization-server | jq .
```

Look for:

```json
{
  "resource_indicator_supported": true
}
```

If not present, the server may still support it (not all servers advertise it).

### Migration Strategy

When migrating from non-RFC 8707 to RFC 8707:

1. **Phase 1**: Enable on development servers first
2. **Phase 2**: Verify token audience claims are validated
3. **Phase 3**: Enable on staging
4. **Phase 4**: Roll out to production
5. **Phase 5**: Make resource parameter mandatory (remove `--oauth-skip-resource-param` option)

## See Also

- [Discovery](discovery.md): How resource metadata is discovered
- [Security](security.md): Other security features and best practices
- [Testing](testing.md): Using compatibility flags
- [Configuration](configuration.md): All OAuth configuration options
- [RFC 8707](https://www.rfc-editor.org/rfc/rfc8707.html): Resource Indicators specification
- [MCP Authorization Spec](https://spec.modelcontextprotocol.io/specification/2025-11-25/basic/authorization/): MCP resource indicator requirements

