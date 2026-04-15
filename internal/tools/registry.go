package tools

import (
	"context"
	"fmt"

	"github.com/nice-code/codego/internal/types"
)

// Tool is the interface that all tools must implement.
type Tool interface {
	// Name returns the tool name (used as identifier in tool calls).
	Name() string

	// Description returns a human-readable description for the LLM.
	Description() string

	// InputSchema returns the JSON Schema for the tool's input.
	InputSchema() *types.JSONSchema

	// Execute runs the tool with the given input.
	Execute(ctx context.Context, input types.ToolInput) (*types.ToolResult, error)
}

// Registry holds all available tools.
type Registry struct {
	tools map[string]Tool
}

// NewRegistry creates an empty tool registry.
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

// Register adds a tool to the registry.
func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
}

// Get retrieves a tool by name.
func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// All returns all registered tools.
func (r *Registry) All() []Tool {
	tools := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		tools = append(tools, t)
	}
	return tools
}

// Names returns all registered tool names.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// ToolDefs returns tool definitions for the API request.
func (r *Registry) ToolDefs() []types.ToolDef {
	defs := make([]types.ToolDef, 0, len(r.tools))
	for _, t := range r.tools {
		defs = append(defs, types.ToolDef{
			Name:        t.Name(),
			Description: t.Description(),
			InputSchema: t.InputSchema(),
		})
	}
	return defs
}

// Count returns the number of registered tools.
func (r *Registry) Count() int {
	return len(r.tools)
}

// Validate checks that a tool input matches the schema.
// Returns an error for missing required fields.
func Validate(input types.ToolInput, schema *types.JSONSchema) error {
	if schema == nil {
		return nil
	}
	if schema.Type != "object" {
		return nil
	}
	for _, field := range schema.Required {
		if !input.Has(field) {
			return fmt.Errorf("missing required field: %q", field)
		}
	}
	return nil
}
