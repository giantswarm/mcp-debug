package agent

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/client"
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
func NewMCPServer(endpoint, clientTransport, serverTransport string, logger *Logger, notifyClients bool) (*MCPServer, error) {
	client := NewClient(endpoint, clientTransport, logger)

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
	// Connect to server first
	if err := m.connectToServer(ctx); err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	// Start the server with the specified transport
	switch m.serverTransport {
	case "stdio":
		return server.ServeStdio(m.mcpServer)
	case "streamable-http":
		httpServer := server.NewStreamableHTTPServer(m.mcpServer)
		return httpServer.Start(listenAddr)
	default:
		return fmt.Errorf("unsupported server transport: %s", m.serverTransport)
	}
}

// connectToServer establishes connection to the MCP server
func (m *MCPServer) connectToServer(ctx context.Context) error {
	m.logger.Info("Connecting to MCP server at %s...", m.client.endpoint)

	var mcpClient *client.Client
	var err error

	switch m.client.transport {
	case "sse":
		mcpClient, err = client.NewSSEMCPClient(m.client.endpoint)
		if err != nil {
			return fmt.Errorf("failed to create SSE client: %w", err)
		}
	case "streamable-http":
		mcpClient, err = client.NewStreamableHttpClient(m.client.endpoint)
		if err != nil {
			return fmt.Errorf("failed to create streamable HTTP client: %w", err)
		}
	default:
		return fmt.Errorf("unsupported transport to connect to upstream: %s", m.client.transport)
	}
	m.client.client = mcpClient

	// Start the transport
	if err := mcpClient.Start(ctx); err != nil {
		return fmt.Errorf("failed to start client: %w", err)
	}

	// Initialize the session
	if err := m.client.initialize(ctx); err != nil {
		mcpClient.Close()
		return fmt.Errorf("initialization failed: %w", err)
	}

	// List initial items
	if err := m.client.listTools(ctx, true); err != nil {
		m.logger.Error("Failed to list tools: %v", err)
	}

	if err := m.client.listResources(ctx, true); err != nil {
		m.logger.Error("Failed to list resources: %v", err)
	}

	if err := m.client.listPrompts(ctx, true); err != nil {
		m.logger.Error("Failed to list prompts: %v", err)
	}

	return nil
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
