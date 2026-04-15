package tools

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/nice-code/codego/internal/types"
)

// TodoStatus represents the status of a todo item.
type TodoStatus string

const (
	TodoPending    TodoStatus = "pending"
	TodoInProgress TodoStatus = "in_progress"
	TodoCompleted  TodoStatus = "completed"
	TodoCancelled  TodoStatus = "cancelled"
)

// TodoItem is a single task in the todo list.
type TodoItem struct {
	ID     int        `json:"id"`
	Text   string     `json:"text"`
	Status TodoStatus `json:"status"`
}

// TodoWriteTool manages an in-session task list.
type TodoWriteTool struct {
	mu    sync.Mutex
	items []TodoItem
	next  int
}

func NewTodoWriteTool() *TodoWriteTool {
	return &TodoWriteTool{}
}

func (t *TodoWriteTool) Name() string { return "todo" }
func (t *TodoWriteTool) Description() string {
	return "Manage a task list for the current session. Add, update, and list tasks to track progress on multi-step work."
}
func (t *TodoWriteTool) InputSchema() *types.JSONSchema {
	return types.NewObjectSchema(
		"Todo input",
		map[string]*types.JSONSchema{
			"action": types.NewStringSchema("Action: add, update, list, clear"),
			"text":   types.NewStringSchema("Task text (for add)"),
			"id":     types.NewNumberSchema("Task ID (for update)"),
			"status": types.NewStringSchema("New status: pending, in_progress, completed, cancelled (for update)"),
		},
		"action",
	)
}

func (t *TodoWriteTool) Execute(_ context.Context, input types.ToolInput) (*types.ToolResult, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	action := input.GetString("action")
	switch action {
	case "add":
		return t.add(input)
	case "update":
		return t.update(input)
	case "list":
		return t.list()
	case "clear":
		return t.clear()
	default:
		return types.NewToolError(fmt.Sprintf("unknown action: %q (use add, update, list, clear)", action)), nil
	}
}

func (t *TodoWriteTool) add(input types.ToolInput) (*types.ToolResult, error) {
	text := input.GetString("text")
	if text == "" {
		return types.NewToolError("text is required for add"), nil
	}
	t.next++
	item := TodoItem{ID: t.next, Text: text, Status: TodoPending}
	t.items = append(t.items, item)
	return types.NewToolResult(fmt.Sprintf("Added task #%d: %s", item.ID, text)), nil
}

func (t *TodoWriteTool) update(input types.ToolInput) (*types.ToolResult, error) {
	id := int(input.GetFloat("id"))
	if id <= 0 {
		return types.NewToolError("id is required for update"), nil
	}
	status := TodoStatus(input.GetString("status"))
	if status == "" {
		return types.NewToolError("status is required for update"), nil
	}

	for i, item := range t.items {
		if item.ID == id {
			t.items[i].Status = status
			return types.NewToolResult(fmt.Sprintf("Updated task #%d: %s → %s", id, item.Text, status)), nil
		}
	}
	return types.NewToolError(fmt.Sprintf("task #%d not found", id)), nil
}

func (t *TodoWriteTool) list() (*types.ToolResult, error) {
	if len(t.items) == 0 {
		return types.NewToolResult("No tasks"), nil
	}

	var sb strings.Builder
	for _, item := range t.items {
		icon := statusIcon(item.Status)
		sb.WriteString(fmt.Sprintf("%s #%d: %s\n", icon, item.ID, item.Text))
	}
	return types.NewToolResult(sb.String()), nil
}

func (t *TodoWriteTool) clear() (*types.ToolResult, error) {
	count := len(t.items)
	t.items = nil
	t.next = 0
	return types.NewToolResult(fmt.Sprintf("Cleared %d tasks", count)), nil
}

func statusIcon(s TodoStatus) string {
	switch s {
	case TodoPending:
		return "[ ]"
	case TodoInProgress:
		return "[~]"
	case TodoCompleted:
		return "[x]"
	case TodoCancelled:
		return "[-]"
	default:
		return "[?]"
	}
}
