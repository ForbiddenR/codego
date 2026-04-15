package agent

import (
	"context"
	"fmt"
	"testing"

	"github.com/nice-code/codego/internal/api"
	"github.com/nice-code/codego/internal/tools"
	"github.com/nice-code/codego/internal/types"
)

// ─── Mock API ───

type mockAPI struct {
	responses []mockResponse
	callCount int
}

type mockResponse struct {
	events []types.StreamEvent
	err    error
}

func (m *mockAPI) Model() string { return "mock-model" }

func (m *mockAPI) StreamMessage(ctx context.Context, req *api.CreateMessageRequest) (<-chan types.StreamEvent, <-chan error) {
	events := make(chan types.StreamEvent, 64)
	errs := make(chan error, 1)

	go func() {
		defer close(errs)

		if m.callCount >= len(m.responses) {
			errs <- fmt.Errorf("no more mock responses (call %d)", m.callCount)
			close(events)
			return
		}

		resp := m.responses[m.callCount]
		m.callCount++

		if resp.err != nil {
			errs <- resp.err
			close(events)
			return
		}

		for _, event := range resp.events {
			select {
			case events <- event:
			case <-ctx.Done():
				errs <- ctx.Err()
				close(events)
				return
			}
		}
		close(events)
		// Always send nil on success (step drains errCh after events close)
		errs <- nil
	}()

	return events, errs
}

// ─── Helpers ───

func ptr(s string) *string { return &s }

func textEvents(text string) []types.StreamEvent {
	bt := types.ContentTypeText
	return []types.StreamEvent{
		{Type: types.EventContentBlockStart, Index: 0, ContentBlock: &types.ContentBlock{Type: bt}},
		{Type: types.EventContentBlockDelta, Index: 0, ContentBlock: &types.ContentBlock{Type: bt}, Delta: &text},
		{Type: types.EventContentBlockStop, Index: 0},
		{Type: types.EventMessageDelta, Usage: &types.Usage{OutputTokens: 10}},
	}
}

func toolUseEvents(id, name string, inputJSON string) []types.StreamEvent {
	return []types.StreamEvent{
		{Type: types.EventContentBlockStart, Index: 0, ContentBlock: &types.ContentBlock{
			Type: types.ContentTypeToolUse, ID: id, Name: name,
		}},
		{Type: types.EventContentBlockDelta, Index: 0, ContentBlock: &types.ContentBlock{Type: types.ContentTypeToolUse}, Delta: &inputJSON},
		{Type: types.EventContentBlockStop, Index: 0},
		{Type: types.EventMessageDelta, Usage: &types.Usage{OutputTokens: 20}},
	}
}

func makeInfiniteToolLoop(count int) []mockResponse {
	responses := make([]mockResponse, count)
	for i := range responses {
		responses[i] = mockResponse{events: toolUseEvents("t1", "loop", `{}`)}
	}
	return responses
}

type stubTool struct {
	name     string
	response *types.ToolResult
	execErr  error
}

func (t *stubTool) Name() string                  { return t.name }
func (t *stubTool) Description() string            { return "stub" }
func (t *stubTool) InputSchema() *types.JSONSchema { return nil }
func (t *stubTool) Execute(_ context.Context, _ types.ToolInput) (*types.ToolResult, error) {
	if t.execErr != nil {
		return nil, t.execErr
	}
	return t.response, nil
}

// ─── Tests ───

func TestNew(t *testing.T) {
	a := New(&mockAPI{}, tools.NewRegistry())
	if a.Model() != "mock-model" {
		t.Errorf("model = %q, want %q", a.Model(), "mock-model")
	}
	if a.maxIterations != 50 {
		t.Errorf("maxIterations = %d, want 50", a.maxIterations)
	}
	if a.maxTokens != 8192 {
		t.Errorf("maxTokens = %d, want 8192", a.maxTokens)
	}
}

