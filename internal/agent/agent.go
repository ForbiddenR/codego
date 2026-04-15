package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/nice-code/codego/internal/api"
	"github.com/nice-code/codego/internal/tools"
	"github.com/nice-code/codego/internal/types"
)

// APIClient is the interface the agent needs from the API layer.
type APIClient interface {
	Model() string
	StreamMessage(ctx context.Context, req *api.CreateMessageRequest) (<-chan types.StreamEvent, <-chan error)
}

// Agent runs the conversation loop.
type Agent struct {
	api           APIClient
	tools         *tools.Registry
	messages      []types.Message
	systemPrompt  string
	model         string
	maxTokens     int
	maxIterations int

	OnText      func(text string)
	OnThinking  func(text string)
	OnToolStart func(name string, input types.ToolInput)
	OnToolEnd   func(name string, result *types.ToolResult)
	OnUsage     func(usage types.Usage)
}

type Option func(*Agent)

func WithSystemPrompt(p string) Option   { return func(a *Agent) { a.systemPrompt = p } }
func WithModel(m string) Option          { return func(a *Agent) { a.model = m } }
func WithMaxTokens(n int) Option         { return func(a *Agent) { a.maxTokens = n } }
func WithMaxIterations(n int) Option     { return func(a *Agent) { a.maxIterations = n } }
func WithMessages(msgs []types.Message) Option {
	return func(a *Agent) { a.messages = msgs }
}

