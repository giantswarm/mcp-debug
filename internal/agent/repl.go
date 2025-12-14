package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/chzyer/readline"
)

// errExit is a sentinel error used to signal REPL exit
var errExit = errors.New("exit")

// REPL represents the Read-Eval-Print Loop for MCP interaction
type REPL struct {
	client          *Client
	logger          *Logger
	rl              *readline.Instance
	stopChan        chan struct{}
	wg              sync.WaitGroup
	commandHandlers map[string]commandHandler
}

// NewREPL creates a new REPL instance
func NewREPL(client *Client, logger *Logger) *REPL {
	r := &REPL{
		client:   client,
		logger:   logger,
		stopChan: make(chan struct{}),
	}
	r.commandHandlers = r.buildCommandHandlers()
	return r
}

// Run starts the REPL
func (r *REPL) Run(ctx context.Context) error {
	// Set up readline with tab completion
	completer := r.createCompleter()
	historyFile := filepath.Join(os.TempDir(), ".mcp_debug_history")

	config := &readline.Config{
		Prompt:          "MCP> ",
		HistoryFile:     historyFile,
		AutoComplete:    completer,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",

		HistorySearchFold:   true,
		FuncFilterInputRune: filterInput,
	}

	rl, err := readline.NewEx(config)
	if err != nil {
		return fmt.Errorf("failed to create readline instance: %w", err)
	}
	defer func() { _ = rl.Close() }()
	r.rl = rl

	// Start notification listener in background
	r.wg.Add(1)
	go r.notificationListener(ctx)

	// Display welcome message
	r.logger.Info("MCP REPL started. Type 'help' for available commands. Use TAB for completion.")
	fmt.Println()

	// Main REPL loop
	for {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			close(r.stopChan)
			r.wg.Wait()
			r.logger.Info("REPL shutting down...")
			return nil
		default:
		}

		// Read input
		line, err := rl.Readline()
		if err == readline.ErrInterrupt {
			if len(line) == 0 {
				continue
			}
		} else if err == io.EOF {
			close(r.stopChan)
			r.wg.Wait()
			r.logger.Info("Goodbye!")
			return nil
		} else if err != nil {
			return fmt.Errorf("readline error: %w", err)
		}

		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}

		// Parse and execute command
		if err := r.executeCommand(ctx, input); err != nil {
			if errors.Is(err, errExit) {
				close(r.stopChan)
				r.wg.Wait()
				r.logger.Info("Goodbye!")
				return nil
			}
			r.logger.Error("Error: %v", err)
		}

		fmt.Println()
	}
}

// completerCache holds cached completion items for different capabilities
type completerCache struct {
	tools     []string
	resources []string
	prompts   []string
}

// getCompletionNames retrieves names for tab completion from the client cache
func (r *REPL) getCompletionNames() completerCache {
	r.client.mu.RLock()
	defer r.client.mu.RUnlock()

	cache := completerCache{}

	if r.client.ServerSupportsTools() {
		cache.tools = make([]string, len(r.client.toolCache))
		for i, tool := range r.client.toolCache {
			cache.tools[i] = tool.Name
		}
	}

	if r.client.ServerSupportsResources() {
		cache.resources = make([]string, len(r.client.resourceCache))
		for i, resource := range r.client.resourceCache {
			cache.resources[i] = resource.URI
		}
	}

	if r.client.ServerSupportsPrompts() {
		cache.prompts = make([]string, len(r.client.promptCache))
		for i, prompt := range r.client.promptCache {
			cache.prompts[i] = prompt.Name
		}
	}

	return cache
}

// buildPcItems converts a slice of strings to readline completer items
func buildPcItems(names []string) []readline.PrefixCompleterInterface {
	items := make([]readline.PrefixCompleterInterface, len(names))
	for i, name := range names {
		items[i] = readline.PcItem(name)
	}
	return items
}

// buildListItems creates list command completion items based on server capabilities
func (r *REPL) buildListItems() []readline.PrefixCompleterInterface {
	var items []readline.PrefixCompleterInterface
	if r.client.ServerSupportsTools() {
		items = append(items, readline.PcItem("tools"))
	}
	if r.client.ServerSupportsResources() {
		items = append(items, readline.PcItem("resources"))
	}
	if r.client.ServerSupportsPrompts() {
		items = append(items, readline.PcItem("prompts"))
	}
	return items
}

