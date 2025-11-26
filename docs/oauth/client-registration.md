# Client Registration

`mcp-debug` supports multiple client registration methods with intelligent fallback to ensure you can connect to OAuth-protected MCP servers even without pre-configured credentials.

## Table of Contents

- [Overview](#overview)
- [Registration Priority](#registration-priority)
- [Pre-Registration](#pre-registration)
- [Client ID Metadata Documents](#client-id-metadata-documents)
- [Dynamic Client Registration](#dynamic-client-registration)
- [Configuration](#configuration)
- [Examples](#examples)
- [Security Considerations](#security-considerations)
- [Troubleshooting](#troubleshooting)

## Overview

Client registration establishes your identity with the authorization server. `mcp-debug` supports three registration methods:

1. **Pre-Registration**: Use existing OAuth client credentials
2. **Client ID Metadata Documents (CIMD)**: Use HTTPS URL as client identifier
3. **Dynamic Client Registration (DCR)**: Automatically register on first connection

The registration happens automatically during the OAuth flow.

## Registration Priority

`mcp-debug` follows the MCP specification's recommended priority order:

```mermaid
graph TD
    A[Start OAuth Flow] --> B{Pre-registered<br/>credentials?}
    B -->|Yes| C[Use Pre-Registered Client ID]
    B -->|No| D{AS supports<br/>CIMD?}
    D -->|Yes| E[Use Client ID Metadata Documents]
    D -->|No| F{AS supports<br/>DCR?}
    F -->|Yes| G[Dynamic Client Registration]
    F -->|No| H[Prompt User for Credentials]
    
    C --> I[Proceed to Authorization]
    E --> I
    G --> I
    H --> I
    
    style C fill:#90EE90
    style E fill:#87CEEB
    style G fill:#FFD700
    style H fill:#FFA07A
```

### Why This Priority?

1. **Pre-Registration** (highest priority)
   - Most explicit - user provided credentials
   - Best for production use
   - Credentials vetted by organization

2. **Client ID Metadata Documents**
   - Standards-based (draft-ietf-oauth-client-id-metadata-document)
   - No credentials needed
   - Self-describing client

3. **Dynamic Client Registration**
   - Automatic, zero-config
   - Standardized (RFC 7591)
   - May require registration token

4. **Manual Entry** (lowest priority)
   - Last resort
   - User must obtain credentials manually

## Pre-Registration

Use OAuth client credentials you already have.

### When to Use

- Production deployments
- Organization has pre-registered OAuth clients
- Testing with specific client configurations
- Need predictable client identifiers

### Configuration

```bash
./mcp-debug --oauth \
  --oauth-client-id "your-client-id" \
  --oauth-client-secret "your-client-secret" \
  --endpoint https://mcp.example.com/mcp
```

### Public vs Confidential Clients

**Confidential Client** (with secret):

```bash
./mcp-debug --oauth \
  --oauth-client-id "confidential-client-id" \
  --oauth-client-secret "client-secret-abc123" \
  --endpoint https://mcp.example.com/mcp
```

**Public Client** (no secret):

```bash
./mcp-debug --oauth \
  --oauth-client-id "public-client-id" \
  --endpoint https://mcp.example.com/mcp
```

Per OAuth 2.1, native/desktop applications like `mcp-debug` should use public clients with PKCE (no client secret).

### Obtaining Pre-Registered Credentials

**For Google OAuth:**

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create or select a project
3. Navigate to "APIs & Services" → "Credentials"
4. Click "Create Credentials" → "OAuth 2.0 Client ID"
5. Select "Desktop app" or "Web application"
6. Add redirect URI: `http://localhost:8765/callback`
7. Copy the client ID and secret

**For Other Providers:**

- Check authorization server documentation
- Contact server administrator
- Use server's developer portal

## Client ID Metadata Documents

[Client ID Metadata Documents](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-client-id-metadata-document-00) allow using an HTTPS URL as your client identifier.

### What are Client ID Metadata Documents?

Instead of registering with the authorization server, you host a metadata document describing your client:

```
Client ID: https://app.example.com/oauth/client-metadata.json
```

The authorization server fetches this document to learn about your client.

### Current Status

`mcp-debug` currently **detects** authorization server support for CIMD but does not yet implement full client-side CIMD functionality.

**Support Detection:**

```bash
./mcp-debug --oauth --verbose --endpoint https://mcp.example.com/mcp
```

If the authorization server supports CIMD:

```
[INFO] Authorization server metadata:
[INFO]   client_id_metadata_document_supported: true
[INFO]   ... (other metadata)
```

**Implementation Status:**

- ✅ Detection of AS support in metadata
- ⏳ Hosting/generating metadata documents (planned)
- ⏳ Using HTTPS URL as client_id (planned)

### How It Will Work (When Implemented)

**Step 1: Generate Metadata Document**

```bash
# Generate metadata JSON (future feature)
./mcp-debug --oauth-generate-client-metadata > client-metadata.json
```

**Step 2: Host the Document**

Host at an HTTPS URL (e.g., `https://yourdomain.com/client-metadata.json`):

```json
{
  "client_id": "https://yourdomain.com/client-metadata.json",
  "client_name": "mcp-debug",
  "client_uri": "https://github.com/giantswarm/mcp-debug",
  "logo_uri": "https://giantswarm.io/logo.png",
  "redirect_uris": [
    "http://localhost:8765/callback"
  ],
  "grant_types": ["authorization_code", "refresh_token"],
  "response_types": ["code"],
  "token_endpoint_auth_method": "none"
}
```

**Step 3: Use URL as Client ID**

```bash
# Use HTTPS URL as client_id (future feature)
./mcp-debug --oauth \
  --oauth-client-id-metadata-url "https://yourdomain.com/client-metadata.json" \
  --endpoint https://mcp.example.com/mcp
```

### Benefits of CIMD

- **No pre-registration**: Self-describing client
- **Distributed trust**: You control client metadata
- **HTTPS-based**: Leverages existing web infrastructure
- **Decentralized**: No central registration authority needed

### Security Requirements

Per the specification:

- Client ID URL **MUST** use HTTPS scheme
- URL **MUST** contain a path component
- Authorization server **MUST** protect against SSRF attacks
- Localhost redirect URIs have security implications

## Dynamic Client Registration

[RFC 7591](https://www.rfc-editor.org/rfc/rfc7591.html) enables automatic client registration at runtime.

### How It Works

When no client credentials are provided:

```bash
./mcp-debug --oauth --endpoint https://mcp.example.com/mcp
```

`mcp-debug`:

1. Discovers the registration endpoint from AS metadata
2. Sends a registration request
3. Receives a client_id (and optional client_secret)
4. Uses these credentials for authorization
5. Continues with normal OAuth flow

### Registration Request

```http
POST /oauth/register HTTP/1.1
Host: auth.example.com
Content-Type: application/json

{
  "client_name": "mcp-debug",
  "client_uri": "https://github.com/giantswarm/mcp-debug",
  "redirect_uris": ["http://localhost:8765/callback"],
  "grant_types": ["authorization_code", "refresh_token"],
  "response_types": ["code"],
  "token_endpoint_auth_method": "none"
}
```

### Registration Response

```http
HTTP/1.1 201 Created
Content-Type: application/json

{
  "client_id": "dcr_generated_abc123",
  "client_id_issued_at": 1735689600,
  "client_name": "mcp-debug",
  "redirect_uris": ["http://localhost:8765/callback"],
  "grant_types": ["authorization_code", "refresh_token"]
}
```

### Example DCR Flow

```bash
$ ./mcp-debug --oauth --endpoint https://mcp.example.com/mcp
[INFO] OAuth enabled - will attempt Dynamic Client Registration
[INFO] OAuth authorization required
[INFO] No client ID configured, attempting dynamic client registration...
[INFO] Registration endpoint: https://auth.example.com/oauth/register
[INFO] Sending registration request...
[INFO] ✓ Client registered successfully with ID: dcr_generated_abc123
[INFO] Opening browser for authorization...
[INFO] ✓ Authorization code received
[INFO] ✓ Access token obtained successfully!
[INFO] ✓ Connected to MCP server
```

### Authenticated DCR

Some authorization servers require a registration access token:

```bash
./mcp-debug --oauth \
  --oauth-registration-token "your-registration-token" \
  --endpoint https://mcp.example.com/mcp
```

The token is included as a Bearer token in the registration request:

```http
POST /oauth/register HTTP/1.1
Host: auth.example.com
Content-Type: application/json
Authorization: Bearer your-registration-token

{
  "client_name": "mcp-debug",
  ...
}
```

### Security of Registration Tokens

Registration tokens are sensitive credentials:

- **HTTPS Only**: Tokens only sent over HTTPS (never HTTP)
- **Endpoint Validation**: Token only included in recognized OAuth registration endpoints
- **No Logging**: Tokens never logged (even in verbose mode)
- **Environment Variables**: Store in environment variables, not code
- **Short-Lived**: Obtain fresh tokens, rotate regularly

```bash
# From environment variable
export OAUTH_REGISTRATION_TOKEN="your-token"
./mcp-debug --oauth \
  --oauth-registration-token "$OAUTH_REGISTRATION_TOKEN" \
  --endpoint https://mcp.example.com/mcp
```

### Disabling DCR

For testing or when DCR is not desired:

```bash
./mcp-debug --oauth \
  --oauth-disable-dcr \
  --endpoint https://mcp.example.com/mcp
```

**Note**: This flag doesn't exist yet but represents the conceptual approach. Currently, skip DCR by providing `--oauth-client-id`.

## Configuration

### Pre-Registration Flags

| Flag | Description | Required |
|------|-------------|----------|
| `--oauth-client-id` | OAuth client identifier | Yes |
| `--oauth-client-secret` | OAuth client secret | No (for public clients) |

### CIMD Flags (Future)

| Flag | Description | Required |
|------|-------------|----------|
| `--oauth-client-id-metadata-url` | HTTPS URL to client metadata | Yes (CIMD mode) |
| `--oauth-generate-client-metadata` | Generate metadata JSON to stdout | No |
| `--oauth-disable-cimd` | Disable CIMD, use DCR instead | No |

### DCR Flags

| Flag | Description | Required |
|------|-------------|----------|
| `--oauth-registration-token` | Registration access token | No (unless AS requires it) |

## Examples

### Pre-Registered Client

```bash
./mcp-debug --oauth \
  --oauth-client-id "abc123xyz" \
  --oauth-client-secret "secret456def" \
  --endpoint https://mcp.example.com/mcp
```

### Public Client (No Secret)

```bash
./mcp-debug --oauth \
  --oauth-client-id "public-client-abc123" \
  --endpoint https://mcp.example.com/mcp
```

### Dynamic Client Registration

```bash
# Automatic - just enable OAuth
./mcp-debug --oauth --endpoint https://mcp.example.com/mcp
```

### DCR with Registration Token

```bash
./mcp-debug --oauth \
  --oauth-registration-token "reg-token-xyz789" \
  --endpoint https://mcp.example.com/mcp
```

### Debug Registration Process

```bash
./mcp-debug --oauth --verbose --endpoint https://mcp.example.com/mcp
```

Logs show:

```
[INFO] Checking for pre-registered client credentials...
[INFO] No client ID configured
[INFO] Checking AS metadata for DCR support...
[INFO] Registration endpoint found: https://auth.example.com/oauth/register
[INFO] Attempting Dynamic Client Registration...
[INFO] Registration request prepared
[INFO] Sending registration request...
[INFO] ✓ Registration successful
[INFO] Client ID: dcr_gen_abc123
```

### Fallback Chain Example

```bash
# Try pre-reg → CIMD → DCR → manual
./mcp-debug --oauth --endpoint https://mcp.example.com/mcp
```

If no pre-registered credentials:

```
[INFO] No client ID configured
[INFO] Checking AS support for Client ID Metadata Documents...
[INFO] AS does not support client_id_metadata_document
[INFO] Checking AS support for Dynamic Client Registration...
[INFO] Registration endpoint: https://auth.example.com/oauth/register
[INFO] Attempting Dynamic Client Registration...
[INFO] ✓ Client registered successfully
```

## Security Considerations

### Pre-Registration Security

- **Credential Storage**: Never commit client secrets to version control
- **Environment Variables**: Use environment variables for secrets
- **Rotation**: Rotate client secrets regularly
- **Access Control**: Limit who can view/use client credentials
- **Scope Restrictions**: Pre-register with minimal scopes

```bash
# Good: Secret in environment variable
export OAUTH_CLIENT_SECRET="secret123"
./mcp-debug --oauth \
  --oauth-client-id "abc123" \
  --oauth-client-secret "$OAUTH_CLIENT_SECRET" \
  --endpoint https://mcp.example.com/mcp

# Bad: Secret on command line (visible in history)
./mcp-debug --oauth \
  --oauth-client-id "abc123" \
  --oauth-client-secret "secret123" \
  --endpoint https://mcp.example.com/mcp
```

### CIMD Security

- **HTTPS Required**: Client metadata URLs must use HTTPS
- **SSRF Protection**: Authorization servers must protect against SSRF
- **Content-Type**: Serve metadata with `application/json`
- **Localhost Risks**: Localhost redirect URIs can be impersonated

### DCR Security

- **Registration Tokens**: Protect like passwords, transmit only over HTTPS
- **Token Validation**: `mcp-debug` validates endpoints before sending tokens
- **Client Validation**: Authorization servers should validate registration requests
- **Rate Limiting**: Servers should rate-limit registration endpoints
- **Audit Logging**: DCR requests are logged for security auditing

### Token Exposure Risks

**Command-Line Arguments** (visible in process list):

```bash
# Avoid: Token visible to other users via ps/htop
./mcp-debug --oauth --oauth-registration-token "secret-token" ...
```

**Environment Variables** (better):

```bash
# Preferred: Token not in process list
export OAUTH_REGISTRATION_TOKEN="secret-token"
./mcp-debug --oauth --oauth-registration-token "$OAUTH_REGISTRATION_TOKEN" ...
```

**Configuration Files** (best for production):

```bash
# Best: Token in protected config file
./mcp-debug --oauth --config oauth-config.json ...
```

## Troubleshooting

### DCR Fails with "Registration Access Token Required"

**Problem:**

```
ERROR: Dynamic client registration failed: invalid_token
ERROR: Registration access token required
```

**Solution**: The authorization server requires authenticated DCR:

```bash
./mcp-debug --oauth \
  --oauth-registration-token "your-token" \
  --endpoint https://mcp.example.com/mcp
```

Contact your authorization server administrator to obtain a registration token.

### DCR Fails with Security Error

**Problem:**

```
ERROR: security: registration token can only be sent over HTTPS
```

**Cause**: Attempting to use registration token with HTTP endpoint

**Solution**: Use HTTPS endpoint:

```bash
# Fix: Use HTTPS
./mcp-debug --oauth \
  --oauth-registration-token "token" \
  --endpoint https://mcp.example.com/mcp  # HTTPS, not HTTP
```

### Registration Endpoint Not Found

**Problem:**

```
INFO: No registration endpoint found in AS metadata
INFO: Dynamic Client Registration not available
```

**Solution**: Provide pre-registered credentials:

```bash
./mcp-debug --oauth \
  --oauth-client-id "your-client-id" \
  --endpoint https://mcp.example.com/mcp
```

### CIMD Not Working

**Current Status**: CIMD detection is implemented, but full CIMD support (hosting/using metadata URLs) is planned for future releases.

**Workaround**: Use DCR or pre-registration:

```bash
# Option 1: DCR
./mcp-debug --oauth --endpoint https://mcp.example.com/mcp

# Option 2: Pre-registration
./mcp-debug --oauth \
  --oauth-client-id "your-client-id" \
  --endpoint https://mcp.example.com/mcp
```

## See Also

- [Discovery](discovery.md): How registration endpoints are discovered
- [Security](security.md): Credential security best practices
- [Configuration](configuration.md): Complete configuration reference
- [Examples](examples/): Registration examples
- [RFC 7591](https://www.rfc-editor.org/rfc/rfc7591.html): Dynamic Client Registration specification
- [Client ID Metadata Documents](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-client-id-metadata-document-00): CIMD specification (draft)
- [MCP Authorization Spec](https://spec.modelcontextprotocol.io/specification/2025-11-25/basic/authorization/): MCP client registration requirements

