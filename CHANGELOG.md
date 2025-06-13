# Changelog

## [Unreleased]

### Fixed
- **MCP Server Capability Compatibility**: Fixed issue where `mcp-debug` would crash when connecting to MCP servers that don't support all capabilities (tools, resources, prompts). The client now:
  - Checks server capabilities during initialization
  - Only attempts to list capabilities that the server actually supports
  - Provides graceful error messages when unsupported capabilities are accessed
  - Updates tab completion to only show supported capabilities
  - Makes REPL commands capability-aware

  This change allows `mcp-debug` to work with any MCP server regardless of which combination of capabilities it supports.

### Technical Details
- Added `ServerSupportsTools()`, `ServerSupportsResources()`, and `ServerSupportsPrompts()` methods to the `Client` struct
- Modified initialization sequence in both `client.go` and `repl.go` to be conditional
- Updated REPL command handlers to check capabilities before execution
- Enhanced tab completion to only suggest commands for supported capabilities
- Added test coverage for capability checking functionality 