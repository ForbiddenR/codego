package types

// StreamEvent represents a single event from the streaming API.
type StreamEvent struct {
	Type         string         `json:"type"`
	ContentBlock *ContentBlock  `json:"content_block,omitempty"`
	Delta        *string        `json:"delta,omitempty"`
	Index        int            `json:"index,omitempty"`
	Usage        *Usage         `json:"usage,omitempty"`
}

// Stream event type constants matching the Anthropic API.
const (
	EventMessageStart       = "message_start"
	EventContentBlockStart  = "content_block_start"
	EventContentBlockDelta  = "content_block_delta"
	EventContentBlockStop   = "content_block_stop"
	EventMessageDelta       = "message_delta"
	EventMessageStop        = "message_stop"
	EventPing               = "ping"
	EventError              = "error"
)

// IsTextDelta returns true if this is a text content block delta.
func (e StreamEvent) IsTextDelta() bool {
	return e.Type == EventContentBlockDelta && e.ContentBlock != nil && e.ContentBlock.Type == ContentTypeText
}

// IsThinkingDelta returns true if this is a thinking content block delta.
func (e StreamEvent) IsThinkingDelta() bool {
	return e.Type == EventContentBlockDelta && e.ContentBlock != nil && e.ContentBlock.Type == ContentTypeThinking
}

// IsToolInputDelta returns true if this is a tool_use input delta.
func (e StreamEvent) IsToolInputDelta() bool {
	return e.Type == EventContentBlockDelta && e.ContentBlock != nil && e.ContentBlock.Type == ContentTypeToolUse
}

// DeltaText returns the delta text, or empty string if not a text delta.
func (e StreamEvent) DeltaText() string {
	if e.Delta != nil {
		return *e.Delta
	}
	return ""
}

// Usage contains token usage information from the API response.
type Usage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}

// TotalInputTokens returns the total input tokens including cache.
func (u Usage) TotalInputTokens() int {
	return u.InputTokens + u.CacheCreationInputTokens + u.CacheReadInputTokens
}

// HasCache returns true if any cache tokens were used.
func (u Usage) HasCache() bool {
	return u.CacheCreationInputTokens > 0 || u.CacheReadInputTokens > 0
}

// RunResult is the final result of an agent run.
type RunResult struct {
	Text  string `json:"text"`
	Usage Usage  `json:"usage"`
}

// NewRunResult creates a run result with text and usage.
func NewRunResult(text string, usage Usage) *RunResult {
	return &RunResult{Text: text, Usage: usage}
}
