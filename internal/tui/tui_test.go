package tui

import (
	"strings"
	"testing"

	"github.com/nice-code/codego/internal/types"
)

func TestAppState_String(t *testing.T) {
	tests := []struct {
		state AppState
		want  string
	}{
		{StateInput, "input"},
		{StateThinking, "thinking"},
		{StateToolRunning, "tool_running"},
		{StateResponse, "response"},
		{StateError, "error"},
	}
	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("AppState(%d).String() = %q, want %q", tt.state, got, tt.want)
		}
	}
}

func TestNewUserMessageView(t *testing.T) {
	v := NewUserMessageView("hello")
	if v.Kind != MsgUser {
		t.Errorf("kind = %d, want MsgUser", v.Kind)
	}
	if v.Content != "hello" {
		t.Errorf("content = %q, want %q", v.Content, "hello")
	}
}

func TestNewAssistantMessageView(t *testing.T) {
	v := NewAssistantMessageView()
	if v.Kind != MsgAssistant {
		t.Errorf("kind = %d, want MsgAssistant", v.Kind)
	}
	if v.Content != "" {
		t.Errorf("content = %q, want empty", v.Content)
	}
}

func TestNewToolCallView(t *testing.T) {
	input := map[string]interface{}{"command": "ls"}
	v := NewToolCallView("bash", input)
	if v.Kind != MsgToolCall {
		t.Errorf("kind = %d, want MsgToolCall", v.Kind)
	}
	if v.ToolName != "bash" {
		t.Errorf("tool = %q, want %q", v.ToolName, "bash")
	}
}

func TestNewSystemMessageView(t *testing.T) {
	v := NewSystemMessageView("context compressed")
	if v.Kind != MsgSystem {
		t.Errorf("kind = %d, want MsgSystem", v.Kind)
	}
	if v.Content != "context compressed" {
		t.Errorf("content = %q", v.Content)
	}
}

func TestMessageView_AppendText(t *testing.T) {
	v := NewAssistantMessageView()
	v.AppendText("Hello ")
	v.AppendText("world")

	if v.Content != "Hello world" {
		t.Errorf("content = %q, want %q", v.Content, "Hello world")
	}
}

func TestMessageView_SetResult(t *testing.T) {
	v := NewToolCallView("bash", nil)
	if v.ToolResult != nil {
		t.Error("should start nil")
	}

	result := types.NewToolResult("output")
	v.SetResult(result)

	if v.ToolResult == nil {
		t.Fatal("should not be nil after SetResult")
	}
	if v.ToolResult.Output != "output" {
		t.Errorf("output = %q", v.ToolResult.Output)
	}
}

func TestMessageView_Render_User(t *testing.T) {
	v := NewUserMessageView("explain main.go")
	rendered := v.Render(80)

	if !strings.Contains(rendered, "You") {
		t.Errorf("should contain 'You': %s", rendered)
	}
	if !strings.Contains(rendered, "explain main.go") {
		t.Errorf("should contain text: %s", rendered)
	}
}

func TestMessageView_Render_Assistant_Empty(t *testing.T) {
	v := NewAssistantMessageView()
	rendered := v.Render(80)

	if !strings.Contains(rendered, "thinking") {
		t.Errorf("empty assistant should show thinking: %s", rendered)
	}
}

func TestMessageView_Render_Assistant_WithText(t *testing.T) {
	v := NewAssistantMessageView()
	v.AppendText("Here's the answer.")
	rendered := v.Render(80)

	if !strings.Contains(rendered, "Here's the answer") {
		t.Errorf("should contain text: %s", rendered)
	}
}

func TestMessageView_Render_ToolCall_Pending(t *testing.T) {
	v := NewToolCallView("bash", map[string]interface{}{"command": "ls"})
	rendered := v.Render(80)

	if !strings.Contains(rendered, "bash") {
		t.Errorf("should contain tool name: %s", rendered)
	}
	if !strings.Contains(rendered, "⠋") {
		t.Errorf("pending should show spinner: %s", rendered)
	}
}

func TestMessageView_Render_ToolCall_Success(t *testing.T) {
	v := NewToolCallView("bash", nil)
	v.SetResult(types.NewToolResult("file1.txt\nfile2.txt"))
	rendered := v.Render(80)

	if !strings.Contains(rendered, "bash") {
		t.Errorf("should contain tool name: %s", rendered)
	}
	if !strings.Contains(rendered, "✓") {
		t.Errorf("success should show checkmark: %s", rendered)
	}
	if !strings.Contains(rendered, "file1.txt") {
		t.Errorf("should contain output: %s", rendered)
	}
}

func TestMessageView_Render_ToolCall_Error(t *testing.T) {
	v := NewToolCallView("bash", nil)
	v.SetResult(types.NewToolError("command not found"))
	rendered := v.Render(80)

	if !strings.Contains(rendered, "✗") {
		t.Errorf("error should show x mark: %s", rendered)
	}
	if !strings.Contains(rendered, "command not found") {
		t.Errorf("should contain error: %s", rendered)
	}
}

func TestMessageView_Render_System(t *testing.T) {
	v := NewSystemMessageView("Context compressed: 10 → 4 messages")
	rendered := v.Render(80)

	if !strings.Contains(rendered, "compressed") {
		t.Errorf("should contain text: %s", rendered)
	}
}

func TestMessageView_Render_ToolCall_LongOutput(t *testing.T) {
	v := NewToolCallView("bash", nil)
	longOutput := strings.Repeat("x", 1000)
	v.SetResult(types.NewToolResult(longOutput))
	rendered := v.Render(80)

	if !strings.Contains(rendered, "...") {
		t.Errorf("long output should be truncated: len=%d", len(rendered))
	}
}

func TestAppModel_NewAppModel(t *testing.T) {
	m := NewAppModel(nil)

	if m.state != StateInput {
		t.Errorf("state = %d, want StateInput", m.state)
	}
	if len(m.messages) != 0 {
		t.Errorf("messages = %d, want 0", len(m.messages))
	}
}

func TestAppModel_View_Initialized(t *testing.T) {
	m := NewAppModel(nil)
	m.width = 80
	m.height = 24

	view := m.View()
	if !strings.Contains(view, "CodeGo") {
		t.Errorf("should contain CodeGo: %s", view[:100])
	}
}

func TestAppModel_View_NotInitialized(t *testing.T) {
	m := NewAppModel(nil)
	// width is 0
	view := m.View()
	if view != "Initializing..." {
		t.Errorf("should show initializing: %q", view)
	}
}

func TestAppModel_renderStatusBar(t *testing.T) {
	m := NewAppModel(nil)
	m.width = 80
	m.messages = []MessageView{NewUserMessageView("hi"), NewAssistantMessageView()}

	bar := m.renderStatusBar()
	if !strings.Contains(bar, "CodeGo") {
		t.Errorf("should contain CodeGo: %s", bar)
	}
	if !strings.Contains(bar, "2 msgs") {
		t.Errorf("should contain msg count: %s", bar)
	}
}

func TestAppModel_renderSpinner_Thinking(t *testing.T) {
	m := NewAppModel(nil)
	m.state = StateThinking

	spinner := m.renderSpinner()
	if !strings.Contains(spinner, "Thinking") {
		t.Errorf("should show Thinking: %s", spinner)
	}
}

func TestAppModel_renderSpinner_ToolRunning(t *testing.T) {
	m := NewAppModel(nil)
	m.state = StateToolRunning
	m.activeTool = "bash"

	spinner := m.renderSpinner()
	if !strings.Contains(spinner, "bash") {
		t.Errorf("should show tool name: %s", spinner)
	}
}