// buildDescribeItems creates describe command completion items
func (r *REPL) buildDescribeItems(toolCompleter, resourceCompleter, promptCompleter []readline.PrefixCompleterInterface) []readline.PrefixCompleterInterface {
	var items []readline.PrefixCompleterInterface
	if r.client.ServerSupportsTools() {
		items = append(items, readline.PcItem("tool", toolCompleter...))
	}
	if r.client.ServerSupportsResources() {
		items = append(items, readline.PcItem("resource", resourceCompleter...))
	}
	if r.client.ServerSupportsPrompts() {
		items = append(items, readline.PcItem("prompt", promptCompleter...))
	}
	return items
}

// buildBaseCompleterItems creates the base command completion items
func buildBaseCompleterItems() []readline.PrefixCompleterInterface {
	return []readline.PrefixCompleterInterface{
		readline.PcItem("help"),
		readline.PcItem("?"),
		readline.PcItem("exit"),
		readline.PcItem("quit"),
		readline.PcItem("notifications",
			readline.PcItem("on"),
			readline.PcItem("off"),
		),
	}
}

// createCompleter creates the tab completion configuration
func (r *REPL) createCompleter() *readline.PrefixCompleter {
	cache := r.getCompletionNames()

	toolCompleter := buildPcItems(cache.tools)
	resourceCompleter := buildPcItems(cache.resources)
	promptCompleter := buildPcItems(cache.prompts)

	listItems := r.buildListItems()
	describeItems := r.buildDescribeItems(toolCompleter, resourceCompleter, promptCompleter)

	items := buildBaseCompleterItems()

	if len(listItems) > 0 {
		items = append(items, readline.PcItem("list", listItems...))
	}
	if len(describeItems) > 0 {
		items = append(items, readline.PcItem("describe", describeItems...))
	}
	if r.client.ServerSupportsTools() {
		items = append(items, readline.PcItem("call", toolCompleter...))
	}
	if r.client.ServerSupportsResources() {
		items = append(items, readline.PcItem("get", resourceCompleter...))
	}
	if r.client.ServerSupportsPrompts() {
		items = append(items, readline.PcItem("prompt", promptCompleter...))
	}

	return readline.NewPrefixCompleter(items...)
}

// filterInput filters input characters for readline
func filterInput(r rune) (rune, bool) {
	switch r {
	// block CtrlZ feature
	case readline.CharCtrlZ:
		return r, false
	}
	return r, true
}

// notificationListener handles notifications in the background
func (r *REPL) notificationListener(ctx context.Context) {
	defer r.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stopChan:
			return
		case notification := <-r.client.notificationChan:
			// Temporarily pause readline
			if r.rl != nil {
				_, _ = r.rl.Stdout().Write([]byte("\r\033[K"))
			}

			// Handle the notification (this will log it)
			if err := r.client.handleNotification(ctx, notification); err != nil {
				r.logger.Error("Failed to handle notification: %v", err)
			}

			// Update completer if items changed
			switch notification.Method {
			case notificationToolsListChanged,
				notificationResourcesListChanged,
				notificationPromptsListChanged:
				if r.rl != nil {
					r.rl.Config.AutoComplete = r.createCompleter()
				}
			}

			// Refresh readline prompt
			if r.rl != nil {
				r.rl.Refresh()
			}
		}
	}
}

// commandHandler defines a REPL command with its handler and argument requirements
type commandHandler struct {
	minArgs int
	usage   string
	handler func(ctx context.Context, parts []string) error
}

