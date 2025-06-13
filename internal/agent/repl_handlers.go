package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// handleCallTool executes a tool with the given arguments
func (r *REPL) handleCallTool(ctx context.Context, toolName string, argsStr string) error {
	// Check if server supports tools
	if !r.client.ServerSupportsTools() {
		return fmt.Errorf("server does not support tools capability")
	}

	// Find the tool to validate it exists
	r.client.mu.RLock()
	var tool *mcp.Tool
	for _, t := range r.client.toolCache {
		if t.Name == toolName {
			tool = &t
			break
		}
	}
	r.client.mu.RUnlock()

	if tool == nil {
		return fmt.Errorf("tool not found: %s", toolName)
	}

	// Parse arguments
	var args map[string]interface{}
	if argsStr != "" {
		// Try to parse as JSON
		if err := json.Unmarshal([]byte(argsStr), &args); err != nil {
			// If not valid JSON, provide help
			fmt.Println("Error: Arguments must be valid JSON")
			fmt.Printf("Example: call %s {\"param1\": \"value1\", \"param2\": 123}\n", toolName)
			return fmt.Errorf("invalid JSON arguments: %w", err)
		}
	}

	// Execute the tool
	fmt.Printf("Executing tool: %s...\n", toolName)
	result, err := r.client.CallTool(ctx, toolName, args)
	if err != nil {
		return fmt.Errorf("tool execution failed: %w", err)
	}

	// Display results
	if result.IsError {
		fmt.Println("Tool returned an error:")
		for _, content := range result.Content {
			if textContent, ok := mcp.AsTextContent(content); ok {
				fmt.Printf("  %s\n", textContent.Text)
			}
		}
	} else {
		fmt.Println("Result:")
		for _, content := range result.Content {
			if textContent, ok := mcp.AsTextContent(content); ok {
				// Try to pretty-print if it's JSON
				var jsonData interface{}
				if err := json.Unmarshal([]byte(textContent.Text), &jsonData); err == nil {
					fmt.Println(PrettyJSON(jsonData))
				} else {
					fmt.Println(textContent.Text)
				}
			} else if imageContent, ok := mcp.AsImageContent(content); ok {
				fmt.Printf("[Image: MIME type %s, %d bytes]\n", imageContent.MIMEType, len(imageContent.Data))
			} else if audioContent, ok := mcp.AsAudioContent(content); ok {
				fmt.Printf("[Audio: MIME type %s, %d bytes]\n", audioContent.MIMEType, len(audioContent.Data))
			}
		}
	}

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

// handleGetPrompt retrieves and displays a prompt with arguments
func (r *REPL) handleGetPrompt(ctx context.Context, promptName string, argsStr string) error {
	// Check if server supports prompts
	if !r.client.ServerSupportsPrompts() {
		return fmt.Errorf("server does not support prompts capability")
	}

	// Find the prompt to validate it exists
	r.client.mu.RLock()
	var prompt *mcp.Prompt
	for _, p := range r.client.promptCache {
		if p.Name == promptName {
			prompt = &p
			break
		}
	}
	r.client.mu.RUnlock()

	if prompt == nil {
		return fmt.Errorf("prompt not found: %s", promptName)
	}

	// Parse arguments
	args := make(map[string]string)
	if argsStr != "" {
		// Try to parse as JSON
		var jsonArgs map[string]interface{}
		if err := json.Unmarshal([]byte(argsStr), &jsonArgs); err != nil {
			// If not valid JSON, provide help
			fmt.Println("Error: Arguments must be valid JSON")
			fmt.Printf("Example: prompt %s {\"arg1\": \"value1\", \"arg2\": \"value2\"}\n", promptName)

			// Show required arguments
			if len(prompt.Arguments) > 0 {
				fmt.Println("Required arguments:")
				for _, arg := range prompt.Arguments {
					if arg.Required {
						fmt.Printf("  - %s: %s\n", arg.Name, arg.Description)
					}
				}
			}
			return fmt.Errorf("invalid JSON arguments: %w", err)
		}

		// Convert to string map
		for k, v := range jsonArgs {
			args[k] = fmt.Sprintf("%v", v)
		}
	}

	// Check required arguments
	for _, arg := range prompt.Arguments {
		if arg.Required && args[arg.Name] == "" {
			return fmt.Errorf("missing required argument: %s", arg.Name)
		}
	}

	// Get the prompt
	fmt.Printf("Getting prompt: %s...\n", promptName)
	result, err := r.client.GetPrompt(ctx, promptName, args)
	if err != nil {
		return fmt.Errorf("prompt retrieval failed: %w", err)
	}

	// Display messages
	fmt.Println("Messages:")
	for i, msg := range result.Messages {
		fmt.Printf("\n[%d] Role: %s\n", i+1, msg.Role)
		if textContent, ok := mcp.AsTextContent(msg.Content); ok {
			fmt.Printf("Content: %s\n", textContent.Text)
		} else if imageContent, ok := mcp.AsImageContent(msg.Content); ok {
			fmt.Printf("Content: [Image: MIME type %s, %d bytes]\n", imageContent.MIMEType, len(imageContent.Data))
		} else if audioContent, ok := mcp.AsAudioContent(msg.Content); ok {
			fmt.Printf("Content: [Audio: MIME type %s, %d bytes]\n", audioContent.MIMEType, len(audioContent.Data))
		} else if resource, ok := mcp.AsEmbeddedResource(msg.Content); ok {
			fmt.Printf("Content: [Embedded Resource: %v]\n", resource.Resource)
		} else {
			// Fallback for unknown content types
			fmt.Printf("Content: %+v\n", msg.Content)
		}
	}

	return nil
}
