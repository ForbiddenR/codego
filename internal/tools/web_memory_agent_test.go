package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nice-code/codego/internal/types"
)

// ─── WebFetchTool Tests ───

func TestWebFetchTool_InvalidURL(t *testing.T) {
	tool := NewWebFetchTool()
	result, _ := tool.Execute(context.Background(), types.ToolInput{
		"url": "not-a-valid-url",
	})
	if !result.IsError {
		t.Error("should be error for invalid URL")
	}
}

func TestWebFetchTool_MissingURL(t *testing.T) {
	tool := NewWebFetchTool()
	result, _ := tool.Execute(context.Background(), types.ToolInput{})
	if !result.IsError {
		t.Error("missing url should be error")
	}
}

func TestWebFetchTool_Schema(t *testing.T) {
	tool := NewWebFetchTool()
	if tool.Name() != "web_fetch" {
		t.Errorf("name = %q", tool.Name())
	}
	schema := tool.InputSchema()
	if schema.Type != "object" {
		t.Errorf("schema type = %q", schema.Type)
	}
}

func TestHTMLToText(t *testing.T) {
	html := `<html><body><h1>Title</h1><p>Hello <b>world</b></p><script>ignore</script></body></html>`
	text := htmlToText(html)

	if !strings.Contains(text, "Title") {
		t.Errorf("should contain Title: %s", text)
	}
	if !strings.Contains(text, "Hello") {
		t.Errorf("should contain Hello: %s", text)
	}
	if !strings.Contains(text, "world") {
		t.Errorf("should contain world: %s", text)
	}
	if strings.Contains(text, "ignore") {
		t.Errorf("should not contain script content: %s", text)
	}
	if strings.Contains(text, "<b>") {
		t.Errorf("should not contain tags: %s", text)
	}
}

// ─── WebSearchTool Tests ───

func TestWebSearchTool_MissingQuery(t *testing.T) {
	tool := NewWebSearchTool("")
	result, _ := tool.Execute(context.Background(), types.ToolInput{})
	if !result.IsError {
		t.Error("missing query should be error")
	}
}

func TestWebSearchTool_Schema(t *testing.T) {
	tool := NewWebSearchTool("key")
	if tool.Name() != "web_search" {
		t.Errorf("name = %q", tool.Name())
	}
}

// ─── Memory Tests ───

func TestMemory_New(t *testing.T) {
	dir := t.TempDir()
	m := NewMemoryWithPath(filepath.Join(dir, "memory.md"))

	content, err := m.Load()
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if content != "" {
		t.Errorf("empty memory should return empty string")
	}
}

func TestMemory_Append(t *testing.T) {
	dir := t.TempDir()
	m := NewMemoryWithPath(filepath.Join(dir, "memory.md"))

	m.Append("User prefers Go over Python")
	m.Append("Project uses SQLite for storage")

	content, _ := m.Load()
	if !strings.Contains(content, "Go over Python") {
		t.Errorf("should contain first memory: %s", content)
	}
	if !strings.Contains(content, "SQLite") {
		t.Errorf("should contain second memory: %s", content)
	}
	if !strings.Contains(content, "# CodeGo Memory") {
		t.Errorf("should have header: %s", content)
	}
}

func TestMemory_Clear(t *testing.T) {
	dir := t.TempDir()
	m := NewMemoryWithPath(filepath.Join(dir, "memory.md"))

	m.Append("something")
	m.Clear()

	content, _ := m.Load()
	if content != "" {
		t.Errorf("should be empty after clear: %q", content)
	}
}

func TestMemory_Exists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "memory.md")
	m := NewMemoryWithPath(path)

	if m.Exists() {
		t.Error("should not exist initially")
	}

	m.Save("# Memory")
	if !m.Exists() {
		t.Error("should exist after save")
	}
}

func TestMemoryTool_Add(t *testing.T) {
	dir := t.TempDir()
	tool := &MemoryTool{memory: NewMemoryWithPath(filepath.Join(dir, "memory.md"))}

	result, _ := tool.Execute(context.Background(), types.ToolInput{
		"action": "add",
		"text":   "User likes short commands",
	})

	if result.IsError {
		t.Fatalf("error: %s", result.Output)
	}
	if !strings.Contains(result.Output, "Remembered") {
		t.Errorf("should confirm: %s", result.Output)
	}
}

