package agent

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

// CallTool executes a tool with the given arguments
func (c *Client) CallTool(ctx context.Context, name string, args map[string]interface{}) (*mcp.CallToolResult, error) {
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: args,
		},
	}

	// Log request
	c.logger.Request("tools/call", req.Params)

	// Send request
	result, err := c.client.CallTool(ctx, req)
	if err != nil {
		c.logger.Error("CallTool failed: %v", err)
		return nil, err
	}

	// Log response
	c.logger.Response("tools/call", result)

	return result, nil
}

// GetResource retrieves a resource by URI
func (c *Client) GetResource(ctx context.Context, uri string) (*mcp.ReadResourceResult, error) {
	req := mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{
			URI: uri,
		},
	}

	// Log request
	c.logger.Request("resources/read", req.Params)

	// Send request
	result, err := c.client.ReadResource(ctx, req)
	if err != nil {
		c.logger.Error("ReadResource failed: %v", err)
		return nil, err
	}

	// Log response
	c.logger.Response("resources/read", result)

	return result, nil
}

// GetPrompt retrieves a prompt with arguments
func (c *Client) GetPrompt(ctx context.Context, name string, args map[string]string) (*mcp.GetPromptResult, error) {
	req := mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{
			Name:      name,
			Arguments: args,
		},
	}

	// Log request
	c.logger.Request("prompts/get", req.Params)

	// Send request
	result, err := c.client.GetPrompt(ctx, req)
	if err != nil {
		c.logger.Error("GetPrompt failed: %v", err)
		return nil, err
	}

	// Log response
	c.logger.Response("prompts/get", result)

	return result, nil
} 