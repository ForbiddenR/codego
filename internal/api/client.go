package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/nice-code/codego/internal/types"
)

// Client is the Anthropic API client.
type Client struct {
	apiKey     string
	baseURL    string
	model      string
	maxTokens  int
	httpClient *http.Client
}

// Option configures the Client.
type Option func(*Client)

// WithBaseURL sets a custom API base URL.
func WithBaseURL(url string) Option {
	return func(c *Client) { c.baseURL = strings.TrimRight(url, "/") }
}

// WithModel sets the default model.
func WithModel(model string) Option {
	return func(c *Client) { c.model = model }
}

// WithMaxTokens sets the default max tokens.
func WithMaxTokens(n int) Option {
	return func(c *Client) { c.maxTokens = n }
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// New creates a new Anthropic API client.
func New(apiKey string, opts ...Option) *Client {
	c := &Client{
		apiKey:    apiKey,
		baseURL:   "https://api.anthropic.com",
		model:     "claude-sonnet-4-20250514",
		maxTokens: 8192,
		httpClient: &http.Client{
			Timeout: 0, // no timeout for streaming
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Model returns the current model name.
func (c *Client) Model() string {
	return c.model
}

// buildRequest creates an http.Request for the messages endpoint.
func (c *Client) buildRequest(ctx context.Context, body []byte) (*http.Request, error) {
	url := c.baseURL + "/v1/messages"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	return req, nil
}

// CreateMessage sends a non-streaming request and returns the full response.
func (c *Client) CreateMessage(ctx context.Context, req *CreateMessageRequest) (*MessageResponse, error) {
	req.Stream = false
	if req.Model == "" {
		req.Model = c.model
	}
	if req.MaxTokens == 0 {
		req.MaxTokens = c.maxTokens
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := c.buildRequest(ctx, body)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp.StatusCode, respBody)
	}

	var msgResp MessageResponse
	if err := json.Unmarshal(respBody, &msgResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return &msgResp, nil
}

// StreamMessage sends a streaming request and returns channels for events and errors.
// The caller should range over the events channel until it closes.
// The error channel receives at most one error and then closes.
func (c *Client) StreamMessage(ctx context.Context, req *CreateMessageRequest) (<-chan types.StreamEvent, <-chan error) {
	events := make(chan types.StreamEvent, 32)
	errs := make(chan error, 1)

	go func() {
		defer close(events)
		defer close(errs)

		req.Stream = true
		if req.Model == "" {
			req.Model = c.model
		}
		if req.MaxTokens == 0 {
			req.MaxTokens = c.maxTokens
		}

		body, err := json.Marshal(req)
		if err != nil {
			errs <- fmt.Errorf("marshal request: %w", err)
			return
		}

		httpReq, err := c.buildRequest(ctx, body)
		if err != nil {
			errs <- fmt.Errorf("build request: %w", err)
			return
		}

		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			errs <- fmt.Errorf("http request: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			errs <- parseError(resp.StatusCode, respBody)
			return
		}

		c.parseSSE(ctx, resp.Body, events, errs)
	}()

	return events, errs
}

// parseSSE reads Server-Sent Events from the response body and emits StreamEvents.
func (c *Client) parseSSE(ctx context.Context, body io.Reader, events chan<- types.StreamEvent, errs chan<- error) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer

	var currentEvent string
	for scanner.Scan() {
		line := scanner.Text()

		// Empty line signals end of event
		if line == "" {
			currentEvent = ""
			continue
		}

		// Comment lines start with :
		if strings.HasPrefix(line, ":") {
			continue
		}

		// Parse field: value
		field, value, ok := strings.Cut(line, ": ")
		if !ok {
			field = line
			value = ""
		}

		switch field {
		case "event":
			currentEvent = value
		case "data":
			event, err := c.parseSSEData(currentEvent, value)
			if err != nil {
				// Skip unparseable events but don't fail
				continue
			}
			select {
			case events <- event:
			case <-ctx.Done():
				return
			}
		}
	}

	if err := scanner.Err(); err != nil {
		select {
		case errs <- fmt.Errorf("read stream: %w", err):
		default:
		}
	}
}

// parseSSEData parses a single SSE data line into a StreamEvent.
func (c *Client) parseSSEData(eventType, data string) (types.StreamEvent, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(data), &raw); err != nil {
		return types.StreamEvent{}, err
	}

	event := types.StreamEvent{Type: eventType}

	switch eventType {
	case types.EventContentBlockStart, types.EventContentBlockDelta, types.EventContentBlockStop:
		if idx, ok := raw["index"]; ok {
			json.Unmarshal(idx, &event.Index)
		}
		if cb, ok := raw["content_block"]; ok {
			var block types.ContentBlock
			if err := json.Unmarshal(cb, &block); err == nil {
				event.ContentBlock = &block
			}
		}
		if delta, ok := raw["delta"]; ok {
			// Try to extract text delta
			var deltaObj map[string]json.RawMessage
			if err := json.Unmarshal(delta, &deltaObj); err == nil {
				if text, ok := deltaObj["text"]; ok {
					var s string
					if err := json.Unmarshal(text, &s); err == nil {
						event.Delta = &s
						// Set ContentBlock type for delta events (content_block_delta
						// doesn't carry a content_block field, only delta)
						if event.ContentBlock == nil {
							event.ContentBlock = &types.ContentBlock{Type: types.ContentTypeText}
						}
					}
				}
				// Check delta type for thinking or tool_use deltas
				if deltaType, ok := deltaObj["type"]; ok {
					var dt string
					if err := json.Unmarshal(deltaType, &dt); err == nil {
						switch dt {
						case "thinking_delta":
							if event.ContentBlock == nil {
								event.ContentBlock = &types.ContentBlock{Type: types.ContentTypeThinking}
							}
						case "input_json_delta":
							if event.ContentBlock == nil {
								event.ContentBlock = &types.ContentBlock{Type: types.ContentTypeToolUse}
							}
						}
					}
				}
				// For content_block_start, the content_block is in delta
				if cb, ok := deltaObj["content_block"]; ok {
					var block types.ContentBlock
					if err := json.Unmarshal(cb, &block); err == nil {
						event.ContentBlock = &block
					}
				}
			}
		}

	case types.EventMessageDelta:
		if usage, ok := raw["usage"]; ok {
			var u types.Usage
			if err := json.Unmarshal(usage, &u); err == nil {
				event.Usage = &u
			}
		}

	case types.EventMessageStart:
		// message_start contains the full message object
		// we could parse it but it's mostly useful for the id

	case types.EventMessageStop, types.EventPing:
		// no additional data needed
	}

	return event, nil
}

// parseError creates an appropriate error from a non-200 response.
func parseError(statusCode int, body []byte) error {
	var errResp struct {
		Type  string `json:"type"`
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &errResp); err != nil {
		return &APIError{
			StatusCode: statusCode,
			Message:    string(body),
		}
	}

	return &APIError{
		StatusCode: statusCode,
		ErrorType:  errResp.Error.Type,
		Message:    errResp.Error.Message,
	}
}
