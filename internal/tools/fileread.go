package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/nice-code/codego/internal/types"
)

const maxFileSize = 500 * 1024 // 500KB max for reading

// FileReadTool reads files from the filesystem.
type FileReadTool struct{}

func NewFileReadTool() *FileReadTool { return &FileReadTool{} }

func (t *FileReadTool) Name() string        { return "read" }
func (t *FileReadTool) Description() string  { return "Read a file from the filesystem. Shows content with line numbers." }
func (t *FileReadTool) InputSchema() *types.JSONSchema {
	return types.NewObjectSchema(
		"File read input",
		map[string]*types.JSONSchema{
			"path": types.NewStringSchema("Absolute path to the file to read"),
			"offset": types.NewNumberSchema("Line number to start from (1-indexed, default 1)"),
			"limit": types.NewNumberSchema("Max lines to read (default 2000)"),
		},
		"path",
	)
}

func (t *FileReadTool) Execute(_ context.Context, input types.ToolInput) (*types.ToolResult, error) {
	path := input.GetString("path")
	if path == "" {
		return types.NewToolError("path is required"), nil
	}

	// Check file exists and size
	info, err := os.Stat(path)
	if err != nil {
		return types.NewToolError(fmt.Sprintf("cannot read file: %v", err)), nil
	}
	if info.IsDir() {
		return types.NewToolError(fmt.Sprintf("%s is a directory, not a file", path)), nil
	}
	if info.Size() > int64(maxFileSize) {
		return types.NewToolError(fmt.Sprintf("file too large: %d bytes (max %d)", info.Size(), maxFileSize)), nil
	}

	offset := int(input.GetFloat("offset"))
	if offset < 1 {
		offset = 1
	}
	limit := int(input.GetFloat("limit"))
	if limit <= 0 {
		limit = 2000
	}

	file, err := os.Open(path)
	if err != nil {
		return types.NewToolError(fmt.Sprintf("cannot open file: %v", err)), nil
	}
	defer file.Close()

	// Check for binary content
	header := make([]byte, 512)
	n, _ := file.Read(header)
	file.Seek(0, 0)
	if isBinary(header[:n]) {
		return types.NewToolError("file appears to be binary, cannot read as text"), nil
	}

	var lines []string
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if lineNum < offset {
			continue
		}
		if len(lines) >= limit {
			lines = append(lines, fmt.Sprintf("... (%d more lines)", countRemaining(scanner)))
			break
		}
		lines = append(lines, fmt.Sprintf("%5d\t%s", lineNum, scanner.Text()))
	}

	if err := scanner.Err(); err != nil {
		return types.NewToolError(fmt.Sprintf("error reading file: %v", err)), nil
	}

	if len(lines) == 0 {
		return types.NewToolResult("(empty file)"), nil
	}

	return types.NewToolResult(strings.Join(lines, "\n")), nil
}

// isBinary checks if data contains null bytes (simple binary detection).
func isBinary(data []byte) bool {
	for _, b := range data {
		if b == 0 {
			return true
		}
	}
	return false
}

// countRemaining counts remaining lines in the scanner.
func countRemaining(scanner *bufio.Scanner) int {
	count := 0
	for scanner.Scan() {
		count++
	}
	return count
}
