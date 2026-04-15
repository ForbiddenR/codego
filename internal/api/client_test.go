package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nice-code/codego/internal/types"
)

// mockSSEServer creates a test server that returns SSE events.
func mockSSEServer(events []string, statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers
		if r.Header.Get("x-api-key") == "" {
			http.Error(w, "missing api key", 401)
			return
		}
		if r.Header.Get("anthropic-version") == "" {
			http.Error(w, "missing version", 400)
			return
		}

		if statusCode != http.StatusOK {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(statusCode)
			errResp := map[string]interface{}{
				"type": "error",
				"error": map[string]string{
					"type":    "api_error",
					"message": fmt.Sprintf("status %d", statusCode),
				},
			}
			json.NewEncoder(w).Encode(errResp)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", 500)
			return
		}

		for _, event := range events {
			fmt.Fprint(w, event)
			flusher.Flush()
			time.Sleep(10 * time.Millisecond)
		}
	}))
}

func TestNew(t *testing.T) {
	c := New("test-key")
	if c.apiKey != "test-key" {
		t.Errorf("apiKey = %q, want %q", c.apiKey, "test-key")
	}
	if c.baseURL != "https://api.anthropic.com" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://api.anthropic.com")
	}
	if c.model != "claude-sonnet-4-20250514" {
		t.Errorf("model = %q, want %q", c.model, "claude-sonnet-4-20250514")
	}
}

func TestNew_WithOptions(t *testing.T) {
	c := New("key",
		WithBaseURL("http://localhost:8080/"),
		WithModel("claude-opus-4-20250514"),
		WithMaxTokens(4096),
	)
	if c.baseURL != "http://localhost:8080" {
		t.Errorf("baseURL = %q, want trailing slash removed", c.baseURL)
	}
	if c.model != "claude-opus-4-20250514" {
		t.Errorf("model = %q, want %q", c.model, "claude-opus-4-20250514")
	}
	if c.maxTokens != 4096 {
		t.Errorf("maxTokens = %d, want 4096", c.maxTokens)
	}
}

func TestClient_Model(t *testing.T) {
	c := New("key", WithModel("test-model"))
	if c.Model() != "test-model" {
		t.Errorf("Model() = %q, want %q", c.Model(), "test-model")
	}
}

func TestStreamMessage_Success(t *testing.T) {
	events := []string{
		"event: message_start\ndata: {\"type\":\"message\"}\n\n",
		"event: content_block_start\ndata: {\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n",
		"event: content_block_delta\ndata: {\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"Hello\"}}\n\n",
		"event: content_block_delta\ndata: {\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\" world\"}}\n\n",
		"event: content_block_stop\ndata: {\"index\":0}\n\n",
		"event: message_delta\ndata: {\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":2}}\n\n",
		"event: message_stop\ndata: {}\n\n",
	}

	server := mockSSEServer(events, http.StatusOK)
	defer server.Close()

	c := New("test-key", WithBaseURL(server.URL))
	ctx := context.Background()

	req := &CreateMessageRequest{
		Messages: []types.Message{types.NewUserMessage("hello")},
	}

	eventCh, errCh := c.StreamMessage(ctx, req)

	var textDeltas []string
	var gotStop bool

	for {
		select {
		case event, ok := <-eventCh:
			if !ok {
				goto done
			}
			if event.IsTextDelta() {
				textDeltas = append(textDeltas, event.DeltaText())
			}
			if event.Type == types.EventMessageStop {
				gotStop = true
			}
		case err := <-errCh:
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for events")
		}
	}
done:

	if len(textDeltas) != 2 {
		t.Fatalf("text deltas = %d, want 2", len(textDeltas))
	}
	if textDeltas[0] != "Hello" {
		t.Errorf("delta[0] = %q, want %q", textDeltas[0], "Hello")
	}
	if textDeltas[1] != " world" {
		t.Errorf("delta[1] = %q, want %q", textDeltas[1], " world")
	}
	if !gotStop {
		t.Error("did not receive message_stop")
	}
}

