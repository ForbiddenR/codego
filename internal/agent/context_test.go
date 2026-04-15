package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nice-code/codego/internal/types"
)

// ─── Prompt Builder Tests ───

func TestBuildSystemPrompt(t *testing.T) {
	ctx := &ProjectContext{
		WorkingDir:  "/tmp/myproject",
		IsGitRepo:   true,
		GitBranch:   "main",
		Language:    "Go",
		Framework:   "",
		ProjectName: "myproject",
	}

	prompt := ctx.BuildSystemPrompt()

	if !strings.Contains(prompt, "CodeGo") {
		t.Errorf("should contain CodeGo")
	}
	if !strings.Contains(prompt, "Go") {
		t.Errorf("should contain language Go")
	}
	if !strings.Contains(prompt, "main") {
		t.Errorf("should contain branch main")
	}
	if !strings.Contains(prompt, "myproject") {
		t.Errorf("should contain project name")
	}
}

func TestBuildSystemPrompt_WithClaudeMd(t *testing.T) {
	ctx := &ProjectContext{
		WorkingDir:  "/tmp",
		HasClaudeMd: true,
		ClaudeMd:    "Always use tabs for indentation",
	}

	prompt := ctx.BuildSystemPrompt()
	if !strings.Contains(prompt, "Always use tabs") {
		t.Errorf("should contain CLAUDE.md content")
	}
	if !strings.Contains(prompt, "<project_instructions>") {
		t.Errorf("should wrap in tags")
	}
}

// ─── Context Detection Tests ───

func TestDetectProjectContext_GoProject(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644)

	ctx := DetectProjectContext(dir)
	if ctx.Language != "Go" {
		t.Errorf("language = %q, want Go", ctx.Language)
	}
	if ctx.ProjectName != filepath.Base(dir) {
		t.Errorf("project = %q", ctx.ProjectName)
	}
}

func TestDetectProjectContext_PythonProject(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[project]\nname='test'\ndependencies=['fastapi']\n"), 0o644)

	ctx := DetectProjectContext(dir)
	if ctx.Language != "Python" {
		t.Errorf("language = %q, want Python", ctx.Language)
	}
	if ctx.Framework != "FastAPI" {
		t.Errorf("framework = %q, want FastAPI", ctx.Framework)
	}
}

