package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// findTool finds a tool by name in the cache
func (r *REPL) findTool(toolName string) *mcp.Tool {
	r.client.mu.RLock()
	defer r.client.mu.RUnlock()

	for _, t := range r.client.toolCache {
		if t.Name == toolName {
			return &t
		}
	}
	return nil
}

// parseToolArgs parses JSON arguments for a tool call
func parseToolArgs(argsStr string, toolName string) (map[string]interface{}, error) {
	if argsStr == "" {
		return nil, nil
	}

	var args map[string]interface{}
	if err := json.Unmarshal([]byte(argsStr), &args); err != nil {
		fmt.Println("Error: Arguments must be valid JSON")
		fmt.Printf("Example: call %s {\"param1\": \"value1\", \"param2\": 123}\n", toolName)
		return nil, fmt.Errorf("invalid JSON arguments: %w", err)
	}
	return args, nil
}

// displayToolResultContent displays a single content item from a tool result
func displayToolResultContent(content mcp.Content) {
	if textContent, ok := mcp.AsTextContent(content); ok {
		displayTextContent(textContent.Text)
	} else if imageContent, ok := mcp.AsImageContent(content); ok {
		fmt.Printf("[Image: MIME type %s, %d bytes]\n", imageContent.MIMEType, len(imageContent.Data))
	} else if audioContent, ok := mcp.AsAudioContent(content); ok {
		fmt.Printf("[Audio: MIME type %s, %d bytes]\n", audioContent.MIMEType, len(audioContent.Data))
	}
}

// displayTextContent displays text content, pretty-printing JSON if possible
func displayTextContent(text string) {
	var jsonData interface{}
	if err := json.Unmarshal([]byte(text), &jsonData); err == nil {
		fmt.Println(PrettyJSON(jsonData))
	} else {
		fmt.Println(text)
	}
}

// displayToolResult displays the result of a tool call
func displayToolResult(result *mcp.CallToolResult) {
	if result.IsError {
		fmt.Println("Tool returned an error:")
		for _, content := range result.Content {
			if textContent, ok := mcp.AsTextContent(content); ok {
				fmt.Printf("  %s\n", textContent.Text)
			}
		}
		return
	}

	fmt.Println("Result:")
	for _, content := range result.Content {
		displayToolResultContent(content)
	}
}

// handleCallTool executes a tool with the given arguments
func (r *REPL) handleCallTool(ctx context.Context, toolName string, argsStr string) error {
	if !r.client.ServerSupportsTools() {
		return fmt.Errorf("server does not support tools capability")
	}

	if tool := r.findTool(toolName); tool == nil {
		return fmt.Errorf("tool not found: %s", toolName)
	}

	args, err := parseToolArgs(argsStr, toolName)
	if err != nil {
		return err
	}

	fmt.Printf("Executing tool: %s...\n", toolName)
	result, err := r.client.CallTool(ctx, toolName, args)
	if err != nil {
		return fmt.Errorf("tool execution failed: %w", err)
	}

	displayToolResult(result)
	return nil
}

// handleGetResource retrieves and displays a resource
func (r *REPL) handleGetResource(ctx context.Context, uri string) error {
	// Check if server supports resources
	if !r.client.ServerSupportsResources() {
		return fmt.Errorf("server does not support resources capability")
	}

	// Find the resource to validate it exists
	r.client.mu.RLock()
	var resource *mcp.Resource
	for _, res := range r.client.resourceCache {
		if res.URI == uri {
			resource = &res
			break
		}
	}
	r.client.mu.RUnlock()

	if resource == nil {
		return fmt.Errorf("resource not found: %s", uri)
	}

	// Retrieve the resource
	fmt.Printf("Retrieving resource: %s...\n", uri)
	result, err := r.client.GetResource(ctx, uri)
	if err != nil {
		return fmt.Errorf("resource retrieval failed: %w", err)
	}

	// Display contents
	fmt.Println("Contents:")
	for _, content := range result.Contents {
		if textContent, ok := mcp.AsTextResourceContents(content); ok {
			// Check MIME type for appropriate display
			if resource.MIMEType == "application/json" {
				var jsonData interface{}
				if err := json.Unmarshal([]byte(textContent.Text), &jsonData); err == nil {
					fmt.Println(PrettyJSON(jsonData))
				} else {
					fmt.Println(textContent.Text)
				}
			} else {
				fmt.Println(textContent.Text)
			}
		} else if blobContent, ok := mcp.AsBlobResourceContents(content); ok {
			fmt.Printf("[Binary data: %d bytes]\n", len(blobContent.Blob))
		}
	}

	return nil
}

