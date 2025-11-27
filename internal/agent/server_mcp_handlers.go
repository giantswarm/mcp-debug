package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// handleListTools handles the list_tools tool request
func (m *MCPServer) handleListTools(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	m.client.mu.RLock()
	tools := m.client.toolCache
	m.client.mu.RUnlock()

	// Convert to JSON
	data, err := json.Marshal(tools)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal tools: %v", err)), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

// handleListResources handles the list_resources tool request
func (m *MCPServer) handleListResources(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	m.client.mu.RLock()
	resources := m.client.resourceCache
	m.client.mu.RUnlock()

	// Convert to JSON
	data, err := json.Marshal(resources)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal resources: %v", err)), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

// handleListPrompts handles the list_prompts tool request
func (m *MCPServer) handleListPrompts(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	m.client.mu.RLock()
	prompts := m.client.promptCache
	m.client.mu.RUnlock()

	// Convert to JSON
	data, err := json.Marshal(prompts)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal prompts: %v", err)), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

// handleDescribeTool handles the describe_tool request
func (m *MCPServer) handleDescribeTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get tool name from arguments
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("invalid arguments type"), nil
	}

	name, ok := args["name"].(string)
	if !ok || name == "" {
		return mcp.NewToolResultError("missing or invalid 'name' argument"), nil
	}

	// Find the tool
	m.client.mu.RLock()
	var tool *mcp.Tool
	for _, t := range m.client.toolCache {
		if t.Name == name {
			tool = &t
			break
		}
	}
	m.client.mu.RUnlock()

	if tool == nil {
		return mcp.NewToolResultError(fmt.Sprintf("tool not found: %s", name)), nil
	}

	// Convert to JSON
	data, err := json.Marshal(tool)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal tool: %v", err)), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

// handleDescribeResource handles the describe_resource request
func (m *MCPServer) handleDescribeResource(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get resource URI from arguments
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("invalid arguments type"), nil
	}

	uri, ok := args["uri"].(string)
	if !ok || uri == "" {
		return mcp.NewToolResultError("missing or invalid 'uri' argument"), nil
	}

	// Find the resource
	m.client.mu.RLock()
	var resource *mcp.Resource
	for _, r := range m.client.resourceCache {
		if r.URI == uri {
			resource = &r
			break
		}
	}
	m.client.mu.RUnlock()

	if resource == nil {
		return mcp.NewToolResultError(fmt.Sprintf("resource not found: %s", uri)), nil
	}

	// Convert to JSON
	data, err := json.Marshal(resource)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal resource: %v", err)), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

// handleDescribePrompt handles the describe_prompt request
func (m *MCPServer) handleDescribePrompt(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get prompt name from arguments
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("invalid arguments type"), nil
	}

	name, ok := args["name"].(string)
	if !ok || name == "" {
		return mcp.NewToolResultError("missing or invalid 'name' argument"), nil
	}

	// Find the prompt
	m.client.mu.RLock()
	var prompt *mcp.Prompt
	for _, p := range m.client.promptCache {
		if p.Name == name {
			prompt = &p
			break
		}
	}
	m.client.mu.RUnlock()

	if prompt == nil {
		return mcp.NewToolResultError(fmt.Sprintf("prompt not found: %s", name)), nil
	}

	// Convert to JSON
	data, err := json.Marshal(prompt)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal prompt: %v", err)), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

// handleCallTool handles the call_tool request
func (m *MCPServer) handleCallTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get arguments
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("invalid arguments type"), nil
	}

	toolName, ok := args["name"].(string)
	if !ok || toolName == "" {
		return mcp.NewToolResultError("missing or invalid 'name' argument"), nil
	}

	// Get tool arguments (optional)
	var toolArgs map[string]interface{}
	if argValue, exists := args["arguments"]; exists {
		toolArgs, _ = argValue.(map[string]interface{})
	}

	// Call the tool
	result, err := m.client.CallTool(ctx, toolName, toolArgs)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("tool call failed: %v", err)), nil
	}

	// Convert result to JSON
	data, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

// handleGetResource handles the get_resource request
func (m *MCPServer) handleGetResource(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get resource URI from arguments
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("invalid arguments type"), nil
	}

	uri, ok := args["uri"].(string)
	if !ok || uri == "" {
		return mcp.NewToolResultError("missing or invalid 'uri' argument"), nil
	}

	// Get the resource
	result, err := m.client.GetResource(ctx, uri)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("resource retrieval failed: %v", err)), nil
	}

	// Convert result to JSON
	data, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

// handleGetPrompt handles the get_prompt request
func (m *MCPServer) handleGetPrompt(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get arguments
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("invalid arguments type"), nil
	}

	promptName, ok := args["name"].(string)
	if !ok || promptName == "" {
		return mcp.NewToolResultError("missing or invalid 'name' argument"), nil
	}

	// Get prompt arguments (optional)
	promptArgs := make(map[string]string)
	if argValue, exists := args["arguments"]; exists {
		if argsMap, ok := argValue.(map[string]interface{}); ok {
			for k, v := range argsMap {
				promptArgs[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	// Get the prompt
	result, err := m.client.GetPrompt(ctx, promptName, promptArgs)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("prompt retrieval failed: %v", err)), nil
	}

	// Convert result to JSON
	data, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}