// buildCommandHandlers creates the map of command handlers
func (r *REPL) buildCommandHandlers() map[string]commandHandler {
	return map[string]commandHandler{
		"help": {minArgs: 1, handler: func(ctx context.Context, parts []string) error {
			return r.showHelp()
		}},
		"?": {minArgs: 1, handler: func(ctx context.Context, parts []string) error {
			return r.showHelp()
		}},
		"exit": {minArgs: 1, handler: func(ctx context.Context, parts []string) error {
			return errExit
		}},
		"quit": {minArgs: 1, handler: func(ctx context.Context, parts []string) error {
			return errExit
		}},
		"list": {
			minArgs: 2,
			usage:   "usage: list <tools|resources|prompts>",
			handler: func(ctx context.Context, parts []string) error {
				return r.handleList(ctx, parts[1])
			},
		},
		"describe": {
			minArgs: 3,
			usage:   "usage: describe <tool|resource|prompt> <name>",
			handler: func(ctx context.Context, parts []string) error {
				return r.handleDescribe(ctx, parts[1], strings.Join(parts[2:], " "))
			},
		},
		"notifications": {
			minArgs: 2,
			usage:   "usage: notifications <on|off>",
			handler: func(ctx context.Context, parts []string) error {
				return r.handleNotifications(parts[1])
			},
		},
		"call": {
			minArgs: 2,
			usage:   "usage: call <tool-name> [args...]",
			handler: func(ctx context.Context, parts []string) error {
				return r.handleCallTool(ctx, parts[1], strings.Join(parts[2:], " "))
			},
		},
		"get": {
			minArgs: 2,
			usage:   "usage: get <resource-uri>",
			handler: func(ctx context.Context, parts []string) error {
				return r.handleGetResource(ctx, parts[1])
			},
		},
		"prompt": {
			minArgs: 2,
			usage:   "usage: prompt <prompt-name> [args...]",
			handler: func(ctx context.Context, parts []string) error {
				return r.handleGetPrompt(ctx, parts[1], strings.Join(parts[2:], " "))
			},
		},
	}
}

// executeCommand parses and executes a command
func (r *REPL) executeCommand(ctx context.Context, input string) error {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil
	}

	command := strings.ToLower(parts[0])

	handler, exists := r.commandHandlers[command]
	if !exists {
		return fmt.Errorf("unknown command: %s. Type 'help' for available commands", command)
	}

	if len(parts) < handler.minArgs {
		return errors.New(handler.usage)
	}

	return handler.handler(ctx, parts)
}

// showHelp displays available commands
func (r *REPL) showHelp() error {
	fmt.Println("Available commands:")
	fmt.Println("  help, ?                      - Show this help message")
	fmt.Println("  list tools                   - List all available tools")
	fmt.Println("  list resources               - List all available resources")
	fmt.Println("  list prompts                 - List all available prompts")
	fmt.Println("  describe tool <name>         - Show detailed information about a tool")
	fmt.Println("  describe resource <uri>      - Show detailed information about a resource")
	fmt.Println("  describe prompt <name>       - Show detailed information about a prompt")
	fmt.Println("  call <tool> {json}           - Execute a tool with JSON arguments")
	fmt.Println("  get <resource-uri>           - Retrieve a resource")
	fmt.Println("  prompt <name> {json}         - Get a prompt with JSON arguments")
	fmt.Println("  notifications <on|off>       - Enable/disable notification display")
	fmt.Println("  exit, quit                   - Exit the REPL")
	fmt.Println()
	fmt.Println("Keyboard shortcuts:")
	fmt.Println("  TAB                          - Auto-complete commands and arguments")
	fmt.Println("  ↑/↓ (arrow keys)             - Navigate command history")
	fmt.Println("  Ctrl+R                       - Search command history")
	fmt.Println("  Ctrl+C                       - Cancel current line")
	fmt.Println("  Ctrl+D                       - Exit REPL")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  call calculate {\"operation\": \"add\", \"x\": 5, \"y\": 3}")
	fmt.Println("  get docs://readme")
	fmt.Println("  prompt greeting {\"name\": \"Alice\"}")
	return nil
}

// handleList handles list commands
func (r *REPL) handleList(ctx context.Context, target string) error {
	switch strings.ToLower(target) {
	case "tools", "tool":
		if !r.client.ServerSupportsTools() {
			fmt.Println("Server does not support tools capability.")
			return nil
		}
		return r.listTools(ctx)
	case "resources", "resource":
		if !r.client.ServerSupportsResources() {
			fmt.Println("Server does not support resources capability.")
			return nil
		}
		return r.listResources(ctx)
	case "prompts", "prompt":
		if !r.client.ServerSupportsPrompts() {
			fmt.Println("Server does not support prompts capability.")
			return nil
		}
		return r.listPrompts(ctx)
	default:
		return fmt.Errorf("unknown list target: %s. Use 'tools', 'resources', or 'prompts'", target)
	}
}

// listTools displays available tools
func (r *REPL) listTools(ctx context.Context) error {
	r.client.mu.RLock()
	tools := r.client.toolCache
	r.client.mu.RUnlock()

	if len(tools) == 0 {
		fmt.Println("No tools available.")
		return nil
	}

	fmt.Printf("Available tools (%d):\n", len(tools))
	for i, tool := range tools {
		fmt.Printf("  %d. %-30s - %s\n", i+1, tool.Name, tool.Description)
	}
	return nil
}

