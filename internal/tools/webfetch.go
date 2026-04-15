package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nice-code/codego/internal/types"
)

// WebFetchTool fetches content from URLs.
type WebFetchTool struct {
	Timeout time.Duration
}

func NewWebFetchTool() *WebFetchTool {
	return &WebFetchTool{Timeout: 30 * time.Second}
}

func (t *WebFetchTool) Name() string        { return "web_fetch" }
func (t *WebFetchTool) Description() string  { return "Fetch content from a URL. Returns the page text (HTML stripped to readable content)." }
func (t *WebFetchTool) InputSchema() *types.JSONSchema {
	return types.NewObjectSchema(
		"Web fetch input",
		map[string]*types.JSONSchema{
			"url":     types.NewStringSchema("The URL to fetch"),
			"max_chars": types.NewNumberSchema("Max characters to return (default 10000)"),
		},
		"url",
	)
}

func (t *WebFetchTool) Execute(ctx context.Context, input types.ToolInput) (*types.ToolResult, error) {
	url := input.GetString("url")
	if url == "" {
		return types.NewToolError("url is required"), nil
	}

	maxChars := int(input.GetFloat("max_chars"))
	if maxChars <= 0 {
		maxChars = 10000
	}

	ctx, cancel := context.WithTimeout(ctx, t.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return types.NewToolError(fmt.Sprintf("invalid URL: %v", err)), nil
	}
	req.Header.Set("User-Agent", "CodeGo/0.1")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return types.NewToolError(fmt.Sprintf("fetch failed: %v", err)), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return types.NewToolError(fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Status)), nil
	}

	// Read body with limit
	limited := io.LimitReader(resp.Body, int64(maxChars*10)) // read more, we'll strip HTML
	body, err := io.ReadAll(limited)
	if err != nil {
		return types.NewToolError(fmt.Sprintf("read error: %v", err)), nil
	}

	// Simple HTML to text conversion
	text := htmlToText(string(body))
	if len(text) > maxChars {
		text = text[:maxChars] + "\n\n[...truncated]"
	}

	return types.NewToolResult(text), nil
}

// htmlToText strips HTML tags and normalizes whitespace.
func htmlToText(html string) string {
	// Remove script and style blocks
	html = removeBlock(html, "<script", "</script>")
	html = removeBlock(html, "<style", "</style>")

	// Replace common block elements with newlines
	for _, tag := range []string{"<br", "<p", "<div", "<li", "<h1", "<h2", "<h3", "<h4", "<h5", "<h6", "<tr", "<table"} {
		html = strings.ReplaceAll(html, tag, "\n"+tag)
	}

	// Strip tags
	var sb strings.Builder
	inTag := false
	for _, ch := range html {
		if ch == '<' {
			inTag = true
			continue
		}
		if ch == '>' {
			inTag = false
			sb.WriteRune('\n')
			continue
		}
		if !inTag {
			sb.WriteRune(ch)
		}
	}

	// Normalize whitespace
	lines := strings.Split(sb.String(), "\n")
	var result []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n")
}

func removeBlock(s, start, end string) string {
	for {
		i := strings.Index(s, start)
		if i < 0 {
			break
		}
		j := strings.Index(s[i:], end)
		if j < 0 {
			break
		}
		s = s[:i] + s[i+j+len(end):]
	}
	return s
}
