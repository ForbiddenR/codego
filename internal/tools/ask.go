package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/nice-code/codego/internal/types"
)

// AskTool prompts the user for input via stdin.
type AskTool struct{}

func NewAskTool() *AskTool { return &AskTool{} }

func (t *AskTool) Name() string        { return "ask" }
func (t *AskTool) Description() string  { return "Ask the user a question and wait for their input. Use this when you need clarification or a decision." }
func (t *AskTool) InputSchema() *types.JSONSchema {
	return types.NewObjectSchema(
		"Ask input",
		map[string]*types.JSONSchema{
			"question": types.NewStringSchema("The question to ask the user"),
			"options":  types.NewStringSchema("Comma-separated list of options (optional)"),
		},
		"question",
	)
}

func (t *AskTool) Execute(_ context.Context, input types.ToolInput) (*types.ToolResult, error) {
	question := input.GetString("question")
	if question == "" {
		return types.NewToolError("question is required"), nil
	}

	options := input.GetString("options")

	fmt.Printf("\n  ? %s\n", question)
	if options != "" {
		fmt.Printf("  Options: %s\n", options)
	}
	fmt.Print("  > ")

	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return types.NewToolError(fmt.Sprintf("failed to read input: %v", err)), nil
	}

	answer = strings.TrimSpace(answer)
	if answer == "" {
		return types.NewToolResult("(user provided no answer)"), nil
	}

	return types.NewToolResult(fmt.Sprintf("User answered: %s", answer)), nil
}
