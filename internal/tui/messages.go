package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nice-code/codego/internal/types"
)

// MessageKind is the type of message in the view.
type MessageKind int

const (
	MsgUser MessageKind = iota
	MsgAssistant
	MsgToolCall
	MsgSystem
)

// MessageView is a message rendered for the TUI.
type MessageView struct {
	Kind       MessageKind
	Content    string // accumulated text
	ToolName   string
	ToolInput  map[string]interface{}
	ToolResult *types.ToolResult
}

// NewUserMessageView creates a user message view.
func NewUserMessageView(text string) MessageView {
	return MessageView{Kind: MsgUser, Content: text}
}

// NewAssistantMessageView creates an empty assistant message view (for streaming).
func NewAssistantMessageView() MessageView {
	return MessageView{Kind: MsgAssistant}
}

// NewToolCallView creates a tool call view.
func NewToolCallView(name string, input map[string]interface{}) MessageView {
	return MessageView{Kind: MsgToolCall, ToolName: name, ToolInput: input}
}

// NewSystemMessageView creates a system message view.
func NewSystemMessageView(text string) MessageView {
	return MessageView{Kind: MsgSystem, Content: text}
}

// AppendText appends streaming text to the message.
func (m *MessageView) AppendText(text string) {
	m.Content += text
}

// SetResult sets the tool result for a tool call view.
func (m *MessageView) SetResult(r *types.ToolResult) {
	m.ToolResult = r
}

// Render renders the message for display.
func (m MessageView) Render(width int) string {
	if width <= 0 {
		width = 80
	}

	switch m.Kind {
	case MsgUser:
		return renderUser(m.Content, width)
	case MsgAssistant:
		return renderAssistant(m.Content, width)
	case MsgToolCall:
		return renderToolCall(m.ToolName, m.ToolResult, width)
	case MsgSystem:
		return renderSystem(m.Content, width)
	}
	return ""
}

// Styles
var (
	userLabelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("14")) // cyan

	userContentStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	assistantStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	toolHeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			BorderLeft(true).
			BorderForeground(lipgloss.Color("240")).
			PaddingLeft(1)

	toolSuccessStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
			BorderLeft(true).
			BorderForeground(lipgloss.Color("2")).
			PaddingLeft(1)

	toolErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			BorderLeft(true).
			BorderForeground(lipgloss.Color("196")).
			PaddingLeft(1)

	systemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true)
)

func renderUser(text string, width int) string {
	label := userLabelStyle.Render("You")
	content := userContentStyle.Render(text)
	return label + "\n" + content
}

func renderAssistant(text string, width int) string {
	if text == "" {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("⠋ thinking...")
	}
	return assistantStyle.Render(text)
}

func renderToolCall(name string, result *types.ToolResult, width int) string {
	header := fmt.Sprintf("  %s", name)

	if result == nil {
		return toolHeaderStyle.Render(header + " ⠋")
	}

	output := result.Output
	if len(output) > 500 {
		output = output[:497] + "..."
	}
	// Indent output lines
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		lines[i] = "  " + line
	}

	style := toolSuccessStyle
	status := " ✓"
	if result.IsError {
		style = toolErrorStyle
		status = " ✗"
	}

	return style.Render(header+status) + "\n" + style.Render(strings.Join(lines, "\n"))
}

func renderSystem(text string, width int) string {
	return systemStyle.Render("  " + text)
}
