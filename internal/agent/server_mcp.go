package agent

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// MCPServer wraps the agent functionality and exposes it via MCP
type MCPServer struct {
	client          *Client
	logger          *Logger
	mcpServer       *server.MCPServer
	notifyClients   bool
	serverTransport string
}

// NewMCPServer creates a new MCP server that exposes agent functionality
func NewMCPServer(client *Client, serverTransport string, logger *Logger, notifyClients bool) (*MCPServer, error) {
	// Create MCP server
	mcpServer := server.NewMCPServer(
		"mcp-debug-agent",
		"1.0.0",
		server.WithToolCapabilities(notifyClients),
		server.WithResourceCapabilities(false, false),
		server.WithPromptCapabilities(false),
	)

	ms := &MCPServer{
		client:          client,
		logger:          logger,
		mcpServer:       mcpServer,
		notifyClients:   notifyClients,
		serverTransport: serverTransport,
	}

	// Register all tools
	ms.registerTools()

	return ms, nil
}

// Start starts the MCP server using stdio or streamable-http transport
func (m *MCPServer) Start(ctx context.Context, listenAddr string) error {
	// Start the server with the specified transport
	switch m.serverTransport {
	case "stdio":
		return server.ServeStdio(m.mcpServer)
	case "streamable-http":
		httpServer := server.NewStreamableHTTPServer(
			m.mcpServer,
			server.WithEndpointPath("/mcp"),
		)
		return httpServer.Start(listenAddr)
	default:
		return fmt.Errorf("unsupported server transport: %s", m.serverTransport)
	}
}

// registerTools registers all MCP tools
func (m *MCPServer) registerTools() {
	// List tools
	listToolsTool := mcp.NewTool("list_tools",
		mcp.WithDescription("List all available tools from connected MCP servers"),
	)
	m.mcpServer.AddTool(listToolsTool, m.handleListTools)

	// List resources
	listResourcesTool := mcp.NewTool("list_resources",
		mcp.WithDescription("List all available resources from connected MCP servers"),
	)
	m.mcpServer.AddTool(listResourcesTool, m.handleListResources)

	// List prompts
	listPromptsTool := mcp.NewTool("list_prompts",
		mcp.WithDescription("List all available prompts from connected MCP servers"),
	)
	m.mcpServer.AddTool(listPromptsTool, m.handleListPrompts)

	// Describe tool
	describeToolTool := mcp.NewTool("describe_tool",
		mcp.WithDescription("Get detailed information about a specific tool"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the tool to describe"),
		),
	)
	m.mcpServer.AddTool(describeToolTool, m.handleDescribeTool)

	// Describe resource
	describeResourceTool := mcp.NewTool("describe_resource",
		mcp.WithDescription("Get detailed information about a specific resource"),
		mcp.WithString("uri",
			mcp.Required(),
			mcp.Description("URI of the resource to describe"),
		),
	)
	m.mcpServer.AddTool(describeResourceTool, m.handleDescribeResource)

	// Describe prompt
	describePromptTool := mcp.NewTool("describe_prompt",
		mcp.WithDescription("Get detailed information about a specific prompt"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the prompt to describe"),
		),
	)
	m.mcpServer.AddTool(describePromptTool, m.handleDescribePrompt)

	// Call tool
	callToolTool := mcp.NewTool("call_tool",
		mcp.WithDescription("Execute a tool with the given arguments"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the tool to call"),
		),
		mcp.WithObject("arguments",
			mcp.Description("Arguments to pass to the tool (as JSON object)"),
		),
	)
	m.mcpServer.AddTool(callToolTool, m.handleCallTool)

	// Get resource
	getResourceTool := mcp.NewTool("get_resource",
		mcp.WithDescription("Retrieve the contents of a resource"),
		mcp.WithString("uri",
			mcp.Required(),
			mcp.Description("URI of the resource to retrieve"),
		),
	)
	m.mcpServer.AddTool(getResourceTool, m.handleGetResource)

	// Get prompt
	getPromptTool := mcp.NewTool("get_prompt",
		mcp.WithDescription("Get a prompt with the given arguments"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the prompt to get"),
		),
		mcp.WithObject("arguments",
			mcp.Description("Arguments to pass to the prompt (as JSON object with string values)"),
		),
	)
	m.mcpServer.AddTool(getPromptTool, m.handleGetPrompt)
}