func New(client APIClient, toolRegistry *tools.Registry, opts ...Option) *Agent {
	a := &Agent{
		api:           client,
		tools:         toolRegistry,
		maxTokens:     8192,
		maxIterations: 50,
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

func (a *Agent) Model() string {
	if a.model != "" {
		return a.model
	}
	return a.api.Model()
}

func (a *Agent) Messages() []types.Message        { return a.messages }
func (a *Agent) SetMessages(msgs []types.Message)  { a.messages = msgs }
func (a *Agent) Reset()                            { a.messages = nil }

func (a *Agent) Run(ctx context.Context, userMessage string) (*types.RunResult, error) {
	a.messages = append(a.messages, types.NewUserMessage(userMessage))
	return a.run(ctx)
}

func (a *Agent) Continue(ctx context.Context) (*types.RunResult, error) {
	return a.run(ctx)
}

func (a *Agent) run(ctx context.Context) (*types.RunResult, error) {
	var totalUsage types.Usage

	for i := 0; i < a.maxIterations; i++ {
		done, usage, err := a.step(ctx)
		if err != nil {
			return nil, err
		}
		totalUsage = mergeUsage(totalUsage, usage)
		if done {
			return types.NewRunResult("", totalUsage), nil
		}
	}

	return nil, fmt.Errorf("%w after %d iterations", api.ErrMaxIterations, a.maxIterations)
}

func (a *Agent) step(ctx context.Context) (done bool, usage types.Usage, err error) {
	req := a.buildRequest()
	eventCh, errCh := a.api.StreamMessage(ctx, req)

	var (
		toolCalls   []types.ContentBlock
		assistantText strings.Builder
		toolInputs  = make(map[int]*strings.Builder)
		lastUsage   types.Usage
	)

	// Process events until channel closes
	for event := range eventCh {
		switch event.Type {
		case types.EventContentBlockStart:
			if event.ContentBlock != nil && event.ContentBlock.IsToolUse() {
				for len(toolCalls) <= event.Index {
					toolCalls = append(toolCalls, types.ContentBlock{})
				}
				toolCalls[event.Index] = *event.ContentBlock
				toolInputs[event.Index] = &strings.Builder{}
			}

		case types.EventContentBlockDelta:
			switch {
			case event.IsTextDelta():
				text := event.DeltaText()
				assistantText.WriteString(text)
				if a.OnText != nil {
					a.OnText(text)
				}
			case event.IsThinkingDelta():
				if a.OnThinking != nil {
					a.OnThinking(event.DeltaText())
				}
			case event.IsToolInputDelta():
				if builder, ok := toolInputs[event.Index]; ok && event.Delta != nil {
					builder.WriteString(*event.Delta)
				}
			}

		case types.EventMessageDelta:
			if event.Usage != nil {
				lastUsage = *event.Usage
			}
		}
	}

	// Channel closed — drain error (always sent by API client)
	err = <-errCh
	if err != nil {
		return false, lastUsage, err
	}

	// Build assistant message
	var assistantBlocks []types.ContentBlock
	if text := assistantText.String(); text != "" {
		assistantBlocks = append(assistantBlocks, types.NewTextBlock(text))
	}

	for i, tc := range toolCalls {
		if tc.IsToolUse() {
			if builder, ok := toolInputs[i]; ok {
				inputJSON := builder.String()
				if inputJSON != "" {
					_ = json.Unmarshal([]byte(inputJSON), &tc.Input)
				}
			}
			assistantBlocks = append(assistantBlocks, tc)
		}
	}

	if len(assistantBlocks) > 0 {
		a.messages = append(a.messages, types.NewAssistantMessage(assistantBlocks...))
	}

	if a.OnUsage != nil {
		a.OnUsage(lastUsage)
	}

	if len(toolCalls) == 0 {
		return true, lastUsage, nil
	}

	// Execute tools concurrently
	results := a.executeTools(ctx, toolCalls)
	for i, tc := range toolCalls {
		if tc.IsToolUse() {
			a.messages = append(a.messages, types.NewToolResultMessage(tc.ID, results[i].Output, results[i].IsError))
		}
	}

	return false, lastUsage, nil
}

func (a *Agent) buildRequest() *api.CreateMessageRequest {
	req := &api.CreateMessageRequest{
		MaxTokens: a.maxTokens,
		Messages:  a.messages,
		Stream:    true,
	}
	if a.model != "" {
		req.Model = a.model
	}
	if a.systemPrompt != "" {
		req.System = a.systemPrompt
	}
	if a.tools != nil && a.tools.Count() > 0 {
		req.Tools = a.tools.ToolDefs()
	}
	return req
}

func (a *Agent) executeTools(ctx context.Context, calls []types.ContentBlock) []*types.ToolResult {
	results := make([]*types.ToolResult, len(calls))
	var wg sync.WaitGroup

	for i, call := range calls {
		if !call.IsToolUse() {
			continue
		}
		wg.Add(1)
		go func(idx int, c types.ContentBlock) {
			defer wg.Done()
			if a.OnToolStart != nil {
				a.OnToolStart(c.Name, c.Input)
			}
			results[idx] = a.executeSingleTool(ctx, c)
			if a.OnToolEnd != nil {
				a.OnToolEnd(c.Name, results[idx])
			}
		}(i, call)
	}
	wg.Wait()
	return results
}

func (a *Agent) executeSingleTool(ctx context.Context, call types.ContentBlock) *types.ToolResult {
	tool, ok := a.tools.Get(call.Name)
	if !ok {
		return types.NewToolError(fmt.Sprintf("tool not found: %s", call.Name))
	}
	result, err := tool.Execute(ctx, call.Input)
	if err != nil {
		return types.NewToolError(fmt.Sprintf("tool error: %v", err))
	}
	return result
}

func mergeUsage(a, b types.Usage) types.Usage {
	return types.Usage{
		InputTokens:              a.InputTokens + b.InputTokens,
		OutputTokens:             a.OutputTokens + b.OutputTokens,
		CacheCreationInputTokens: a.CacheCreationInputTokens + b.CacheCreationInputTokens,
		CacheReadInputTokens:     a.CacheReadInputTokens + b.CacheReadInputTokens,
	}
}

func parseToolInput(s string) map[string]interface{} {
	input := make(map[string]interface{})
	s = strings.TrimSpace(s)
	if s == "" || s == "{}" {
		return input
	}
	_ = json.Unmarshal([]byte(s), &input)
	return input
}