// findPrompt finds a prompt by name in the cache
func (r *REPL) findPrompt(promptName string) *mcp.Prompt {
	r.client.mu.RLock()
	defer r.client.mu.RUnlock()

	for _, p := range r.client.promptCache {
		if p.Name == promptName {
			return &p
		}
	}
	return nil
}

// showPromptArgumentHelp displays help for prompt arguments
func showPromptArgumentHelp(promptName string, arguments []mcp.PromptArgument) {
	fmt.Println("Error: Arguments must be valid JSON")
	fmt.Printf("Example: prompt %s {\"arg1\": \"value1\", \"arg2\": \"value2\"}\n", promptName)

	if len(arguments) == 0 {
		return
	}

	fmt.Println("Required arguments:")
	for _, arg := range arguments {
		if arg.Required {
			fmt.Printf("  - %s: %s\n", arg.Name, arg.Description)
		}
	}
}

// parsePromptArgs parses and validates prompt arguments
func parsePromptArgs(argsStr string, prompt *mcp.Prompt) (map[string]string, error) {
	args := make(map[string]string)

	if argsStr != "" {
		var jsonArgs map[string]interface{}
		if err := json.Unmarshal([]byte(argsStr), &jsonArgs); err != nil {
			showPromptArgumentHelp(prompt.Name, prompt.Arguments)
			return nil, fmt.Errorf("invalid JSON arguments: %w", err)
		}

		for k, v := range jsonArgs {
			args[k] = fmt.Sprintf("%v", v)
		}
	}

	// Check required arguments
	for _, arg := range prompt.Arguments {
		if arg.Required && args[arg.Name] == "" {
			return nil, fmt.Errorf("missing required argument: %s", arg.Name)
		}
	}

	return args, nil
}

// displayPromptMessageContent displays the content of a prompt message
func displayPromptMessageContent(content mcp.Content) {
	if textContent, ok := mcp.AsTextContent(content); ok {
		fmt.Printf("Content: %s\n", textContent.Text)
		return
	}
	if imageContent, ok := mcp.AsImageContent(content); ok {
		fmt.Printf("Content: [Image: MIME type %s, %d bytes]\n", imageContent.MIMEType, len(imageContent.Data))
		return
	}
	if audioContent, ok := mcp.AsAudioContent(content); ok {
		fmt.Printf("Content: [Audio: MIME type %s, %d bytes]\n", audioContent.MIMEType, len(audioContent.Data))
		return
	}
	if resource, ok := mcp.AsEmbeddedResource(content); ok {
		fmt.Printf("Content: [Embedded Resource: %v]\n", resource.Resource)
		return
	}
	fmt.Printf("Content: %+v\n", content)
}

// displayPromptResult displays the result of a prompt retrieval
func displayPromptResult(result *mcp.GetPromptResult) {
	fmt.Println("Messages:")
	for i, msg := range result.Messages {
		fmt.Printf("\n[%d] Role: %s\n", i+1, msg.Role)
		displayPromptMessageContent(msg.Content)
	}
}

// handleGetPrompt retrieves and displays a prompt with arguments
func (r *REPL) handleGetPrompt(ctx context.Context, promptName string, argsStr string) error {
	if !r.client.ServerSupportsPrompts() {
		return fmt.Errorf("server does not support prompts capability")
	}

	prompt := r.findPrompt(promptName)
	if prompt == nil {
		return fmt.Errorf("prompt not found: %s", promptName)
	}

	args, err := parsePromptArgs(argsStr, prompt)
	if err != nil {
		return err
	}

	fmt.Printf("Getting prompt: %s...\n", promptName)
	result, err := r.client.GetPrompt(ctx, promptName, args)
	if err != nil {
		return fmt.Errorf("prompt retrieval failed: %w", err)
	}

	displayPromptResult(result)
	return nil
}
