package tools

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/nice-code/codego/internal/types"
)

// FileEditTool performs find-and-replace edits on files.
type FileEditTool struct{}

func NewFileEditTool() *FileEditTool { return &FileEditTool{} }

func (t *FileEditTool) Name() string { return "edit" }
func (t *FileEditTool) Description() string {
	return "Edit a file by finding and replacing text. Shows a diff of the changes. Use for modifying existing files."
}
func (t *FileEditTool) InputSchema() *types.JSONSchema {
	return types.NewObjectSchema(
		"File edit input",
		map[string]*types.JSONSchema{
			"path":         types.NewStringSchema("Absolute path to the file to edit"),
			"old_string":   types.NewStringSchema("The exact text to find and replace"),
			"new_string":   types.NewStringSchema("The replacement text"),
			"replace_all":  types.NewBooleanSchema("Replace all occurrences (default false)"),
		},
		"path", "old_string", "new_string",
	)
}

func (t *FileEditTool) Execute(_ context.Context, input types.ToolInput) (*types.ToolResult, error) {
	path := input.GetString("path")
	if path == "" {
		return types.NewToolError("path is required"), nil
	}

	oldString := input.GetString("old_string")
	if oldString == "" {
		return types.NewToolError("old_string is required"), nil
	}

	newString := input.GetString("new_string")
	replaceAll := input.GetBool("replace_all")

	// Read file
	content, err := os.ReadFile(path)
	if err != nil {
		return types.NewToolError(fmt.Sprintf("cannot read file: %v", err)), nil
	}

	oldContent := string(content)

	// Check old_string exists
	if !strings.Contains(oldContent, oldString) {
		return types.NewToolError("old_string not found in file"), nil
	}

	// Count occurrences
	count := strings.Count(oldContent, oldString)
	if count > 1 && !replaceAll {
		return types.NewToolError(fmt.Sprintf("found %d occurrences of old_string. Set replace_all=true to replace all, or make old_string more specific", count)), nil
	}

	// Perform replacement
	var newContent string
	if replaceAll {
		newContent = strings.ReplaceAll(oldContent, oldString, newString)
	} else {
		newContent = strings.Replace(oldContent, oldString, newString, 1)
	}

	// Atomic write: temp file + rename
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(newContent), 0o644); err != nil {
		return types.NewToolError(fmt.Sprintf("cannot write temp file: %v", err)), nil
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return types.NewToolError(fmt.Sprintf("cannot rename file: %v", err)), nil
	}

	// Build diff
	replaced := count
	if !replaceAll {
		replaced = 1
	}
	diff := buildDiff(oldString, newString, path)

	return types.NewToolResult(fmt.Sprintf("Replaced %d occurrence(s) in %s\n\n%s", replaced, path, diff)), nil
}

// buildDiff creates a simple unified-style diff.
func buildDiff(old, new, path string) string {
	oldLines := strings.Split(old, "\n")
	newLines := strings.Split(new, "\n")

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("--- %s (before)\n", path))
	sb.WriteString(fmt.Sprintf("+++ %s (after)\n", path))

	for _, line := range oldLines {
		sb.WriteString(fmt.Sprintf("-%s\n", line))
	}
	for _, line := range newLines {
		sb.WriteString(fmt.Sprintf("+%s\n", line))
	}
	return sb.String()
}
