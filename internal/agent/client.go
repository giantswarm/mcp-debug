package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// Client represents an MCP agent client
type Client struct {
	endpoint           string
	transport          string
	logger             *Logger
	client             client.MCPClient
	toolCache          []mcp.Tool
	resourceCache      []mcp.Resource
	promptCache        []mcp.Prompt
	mu                 sync.RWMutex
	notificationChan   chan mcp.JSONRPCNotification
	serverCapabilities *mcp.ServerCapabilities
	oauthConfig        *OAuthConfig
}

// NewClient creates a new agent client
func NewClient(endpoint, transport string, logger *Logger, oauthConfig *OAuthConfig) *Client {
	return &Client{
		endpoint:         endpoint,
		transport:        transport,
		logger:           logger,
		toolCache:        []mcp.Tool{},
		resourceCache:    []mcp.Resource{},
		promptCache:      []mcp.Prompt{},
		notificationChan: make(chan mcp.JSONRPCNotification, 10),
		oauthConfig:      oauthConfig,
	}
}

// Run executes the agent workflow
func (c *Client) Run(ctx context.Context) error {
	return c.connectAndInitialize(ctx)
}

func (c *Client) Reconnect(ctx context.Context) error {
	c.logger.Info("Attempting to reconnect to MCP server...")
	if c.client != nil {
		c.client.Close()
	}
	return c.connectAndInitialize(ctx)
}

func (c *Client) connectAndInitialize(ctx context.Context) error {
	c.logger.Info("Connecting to MCP server at %s using %s transport...", c.endpoint, c.transport)

	var mcpClient *client.Client
	var err error

	// Handle OAuth authentication if enabled
	if c.oauthConfig != nil && c.oauthConfig.Enabled {
		if err := c.oauthConfig.Validate(); err != nil {
			return fmt.Errorf("invalid OAuth configuration: %w", err)
		}

		c.logger.Info("OAuth authentication enabled")

		// Create token store for mcp-go
		tokenStore := client.NewMemoryTokenStore()

		// Create mcp-go OAuth config
		mcpOAuthConfig := client.OAuthConfig{
			ClientID:     c.oauthConfig.ClientID,
			ClientSecret: c.oauthConfig.ClientSecret,
			RedirectURI:  c.oauthConfig.RedirectURL,
			Scopes:       c.oauthConfig.Scopes,
			TokenStore:   tokenStore,
			PKCEEnabled:  c.oauthConfig.UsePKCE,
		}

		// Create OAuth client using mcp-go's native support
		mcpClient, err = client.NewOAuthStreamableHttpClient(c.endpoint, mcpOAuthConfig)
		if err != nil {
			return fmt.Errorf("failed to create OAuth client: %w", err)
		}
		c.logger.Success("OAuth client created")
	} else {
		// Create regular non-OAuth client
		mcpClient, err = client.NewStreamableHttpClient(c.endpoint)
		if err != nil {
			return fmt.Errorf("failed to create streamable HTTP client: %w", err)
		}
	}

	c.client = mcpClient

	// Start the transport
	if err := mcpClient.Start(ctx); err != nil {
		// Check if OAuth authorization is required
		if client.IsOAuthAuthorizationRequiredError(err) {
			c.logger.Info("OAuth authorization required, starting authorization flow...")
			if err := c.handleOAuthAuthorization(ctx, err); err != nil {
				return fmt.Errorf("OAuth authorization failed: %w", err)
			}
			// Retry starting the client
			if err := mcpClient.Start(ctx); err != nil {
				return fmt.Errorf("failed to start client after authorization: %w", err)
			}
		} else {
			return fmt.Errorf("failed to start client: %w", err)
		}
	}

	// Set up notification handler
	mcpClient.OnNotification(func(notification mcp.JSONRPCNotification) {
		select {
		case c.notificationChan <- notification:
		case <-ctx.Done():
		}
	})

	// Initialize the session
	if err := c.initialize(ctx); err != nil {
		// Check if OAuth authorization is required
		if client.IsOAuthAuthorizationRequiredError(err) {
			c.logger.Info("OAuth authorization required during initialization, starting authorization flow...")
			if err := c.handleOAuthAuthorization(ctx, err); err != nil {
				return fmt.Errorf("OAuth authorization failed: %w", err)
			}
			// Retry initialization
			if err := c.initialize(ctx); err != nil {
				return fmt.Errorf("initialization failed after authorization: %w", err)
			}
		} else {
			return fmt.Errorf("initialization failed: %w", err)
		}
	}

	// List capabilities conditionally based on what the server supports
	if c.ServerSupportsTools() {
		if err := c.listTools(ctx, true); err != nil {
			return fmt.Errorf("initial tool listing failed: %w", err)
		}
	} else {
		c.logger.Info("Server does not support tools capability")
	}

	if c.ServerSupportsResources() {
		if err := c.listResources(ctx, true); err != nil {
			return fmt.Errorf("initial resource listing failed: %w", err)
		}
	} else {
		c.logger.Info("Server does not support resources capability")
	}

	if c.ServerSupportsPrompts() {
		if err := c.listPrompts(ctx, true); err != nil {
			return fmt.Errorf("initial prompt listing failed: %w", err)
		}
	} else {
		c.logger.Info("Server does not support prompts capability")
	}

	return nil
}

