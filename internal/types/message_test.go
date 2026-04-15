package types

import "testing"

func TestRole_IsValid(t *testing.T) {
	tests := []struct {
		role Role
		want bool
	}{
		{RoleUser, true},
		{RoleAssistant, true},
		{RoleSystem, true},
		{Role("invalid"), false},
		{Role(""), false},
	}
	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			if got := tt.role.IsValid(); got != tt.want {
				t.Errorf("Role(%q).IsValid() = %v, want %v", tt.role, got, tt.want)
			}
		})
	}
}

func TestRole_String(t *testing.T) {
	if RoleUser.String() != "user" {
		t.Errorf("RoleUser.String() = %q, want %q", RoleUser.String(), "user")
	}
	if RoleAssistant.String() != "assistant" {
		t.Errorf("RoleAssistant.String() = %q, want %q", RoleAssistant.String(), "assistant")
	}
}

func TestContentBlock_IsText(t *testing.T) {
	text := ContentBlock{Type: ContentTypeText, Text: "hello"}
	tool := ContentBlock{Type: ContentTypeToolUse, Name: "bash"}

	if !text.IsText() {
		t.Error("text block should be text")
	}
	if tool.IsText() {
		t.Error("tool_use block should not be text")
	}
}

func TestContentBlock_IsToolUse(t *testing.T) {
	tool := ContentBlock{Type: ContentTypeToolUse, Name: "bash"}
	text := ContentBlock{Type: ContentTypeText, Text: "hello"}

	if !tool.IsToolUse() {
		t.Error("tool block should be tool_use")
	}
	if text.IsToolUse() {
		t.Error("text block should not be tool_use")
	}
}

func TestContentBlock_IsToolResult(t *testing.T) {
	result := ContentBlock{Type: ContentTypeToolResult, ToolUseID: "123"}
	if !result.IsToolResult() {
		t.Error("should be tool_result")
	}
}

func TestContentBlock_IsThinking(t *testing.T) {
	thinking := ContentBlock{Type: ContentTypeThinking, Text: "thinking..."}
	if !thinking.IsThinking() {
		t.Error("should be thinking")
	}
}

func TestNewUserMessage(t *testing.T) {
	msg := NewUserMessage("hello")
	if msg.Role != RoleUser {
		t.Errorf("role = %q, want %q", msg.Role, RoleUser)
	}
	if len(msg.Content) != 1 {
		t.Fatalf("content blocks = %d, want 1", len(msg.Content))
	}
	if msg.Content[0].Type != ContentTypeText {
		t.Errorf("content type = %q, want %q", msg.Content[0].Type, ContentTypeText)
	}
	if msg.Content[0].Text != "hello" {
		t.Errorf("text = %q, want %q", msg.Content[0].Text, "hello")
	}
}

func TestNewAssistantMessage(t *testing.T) {
	msg := NewAssistantMessage(
		NewTextBlock("Here is the answer"),
		NewToolUseBlock("id1", "bash", map[string]interface{}{"command": "ls"}),
	)
	if msg.Role != RoleAssistant {
		t.Errorf("role = %q, want %q", msg.Role, RoleAssistant)
	}
	if len(msg.Content) != 2 {
		t.Fatalf("content blocks = %d, want 2", len(msg.Content))
	}
}

func TestNewAssistantText(t *testing.T) {
	msg := NewAssistantText("response")
	if msg.Role != RoleAssistant {
		t.Errorf("role = %q, want %q", msg.Role, RoleAssistant)
	}
	if msg.TextContent() != "response" {
		t.Errorf("text = %q, want %q", msg.TextContent(), "response")
	}
}

func TestNewSystemMessage(t *testing.T) {
	msg := NewSystemMessage("system prompt")
	if msg.Role != RoleSystem {
		t.Errorf("role = %q, want %q", msg.Role, RoleSystem)
	}
	if msg.TextContent() != "system prompt" {
		t.Errorf("text = %q, want %q", msg.TextContent(), "system prompt")
	}
}