// listResources displays available resources
func (r *REPL) listResources(ctx context.Context) error {
	r.client.mu.RLock()
	resources := r.client.resourceCache
	r.client.mu.RUnlock()

	if len(resources) == 0 {
		fmt.Println("No resources available.")
		return nil
	}

	fmt.Printf("Available resources (%d):\n", len(resources))
	for i, resource := range resources {
		desc := resource.Description
		if desc == "" {
			desc = resource.Name
		}
		fmt.Printf("  %d. %-40s - %s\n", i+1, resource.URI, desc)
	}
	return nil
}

// listPrompts displays available prompts
func (r *REPL) listPrompts(ctx context.Context) error {
	r.client.mu.RLock()
	prompts := r.client.promptCache
	r.client.mu.RUnlock()

	if len(prompts) == 0 {
		fmt.Println("No prompts available.")
		return nil
	}

	fmt.Printf("Available prompts (%d):\n", len(prompts))
	for i, prompt := range prompts {
		fmt.Printf("  %d. %-30s - %s\n", i+1, prompt.Name, prompt.Description)
	}
	return nil
}

// handleDescribe handles describe commands
func (r *REPL) handleDescribe(ctx context.Context, targetType, name string) error {
	switch strings.ToLower(targetType) {
	case "tool":
		if !r.client.ServerSupportsTools() {
			return fmt.Errorf("server does not support tools capability")
		}
		return r.describeTool(ctx, name)
	case "resource":
		if !r.client.ServerSupportsResources() {
			return fmt.Errorf("server does not support resources capability")
		}
		return r.describeResource(ctx, name)
	case "prompt":
		if !r.client.ServerSupportsPrompts() {
			return fmt.Errorf("server does not support prompts capability")
		}
		return r.describePrompt(ctx, name)
	default:
		return fmt.Errorf("unknown describe target: %s. Use 'tool', 'resource', or 'prompt'", targetType)
	}
}

// describeTool shows detailed information about a tool
func (r *REPL) describeTool(ctx context.Context, name string) error {
	r.client.mu.RLock()
	defer r.client.mu.RUnlock()

	for _, tool := range r.client.toolCache {
		if tool.Name == name {
			fmt.Printf("Tool: %s\n", tool.Name)
			fmt.Printf("Description: %s\n", tool.Description)
			fmt.Println("Input Schema:")
			fmt.Printf("%s\n", PrettyJSON(tool.InputSchema))
			return nil
		}
	}

	return fmt.Errorf("tool not found: %s", name)
}

// describeResource shows detailed information about a resource
func (r *REPL) describeResource(ctx context.Context, uri string) error {
	r.client.mu.RLock()
	defer r.client.mu.RUnlock()

	for _, resource := range r.client.resourceCache {
		if resource.URI == uri {
			fmt.Printf("Resource: %s\n", resource.URI)
			fmt.Printf("Name: %s\n", resource.Name)
			if resource.Description != "" {
				fmt.Printf("Description: %s\n", resource.Description)
			}
			if resource.MIMEType != "" {
				fmt.Printf("MIME Type: %s\n", resource.MIMEType)
			}
			return nil
		}
	}

	return fmt.Errorf("resource not found: %s", uri)
}

// describePrompt shows detailed information about a prompt
func (r *REPL) describePrompt(ctx context.Context, name string) error {
	r.client.mu.RLock()
	defer r.client.mu.RUnlock()

	for _, prompt := range r.client.promptCache {
		if prompt.Name == name {
			fmt.Printf("Prompt: %s\n", prompt.Name)
			fmt.Printf("Description: %s\n", prompt.Description)
			if len(prompt.Arguments) > 0 {
				fmt.Println("Arguments:")
				for _, arg := range prompt.Arguments {
					required := ""
					if arg.Required {
						required = " (required)"
					}
					fmt.Printf("  - %s%s: %s\n", arg.Name, required, arg.Description)
				}
			}
			return nil
		}
	}

	return fmt.Errorf("prompt not found: %s", name)
}

// handleNotifications enables or disables notification display
func (r *REPL) handleNotifications(setting string) error {
	switch strings.ToLower(setting) {
	case "on":
		r.logger.SetVerbose(true)
		fmt.Println("Notifications enabled")
	case "off":
		r.logger.SetVerbose(false)
		fmt.Println("Notifications disabled")
	default:
		return fmt.Errorf("invalid setting: %s. Use 'on' or 'off'", setting)
	}
	return nil
}