func (c *Client) Listen(ctx context.Context) error {
	// Wait for notifications
	c.logger.Info("Waiting for notifications (press Ctrl+C to exit)...")

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Shutting down...")
			return nil

		case notification := <-c.notificationChan:
			if err := c.handleNotification(ctx, notification); err != nil {
				c.logger.Error("Failed to handle notification: %v", err)
			}
		}
	}
}

// initialize performs the MCP protocol handshake
func (c *Client) initialize(ctx context.Context) error {
	req := mcp.InitializeRequest{
		Params: struct {
			ProtocolVersion string                 `json:"protocolVersion"`
			Capabilities    mcp.ClientCapabilities `json:"capabilities"`
			ClientInfo      mcp.Implementation     `json:"clientInfo"`
		}{
			ProtocolVersion: "2024-11-05",
			ClientInfo: mcp.Implementation{
				Name:    "mcp-debug-agent",
				Version: "1.0.0",
			},
			Capabilities: mcp.ClientCapabilities{},
		},
	}

	// Log request
	c.logger.Request("initialize", req.Params)

	// Send request
	result, err := c.client.Initialize(ctx, req)
	if err != nil {
		c.logger.Error("Initialize failed: %v", err)
		return err
	}

	// Log response
	c.logger.Response("initialize", result)

	// Store server capabilities for conditional feature usage
	c.mu.Lock()
	c.serverCapabilities = &result.Capabilities
	c.mu.Unlock()

	return nil
}

// listTools lists all available tools
func (c *Client) listTools(ctx context.Context, initial bool) error {
	req := mcp.ListToolsRequest{}

	// Log request
	c.logger.Request("tools/list", req.Params)

	// Send request
	result, err := c.client.ListTools(ctx, req)
	if err != nil {
		c.logger.Error("ListTools failed: %v", err)
		return err
	}

	// Log response
	c.logger.Response("tools/list", result)

	// Compare with cache if not initial
	if !initial {
		c.mu.RLock()
		oldTools := c.toolCache
		c.mu.RUnlock()

		c.mu.Lock()
		c.toolCache = result.Tools
		c.mu.Unlock()

		// Show differences
		c.showToolDiff(oldTools, result.Tools)
	} else {
		c.mu.Lock()
		c.toolCache = result.Tools
		c.mu.Unlock()
	}

	return nil
}

