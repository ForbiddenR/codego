package tui

import (
	"encoding/json"
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// ApprovalDialog shows a tool approval prompt.
type ApprovalDialog struct {
	Visible   bool
	ToolName  string
	ToolInput map[string]interface{}
	Theme     Theme
}

// NewApprovalDialog creates a new approval dialog.
func NewApprovalDialog(name string, input map[string]interface{}) ApprovalDialog {
	return ApprovalDialog{
		Visible:   true,
		ToolName:  name,
		ToolInput: input,
		Theme:     DefaultTheme,
	}
}

// Render renders the approval dialog.
func (d ApprovalDialog) Render(width int) string {
	if !d.Visible {
		return ""
	}

	if width < 20 {
		width = 20
	}
	dialogWidth := width - 8
	if dialogWidth > 60 {
		dialogWidth = 60
	}

	style := d.Theme.ApprovalBorder.Width(dialogWidth)
	headerStyle := d.Theme.ApprovalHeader

	header := headerStyle.Render(fmt.Sprintf("Allow tool: %s?", d.ToolName))

	inputStr := formatInput(d.ToolInput)
	if len(inputStr) > dialogWidth-4 {
		inputStr = inputStr[:dialogWidth-7] + "..."
	}
	inputLine := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render("Input: " + inputStr)

	prompt := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("[y] Yes  [n] No  [a] Always allow")

	content := header + "\n\n" + inputLine + "\n\n" + prompt
	return style.Render(content)
}

func formatInput(input map[string]interface{}) string {
	if input == nil {
		return "{}"
	}
	data, err := json.Marshal(input)
	if err != nil {
		return fmt.Sprintf("%v", input)
	}
	return string(data)
}