func TestMemoryTool_Show(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "memory.md")
	tool := &MemoryTool{memory: NewMemoryWithPath(path)}

	tool.Execute(context.Background(), types.ToolInput{"action": "add", "text": "fact 1"})
	tool.Execute(context.Background(), types.ToolInput{"action": "add", "text": "fact 2"})

	result, _ := tool.Execute(context.Background(), types.ToolInput{"action": "show"})
	if !strings.Contains(result.Output, "fact 1") {
		t.Errorf("should show fact 1: %s", result.Output)
	}
	if !strings.Contains(result.Output, "fact 2") {
		t.Errorf("should show fact 2: %s", result.Output)
	}
}

func TestMemoryTool_Show_Empty(t *testing.T) {
	dir := t.TempDir()
	tool := &MemoryTool{memory: NewMemoryWithPath(filepath.Join(dir, "memory.md"))}

	result, _ := tool.Execute(context.Background(), types.ToolInput{"action": "show"})
	if !strings.Contains(result.Output, "No memories") {
		t.Errorf("empty: %s", result.Output)
	}
}

func TestMemoryTool_Clear(t *testing.T) {
	dir := t.TempDir()
	tool := &MemoryTool{memory: NewMemoryWithPath(filepath.Join(dir, "memory.md"))}

	tool.Execute(context.Background(), types.ToolInput{"action": "add", "text": "test"})
	result, _ := tool.Execute(context.Background(), types.ToolInput{"action": "clear"})

	if !strings.Contains(result.Output, "cleared") {
		t.Errorf("should confirm clear: %s", result.Output)
	}
}

func TestMemoryTool_Unknown(t *testing.T) {
	tool := NewMemoryTool()
	result, _ := tool.Execute(context.Background(), types.ToolInput{"action": "bad"})
	if !result.IsError {
		t.Error("unknown action should be error")
	}
}

func TestMemoryTool_EmptyText(t *testing.T) {
	tool := NewMemoryTool()
	result, _ := tool.Execute(context.Background(), types.ToolInput{"action": "add"})
	if !result.IsError {
		t.Error("add without text should be error")
	}
}

func TestLoadMemoryForPrompt(t *testing.T) {
	// With no memory file, should return empty
	prompt := LoadMemoryForPrompt()
	_ = prompt // just verify it doesn't panic

	// Create a temp memory
	dir := t.TempDir()
	os.MkdirAll(dir, 0o755)
	os.Setenv("HOME", dir)
	_ = os.MkdirAll(filepath.Join(dir, ".codego"), 0o755)
	os.WriteFile(filepath.Join(dir, ".codego", "memory.md"), []byte("# Memory\n- user likes Go"), 0o644)
}

// ─── AgentTool Tests ───

type mockAgentExecutor struct {
	result *types.RunResult
	err    error
}

func (m *mockAgentExecutor) Run(_ context.Context, _ string) (*types.RunResult, error) {
	return m.result, m.err
}

func TestAgentTool_MissingPrompt(t *testing.T) {
	tool := NewAgentTool(nil)
	result, _ := tool.Execute(context.Background(), types.ToolInput{})
	if !result.IsError {
		t.Error("missing prompt should be error")
	}
}

func TestAgentTool_NoFactory(t *testing.T) {
	tool := NewAgentTool(nil)
	result, _ := tool.Execute(context.Background(), types.ToolInput{"prompt": "test"})
	if !result.IsError {
		t.Error("nil factory should be error")
	}
}

func TestAgentTool_Success(t *testing.T) {
	mock := &mockAgentExecutor{
		result: types.NewRunResult("Done! Created 3 files.", types.Usage{InputTokens: 100, OutputTokens: 50}),
	}
	tool := NewAgentTool(func() AgentExecutor { return mock })

	result, _ := tool.Execute(context.Background(), types.ToolInput{
		"prompt": "create a REST API",
	})

	if result.IsError {
		t.Fatalf("should succeed: %s", result.Output)
	}
	if !strings.Contains(result.Output, "Done!") {
		t.Errorf("should contain result: %s", result.Output)
	}
	if !strings.Contains(result.Output, "100 in") {
		t.Errorf("should contain usage: %s", result.Output)
	}
}

func TestAgentTool_Error(t *testing.T) {
	mock := &mockAgentExecutor{err: fmt.Errorf("timeout")}
	tool := NewAgentTool(func() AgentExecutor { return mock })

	result, _ := tool.Execute(context.Background(), types.ToolInput{"prompt": "test"})
	if !result.IsError {
		t.Error("agent error should be tool error")
	}
}

func TestAgentTool_Schema(t *testing.T) {
	tool := NewAgentTool(nil)
	if tool.Name() != "agent" {
		t.Errorf("name = %q", tool.Name())
	}
}