// listResources lists all available resources
func (c *Client) listResources(ctx context.Context, initial bool) error {
	req := mcp.ListResourcesRequest{}

	// Log request
	c.logger.Request("resources/list", req.Params)

	// Send request
	result, err := c.client.ListResources(ctx, req)
	if err != nil {
		c.logger.Error("ListResources failed: %v", err)
		return err
	}

	// Log response
	c.logger.Response("resources/list", result)

	// Compare with cache if not initial
	if !initial {
		c.mu.RLock()
		oldResources := c.resourceCache
		c.mu.RUnlock()

		c.mu.Lock()
		c.resourceCache = result.Resources
		c.mu.Unlock()

		// Show differences
		c.showResourceDiff(oldResources, result.Resources)
	} else {
		c.mu.Lock()
		c.resourceCache = result.Resources
		c.mu.Unlock()
	}

	return nil
}

// listPrompts lists all available prompts
func (c *Client) listPrompts(ctx context.Context, initial bool) error {
	req := mcp.ListPromptsRequest{}

	// Log request
	c.logger.Request("prompts/list", req.Params)

	// Send request
	result, err := c.client.ListPrompts(ctx, req)
	if err != nil {
		c.logger.Error("ListPrompts failed: %v", err)
		return err
	}

	// Log response
	c.logger.Response("prompts/list", result)

	// Compare with cache if not initial
	if !initial {
		c.mu.RLock()
		oldPrompts := c.promptCache
		c.mu.RUnlock()

		c.mu.Lock()
		c.promptCache = result.Prompts
		c.mu.Unlock()

		// Show differences
		c.showPromptDiff(oldPrompts, result.Prompts)
	} else {
		c.mu.Lock()
		c.promptCache = result.Prompts
		c.mu.Unlock()
	}

	return nil
}

// handleNotification processes incoming notifications
func (c *Client) handleNotification(ctx context.Context, notification mcp.JSONRPCNotification) error {
	// Log the notification
	c.logger.Notification(notification.Method, notification.Params)

	// Handle specific notifications only if the server supports the corresponding capability
	switch notification.Method {
	case "notifications/tools/list_changed":
		if c.ServerSupportsTools() {
			return c.listTools(ctx, false)
		}

	case "notifications/resources/list_changed":
		if c.ServerSupportsResources() {
			return c.listResources(ctx, false)
		}

	case "notifications/prompts/list_changed":
		if c.ServerSupportsPrompts() {
			return c.listPrompts(ctx, false)
		}

	default:
		// Unknown notification type
	}

	return nil
}

// showToolDiff displays the differences between old and new tool lists
func (c *Client) showToolDiff(oldTools, newTools []mcp.Tool) {
	// Create maps for easier comparison
	oldMap := make(map[string]mcp.Tool)
	for _, tool := range oldTools {
		oldMap[tool.Name] = tool
	}

	newMap := make(map[string]mcp.Tool)
	for _, tool := range newTools {
		newMap[tool.Name] = tool
	}

	// Check for changes
	var added []string
	var removed []string
	var unchanged []string

	// Find added and unchanged
	for name := range newMap {
		if _, exists := oldMap[name]; exists {
			unchanged = append(unchanged, name)
		} else {
			added = append(added, name)
		}
	}

	// Find removed
	for name := range oldMap {
		if _, exists := newMap[name]; !exists {
			removed = append(removed, name)
		}
	}

	// Display changes
	if len(added) > 0 || len(removed) > 0 {
		c.logger.Info("Tool changes detected:")
		for _, name := range unchanged {
			c.logger.Success("  ✓ Unchanged: %s", name)
		}
		for _, name := range added {
			c.logger.Success("  + Added: %s", name)
		}
		for _, name := range removed {
			c.logger.Error("  - Removed: %s", name)
		}
	} else {
		c.logger.Info("No tool changes detected")
	}
}

