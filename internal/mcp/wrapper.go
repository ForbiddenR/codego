package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/nice-code/codego/internal/tools"
	"github.com/nice-code/codego/internal/types"
)

// ToolWrapper wraps an MCP tool as a tools.Tool for registry integration.
type ToolWrapper struct {
	serverName string
	toolName   string
	description string
	schema     interface{}
	manager    *Manager
}

// NewToolWrapper creates a wrapper for an MCP tool.
func NewToolWrapper(serverName, toolName, description string, schema interface{}, manager *Manager) *ToolWrapper {
	return &ToolWrapper{
		serverName:  serverName,
		toolName:    toolName,
		description: description,
		schema:      schema,
		manager:     manager,
	}
}

func (t *ToolWrapper) Name() string {
	return fmt.Sprintf("mcp__%s__%s", t.serverName, t.toolName)
}

func (t *ToolWrapper) Description() string {
	return fmt.Sprintf("[MCP:%s] %s", t.serverName, t.description)
}

func (t *ToolWrapper) InputSchema() *types.JSONSchema {
	// MCP schemas are interface{} — return as-is wrapped
	if t.schema != nil {
		return &types.JSONSchema{
			Type: "object",
		}
	}
	return nil
}

func (t *ToolWrapper) Execute(ctx context.Context, input types.ToolInput) (*types.ToolResult, error) {
	return t.manager.CallTool(ctx, t.serverName, t.toolName, input)
}

// RegisterAllTools registers all MCP tools from all connected servers into a tool registry.
func (m *Manager) RegisterAllTools(registry *tools.Registry) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, conn := range m.servers {
		for _, t := range conn.Tools {
			wrapper := NewToolWrapper(
				conn.Name,
				t.Name,
				t.Description,
				t.InputSchema,
				m,
			)
			registry.Register(wrapper)
			count++
		}
	}
	return count
}

// ParseMCPToolName parses "mcp__servername__toolname" into (serverName, toolName).
func ParseMCPToolName(fullName string) (serverName, toolName string, ok bool) {
	if !strings.HasPrefix(fullName, "mcp__") {
		return "", "", false
	}
	rest := fullName[5:]
	idx := strings.Index(rest, "__")
	if idx < 0 {
		return "", "", false
	}
	return rest[:idx], rest[idx+2:], true
}
