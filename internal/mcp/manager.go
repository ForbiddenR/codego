package mcp

import (
	"context"
	"fmt"
	"sync"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/nice-code/codego/internal/types"
)

// Manager manages connections to MCP servers.
type Manager struct {
	mu      sync.RWMutex
	servers map[string]*ServerConnection
}

// ServerConnection holds a connected MCP server.
type ServerConnection struct {
	Name    string
	Command string
	Args    []string
	Client  mcpclient.MCPClient
	Tools   []mcp.Tool
}

// NewManager creates a new MCP manager.
func NewManager() *Manager {
	return &Manager{servers: make(map[string]*ServerConnection)}
}

// Connect connects to an MCP server via stdio.
func (m *Manager) Connect(ctx context.Context, name, command string, args []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.servers[name]; exists {
		return fmt.Errorf("server %q already connected", name)
	}

	client, err := mcpclient.NewStdioMCPClient(command, nil, args...)
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}

	// Initialize
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{
		Name:    "codego",
		Version: "0.1.0",
	}
	initReq.Params.Capabilities = mcp.ClientCapabilities{}

	_, err = client.Initialize(ctx, initReq)
	if err != nil {
		client.Close()
		return fmt.Errorf("initialize: %w", err)
	}

	// Discover tools
	toolsReq := mcp.ListToolsRequest{}
	toolsResp, err := client.ListTools(ctx, toolsReq)
	if err != nil {
		// Some servers don't have tools
		toolsResp = &mcp.ListToolsResult{}
	}

	conn := &ServerConnection{
		Name:    name,
		Command: command,
		Args:    args,
		Client:  client,
		Tools:   toolsResp.Tools,
	}

	m.servers[name] = conn
	return nil
}

// Disconnect disconnects from an MCP server.
func (m *Manager) Disconnect(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	conn, ok := m.servers[name]
	if !ok {
		return fmt.Errorf("server %q not found", name)
	}

	err := conn.Client.Close()
	delete(m.servers, name)
	return err
}

// DisconnectAll disconnects from all servers.
func (m *Manager) DisconnectAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, conn := range m.servers {
		conn.Client.Close()
		delete(m.servers, name)
	}
}

// ListServers returns info about all connected servers.
func (m *Manager) ListServers() []ServerInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var infos []ServerInfo
	for _, conn := range m.servers {
		info := ServerInfo{
			Name:      conn.Name,
			Command:   conn.Command,
			Args:      conn.Args,
			ToolCount: len(conn.Tools),
		}
		for _, t := range conn.Tools {
			info.Tools = append(info.Tools, t.Name)
		}
		infos = append(infos, info)
	}
	return infos
}

// ListAllTools returns tool definitions for all MCP tools across servers.
func (m *Manager) ListAllTools() []types.ToolDef {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var defs []types.ToolDef
	for _, conn := range m.servers {
		for _, t := range conn.Tools {
			defs = append(defs, types.ToolDef{
				Name:        fmt.Sprintf("mcp__%s__%s", conn.Name, t.Name),
				Description: t.Description,
				InputSchema: t.InputSchema,
			})
		}
	}
	return defs
}

// CallTool calls a tool on a specific MCP server.
func (m *Manager) CallTool(ctx context.Context, serverName, toolName string, input map[string]interface{}) (*types.ToolResult, error) {
	m.mu.RLock()
	conn, ok := m.servers[serverName]
	m.mu.RUnlock()

	if !ok {
		return types.NewToolError(fmt.Sprintf("MCP server %q not found", serverName)), nil
	}

	req := mcp.CallToolRequest{}
	req.Params.Name = toolName
	req.Params.Arguments = input

	result, err := conn.Client.CallTool(ctx, req)
	if err != nil {
		return types.NewToolError(fmt.Sprintf("MCP tool error: %v", err)), nil
	}

	// Convert result to string
	var output string
	for _, content := range result.Content {
		if tc, ok := content.(mcp.TextContent); ok {
			output += tc.Text
		}
	}

	if output == "" {
		output = "(no output)"
	}

	if result.IsError {
		return types.NewToolError(output), nil
	}

	return types.NewToolResult(output), nil
}

// GetServer returns a server connection by name.
func (m *Manager) GetServer(name string) (*ServerConnection, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	conn, ok := m.servers[name]
	return conn, ok
}

// Count returns the number of connected servers.
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.servers)
}

// ServerInfo is a summary of a connected server.
type ServerInfo struct {
	Name      string
	Command   string
	Args      []string
	ToolCount int
	Tools     []string
}
