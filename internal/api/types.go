package api

import (
	"errors"
	"fmt"

	"github.com/nice-code/codego/internal/types"
)

// CreateMessageRequest is the request body for the /v1/messages endpoint.
type CreateMessageRequest struct {
	Model       string            `json:"model"`
	MaxTokens   int               `json:"max_tokens"`
	Messages    []types.Message   `json:"messages"`
	System      string            `json:"system,omitempty"`
	Tools       []types.ToolDef   `json:"tools,omitempty"`
	Stream      bool              `json:"stream"`
	Temperature *float32          `json:"temperature,omitempty"`
	Thinking    *ThinkingConfig   `json:"thinking,omitempty"`
}

// ThinkingConfig enables extended thinking on supported models.
type ThinkingConfig struct {
	Type         string `json:"type"`
	BudgetTokens int    `json:"budget_tokens"`
}

// MessageResponse is the non-streaming response from /v1/messages.
type MessageResponse struct {
	ID           string              `json:"id"`
	Type         string              `json:"type"`
	Role         types.Role          `json:"role"`
	Content      []types.ContentBlock `json:"content"`
	Model        string              `json:"model"`
	StopReason   string              `json:"stop_reason"`
	StopSequence *string             `json:"stop_sequence"`
	Usage        types.Usage         `json:"usage"`
}

// TextContent concatenates all text blocks in the response.
func (r *MessageResponse) TextContent() string {
	var text string
	for _, b := range r.Content {
		if b.IsText() {
			text += b.Text
		}
	}
	return text
}

// ToolCalls returns all tool_use blocks from the response.
func (r *MessageResponse) ToolCalls() []types.ContentBlock {
	var calls []types.ContentBlock
	for _, b := range r.Content {
		if b.IsToolUse() {
			calls = append(calls, b)
		}
	}
	return calls
}

// APIError represents an error response from the Anthropic API.
type APIError struct {
	StatusCode int
	ErrorType  string
	Message    string
}

func (e *APIError) Error() string {
	if e.ErrorType != "" {
		return fmt.Sprintf("API error %d (%s): %s", e.StatusCode, e.ErrorType, e.Message)
	}
	return fmt.Sprintf("API error %d: %s", e.StatusCode, e.Message)
}

// IsRateLimit returns true if this is a rate limit error (429).
func (e *APIError) IsRateLimit() bool {
	return e.StatusCode == 429
}

// IsAuthError returns true if this is an authentication error (401).
func (e *APIError) IsAuthError() bool {
	return e.StatusCode == 401
}

// IsOverloaded returns true if the API is overloaded (529).
func (e *APIError) IsOverloaded() bool {
	return e.StatusCode == 529
}

// IsRetryable returns true if the request should be retried.
func (e *APIError) IsRetryable() bool {
	return e.StatusCode == 429 || e.StatusCode == 529 || e.StatusCode >= 500
}

// Sentinel errors for common cases.
var (
	ErrMaxIterations = errors.New("max iterations reached")
	ErrNoAPIKey      = errors.New("ANTHROPIC_API_KEY not set")
)
