package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/nice-code/codego/internal/types"
)

// BashTool executes shell commands.
type BashTool struct {
	WorkingDir string
	Timeout    time.Duration
}

// NewBashTool creates a BashTool with default settings.
func NewBashTool(workingDir string) *BashTool {
	return &BashTool{
		WorkingDir: workingDir,
		Timeout:    2 * time.Minute,
	}
}

func (t *BashTool) Name() string { return "bash" }

func (t *BashTool) Description() string {
	return "Execute a shell command. Use this for running code, installing packages, git operations, and any system tasks. Returns stdout and stderr combined."
}

func (t *BashTool) InputSchema() *types.JSONSchema {
	return types.NewObjectSchema(
		"Bash command input",
		map[string]*types.JSONSchema{
			"command": types.NewStringSchema("The shell command to execute"),
			"timeout": types.NewNumberSchema("Optional timeout in milliseconds (default 120000)"),
		},
		"command",
	)
}

func (t *BashTool) Execute(ctx context.Context, input types.ToolInput) (*types.ToolResult, error) {
	command := input.GetString("command")
	if command == "" {
		return types.NewToolError("command is required"), nil
	}

	timeout := t.Timeout
	if input.Has("timeout") {
		ms := input.GetFloat("timeout")
		if ms > 0 {
			timeout = time.Duration(ms) * time.Millisecond
		}
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	cmd.Dir = t.WorkingDir

	output, err := cmd.CombinedOutput()
	result := strings.TrimRight(string(output), "\n")

	if ctx.Err() == context.DeadlineExceeded {
		if result != "" {
			result += "\n"
		}
		result += fmt.Sprintf("\n[Command timed out after %v]", timeout)
		return types.NewToolError(result), nil
	}

	if err != nil {
		if result != "" {
			result += "\n"
		}
		result += fmt.Sprintf("\n[Exit code: %v]", err)
		return types.NewToolError(result), nil
	}

	if result == "" {
		result = "(no output)"
	}

	return types.NewToolResult(result), nil
}
