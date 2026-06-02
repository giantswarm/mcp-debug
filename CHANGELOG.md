# Changelog

## [0.0.93](https://github.com/giantswarm/mcp-debug/compare/v0.0.92...v0.0.93) (2026-06-02)


### Changed

* align files according to platform standards ([#121](https://github.com/giantswarm/mcp-debug/issues/121)) ([61cbf8e](https://github.com/giantswarm/mcp-debug/commit/61cbf8e0ef85acd259c881872d5a30cbdffa2c89))
* **deps:** update actions/checkout action to v6.0.3 ([#122](https://github.com/giantswarm/mcp-debug/issues/122)) ([d796bd6](https://github.com/giantswarm/mcp-debug/commit/d796bd64ef0d92da981348283dd71fd170502fe4))
* **main:** release 0.0.92 ([#120](https://github.com/giantswarm/mcp-debug/issues/120)) ([790c965](https://github.com/giantswarm/mcp-debug/commit/790c965087bf5035e775892adecd582a146732e8))

## [0.0.92](https://github.com/giantswarm/mcp-debug/compare/v0.0.91...v0.0.92) (2026-06-01)


### Changed

* align files according to platform standards ([#119](https://github.com/giantswarm/mcp-debug/issues/119)) ([ddf89e5](https://github.com/giantswarm/mcp-debug/commit/ddf89e586da44e5dcb074fc284610843a230010b))
* align files according to platform standards ([#121](https://github.com/giantswarm/mcp-debug/issues/121)) ([61cbf8e](https://github.com/giantswarm/mcp-debug/commit/61cbf8e0ef85acd259c881872d5a30cbdffa2c89))

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
