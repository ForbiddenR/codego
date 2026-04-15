package types

import "testing"

func TestToolInput_GetString(t *testing.T) {
	ti := ToolInput{"name": "hello", "count": 42}

	if got := ti.GetString("name"); got != "hello" {
		t.Errorf("GetString(name) = %q, want %q", got, "hello")
	}
	if got := ti.GetString("missing"); got != "" {
		t.Errorf("GetString(missing) = %q, want empty", got)
	}
	if got := ti.GetString("count"); got != "" {
		t.Errorf("GetString(count) = %q, want empty (not a string)", got)
	}
}

func TestToolInput_GetFloat(t *testing.T) {
	ti := ToolInput{
		"float64": float64(3.14),
		"float32": float32(2.5),
		"int":     42,
		"string":  "not a number",
	}

	tests := []struct {
		key  string
		want float64
	}{
		{"float64", 3.14},
		{"float32", 2.5},
		{"int", 42.0},
		{"string", 0},
		{"missing", 0},
	}
	for _, tt := range tests {
		got := ti.GetFloat(tt.key)
		if got != tt.want {
			t.Errorf("GetFloat(%q) = %v, want %v", tt.key, got, tt.want)
		}
	}
}

func TestToolInput_GetBool(t *testing.T) {
	ti := ToolInput{"yes": true, "no": false, "str": "true"}

	if got := ti.GetBool("yes"); !got {
		t.Error("GetBool(yes) should be true")
	}
	if got := ti.GetBool("no"); got {
		t.Error("GetBool(no) should be false")
	}
	if got := ti.GetBool("missing"); got {
		t.Error("GetBool(missing) should be false")
	}
	if got := ti.GetBool("str"); got {
		t.Error("GetBool(str) should be false (not a bool)")
	}
}

func TestToolInput_Has(t *testing.T) {
	ti := ToolInput{"key": nil}

	if !ti.Has("key") {
		t.Error("Has(key) should be true even if value is nil")
	}
	if ti.Has("missing") {
		t.Error("Has(missing) should be false")
	}
}

func TestNewToolResult(t *testing.T) {
	r := NewToolResult("success")
	if r.Output != "success" {
		t.Errorf("output = %q, want %q", r.Output, "success")
	}
	if r.IsError {
		t.Error("is_error should be false")
	}
}

func TestNewToolError(t *testing.T) {
	r := NewToolError("something went wrong")
	if r.Output != "something went wrong" {
		t.Errorf("output = %q, want %q", r.Output, "something went wrong")
	}
	if !r.IsError {
		t.Error("is_error should be true")
	}
}

func TestToolResult_Metadata(t *testing.T) {
	r := &ToolResult{
		Output:   "ok",
		Metadata: map[string]interface{}{"exit_code": 0},
	}
	if r.Metadata["exit_code"] != 0 {
		t.Errorf("metadata = %v, want exit_code=0", r.Metadata)
	}
}

func TestNewStringSchema(t *testing.T) {
	s := NewStringSchema("A name")
	if s.Type != "string" {
		t.Errorf("type = %q, want %q", s.Type, "string")
	}
	if s.Description != "A name" {
		t.Errorf("description = %q, want %q", s.Description, "A name")
	}
}

func TestNewNumberSchema(t *testing.T) {
	s := NewNumberSchema("A count")
	if s.Type != "number" {
		t.Errorf("type = %q, want %q", s.Type, "number")
	}
}

func TestNewBooleanSchema(t *testing.T) {
	s := NewBooleanSchema("A flag")
	if s.Type != "boolean" {
		t.Errorf("type = %q, want %q", s.Type, "boolean")
	}
}

func TestNewObjectSchema(t *testing.T) {
	s := NewObjectSchema(
		"Command input",
		map[string]*JSONSchema{
			"command": NewStringSchema("The command to run"),
			"timeout": NewNumberSchema("Timeout in ms"),
		},
		"command",
	)
	if s.Type != "object" {
		t.Errorf("type = %q, want %q", s.Type, "object")
	}
	if len(s.Properties) != 2 {
		t.Errorf("properties count = %d, want 2", len(s.Properties))
	}
	if len(s.Required) != 1 || s.Required[0] != "command" {
		t.Errorf("required = %v, want [command]", s.Required)
	}
}

func TestToolDef(t *testing.T) {
	def := ToolDef{
		Name:        "bash",
		Description: "Run a shell command",
		InputSchema: NewObjectSchema(
			"Bash input",
			map[string]*JSONSchema{
				"command": NewStringSchema("The command to execute"),
			},
			"command",
		),
	}
	if def.Name != "bash" {
		t.Errorf("name = %q, want %q", def.Name, "bash")
	}
	if def.Description != "Run a shell command" {
		t.Errorf("description = %q, want %q", def.Description, "Run a shell command")
	}
}
