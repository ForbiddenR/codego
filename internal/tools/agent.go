package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/nice-code/codego/internal/types"
)

// AgentExecutor is the interface for running a sub-agent.
type AgentExecutor interface {
	Run(ctx context.Context, prompt string) (*types.RunResult, error)
}

// AgentTool spawns a sub-agent to handle complex tasks.
type AgentTool struct {
	Factory  func() AgentExecutor
	Timeout  time.Duration
}

func NewAgentTool(factory func() AgentExecutor) *AgentTool {
	return &AgentTool{
		Factory: factory,
		Timeout: 5 * time.Minute,
	}
}

func (t *AgentTool) Name() string { return "agent" }
func (t *AgentTool) Description() string {
	return "Spawn a sub-agent to handle a complex task independently. The sub-agent has its own context and tools. Returns a summary of what it accomplished."
}
func (t *AgentTool) InputSchema() *types.JSONSchema {
	return types.NewObjectSchema(
		"Agent input",
		map[string]*types.JSONSchema{
			"prompt":  types.NewStringSchema("The task for the sub-agent to execute"),
			"timeout": types.NewNumberSchema("Timeout in seconds (default 300)"),
		},
		"prompt",
	)
}

func (t *AgentTool) Execute(ctx context.Context, input types.ToolInput) (*types.ToolResult, error) {
	prompt := input.GetString("prompt")
	if prompt == "" {
		return types.NewToolError("prompt is required"), nil
	}

	timeout := t.Timeout
	if input.Has("timeout") {
		s := input.GetFloat("timeout")
		if s > 0 {
			timeout = time.Duration(s) * time.Second
		}
	}

	if t.Factory == nil {
		return types.NewToolError("agent factory not configured"), nil
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	subAgent := t.Factory()
	result, err := subAgent.Run(ctx, prompt)
	if err != nil {
		return types.NewToolError(fmt.Sprintf("sub-agent error: %v", err)), nil
	}

	output := "Sub-agent completed.\n"
	if result.Text != "" {
		output += result.Text
	}
	if result.Usage.OutputTokens > 0 {
		output += fmt.Sprintf("\n\n(Tokens: %d in, %d out)",
			result.Usage.InputTokens, result.Usage.OutputTokens)
	}

	return types.NewToolResult(output), nil
}