// showResourceDiff displays the differences between old and new resource lists
func (c *Client) showResourceDiff(oldResources, newResources []mcp.Resource) {
	// Create maps for easier comparison
	oldMap := make(map[string]mcp.Resource)
	for _, resource := range oldResources {
		oldMap[resource.URI] = resource
	}

	newMap := make(map[string]mcp.Resource)
	for _, resource := range newResources {
		newMap[resource.URI] = resource
	}

	// Check for changes
	var added []string
	var removed []string
	var unchanged []string

	// Find added and unchanged
	for uri := range newMap {
		if _, exists := oldMap[uri]; exists {
			unchanged = append(unchanged, uri)
		} else {
			added = append(added, uri)
		}
	}

	// Find removed
	for uri := range oldMap {
		if _, exists := newMap[uri]; !exists {
			removed = append(removed, uri)
		}
	}

	// Display changes
	if len(added) > 0 || len(removed) > 0 {
		c.logger.Info("Resource changes detected:")
		for _, uri := range unchanged {
			c.logger.Success("  ✓ Unchanged: %s", uri)
		}
		for _, uri := range added {
			c.logger.Success("  + Added: %s", uri)
		}
		for _, uri := range removed {
			c.logger.Error("  - Removed: %s", uri)
		}
	} else {
		c.logger.Info("No resource changes detected")
	}
}

// showPromptDiff displays the differences between old and new prompt lists
func (c *Client) showPromptDiff(oldPrompts, newPrompts []mcp.Prompt) {
	// Create maps for easier comparison
	oldMap := make(map[string]mcp.Prompt)
	for _, prompt := range oldPrompts {
		oldMap[prompt.Name] = prompt
	}

	newMap := make(map[string]mcp.Prompt)
	for _, prompt := range newPrompts {
		newMap[prompt.Name] = prompt
	}

	// Check for changes
	var added []string
	var removed []string
	var unchanged []string

	// Find added and unchanged
	for name := range newMap {
		if _, exists := oldMap[name]; exists {
			unchanged = append(unchanged, name)
		} else {
			added = append(added, name)
		}
	}

	// Find removed
	for name := range oldMap {
		if _, exists := newMap[name]; !exists {
			removed = append(removed, name)
		}
	}

	// Display changes
	if len(added) > 0 || len(removed) > 0 {
		c.logger.Info("Prompt changes detected:")
		for _, name := range unchanged {
			c.logger.Success("  ✓ Unchanged: %s", name)
		}
		for _, name := range added {
			c.logger.Success("  + Added: %s", name)
		}
		for _, name := range removed {
			c.logger.Error("  - Removed: %s", name)
		}
	} else {
		c.logger.Info("No prompt changes detected")
	}
}

// OnNotification is a helper type for type-safe notification handling
type NotificationHandler func(notification mcp.JSONRPCNotification)

// PrettyJSON pretty-prints JSON for logging
func PrettyJSON(v interface{}) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%+v", v)
	}
	return string(b)
}

// prettyJSON is a wrapper for backward compatibility
func prettyJSON(v interface{}) string {
	return PrettyJSON(v)
}

// Helper methods to check server capabilities
func (c *Client) ServerSupportsTools() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.serverCapabilities != nil && c.serverCapabilities.Tools != nil
}

func (c *Client) ServerSupportsResources() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.serverCapabilities != nil && c.serverCapabilities.Resources != nil
}

func (c *Client) ServerSupportsPrompts() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.serverCapabilities != nil && c.serverCapabilities.Prompts != nil
}

// handleOAuthAuthorization handles the OAuth authorization flow using mcp-go's built-in support
func (c *Client) handleOAuthAuthorization(ctx context.Context, authErr error) error {
	// Get the OAuth handler from mcp-go's error
	oauthHandler := client.GetOAuthHandler(authErr)
	if oauthHandler == nil {
		return fmt.Errorf("no OAuth handler available in error")
	}

	// Use our wrapper that provides better UX while leveraging mcp-go's OAuth handler
	return c.handleMCPOAuthFlow(ctx, oauthHandler)
}

func shouldReconnect(err error) bool {
	if err == nil {
		return false
	}

	// Check for context cancellation, which can happen on disconnect
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	errMsg := strings.ToLower(err.Error())
	if strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "connection reset by peer") ||
		strings.Contains(errMsg, "transport is closing") ||
		strings.Contains(errMsg, "broken pipe") ||
		strings.Contains(errMsg, "unexpected eof") {
		return true
	}

	return false
}
