package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nice-code/codego/internal/types"
)

// WebSearchTool searches the web.
type WebSearchTool struct {
	APIKey  string
	Engine  string // "brave" or "duckduckgo" (fallback)
	Timeout time.Duration
}

func NewWebSearchTool(apiKey string) *WebSearchTool {
	engine := "brave"
	if apiKey == "" {
		engine = "duckduckgo"
	}
	return &WebSearchTool{
		APIKey:  apiKey,
		Engine:  engine,
		Timeout: 15 * time.Second,
	}
}

func (t *WebSearchTool) Name() string { return "web_search" }
func (t *WebSearchTool) Description() string {
	return "Search the web for information. Returns search results with titles, URLs, and snippets."
}
func (t *WebSearchTool) InputSchema() *types.JSONSchema {
	return types.NewObjectSchema(
		"Web search input",
		map[string]*types.JSONSchema{
			"query":  types.NewStringSchema("The search query"),
			"count":  types.NewNumberSchema("Number of results (default 5, max 10)"),
		},
		"query",
	)
}

func (t *WebSearchTool) Execute(ctx context.Context, input types.ToolInput) (*types.ToolResult, error) {
	query := input.GetString("query")
	if query == "" {
		return types.NewToolError("query is required"), nil
	}

	count := int(input.GetFloat("count"))
	if count <= 0 {
		count = 5
	}
	if count > 10 {
		count = 10
	}

	ctx, cancel := context.WithTimeout(ctx, t.Timeout)
	defer cancel()

	switch t.Engine {
	case "brave":
		return t.searchBrave(ctx, query, count)
	default:
		return t.searchDuckDuckGo(ctx, query, count)
	}
}

func (t *WebSearchTool) searchBrave(ctx context.Context, query string, count int) (*types.ToolResult, error) {
	endpoint := fmt.Sprintf("https://api.search.brave.com/res/v1/web/search?q=%s&count=%d",
		url.QueryEscape(query), count)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return types.NewToolError(fmt.Sprintf("request error: %v", err)), nil
	}
	req.Header.Set("X-Subscription-Token", t.APIKey)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return types.NewToolError(fmt.Sprintf("search failed: %v", err)), nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return types.NewToolError(fmt.Sprintf("Brave API error %d: %s", resp.StatusCode, string(body))), nil
	}

	var result struct {
		Web struct {
			Results []struct {
				Title   string `json:"title"`
				URL     string `json:"url"`
				Snippet string `json:"description"`
			} `json:"results"`
		} `json:"web"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return types.NewToolError(fmt.Sprintf("parse error: %v", err)), nil
	}

		var results []searchResult
		for _, r := range result.Web.Results {
			results = append(results, searchResult{
				Title:   r.Title,
				URL:     r.URL,
				Snippet: r.Snippet,
			})
		}
		return formatSearchResults(query, results)
}

func (t *WebSearchTool) searchDuckDuckGo(ctx context.Context, query string, count int) (*types.ToolResult, error) {
	// Use DuckDuckGo instant answer API (limited but no key needed)
	endpoint := fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json&no_redirect=1",
		url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return types.NewToolError(fmt.Sprintf("request error: %v", err)), nil
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return types.NewToolError(fmt.Sprintf("search failed: %v", err)), nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Abstract   string `json:"Abstract"`
		AbstractURL string `json:"AbstractURL"`
		Heading    string `json:"Heading"`
		RelatedTopics []struct {
			Text string `json:"Text"`
			URL  string `json:"FirstURL"`
		} `json:"RelatedTopics"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return types.NewToolError(fmt.Sprintf("parse error: %v", err)), nil
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("Search results for: %s\n\n", query))

	if result.Abstract != "" {
		output.WriteString(fmt.Sprintf("Abstract: %s\n", result.Abstract))
		if result.AbstractURL != "" {
			output.WriteString(fmt.Sprintf("Source: %s\n", result.AbstractURL))
		}
		output.WriteString("\n")
	}

	shown := 0
	for _, topic := range result.RelatedTopics {
		if shown >= count {
			break
		}
		if topic.Text != "" && topic.URL != "" {
			output.WriteString(fmt.Sprintf("- %s\n  %s\n\n", topic.Text, topic.URL))
			shown++
		}
	}

	if output.Len() == 0 {
		return types.NewToolResult("No results found. Try setting BRAVE_API_KEY for better search results."), nil
	}

	return types.NewToolResult(output.String()), nil
}

type searchResult struct {
	Title   string
	URL     string
	Snippet string
}

func formatSearchResults(query string, results []searchResult) (*types.ToolResult, error) {
	if len(results) == 0 {
		return types.NewToolResult("No results found."), nil
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("Search results for: %s\n\n", query))

	for i, r := range results {
		output.WriteString(fmt.Sprintf("%d. %s\n", i+1, r.Title))
		output.WriteString(fmt.Sprintf("   %s\n", r.URL))
		if r.Snippet != "" {
			output.WriteString(fmt.Sprintf("   %s\n", r.Snippet))
		}
		output.WriteString("\n")
	}

	return types.NewToolResult(output.String()), nil
}
