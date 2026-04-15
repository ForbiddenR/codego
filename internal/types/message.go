package types

// Role represents the sender of a message in the conversation.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
)

// IsValid returns true if the role is one of the defined constants.
func (r Role) IsValid() bool {
	switch r {
	case RoleUser, RoleAssistant, RoleSystem:
		return true
	}
	return false
}

// String returns the string representation of the role.
func (r Role) String() string {
	return string(r)
}

// ContentBlockType represents the type of a content block.
type ContentBlockType string

const (
	ContentTypeText     ContentBlockType = "text"
	ContentTypeToolUse  ContentBlockType = "tool_use"
	ContentTypeToolResult ContentBlockType = "tool_result"
	ContentTypeThinking ContentBlockType = "thinking"
)

// ContentBlock represents a single block of content within a message.
// This maps directly to the Anthropic API content block format.
type ContentBlock struct {
	Type      ContentBlockType       `json:"type"`
	Text      string                 `json:"text,omitempty"`
	ID        string                 `json:"id,omitempty"`
	Name      string                 `json:"name,omitempty"`
	Input     map[string]interface{} `json:"input,omitempty"`
	ToolUseID string                 `json:"tool_use_id,omitempty"`
	Content   string                 `json:"content,omitempty"`   // for tool_result
	IsError   bool                   `json:"is_error,omitempty"`  // for tool_result
}

// IsText returns true if this is a text content block.
func (cb ContentBlock) IsText() bool {
	return cb.Type == ContentTypeText
}

// IsToolUse returns true if this is a tool_use content block.
func (cb ContentBlock) IsToolUse() bool {
	return cb.Type == ContentTypeToolUse
}

// IsToolResult returns true if this is a tool_result content block.
func (cb ContentBlock) IsToolResult() bool {
	return cb.Type == ContentTypeToolResult
}

// IsThinking returns true if this is a thinking content block.
func (cb ContentBlock) IsThinking() bool {
	return cb.Type == ContentTypeThinking
}

// Message represents a single message in the conversation.
type Message struct {
	Role    Role           `json:"role"`
	Content []ContentBlock `json:"content"`
}

// NewUserMessage creates a new user message with a single text block.
func NewUserMessage(text string) Message {
	return Message{
		Role: RoleUser,
		Content: []ContentBlock{
			{Type: ContentTypeText, Text: text},
		},
	}
}

// NewAssistantMessage creates a new assistant message with content blocks.
func NewAssistantMessage(blocks ...ContentBlock) Message {
	return Message{
		Role:    RoleAssistant,
		Content: blocks,
	}
}

// NewAssistantText creates a new assistant message with a single text block.
func NewAssistantText(text string) Message {
	return NewAssistantMessage(ContentBlock{Type: ContentTypeText, Text: text})
}

// NewSystemMessage creates a new system message with a single text block.
func NewSystemMessage(text string) Message {
	return Message{
		Role: RoleSystem,
		Content: []ContentBlock{
			{Type: ContentTypeText, Text: text},
		},
	}
}

// NewToolResultMessage creates a new user message containing a tool result.
// Tool results are always sent as user messages in the Anthropic API.
func NewToolResultMessage(toolUseID, content string, isError bool) Message {
	return Message{
		Role: RoleUser,
		Content: []ContentBlock{
			{
				Type:      ContentTypeToolResult,
				ToolUseID: toolUseID,
				Content:   content,
				IsError:   isError,
			},
		},
	}
}

// NewToolUseBlock creates a content block for a tool use.
func NewToolUseBlock(id, name string, input map[string]interface{}) ContentBlock {
	return ContentBlock{
		Type:  ContentTypeToolUse,
		ID:    id,
		Name:  name,
		Input: input,
	}
}

// NewTextBlock creates a content block for text.
func NewTextBlock(text string) ContentBlock {
	return ContentBlock{
		Type: ContentTypeText,
		Text: text,
	}
}

// NewThinkingBlock creates a content block for thinking.
func NewThinkingBlock(text string) ContentBlock {
	return ContentBlock{
		Type: ContentTypeThinking,
		Text: text,
	}
}

// HasToolCalls returns true if the message contains any tool_use blocks.
func (m Message) HasToolCalls() bool {
	for _, b := range m.Content {
		if b.IsToolUse() {
			return true
		}
	}
	return false
}

// ToolCalls returns only the tool_use content blocks from the message.
func (m Message) ToolCalls() []ContentBlock {
	var calls []ContentBlock
	for _, b := range m.Content {
		if b.IsToolUse() {
			calls = append(calls, b)
		}
	}
	return calls
}

// TextContent concatenates all text blocks into a single string.
func (m Message) TextContent() string {
	var text string
	for _, b := range m.Content {
		if b.IsText() {
			text += b.Text
		}
	}
	return text
}

// AllText concatenates text and thinking blocks into a single string.
func (m Message) AllText() string {
	var text string
	for _, b := range m.Content {
		if b.IsText() || b.IsThinking() {
			text += b.Text
		}
	}
	return text
}