func TestStreamMessage_ToolUse(t *testing.T) {
	events := []string{
		"event: message_start\ndata: {}\n\n",
		"event: content_block_start\ndata: {\"index\":0,\"content_block\":{\"type\":\"tool_use\",\"id\":\"tool_1\",\"name\":\"bash\"}}\n\n",
		"event: content_block_delta\ndata: {\"index\":0,\"delta\":\"{\\\"command\\\": \"ls\"}\"}\n\n",
		"event: content_block_stop\ndata: {\"index\":0}\n\n",
		"event: message_stop\ndata: {}\n\n",
	}

	server := mockSSEServer(events, http.StatusOK)
	defer server.Close()

	c := New("test-key", WithBaseURL(server.URL))
	ctx := context.Background()

	eventCh, errCh := c.StreamMessage(ctx, &CreateMessageRequest{})

	var toolBlock *types.ContentBlock
	for event := range eventCh {
		if event.Type == types.EventContentBlockStart && event.ContentBlock != nil {
			if event.ContentBlock.IsToolUse() {
				toolBlock = event.ContentBlock
			}
		}
	}
	// Drain error channel
	select {
	case <-errCh:
	default:
	}

	if toolBlock == nil {
		t.Fatal("did not get tool_use block")
	}
	if toolBlock.Name != "bash" {
		t.Errorf("tool name = %q, want %q", toolBlock.Name, "bash")
	}
	if toolBlock.ID != "tool_1" {
		t.Errorf("tool id = %q, want %q", toolBlock.ID, "tool_1")
	}
}

func TestStreamMessage_AuthError(t *testing.T) {
	server := mockSSEServer(nil, http.StatusUnauthorized)
	defer server.Close()

	c := New("bad-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, errCh := c.StreamMessage(ctx, &CreateMessageRequest{})

	err := <-errCh
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if !apiErr.IsAuthError() {
		t.Error("should be auth error")
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("status = %d, want 401", apiErr.StatusCode)
	}
}

func TestStreamMessage_RateLimit(t *testing.T) {
	server := mockSSEServer(nil, http.StatusTooManyRequests)
	defer server.Close()

	c := New("key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, errCh := c.StreamMessage(ctx, &CreateMessageRequest{})

	err := <-errCh
	apiErr := err.(*APIError)
	if !apiErr.IsRateLimit() {
		t.Error("should be rate limit error")
	}
	if !apiErr.IsRetryable() {
		t.Error("rate limit should be retryable")
	}
}

func TestStreamMessage_ServerError(t *testing.T) {
	server := mockSSEServer(nil, http.StatusInternalServerError)
	defer server.Close()

	c := New("key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, errCh := c.StreamMessage(ctx, &CreateMessageRequest{})

	err := <-errCh
	apiErr := err.(*APIError)
	if apiErr.StatusCode != 500 {
		t.Errorf("status = %d, want 500", apiErr.StatusCode)
	}
	if !apiErr.IsRetryable() {
		t.Error("500 should be retryable")
	}
}

func TestStreamMessage_Overloaded(t *testing.T) {
	server := mockSSEServer(nil, 529)
	defer server.Close()

	c := New("key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, errCh := c.StreamMessage(ctx, &CreateMessageRequest{})

	err := <-errCh
	apiErr := err.(*APIError)
	if !apiErr.IsOverloaded() {
		t.Error("should be overloaded error")
	}
	if !apiErr.IsRetryable() {
		t.Error("529 should be retryable")
	}
}

func TestStreamMessage_ContextCancel(t *testing.T) {
	// Server that never finishes sending
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("event: message_start\ndata: {}\n\n"))
		w.(http.Flusher).Flush()
		// Block forever
		<-r.Context().Done()
	}))
	defer server.Close()

	c := New("key", WithBaseURL(server.URL))
	ctx, cancel := context.WithCancel(context.Background())

	eventCh, errCh := c.StreamMessage(ctx, &CreateMessageRequest{})

	// Read one event then cancel
	select {
	case <-eventCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}

	cancel()

	// Channels should eventually close
	timeout := time.After(3 * time.Second)
	for {
		select {
		case _, ok := <-eventCh:
			if !ok {
				return // channel closed, good
			}
		case err := <-errCh:
			if err != nil {
				return // got error, good
			}
		case <-timeout:
			t.Fatal("channels did not close after context cancel")
		}
	}
}

