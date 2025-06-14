package agent

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// CallTool executes a tool with the given arguments, with reconnection logic.
func (c *Client) CallTool(ctx context.Context, name string, args map[string]interface{}) (*mcp.CallToolResult, error) {
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: args,
		},
	}

	c.logger.Request("tools/call", req.Params)

	const maxRetries = 1
	var result *mcp.CallToolResult
	var err error

	for i := 0; i <= maxRetries; i++ {
		result, err = c.client.CallTool(ctx, req)
		if err == nil {
			c.logger.Response("tools/call", result)
			return result, nil // Success
		}

		if shouldReconnect(err) {
			if i < maxRetries {
				c.logger.Error("Connection lost during tool call. Attempting to reconnect...")
				if reconnErr := c.Reconnect(ctx); reconnErr != nil {
					err = fmt.Errorf("failed to reconnect: %w", reconnErr)
					break // Don't retry if reconnect fails
				}
				c.logger.Info("Reconnected successfully. Retrying tool call...")
				continue
			}
		}
		// Break on non-reconnectable error or after last retry
		break
	}

	c.logger.Error("CallTool failed: %v", err)
	return nil, err
}

// GetResource retrieves a resource by URI, with reconnection logic.
func (c *Client) GetResource(ctx context.Context, uri string) (*mcp.ReadResourceResult, error) {
	req := mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{
			URI: uri,
		},
	}
	c.logger.Request("resources/read", req.Params)

	const maxRetries = 1
	var result *mcp.ReadResourceResult
	var err error

	for i := 0; i <= maxRetries; i++ {
		result, err = c.client.ReadResource(ctx, req)
		if err == nil {
			c.logger.Response("resources/read", result)
			return result, nil // Success
		}

		if shouldReconnect(err) {
			if i < maxRetries {
				c.logger.Error("Connection lost during resource fetch. Attempting to reconnect...")
				if reconnErr := c.Reconnect(ctx); reconnErr != nil {
					err = fmt.Errorf("failed to reconnect: %w", reconnErr)
					break // Don't retry if reconnect fails
				}
				c.logger.Info("Reconnected successfully. Retrying resource fetch...")
				continue
			}
		}
		// Break on non-reconnectable error or after last retry
		break
	}

	c.logger.Error("ReadResource failed: %v", err)
	return nil, err
}

// GetPrompt retrieves a prompt with arguments, with reconnection logic.
func (c *Client) GetPrompt(ctx context.Context, name string, args map[string]string) (*mcp.GetPromptResult, error) {
	req := mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{
			Name:      name,
			Arguments: args,
		},
	}
	c.logger.Request("prompts/get", req.Params)

	const maxRetries = 1
	var result *mcp.GetPromptResult
	var err error

	for i := 0; i <= maxRetries; i++ {
		result, err = c.client.GetPrompt(ctx, req)
		if err == nil {
			c.logger.Response("prompts/get", result)
			return result, nil // Success
		}

		if shouldReconnect(err) {
			if i < maxRetries {
				c.logger.Error("Connection lost during prompt fetch. Attempting to reconnect...")
				if reconnErr := c.Reconnect(ctx); reconnErr != nil {
					err = fmt.Errorf("failed to reconnect: %w", reconnErr)
					break // Don't retry if reconnect fails
				}
				c.logger.Info("Reconnected successfully. Retrying prompt fetch...")
				continue
			}
		}
		// Break on non-reconnectable error or after last retry
		break
	}

	c.logger.Error("GetPrompt failed: %v", err)
	return nil, err
}
