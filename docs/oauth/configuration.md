# OAuth Configuration Reference

Complete reference for all OAuth 2.1 configuration options in `mcp-debug`.

## Table of Contents

- [Command-Line Flags](#command-line-flags)
- [Configuration File](#configuration-file)
- [Environment Variables](#environment-variables)
- [OAuthConfig Structure](#oauthconfig-structure)
- [Default Values](#default-values)
- [Validation Rules](#validation-rules)
- [Examples](#examples)

## Command-Line Flags

### Core OAuth Flags

| Flag | Type | Description | Default |
|------|------|-------------|---------|
| `--oauth` | boolean | Enable OAuth 2.1 authentication | `false` |
| `--oauth-client-id` | string | OAuth client identifier | `""` |
| `--oauth-client-secret` | string | OAuth client secret (optional for public clients) | `""` |
| `--oauth-redirect-url` | string | Callback URL for OAuth flow | `http://localhost:8765/callback` |
| `--oauth-timeout` | duration | Maximum time to wait for user authorization | `5m` |

### Scope Management Flags

| Flag | Type | Description | Default |
|------|------|-------------|---------|
| `--oauth-scopes` | string | Comma-separated OAuth scopes to request | `""` |
| `--oauth-scope-mode` | string | Scope selection mode: `auto` or `manual` | `auto` |

### Security Feature Flags

| Flag | Type | Description | Default |
|------|------|-------------|---------|
| `--oauth-pkce` | boolean | Use PKCE for authorization | `true` |
| `--oauth-oidc` | boolean | Enable OpenID Connect features (nonce validation) | `false` |

### Discovery Flags

| Flag | Type | Description | Default |
|------|------|-------------|---------|
| `--oauth-resource-uri` | string | Target resource URI for RFC 8707 (auto-derived if not specified) | `""` (auto) |
| `--oauth-preferred-auth-server` | string | Preferred authorization server URL when multiple are available | `""` |

### Client Registration Flags

| Flag | Type | Description | Default |
|------|------|-------------|---------|
| `--oauth-registration-token` | string | OAuth registration access token for authenticated DCR | `""` |

### Step-Up Authorization Flags

| Flag | Type | Description | Default |
|------|------|-------------|---------|
| `--oauth-disable-step-up` | boolean | Disable automatic step-up authorization | `false` (enabled) |
| `--oauth-step-up-max-retries` | integer | Maximum retry attempts per operation | `2` |
| `--oauth-step-up-prompt` | boolean | Ask user before requesting additional scopes | `false` (automatic) |

### Compatibility/Testing Flags

| Flag | Type | Description | Default | Security Impact |
|------|------|-------------|---------|-----------------|
| `--oauth-skip-resource-param` | boolean | Skip RFC 8707 resource parameter | `false` | **HIGH** |
| `--oauth-skip-resource-metadata` | boolean | Skip RFC 9728 Protected Resource Metadata discovery | `false` | **MEDIUM** |
| `--oauth-skip-pkce-validation` | boolean | Skip PKCE support validation | `false` | **CRITICAL** |
| `--oauth-skip-auth-server-discovery` | boolean | Skip RFC 8414 AS Metadata discovery | `false` | **LOW** |

## Configuration File

While `mcp-debug` primarily uses command-line flags, you can create a configuration file for complex setups.

### Example oauth-config.json

```json
{
  "oauth": {
    "enabled": true,
    "client_id": "your-client-id",
    "client_secret": "your-client-secret",
    "redirect_url": "http://localhost:8765/callback",
    "scopes": ["mcp:read", "mcp:write"],
    "scope_mode": "auto",
    "use_pkce": true,
    "timeout": "5m",
    "enable_step_up": true,
    "step_up_max_retries": 2
  }
}
```

**Usage:**

```bash
./mcp-debug --config oauth-config.json --endpoint https://mcp.example.com/mcp
```

**Security:** Protect configuration files containing secrets:

```bash
chmod 600 oauth-config.json
```

## Environment Variables

Store sensitive credentials in environment variables:

### Supported Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `OAUTH_CLIENT_ID` | OAuth client identifier | `abc123xyz` |
| `OAUTH_CLIENT_SECRET` | OAuth client secret | `secret-abc-123` |
| `OAUTH_REGISTRATION_TOKEN` | Registration access token | `reg-token-xyz` |

### Usage

```bash
export OAUTH_CLIENT_ID="your-client-id"
export OAUTH_CLIENT_SECRET="your-secret"

./mcp-debug --oauth \
  --oauth-client-id="$OAUTH_CLIENT_ID" \
  --oauth-client-secret="$OAUTH_CLIENT_SECRET" \
  --endpoint https://mcp.example.com/mcp
```

## OAuthConfig Structure

The internal `OAuthConfig` structure (from `internal/agent/oauth_config.go`):

```go
type OAuthConfig struct {
    // Core Configuration
    Enabled              bool     // Enable OAuth authentication
    ClientID             string   // OAuth client identifier
    ClientSecret         string   // OAuth client secret (optional)
    RedirectURL          string   // Callback URL (default: http://localhost:8765/callback)
    AuthorizationTimeout duration // Max wait for authorization (default: 5m)
    
    // Scope Management
    Scopes             []string // OAuth scopes to request
    ScopeSelectionMode string   // "auto" (default) or "manual"
    
    // Security Features
    UsePKCE   bool // Use PKCE (default: true)
    UseOIDC   bool // Enable OIDC features (default: false)
    
    // Discovery
    ResourceURI          string // Target resource URI (auto-derived if empty)
    PreferredAuthServer  string // Preferred AS when multiple available
    
    // Client Registration
    RegistrationToken string // Registration access token for DCR
    
    // Step-Up Authorization
    EnableStepUpAuth   bool // Enable step-up (default: true)
    StepUpMaxRetries   int  // Max retries (default: 2)
    StepUpUserPrompt   bool // Ask user before step-up (default: false)
    
    // Compatibility/Testing
    SkipResourceParam          bool // Skip RFC 8707 (default: false)
    SkipResourceMetadata       bool // Skip RFC 9728 (default: false)
    SkipPKCEValidation         bool // Skip PKCE check (default: false)
    SkipAuthServerDiscovery    bool // Skip RFC 8414 (default: false)
}
```

## Default Values

When fields are not specified, these defaults are used:

| Field | Default Value | Reason |
|-------|---------------|--------|
| `Enabled` | `false` | OAuth must be explicitly enabled |
| `ScopeSelectionMode` | `auto` | Follow MCP spec (secure by default) |
| `RedirectURL` | `http://localhost:8765/callback` | Standard localhost callback |
| `UsePKCE` | `true` | Required by MCP spec |
| `AuthorizationTimeout` | `5m` | Reasonable time for user action |
| `UseOIDC` | `false` | Most MCP servers use OAuth, not OIDC |
| `EnableStepUpAuth` | `true` | Better UX, secure by default |
| `StepUpMaxRetries` | `2` | Prevent infinite loops |
| `StepUpUserPrompt` | `false` | Automatic for better UX |
| `SkipResourceParam` | `false` | Security feature enabled by default |
| `SkipResourceMetadata` | `false` | Discovery enabled by default |
| `SkipPKCEValidation` | `false` | Security check enabled by default |
| `SkipAuthServerDiscovery` | `false` | Discovery enabled by default |

## Validation Rules

The configuration is validated before use:

### Scope Selection Mode

```
Valid: "auto", "manual"
Invalid: anything else
Error: "invalid scope selection mode: xyz (must be 'auto' or 'manual')"
```

### Redirect URL

**Required:** Yes

**Format:** Valid URL

**HTTP restrictions:**

- HTTP only allowed for: `localhost`, `127.0.0.1`, `::1`, `0:0:0:0:0:0:0:1`
- HTTPS not supported (callback server is localhost-only)
- Other schemes rejected

**Examples:**

```bash
# Valid
--oauth-redirect-url "http://localhost:8765/callback"
--oauth-redirect-url "http://127.0.0.1:9000/callback"
--oauth-redirect-url "http://[::1]:8765/callback"

# Invalid
--oauth-redirect-url "http://example.com/callback"  # Non-localhost HTTP
--oauth-redirect-url "https://localhost:8765/callback"  # HTTPS not supported
--oauth-redirect-url "custom://callback"  # Invalid scheme
```

### Authorization Timeout

**Required:** Yes (after defaults applied)

**Format:** Go duration string (e.g., `5m`, `30s`, `1h30m`)

**Minimum:** Typically > 0

**Examples:**

```bash
--oauth-timeout 30s   # 30 seconds (very short)
--oauth-timeout 5m    # 5 minutes (default)
--oauth-timeout 1h    # 1 hour (very long)
```

## Examples

### Minimal OAuth (Auto Discovery)

```bash
./mcp-debug --oauth --endpoint https://mcp.example.com/mcp
```

**Results in:**

- Enabled: `true`
- Client ID: (determined via DCR)
- PKCE: `true`
- Scope mode: `auto`
- All discovery enabled

### Pre-Registered Client

```bash
./mcp-debug --oauth \
  --oauth-client-id "abc123" \
  --oauth-client-secret "secret456" \
  --endpoint https://mcp.example.com/mcp
```

**Results in:**

- Client ID: `abc123`
- Client Secret: `secret456`
- PKCE: `true`
- All other defaults

### Manual Scope Selection

```bash
./mcp-debug --oauth \
  --oauth-scope-mode manual \
  --oauth-scopes "mcp:read,mcp:write,user:profile" \
  --endpoint https://mcp.example.com/mcp
```

**Results in:**

- Scope mode: `manual`
- Scopes: `[mcp:read, mcp:write, user:profile]`
- No automatic scope discovery

### Custom Redirect Port

```bash
./mcp-debug --oauth \
  --oauth-redirect-url "http://localhost:9000/callback" \
  --endpoint https://mcp.example.com/mcp
```

**Results in:**

- Callback server on port 9000 instead of 8765
- Must register this URL with OAuth provider

### Extended Authorization Timeout

```bash
./mcp-debug --oauth \
  --oauth-timeout 10m \
  --endpoint https://mcp.example.com/mcp
```

**Results in:**

- Wait up to 10 minutes for user authorization

### Authenticated DCR

```bash
./mcp-debug --oauth \
  --oauth-registration-token "reg-token-xyz" \
  --endpoint https://mcp.example.com/mcp
```

**Results in:**

- DCR with registration token
- Token sent as Bearer token in registration request

### Disable Step-Up for Testing

```bash
./mcp-debug --oauth \
  --oauth-disable-step-up \
  --endpoint https://mcp.example.com/mcp
```

**Results in:**

- Insufficient_scope errors returned directly
- No automatic re-authorization

### Legacy Server (Multiple Compatibility Flags)

```bash
./mcp-debug --oauth \
  --oauth-skip-resource-param \
  --oauth-skip-resource-metadata \
  --oauth-skip-pkce-validation \
  --endpoint https://legacy-server.com/mcp
```

**Results in:**

- No resource parameter
- No RFC 9728 discovery
- No PKCE validation
- **Significantly reduced security**

### OIDC Mode

```bash
./mcp-debug --oauth \
  --oauth-oidc \
  --endpoint https://oidc-server.com/mcp
```

**Results in:**

- OIDC features enabled
- Nonce parameter generation and validation
- ID token awareness

### Complete Configuration Example

```bash
./mcp-debug --oauth \
  --oauth-client-id "prod-client-123" \
  --oauth-client-secret "$OAUTH_CLIENT_SECRET" \
  --oauth-scope-mode auto \
  --oauth-redirect-url "http://localhost:8765/callback" \
  --oauth-timeout 5m \
  --oauth-preferred-auth-server "https://auth-primary.example.com" \
  --oauth-resource-uri "https://mcp.example.com/api/v1" \
  --endpoint https://mcp.example.com:8443/api/v1/mcp
```

**Results in:**

- Pre-registered client
- Auto scope selection
- Custom resource URI
- Preferred auth server specified
- All security features enabled

## See Also

- [Security](security.md): Security implications of configuration choices
- [Testing](testing.md): Compatibility flags and their impact
- [Scopes](scopes.md): Scope selection modes
- [Discovery](discovery.md): What auto-discovery configures
- [Troubleshooting](troubleshooting.md): Configuration-related errors
- [Examples](examples/): Complete configuration examples