func TestCreateMessage_Success(t *testing.T) {
	respBody := `{
		"id": "msg_123",
		"type": "message",
		"role": "assistant",
		"content": [{"type": "text", "text": "Hello!"}],
		"model": "claude-sonnet-4-20250514",
		"stop_reason": "end_turn",
		"usage": {"input_tokens": 10, "output_tokens": 5}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(respBody))
	}))
	defer server.Close()

	c := New("key", WithBaseURL(server.URL))
	resp, err := c.CreateMessage(context.Background(), &CreateMessageRequest{
		Messages: []types.Message{types.NewUserMessage("hi")},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != "msg_123" {
		t.Errorf("id = %q, want %q", resp.ID, "msg_123")
	}
	if resp.TextContent() != "Hello!" {
		t.Errorf("text = %q, want %q", resp.TextContent(), "Hello!")
	}
	if resp.Usage.InputTokens != 10 {
		t.Errorf("input tokens = %d, want 10", resp.Usage.InputTokens)
	}
}

func TestCreateMessage_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{"type":"error","error":{"type":"authentication_error","message":"invalid key"}}`))
	}))
	defer server.Close()

	c := New("bad", WithBaseURL(server.URL))
	_, err := c.CreateMessage(context.Background(), &CreateMessageRequest{})

	if err == nil {
		t.Fatal("expected error")
	}
	apiErr := err.(*APIError)
	if apiErr.ErrorType != "authentication_error" {
		t.Errorf("error type = %q, want %q", apiErr.ErrorType, "authentication_error")
	}
}

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  APIError
		want string
	}{
		{
			name: "with type",
			err:  APIError{StatusCode: 401, ErrorType: "auth_error", Message: "bad key"},
			want: "API error 401 (auth_error): bad key",
		},
		{
			name: "without type",
			err:  APIError{StatusCode: 500, Message: "internal"},
			want: "API error 500: internal",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMessageResponse_TextContent(t *testing.T) {
	resp := &MessageResponse{
		Content: []types.ContentBlock{
			types.NewTextBlock("Hello "),
			types.NewTextBlock("world"),
			types.NewToolUseBlock("id1", "bash", nil),
		},
	}
	if got := resp.TextContent(); got != "Hello world" {
		t.Errorf("TextContent() = %q, want %q", got, "Hello world")
	}
}

func TestMessageResponse_ToolCalls(t *testing.T) {
	resp := &MessageResponse{
		Content: []types.ContentBlock{
			types.NewTextBlock("let me run a command"),
			types.NewToolUseBlock("id1", "bash", map[string]interface{}{"command": "ls"}),
			types.NewToolUseBlock("id2", "read", map[string]interface{}{"path": "main.go"}),
		},
	}
	calls := resp.ToolCalls()
	if len(calls) != 2 {
		t.Fatalf("ToolCalls() = %d, want 2", len(calls))
	}
	if calls[0].Name != "bash" {
		t.Errorf("tool[0] = %q, want %q", calls[0].Name, "bash")
	}
}

func TestStreamEvent_IsTextDelta(t *testing.T) {
	text := "chunk"
	event := types.StreamEvent{
		Type:         types.EventContentBlockDelta,
		ContentBlock: &types.ContentBlock{Type: types.ContentTypeText},
		Delta:        &text,
	}
	if !event.IsTextDelta() {
		t.Error("should be text delta")
	}
	if event.DeltaText() != "chunk" {
		t.Errorf("DeltaText() = %q, want %q", event.DeltaText(), "chunk")
	}
}

func TestStreamEvent_ThinkingDelta(t *testing.T) {
	event := types.StreamEvent{
		Type:         types.EventContentBlockDelta,
		ContentBlock: &types.ContentBlock{Type: types.ContentTypeThinking},
	}
	if !event.IsThinkingDelta() {
		t.Error("should be thinking delta")
	}
	if event.IsTextDelta() {
		t.Error("should not be text delta")
	}
}

func TestStreamMessage_InvalidJSON(t *testing.T) {
	// Server returns garbage SSE
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("event: message_start\ndata: {invalid json}\n\n"))
		w.Write([]byte("event: content_block_delta\ndata: {\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"valid\"}}\n\n"))
		w.Write([]byte("event: message_stop\ndata: {}\n\n"))
		w.(http.Flusher).Flush()
	}))
	defer server.Close()

	c := New("key", WithBaseURL(server.URL))
	eventCh, _ := c.StreamMessage(context.Background(), &CreateMessageRequest{})

	var validEvents int
	for range eventCh {
		validEvents++
	}

	// Should have the 2 valid events (invalid JSON skipped)
	if validEvents != 2 {
		t.Errorf("got %d events, want 2 (invalid should be skipped)", validEvents)
	}
}