func TestNew_WithOptions(t *testing.T) {
	a := New(&mockAPI{}, nil,
		WithSystemPrompt("system"),
		WithModel("custom"),
		WithMaxTokens(4096),
		WithMaxIterations(5),
	)
	if a.Model() != "custom" {
		t.Errorf("model = %q", a.Model())
	}
	if a.systemPrompt != "system" {
		t.Errorf("systemPrompt = %q", a.systemPrompt)
	}
	if a.maxTokens != 4096 || a.maxIterations != 5 {
		t.Errorf("maxTokens=%d maxIterations=%d", a.maxTokens, a.maxIterations)
	}
}

func TestAgent_Run_TextResponse(t *testing.T) {
	mock := &mockAPI{
		responses: []mockResponse{{events: textEvents("Hello!")}},
	}
	a := New(mock, tools.NewRegistry())

	var deltas []string
	a.OnText = func(s string) { deltas = append(deltas, s) }

	_, err := a.Run(context.Background(), "hi")
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if len(deltas) != 1 || deltas[0] != "Hello!" {
		t.Errorf("deltas = %v, want [Hello!]", deltas)
	}

	msgs := a.Messages()
	if len(msgs) != 2 {
		t.Fatalf("messages = %d, want 2", len(msgs))
	}
	if msgs[0].Role != types.RoleUser {
		t.Errorf("msg[0] role = %q", msgs[0].Role)
	}
	if msgs[1].Role != types.RoleAssistant {
		t.Errorf("msg[1] role = %q", msgs[1].Role)
	}
	if msgs[1].TextContent() != "Hello!" {
		t.Errorf("assistant text = %q, want %q", msgs[1].TextContent(), "Hello!")
	}
}

func TestAgent_Run_MultiBlockText(t *testing.T) {
	bt := types.ContentTypeText
	events := []types.StreamEvent{
		{Type: types.EventContentBlockStart, Index: 0, ContentBlock: &types.ContentBlock{Type: bt}},
		{Type: types.EventContentBlockDelta, Index: 0, ContentBlock: &types.ContentBlock{Type: bt}, Delta: ptr("Hello ")},
		{Type: types.EventContentBlockDelta, Index: 0, ContentBlock: &types.ContentBlock{Type: bt}, Delta: ptr("world")},
		{Type: types.EventContentBlockStop, Index: 0},
		{Type: types.EventMessageDelta, Usage: &types.Usage{OutputTokens: 5}},
	}

	mock := &mockAPI{responses: []mockResponse{{events: events}}}
	a := New(mock, tools.NewRegistry())

	var deltas []string
	a.OnText = func(s string) { deltas = append(deltas, s) }

	a.Run(context.Background(), "test")

	if len(deltas) != 2 {
		t.Fatalf("deltas = %d, want 2", len(deltas))
	}
	if a.Messages()[1].TextContent() != "Hello world" {
		t.Errorf("text = %q", a.Messages()[1].TextContent())
	}
}

func TestAgent_Run_ToolCall(t *testing.T) {
	mock := &mockAPI{
		responses: []mockResponse{
			{events: toolUseEvents("t1", "echo", `{"text": "hello"}`)},
			{events: textEvents("result is hello")},
		},
	}
	reg := tools.NewRegistry()
	reg.Register(&stubTool{name: "echo", response: types.NewToolResult("echoed: hello")})

	a := New(mock, reg, WithMaxIterations(10))

	var starts, ends []string
	a.OnToolStart = func(n string, _ types.ToolInput) { starts = append(starts, n) }
	a.OnToolEnd = func(n string, _ *types.ToolResult) { ends = append(ends, n) }

	_, err := a.Run(context.Background(), "echo hello")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(starts) != 1 || starts[0] != "echo" {
		t.Errorf("starts = %v", starts)
	}
	if len(ends) != 1 || ends[0] != "echo" {
		t.Errorf("ends = %v", ends)
	}

	msgs := a.Messages()
	// user → assistant(tool_use) → user(tool_result) → assistant(text)
	if len(msgs) != 4 {
		t.Fatalf("messages = %d, want 4", len(msgs))
	}
	if msgs[3].TextContent() != "result is hello" {
		t.Errorf("final text = %q", msgs[3].TextContent())
	}
}

