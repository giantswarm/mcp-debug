// Package agent provides an MCP client and REPL for debugging MCP servers.
package agent

// MCP method and notification constants.
// These are the standard MCP protocol method names used across the package.
const (
	// methodInitialize is the MCP initialization method
	methodInitialize = "initialize"

	// notificationToolsListChanged is sent when the server's tool list changes
	notificationToolsListChanged = "notifications/tools/list_changed"

	// notificationResourcesListChanged is sent when the server's resource list changes
	notificationResourcesListChanged = "notifications/resources/list_changed"

	// notificationPromptsListChanged is sent when the server's prompt list changes
	notificationPromptsListChanged = "notifications/prompts/list_changed"
)

// URL scheme and host constants for validation.
const (
	schemeHTTPS  = "https"
	schemeHTTP   = "http"
	hostLocal    = "localhost"
	hostLoopback = "127.0.0.1"
)

// PKCE code challenge method constant.
const pkceMethodS256 = "S256"
