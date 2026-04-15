package types

import "testing"

func TestStreamEvent_IsTextDelta(t *testing.T) {
	text := "hello"
	tests := []struct {
		name  string
		event StreamEvent
		want  bool
	}{
		{
			name: "text delta",
			event: StreamEvent{
				Type:         EventContentBlockDelta,
				ContentBlock: &ContentBlock{Type: ContentTypeText},
				Delta:        &text,
			},
			want: true,
		},
		{
			name: "tool input delta",
			event: StreamEvent{
				Type:         EventContentBlockDelta,
				ContentBlock: &ContentBlock{Type: ContentTypeToolUse},
			},
			want: false,
		},
		{
			name:  "message start",
			event: StreamEvent{Type: EventMessageStart},
			want:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.event.IsTextDelta(); got != tt.want {
				t.Errorf("IsTextDelta() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStreamEvent_IsThinkingDelta(t *testing.T) {
	event := StreamEvent{
		Type:         EventContentBlockDelta,
		ContentBlock: &ContentBlock{Type: ContentTypeThinking},
	}
	if !event.IsThinkingDelta() {
		t.Error("should be thinking delta")
	}
}

func TestStreamEvent_IsToolInputDelta(t *testing.T) {
	event := StreamEvent{
		Type:         EventContentBlockDelta,
		ContentBlock: &ContentBlock{Type: ContentTypeToolUse},
	}
	if !event.IsToolInputDelta() {
		t.Error("should be tool input delta")
	}
}

func TestStreamEvent_DeltaText(t *testing.T) {
	text := "chunk"
	withDelta := StreamEvent{Delta: &text}
	if got := withDelta.DeltaText(); got != "chunk" {
		t.Errorf("DeltaText() = %q, want %q", got, "chunk")
	}

	noDelta := StreamEvent{}
	if got := noDelta.DeltaText(); got != "" {
		t.Errorf("DeltaText() = %q, want empty", got)
	}
}

func TestUsage_TotalInputTokens(t *testing.T) {
	u := Usage{
		InputTokens:              100,
		CacheCreationInputTokens: 50,
		CacheReadInputTokens:     200,
	}
	if got := u.TotalInputTokens(); got != 350 {
		t.Errorf("TotalInputTokens() = %d, want 350", got)
	}
}

func TestUsage_HasCache(t *testing.T) {
	withCache := Usage{CacheReadInputTokens: 10}
	if !withCache.HasCache() {
		t.Error("should have cache")
	}

	withoutCache := Usage{InputTokens: 100, OutputTokens: 50}
	if withoutCache.HasCache() {
		t.Error("should not have cache")
	}
}

func TestNewRunResult(t *testing.T) {
	usage := Usage{InputTokens: 100, OutputTokens: 50}
	r := NewRunResult("the answer", usage)

	if r.Text != "the answer" {
		t.Errorf("text = %q, want %q", r.Text, "the answer")
	}
	if r.Usage.InputTokens != 100 {
		t.Errorf("input tokens = %d, want 100", r.Usage.InputTokens)
	}
}