func TestAgent_Run_MultipleToolCalls(t *testing.T) {
	events := []types.StreamEvent{
		{Type: types.EventContentBlockStart, Index: 0, ContentBlock: &types.ContentBlock{Type: types.ContentTypeToolUse, ID: "t1", Name: "e"}},
		{Type: types.EventContentBlockDelta, Index: 0, ContentBlock: &types.ContentBlock{Type: types.ContentTypeToolUse}, Delta: ptr(`{"x":"a"}`)},
		{Type: types.EventContentBlockStop, Index: 0},
		{Type: types.EventContentBlockStart, Index: 1, ContentBlock: &types.ContentBlock{Type: types.ContentTypeToolUse, ID: "t2", Name: "e"}},
		{Type: types.EventContentBlockDelta, Index: 1, ContentBlock: &types.ContentBlock{Type: types.ContentTypeToolUse}, Delta: ptr(`{"x":"b"}`)},
		{Type: types.EventContentBlockStop, Index: 1},
		{Type: types.EventMessageDelta, Usage: &types.Usage{OutputTokens: 30}},
	}

	mock := &mockAPI{
		responses: []mockResponse{
			{events: events},
			{events: textEvents("done")},
		},
	}
	reg := tools.NewRegistry()
	reg.Register(&stubTool{name: "e", response: types.NewToolResult("ok")})

	a := New(mock, reg, WithMaxIterations(10))
	_, err := a.Run(context.Background(), "run two")
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	msgs := a.Messages()
	if len(msgs) != 5 {
		t.Fatalf("messages = %d, want 5", len(msgs))
	}
}

func TestAgent_Run_MaxIterations(t *testing.T) {
	mock := &mockAPI{responses: makeInfiniteToolLoop(100)}
	reg := tools.NewRegistry()
	reg.Register(&stubTool{name: "loop", response: types.NewToolResult("again")})

	a := New(mock, reg, WithMaxIterations(3))
	_, err := a.Run(context.Background(), "loop")
	if err == nil {
		t.Fatal("expected error")
	}
	if mock.callCount != 3 {
		t.Errorf("calls = %d, want 3", mock.callCount)
	}
}

