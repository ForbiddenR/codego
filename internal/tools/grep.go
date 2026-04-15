package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/nice-code/codego/internal/types"
)

// GrepTool searches file contents with regex.
type GrepTool struct {
	WorkingDir string
}

func NewGrepTool(workingDir string) *GrepTool {
	return &GrepTool{WorkingDir: workingDir}
}

func (t *GrepTool) Name() string        { return "grep" }
func (t *GrepTool) Description() string  { return "Search file contents using a regex pattern. Returns matching lines with file paths and line numbers." }
func (t *GrepTool) InputSchema() *types.JSONSchema {
	return types.NewObjectSchema(
		"Grep input",
		map[string]*types.JSONSchema{
			"pattern":     types.NewStringSchema("Regex pattern to search for"),
			"path":        types.NewStringSchema("Directory or file to search in (default: working directory)"),
			"include":     types.NewStringSchema("File glob to include (e.g. '*.go')"),
			"context":     types.NewNumberSchema("Number of context lines before/after match (default 0)"),
			"max_results": types.NewNumberSchema("Max results to return (default 100)"),
		},
		"pattern",
	)
}

func (t *GrepTool) Execute(_ context.Context, input types.ToolInput) (*types.ToolResult, error) {
	pattern := input.GetString("pattern")
	if pattern == "" {
		return types.NewToolError("pattern is required"), nil
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return types.NewToolError(fmt.Sprintf("invalid regex: %v", err)), nil
	}

	searchPath := input.GetString("path")
	if searchPath == "" {
		searchPath = t.WorkingDir
	}

	includeGlob := input.GetString("include")
	contextLines := int(input.GetFloat("context"))
	maxResults := int(input.GetFloat("max_results"))
	if maxResults <= 0 {
		maxResults = 100
	}

	var results []string
	err = filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// Skip large files
		if info.Size() > 10*1024*1024 {
			return nil
		}

		// Skip binary-like extensions
		if isBinaryExt(path) {
			return nil
		}

		// Filter by glob
		if includeGlob != "" {
			matched, _ := filepath.Match(includeGlob, filepath.Base(path))
			if !matched {
				return nil
			}
		}

		if len(results) >= maxResults {
			return filepath.SkipDir
		}

		matches := grepFile(path, re, contextLines, searchPath)
		results = append(results, matches...)

		if len(results) >= maxResults {
			results = results[:maxResults]
			return filepath.SkipDir
		}

		return nil
	})

	if err != nil && len(results) == 0 {
		return types.NewToolError(fmt.Sprintf("search error: %v", err)), nil
	}

	if len(results) == 0 {
		return types.NewToolResult("No matches found"), nil
	}

	output := fmt.Sprintf("Found %d match(es):\n\n%s", len(results), strings.Join(results, "\n"))
	return types.NewToolResult(output), nil
}

func grepFile(path string, re *regexp.Regexp, context int, baseDir string) []string {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	// Check for binary
	header := make([]byte, 512)
	n, _ := file.Read(header)
	file.Seek(0, 0)
	if isBinary(header[:n]) {
		return nil
	}

	relPath, _ := filepath.Rel(baseDir, path)

	var lines []string
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	var allLines []string
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}

	matched := make(map[int]bool)
	for i, line := range allLines {
		if re.MatchString(line) {
			// Mark match and context
			for j := max(0, i-context); j <= min(len(allLines)-1, i+context); j++ {
				matched[j] = true
			}
		}
	}

	for i := 0; i < len(allLines); i++ {
		if matched[i] {
			lines = append(lines, fmt.Sprintf("%s:%d: %s", relPath, i+1, allLines[i]))
		}
	}

	return lines
}

func isBinaryExt(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	binaryExts := map[string]bool{
		".exe": true, ".dll": true, ".so": true, ".dylib": true,
		".bin": true, ".o": true, ".a": true, ".pyc": true,
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
		".zip": true, ".tar": true, ".gz": true, ".bz2": true,
		".pdf": true, ".doc": true, ".docx": true, ".xlsx": true,
		".woff": true, ".woff2": true, ".ttf": true, ".eot": true,
		".ico": true, ".svg": true,
	}
	return binaryExts[ext]
}