func TestNewToolResultMessage(t *testing.T) {
	msg := NewToolResultMessage("tool-123", "output", false)
	if msg.Role != RoleUser {
		t.Errorf("role = %q, want %q", msg.Role, RoleUser)
	}
	if len(msg.Content) != 1 {
		t.Fatalf("content blocks = %d, want 1", len(msg.Content))
	}
	block := msg.Content[0]
	if block.Type != ContentTypeToolResult {
		t.Errorf("type = %q, want %q", block.Type, ContentTypeToolResult)
	}
	if block.ToolUseID != "tool-123" {
		t.Errorf("tool_use_id = %q, want %q", block.ToolUseID, "tool-123")
	}
	if block.Content != "output" {
		t.Errorf("content = %q, want %q", block.Content, "output")
	}
	if block.IsError {
		t.Error("is_error should be false")
	}
}

func TestNewToolResultMessage_Error(t *testing.T) {
	msg := NewToolResultMessage("tool-456", "error occurred", true)
	if !msg.Content[0].IsError {
		t.Error("is_error should be true")
	}
}

func TestNewToolUseBlock(t *testing.T) {
	input := map[string]interface{}{"command": "echo hi"}
	block := NewToolUseBlock("id1", "bash", input)

	if block.Type != ContentTypeToolUse {
		t.Errorf("type = %q, want %q", block.Type, ContentTypeToolUse)
	}
	if block.ID != "id1" {
		t.Errorf("id = %q, want %q", block.ID, "id1")
	}
	if block.Name != "bash" {
		t.Errorf("name = %q, want %q", block.Name, "bash")
	}
	if block.Input["command"] != "echo hi" {
		t.Errorf("input = %v, want command=echo hi", block.Input)
	}
}

func TestNewTextBlock(t *testing.T) {
	block := NewTextBlock("hello world")
	if block.Type != ContentTypeText {
		t.Errorf("type = %q, want %q", block.Type, ContentTypeText)
	}
	if block.Text != "hello world" {
		t.Errorf("text = %q, want %q", block.Text, "hello world")
	}
}

func TestNewThinkingBlock(t *testing.T) {
	block := NewThinkingBlock("thinking...")
	if block.Type != ContentTypeThinking {
		t.Errorf("type = %q, want %q", block.Type, ContentTypeThinking)
	}
	if block.Text != "thinking..." {
		t.Errorf("text = %q, want %q", block.Text, "thinking...")
	}
}

func TestMessage_HasToolCalls(t *testing.T) {
	textOnly := NewAssistantText("hello")
	if textOnly.HasToolCalls() {
		t.Error("text-only message should not have tool calls")
	}

	withTools := NewAssistantMessage(
		NewTextBlock("let me check"),
		NewToolUseBlock("id1", "bash", nil),
	)
	if !withTools.HasToolCalls() {
		t.Error("message with tool_use should have tool calls")
	}
}

func TestMessage_ToolCalls(t *testing.T) {
	msg := NewAssistantMessage(
		NewTextBlock("text"),
		NewToolUseBlock("id1", "bash", nil),
		NewToolUseBlock("id2", "read", nil),
	)

	calls := msg.ToolCalls()
	if len(calls) != 2 {
		t.Fatalf("tool calls = %d, want 2", len(calls))
	}
	if calls[0].Name != "bash" {
		t.Errorf("first tool = %q, want %q", calls[0].Name, "bash")
	}
	if calls[1].Name != "read" {
		t.Errorf("second tool = %q, want %q", calls[1].Name, "read")
	}
}

func TestMessage_ToolCalls_Empty(t *testing.T) {
	msg := NewAssistantText("no tools here")
	calls := msg.ToolCalls()
	if calls != nil {
		t.Errorf("expected nil, got %v", calls)
	}
}

func TestMessage_TextContent(t *testing.T) {
	msg := NewAssistantMessage(
		NewTextBlock("hello "),
		NewTextBlock("world"),
		NewToolUseBlock("id1", "bash", nil),
	)
	if msg.TextContent() != "hello world" {
		t.Errorf("text = %q, want %q", msg.TextContent(), "hello world")
	}
}

func TestMessage_TextContent_Empty(t *testing.T) {
	msg := NewAssistantMessage(NewToolUseBlock("id1", "bash", nil))
	if msg.TextContent() != "" {
		t.Errorf("text = %q, want empty", msg.TextContent())
	}
}

func TestMessage_AllText(t *testing.T) {
	msg := NewAssistantMessage(
		NewTextBlock("answer: 42"),
		NewThinkingBlock("let me think..."),
		NewToolUseBlock("id1", "bash", nil),
	)
	all := msg.AllText()
	if all != "answer: 42let me think..." {
		t.Errorf("all text = %q, want %q", all, "answer: 42let me think...")
	}
}