func TestAgent_Run_APIError(t *testing.T) {
	mock := &mockAPI{
		responses: []mockResponse{
			{err: &api.APIError{StatusCode: 500, Message: "boom"}},
		},
	}
	a := New(mock, nil)
	_, err := a.Run(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAgent_Run_ContextCancel(t *testing.T) {
	mock := &mockAPI{responses: []mockResponse{{events: textEvents("hi")}}}
	a := New(mock, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := a.Run(ctx, "test")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAgent_Run_UnknownTool(t *testing.T) {
	mock := &mockAPI{
		responses: []mockResponse{
			{events: toolUseEvents("t1", "nonexistent", `{}`)},
			{events: textEvents("ok")},
		},
	}
	a := New(mock, tools.NewRegistry(), WithMaxIterations(5))
	a.Run(context.Background(), "test")

	msgs := a.Messages()
	// user → assistant(tool) → user(tool_result) → assistant(text)
	if len(msgs) != 4 {
		t.Fatalf("messages = %d, want 4", len(msgs))
	}
	toolResult := msgs[2].Content[0]
	if toolResult.Content != "tool not found: nonexistent" {
		t.Errorf("tool result = %q", toolResult.Content)
	}
}

func TestAgent_Run_ToolError(t *testing.T) {
	mock := &mockAPI{
		responses: []mockResponse{
			{events: toolUseEvents("t1", "bad", `{}`)},
			{events: textEvents("recovered")},
		},
	}
	reg := tools.NewRegistry()
	reg.Register(&stubTool{name: "bad", execErr: fmt.Errorf("boom")})

	a := New(mock, reg, WithMaxIterations(5))
	_, err := a.Run(context.Background(), "test")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
}

func TestAgent_Run_ThinkingEvents(t *testing.T) {
	text := "thinking..."
	thinkEvents := []types.StreamEvent{
		{Type: types.EventContentBlockStart, Index: 0, ContentBlock: &types.ContentBlock{Type: types.ContentTypeThinking}},
		{Type: types.EventContentBlockDelta, Index: 0, ContentBlock: &types.ContentBlock{Type: types.ContentTypeThinking}, Delta: &text},
		{Type: types.EventContentBlockStop, Index: 0},
	}
	all := append(thinkEvents, textEvents("answer")...)
	mock := &mockAPI{responses: []mockResponse{{events: all}}}

	a := New(mock, nil)
	var thinkTexts []string
	a.OnThinking = func(s string) { thinkTexts = append(thinkTexts, s) }

	a.Run(context.Background(), "think")

	if len(thinkTexts) != 1 || thinkTexts[0] != "thinking..." {
		t.Errorf("thinking = %v", thinkTexts)
	}
}

func TestAgent_Reset(t *testing.T) {
	mock := &mockAPI{responses: []mockResponse{{events: textEvents("hi")}}}
	a := New(mock, nil)
	a.Run(context.Background(), "test")

	if len(a.Messages()) != 2 {
		t.Fatalf("before reset: %d messages", len(a.Messages()))
	}

	a.Reset()
	if len(a.Messages()) != 0 {
		t.Fatalf("after reset: %d messages", len(a.Messages()))
	}
}

func TestAgent_SetMessages(t *testing.T) {
	a := New(nil, nil)
	a.SetMessages([]types.Message{types.NewUserMessage("saved")})
	if len(a.Messages()) != 1 {
		t.Fatalf("messages = %d", len(a.Messages()))
	}
}

func TestAgent_Continue(t *testing.T) {
	mock := &mockAPI{responses: []mockResponse{{events: textEvents("response")}}}
	a := New(mock, nil)
	a.SetMessages([]types.Message{types.NewUserMessage("pending")})

	_, err := a.Continue(context.Background())
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(a.Messages()) != 2 {
		t.Fatalf("messages = %d, want 2", len(a.Messages()))
	}
}

func TestAgent_BuildRequest(t *testing.T) {
	a := New(&mockAPI{}, nil,
		WithSystemPrompt("sys"),
		WithModel("test"),
		WithMaxTokens(1000),
	)
	a.SetMessages([]types.Message{types.NewUserMessage("hi")})
	reg := tools.NewRegistry()
	reg.Register(&stubTool{name: "t"})
	a.tools = reg

	req := a.buildRequest()
	if req.Model != "test" || req.System != "sys" || req.MaxTokens != 1000 {
		t.Errorf("model=%q system=%q maxTokens=%d", req.Model, req.System, req.MaxTokens)
	}
	if !req.Stream {
		t.Error("stream should be true")
	}
	if len(req.Tools) != 1 {
		t.Errorf("tools = %d", len(req.Tools))
	}
}

func TestMergeUsage(t *testing.T) {
	a := types.Usage{InputTokens: 10, OutputTokens: 20, CacheReadInputTokens: 5}
	b := types.Usage{InputTokens: 30, OutputTokens: 40, CacheCreationInputTokens: 3}
	m := mergeUsage(a, b)

	if m.InputTokens != 40 || m.OutputTokens != 60 {
		t.Errorf("tokens: input=%d output=%d", m.InputTokens, m.OutputTokens)
	}
	if m.CacheReadInputTokens != 5 || m.CacheCreationInputTokens != 3 {
		t.Errorf("cache: read=%d create=%d", m.CacheReadInputTokens, m.CacheCreationInputTokens)
	}
}

func TestParseToolInput(t *testing.T) {
	tests := []struct {
		input string
		key   string
		want  string
	}{
		{`{"command":"ls"}`, "command", "ls"},
		{`{}`, "", ""},
		{`{"path":"/tmp"}`, "path", "/tmp"},
		{``, "", ""},
	}
	for _, tt := range tests {
		result := parseToolInput(tt.input)
		if tt.key == "" {
			if len(result) != 0 {
				t.Errorf("parse(%q) = %v, want empty", tt.input, result)
			}
			continue
		}
		if got, ok := result[tt.key]; !ok || got != tt.want {
			t.Errorf("parse(%q)[%q] = %v, want %q", tt.input, tt.key, got, tt.want)
		}
	}
}
