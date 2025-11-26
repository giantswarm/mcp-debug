# OAuth 2.1 Authentication for MCP

This directory contains comprehensive documentation for OAuth 2.1 authentication in `mcp-debug`, implementing the [MCP Authorization Specification (2025-11-25)](https://spec.modelcontextprotocol.io/specification/2025-11-25/basic/authorization/).

## Overview

`mcp-debug` implements modern OAuth 2.1 authentication with security-first defaults to connect to protected MCP servers. The implementation follows industry best practices and includes advanced features like automatic discovery, scope management, and step-up authorization.

## Quick Start

### Basic OAuth Connection

Connect to a protected MCP server with automatic discovery:

```bash
./mcp-debug --oauth --endpoint https://mcp.example.com/mcp
```

This will:
1. Discover the authorization server automatically (RFC 9728)
2. Determine required scopes from the server
3. Register as a client if needed (Dynamic Client Registration)
4. Open your browser for authorization
5. Exchange the authorization code for tokens
6. Connect to the MCP server with authentication

### With Pre-Registered Client

If you already have OAuth client credentials:

```bash
./mcp-debug --oauth \
  --oauth-client-id your-client-id \
  --oauth-client-secret your-client-secret \
  --endpoint https://mcp.example.com/mcp
```

### With Authenticated DCR

Some servers require a registration token:

```bash
./mcp-debug --oauth \
  --oauth-registration-token your-registration-token \
  --endpoint https://mcp.example.com/mcp
```

## Security by Default

`mcp-debug` follows a security-first approach with features enabled by default:

- **PKCE (RFC 7636)**: Proof Key for Code Exchange protects against authorization code interception
- **Resource Indicators (RFC 8707)**: Tokens are bound to specific MCP servers
- **Automatic Discovery (RFC 9728, RFC 8414)**: Authorization servers and scopes discovered securely
- **Scope Minimization**: Requests only the scopes the server requires (principle of least privilege)
- **Step-Up Authorization**: Automatically requests additional permissions when needed
- **PKCE Validation**: Refuses connections to servers without PKCE support
- **In-Memory Tokens**: Tokens never persisted to disk

## Key Features

### Discovery

- **[RFC 9728: Protected Resource Metadata](discovery.md#protected-resource-metadata)**: Automatic authorization server discovery
- **[RFC 8414: AS Metadata](discovery.md#authorization-server-metadata)**: Multi-endpoint probing for OAuth 2.0 and OIDC
- **[WWW-Authenticate Header](discovery.md#www-authenticate-header)**: Challenge-based discovery

### Client Registration

- **[Pre-Registration](client-registration.md#pre-registration)**: Use existing OAuth client credentials
- **[Client ID Metadata Documents](client-registration.md#client-id-metadata-documents)**: HTTPS URL as client identifier (draft-ietf-oauth-client-id-metadata-document)
- **[Dynamic Client Registration](client-registration.md#dynamic-client-registration)**: Automatic client registration (RFC 7591)
- **[Registration Priority](client-registration.md#registration-priority)**: Intelligent fallback chain

### Security Features

- **[PKCE](security.md#pkce)**: Code challenge/verifier for enhanced security (required)
- **[Resource Indicators](resource-indicators.md)**: Audience-bound tokens prevent misuse
- **[Scope Management](scopes.md)**: Auto and manual modes for scope selection
- **[Step-Up Authorization](scopes.md#step-up-authorization)**: Runtime permission escalation
- **[Token Security](security.md#token-security)**: In-memory storage, automatic refresh

### Testing & Compatibility

- **[Compatibility Flags](testing.md#compatibility-flags)**: Test with older/non-compliant servers
- **[Testing Workflow](testing.md#testing-workflow)**: Systematic testing approach
- **[Security Warnings](testing.md#security-implications)**: Understand the risks of compatibility mode

## Documentation Structure

- **[Discovery](discovery.md)**: Authorization server and scope discovery mechanisms
- **[Resource Indicators](resource-indicators.md)**: RFC 8707 implementation and configuration
- **[Scopes](scopes.md)**: Scope selection strategies and step-up authorization
- **[Client Registration](client-registration.md)**: Registration methods and priority order
- **[Security](security.md)**: Security features, best practices, and threat model
- **[Testing](testing.md)**: Compatibility modes and testing workflows
- **[Configuration](configuration.md)**: Complete reference of all OAuth options
- **[Troubleshooting](troubleshooting.md)**: Common issues and solutions
- **[Examples](examples/)**: Practical tutorials and use cases

## Common Use Cases

### Connect Without Credentials

Try Dynamic Client Registration first:

```bash
./mcp-debug --oauth --endpoint https://mcp.example.com/mcp
```

### Specify Scopes Manually

Override automatic scope discovery:

```bash
./mcp-debug --oauth \
  --oauth-scope-mode manual \
  --oauth-scopes "mcp:read,mcp:write" \
  --endpoint https://mcp.example.com/mcp
```

### Test with Legacy Servers

Disable modern features for compatibility:

```bash
./mcp-debug --oauth \
  --oauth-skip-resource-param \
  --oauth-skip-pkce-validation \
  --endpoint https://legacy-server.com/mcp
```

**Warning**: Compatibility flags reduce security. Use only for testing.

### Use Specific Authorization Server

When multiple servers are available:

```bash
./mcp-debug --oauth \
  --oauth-preferred-auth-server https://auth-backup.example.com \
  --endpoint https://mcp.example.com/mcp
```

## Interactive REPL with OAuth

All OAuth features work in REPL mode:

```bash
./mcp-debug --repl --oauth --endpoint https://mcp.example.com/mcp
```

Authorization happens once at startup, then all REPL commands use the authenticated session.

## MCP Server Mode with OAuth

`mcp-debug` can act as an MCP server and proxy OAuth-protected MCP servers:

```bash
./mcp-debug --mcp-server --oauth \
  --oauth-client-id your-client-id \
  --endpoint https://mcp.example.com/mcp
```

This allows AI assistants to access protected MCP servers through `mcp-debug`.

## Standards Compliance

`mcp-debug` implements the following standards:

- **[MCP Authorization Specification (2025-11-25)](https://spec.modelcontextprotocol.io/specification/2025-11-25/basic/authorization/)**
- **[RFC 7636: PKCE](https://www.rfc-editor.org/rfc/rfc7636.html)**
- **[RFC 7591: Dynamic Client Registration](https://www.rfc-editor.org/rfc/rfc7591.html)**
- **[RFC 8414: Authorization Server Metadata](https://www.rfc-editor.org/rfc/rfc8414.html)**
- **[RFC 8707: Resource Indicators](https://www.rfc-editor.org/rfc/rfc8707.html)**
- **[RFC 9728: Protected Resource Metadata](https://datatracker.ietf.org/doc/html/rfc9728)**
- **[Client ID Metadata Documents (draft-00)](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-client-id-metadata-document-00)**
- **[OpenID Connect Discovery 1.0](https://openid.net/specs/openid-connect-discovery-1_0.html)**

## Getting Help

- **[Troubleshooting Guide](troubleshooting.md)**: Common issues and solutions
- **[Examples](examples/)**: Step-by-step tutorials
- **[GitHub Issues](https://github.com/giantswarm/mcp-debug/issues)**: Report bugs or request features

## Security Notice

OAuth authentication involves sensitive credentials and tokens. Please review the [Security documentation](security.md) to understand:

- Token storage and lifecycle
- Security implications of compatibility flags
- Best practices for credential management
- Threat model and security boundaries

---

**Next Steps**:
- Read [Discovery](discovery.md) to understand how `mcp-debug` finds authorization servers
- Review [Configuration](configuration.md) for all available options
- Try the [Basic Authentication Example](examples/01-basic-auth.md)