func TestDetectProjectContext_JSProject(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"dependencies":{"next":"14.0.0"}}`), 0o644)

	ctx := DetectProjectContext(dir)
	if ctx.Language != "JavaScript/Node.js" {
		t.Errorf("language = %q", ctx.Language)
	}
	if ctx.Framework != "Next.js" {
		t.Errorf("framework = %q, want Next.js", ctx.Framework)
	}
}

func TestDetectProjectContext_ClaudeMd(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("Use Go 1.22+ features"), 0o644)

	ctx := DetectProjectContext(dir)
	if !ctx.HasClaudeMd {
		t.Error("should detect CLAUDE.md")
	}
	if ctx.ClaudeMd != "Use Go 1.22+ features" {
		t.Errorf("content = %q", ctx.ClaudeMd)
	}
}

func TestDetectProjectContext_NoProject(t *testing.T) {
	dir := t.TempDir()
	ctx := DetectProjectContext(dir)
	if ctx.Language != "" {
		t.Errorf("language should be empty for empty dir: %q", ctx.Language)
	}
	if ctx.ProjectName != filepath.Base(dir) {
		t.Errorf("project = %q", ctx.ProjectName)
	}
}

// ─── Compression Tests ───

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		text string
		min  int
	}{
		{"", 0},
		{"hello", 1},
		{"hello world this is a test", 6},
		{strings.Repeat("a", 100), 25},
	}
	for _, tt := range tests {
		got := EstimateTokens(tt.text)
		if got < tt.min {
			t.Errorf("EstimateTokens(%q) = %d, want >= %d", tt.text, got, tt.min)
		}
	}
}

func TestEstimateConversationTokens(t *testing.T) {
	msgs := []types.Message{
		types.NewUserMessage("hello world"),
		types.NewAssistantText("hi there"),
	}

	tokens := EstimateConversationTokens(msgs)
	if tokens <= 0 {
		t.Errorf("tokens = %d, want > 0", tokens)
	}
}

func TestEstimateMessageTokens_ToolUse(t *testing.T) {
	msg := types.NewAssistantMessage(types.NewToolUseBlock("id1", "bash", map[string]interface{}{
		"command": "ls -la",
	}))
	tokens := EstimateMessageTokens(msg)
	if tokens < 5 {
		t.Errorf("tool use tokens = %d, too low", tokens)
	}
}

func TestShouldCompress(t *testing.T) {
	cfg := DefaultCompressionConfig()

	// Few messages - should not compress
	shortMsgs := []types.Message{
		types.NewUserMessage("hi"),
		types.NewAssistantText("hello"),
	}
	if ShouldCompress(shortMsgs, 8192, cfg) {
		t.Error("short conversation should not need compression")
	}

	// Many messages with long content
	longMsgs := make([]types.Message, 100)
	for i := range longMsgs {
		longMsgs[i] = types.NewUserMessage(strings.Repeat("word ", 100))
	}
	if !ShouldCompress(longMsgs, 8192, cfg) {
		t.Error("long conversation should need compression")
	}
}

func TestCompressMessages(t *testing.T) {
	cfg := CompressionConfig{
		Threshold:         0.70,
		TargetRatio:       0.30,
		MaxMessagesToKeep: 2,
	}

	msgs := []types.Message{
		types.NewUserMessage("question 1"),
		types.NewAssistantText("answer 1"),
		types.NewUserMessage("question 2"),
		types.NewAssistantText("answer 2"),
		types.NewUserMessage("question 3"),
		types.NewAssistantText("answer 3"),
	}

	compressed := CompressMessages(msgs, cfg)

	// Should keep last 2 + add a summary
	if len(compressed) > 3 {
		t.Errorf("compressed = %d messages, want <= 3", len(compressed))
	}

	// Last messages should be preserved
	last := compressed[len(compressed)-1]
	if last.TextContent() != "answer 3" {
		t.Errorf("last message = %q, want 'answer 3'", last.TextContent())
	}
}

func TestCompressMessages_Short(t *testing.T) {
	cfg := DefaultCompressionConfig()
	msgs := []types.Message{
		types.NewUserMessage("hi"),
		types.NewAssistantText("hello"),
	}

	compressed := CompressMessages(msgs, cfg)
	if len(compressed) != 2 {
		t.Errorf("should not compress short conversation, got %d messages", len(compressed))
	}
}

func TestCompressMessages_PreservesOrder(t *testing.T) {
	cfg := CompressionConfig{MaxMessagesToKeep: 2}

	msgs := []types.Message{
		types.NewUserMessage("first"),
		types.NewAssistantText("second"),
		types.NewUserMessage("third"),
		types.NewAssistantText("fourth"),
	}

	compressed := CompressMessages(msgs, cfg)
	last := compressed[len(compressed)-1]
	if last.TextContent() != "fourth" {
		t.Errorf("last = %q, want 'fourth'", last.TextContent())
	}
}

func TestSummarizeMessages(t *testing.T) {
	msgs := []types.Message{
		types.NewUserMessage("How do I write a Go function"),
		types.NewAssistantText("Here's how"),
		types.NewUserMessage("What about tests"),
		types.NewAssistantText("Use testing package"),
	}

	summary := summarizeMessages(msgs)
	if !strings.Contains(summary, "2 user") {
		t.Errorf("should mention user count: %s", summary)
	}
	if !strings.Contains(summary, "2 assistant") {
		t.Errorf("should mention assistant count: %s", summary)
	}
	if !strings.Contains(summary, "How do I write") {
		t.Errorf("should include topic: %s", summary)
	}
}

func TestDefaultCompressionConfig(t *testing.T) {
	cfg := DefaultCompressionConfig()
	if cfg.Threshold != 0.70 {
		t.Errorf("threshold = %f, want 0.70", cfg.Threshold)
	}
	if cfg.MaxMessagesToKeep != 4 {
		t.Errorf("max keep = %d, want 4", cfg.MaxMessagesToKeep)
	}
}
