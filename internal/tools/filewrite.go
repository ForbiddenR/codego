package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nice-code/codego/internal/types"
)

// FileWriteTool writes content to files.
type FileWriteTool struct{}

func NewFileWriteTool() *FileWriteTool { return &FileWriteTool{} }

func (t *FileWriteTool) Name() string        { return "write" }
func (t *FileWriteTool) Description() string  { return "Write content to a file. Creates parent directories if needed. Overwrites existing files." }
func (t *FileWriteTool) InputSchema() *types.JSONSchema {
	return types.NewObjectSchema(
		"File write input",
		map[string]*types.JSONSchema{
			"path":    types.NewStringSchema("Absolute path to the file to write"),
			"content": types.NewStringSchema("The content to write to the file"),
		},
		"path", "content",
	)
}

func (t *FileWriteTool) Execute(_ context.Context, input types.ToolInput) (*types.ToolResult, error) {
	path := input.GetString("path")
	if path == "" {
		return types.NewToolError("path is required"), nil
	}

	content := input.GetString("content")
	if content == "" {
		return types.NewToolError("content is required"), nil
	}

	// Create parent directories
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return types.NewToolError(fmt.Sprintf("cannot create directory: %v", err)), nil
	}

	// Check if file exists (for reporting)
	existed := false
	if _, err := os.Stat(path); err == nil {
		existed = true
	}

	// Write the file
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return types.NewToolError(fmt.Sprintf("cannot write file: %v", err)), nil
	}

	action := "Created"
	if existed {
		action = "Overwrote"
	}

	return types.NewToolResult(fmt.Sprintf("%s %s (%d bytes)", action, path, len(content))), nil
}
