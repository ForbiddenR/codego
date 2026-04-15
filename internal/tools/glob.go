package tools

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/nice-code/codego/internal/types"
)

// GlobTool finds files matching glob patterns.
type GlobTool struct {
	WorkingDir string
}

func NewGlobTool(workingDir string) *GlobTool {
	return &GlobTool{WorkingDir: workingDir}
}

func (t *GlobTool) Name() string        { return "glob" }
func (t *GlobTool) Description() string  { return "Find files matching a glob pattern. Supports ** for recursive matching. Returns sorted list of matching file paths." }
func (t *GlobTool) InputSchema() *types.JSONSchema {
	return types.NewObjectSchema(
		"Glob input",
		map[string]*types.JSONSchema{
			"pattern": types.NewStringSchema("Glob pattern (e.g. '**/*.go', 'src/**/*.ts')"),
			"path":    types.NewStringSchema("Optional directory to search from (default: working directory)"),
		},
		"pattern",
	)
}

func (t *GlobTool) Execute(_ context.Context, input types.ToolInput) (*types.ToolResult, error) {
	pattern := input.GetString("pattern")
	if pattern == "" {
		return types.NewToolError("pattern is required"), nil
	}

	searchDir := input.GetString("path")
	if searchDir == "" {
		searchDir = t.WorkingDir
	}

	// Make pattern relative to search dir
	fullPattern := filepath.Join(searchDir, pattern)

	matches, err := doublestar.FilepathGlob(fullPattern)
	if err != nil {
		return types.NewToolError(fmt.Sprintf("invalid pattern: %v", err)), nil
	}

	// Sort results
	sort.Strings(matches)

	// Make paths relative to search dir for cleaner output
	var result []string
	for _, m := range matches {
		rel, err := filepath.Rel(searchDir, m)
		if err != nil {
			rel = m
		}
		result = append(result, rel)
	}

	if len(result) == 0 {
		return types.NewToolResult("No files found matching pattern"), nil
	}

	output := fmt.Sprintf("Found %d file(s):\n%s", len(result), strings.Join(result, "\n"))
	return types.NewToolResult(output), nil
}
