package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nice-code/codego/internal/types"
)

// MemoryDir returns the memory directory path.
func MemoryDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".codego")
}

// MemoryPath returns the memory file path.
func MemoryPath() string {
	return filepath.Join(MemoryDir(), "memory.md")
}

// Memory manages persistent memory across sessions.
type Memory struct {
	path string
}

// NewMemory creates a new memory manager.
func NewMemory() *Memory {
	return &Memory{path: MemoryPath()}
}

// NewMemoryWithPath creates a memory manager with a custom path.
func NewMemoryWithPath(path string) *Memory {
	return &Memory{path: path}
}

// Load reads all memories from disk.
func (m *Memory) Load() (string, error) {
	data, err := os.ReadFile(m.path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

// Save writes content to the memory file.
func (m *Memory) Save(content string) error {
	if err := os.MkdirAll(filepath.Dir(m.path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(m.path, []byte(content), 0o644)
}

// Append adds a memory entry with timestamp.
func (m *Memory) Append(text string) error {
	existing, _ := m.Load()
	timestamp := time.Now().Format("2006-01-02 15:04")
	entry := fmt.Sprintf("\n- [%s] %s", timestamp, text)

	if existing == "" {
		existing = "# CodeGo Memory\n"
	}

	return m.Save(existing + entry)
}

// Clear removes all memories.
func (m *Memory) Clear() error {
	return m.Save("")
}

// Exists returns true if the memory file exists.
func (m *Memory) Exists() bool {
	_, err := os.Stat(m.path)
	return err == nil
}

// MemoryTool is a tool for managing persistent memory.
type MemoryTool struct {
	memory *Memory
}

func NewMemoryTool() *MemoryTool {
	return &MemoryTool{memory: NewMemory()}
}

func (t *MemoryTool) Name() string { return "memory" }
func (t *MemoryTool) Description() string {
	return "Manage persistent memory across sessions. Actions: add, show, clear. Use to remember important facts, preferences, and context."
}
func (t *MemoryTool) InputSchema() *types.JSONSchema {
	return types.NewObjectSchema(
		"Memory input",
		map[string]*types.JSONSchema{
			"action": types.NewStringSchema("Action: add, show, clear"),
			"text":   types.NewStringSchema("Memory text (for add)"),
		},
		"action",
	)
}

func (t *MemoryTool) Execute(_ context.Context, input types.ToolInput) (*types.ToolResult, error) {
	action := input.GetString("action")
	switch action {
	case "add":
		text := input.GetString("text")
		if text == "" {
			return types.NewToolError("text is required for add"), nil
		}
		if err := t.memory.Append(text); err != nil {
			return types.NewToolError(fmt.Sprintf("save error: %v", err)), nil
		}
		return types.NewToolResult(fmt.Sprintf("Remembered: %s", text)), nil

	case "show":
		content, err := t.memory.Load()
		if err != nil {
			return types.NewToolError(fmt.Sprintf("read error: %v", err)), nil
		}
		if content == "" {
			return types.NewToolResult("No memories stored yet."), nil
		}
		return types.NewToolResult(content), nil

	case "clear":
		if err := t.memory.Clear(); err != nil {
			return types.NewToolError(fmt.Sprintf("clear error: %v", err)), nil
		}
		return types.NewToolResult("Memory cleared."), nil

	default:
		return types.NewToolError(fmt.Sprintf("unknown action: %q (use: add, show, clear)", action)), nil
	}
}

// LoadMemoryForPrompt loads memory content for inclusion in the system prompt.
func LoadMemoryForPrompt() string {
	m := NewMemory()
	content, err := m.Load()
	if err != nil || strings.TrimSpace(content) == "" {
		return ""
	}
	return "<user_memory>\n" + content + "\n</user_memory>"
}
