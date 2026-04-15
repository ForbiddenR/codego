package types

// ToolInput is a map of parameter names to values passed to a tool.
type ToolInput map[string]interface{}

// GetString returns a string value from the input, or empty string if missing.
func (ti ToolInput) GetString(key string) string {
	if v, ok := ti[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetFloat returns a float64 value from the input, or 0 if missing.
func (ti ToolInput) GetFloat(key string) float64 {
	if v, ok := ti[key]; ok {
		switch n := v.(type) {
		case float64:
			return n
		case float32:
			return float64(n)
		case int:
			return float64(n)
		}
	}
	return 0
}

// GetBool returns a bool value from the input, or false if missing.
func (ti ToolInput) GetBool(key string) bool {
	if v, ok := ti[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

// Has returns true if the key exists in the input.
func (ti ToolInput) Has(key string) bool {
	_, ok := ti[key]
	return ok
}

// ToolResult is the output from executing a tool.
type ToolResult struct {
	Output   string                 `json:"output"`
	IsError  bool                   `json:"is_error"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewToolResult creates a successful tool result.
func NewToolResult(output string) *ToolResult {
	return &ToolResult{Output: output}
}

// NewToolError creates an error tool result.
func NewToolError(output string) *ToolResult {
	return &ToolResult{Output: output, IsError: true}
}

// ToolDef is the schema definition for a tool, sent to the API.
type ToolDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"input_schema"`
}

// JSONSchema is a helper for building JSON Schema objects for tool inputs.
type JSONSchema struct {
	Type        string                `json:"type"`
	Properties  map[string]*JSONSchema `json:"properties,omitempty"`
	Required    []string              `json:"required,omitempty"`
	Description string                `json:"description,omitempty"`
	Enum        []string              `json:"enum,omitempty"`
	Items       *JSONSchema           `json:"items,omitempty"`
}

// NewObjectSchema creates a JSON Schema for an object type.
func NewObjectSchema(description string, properties map[string]*JSONSchema, required ...string) *JSONSchema {
	return &JSONSchema{
		Type:        "object",
		Description: description,
		Properties:  properties,
		Required:    required,
	}
}

// NewStringSchema creates a JSON Schema for a string type.
func NewStringSchema(description string) *JSONSchema {
	return &JSONSchema{
		Type:        "string",
		Description: description,
	}
}

// NewNumberSchema creates a JSON Schema for a number type.
func NewNumberSchema(description string) *JSONSchema {
	return &JSONSchema{
		Type:        "number",
		Description: description,
	}
}

// NewBooleanSchema creates a JSON Schema for a boolean type.
func NewBooleanSchema(description string) *JSONSchema {
	return &JSONSchema{
		Type:        "boolean",
		Description: description,
	}
}
