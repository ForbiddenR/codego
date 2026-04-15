package tui

import "github.com/charmbracelet/lipgloss"

// Theme holds all styles for the TUI.
type Theme struct {
	// Status bar
	StatusBar       lipgloss.Style
	StatusBarText   lipgloss.Style

	// Messages
	UserLabel       lipgloss.Style
	UserContent     lipgloss.Style
	AssistantText   lipgloss.Style
	SystemText      lipgloss.Style

	// Tool calls
	ToolHeader      lipgloss.Style
	ToolSuccess     lipgloss.Style
	ToolError       lipgloss.Style
	ToolOutput      lipgloss.Style

	// Input
	InputActive     lipgloss.Style
	InputInactive   lipgloss.Style

	// Spinner
	SpinnerText     lipgloss.Style

	// Help
	HelpText        lipgloss.Style

	// Approval dialog
	ApprovalBorder  lipgloss.Style
	ApprovalHeader  lipgloss.Style

	// Welcome
	WelcomeTitle    lipgloss.Style
	WelcomeInfo     lipgloss.Style
	WelcomeTip      lipgloss.Style
}

// DefaultTheme is the default dark theme.
var DefaultTheme = Theme{
	StatusBar:       lipgloss.NewStyle().Background(lipgloss.Color("236")),
	StatusBarText:   lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
	UserLabel:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14")),
	UserContent:     lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
	AssistantText:   lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
	SystemText:      lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true),
	ToolHeader:      lipgloss.NewStyle().Foreground(lipgloss.Color("240")).BorderLeft(true).BorderForeground(lipgloss.Color("240")).PaddingLeft(1),
	ToolSuccess:     lipgloss.NewStyle().Foreground(lipgloss.Color("240")).BorderLeft(true).BorderForeground(lipgloss.Color("2")).PaddingLeft(1),
	ToolError:       lipgloss.NewStyle().Foreground(lipgloss.Color("196")).BorderLeft(true).BorderForeground(lipgloss.Color("196")).PaddingLeft(1),
	ToolOutput:      lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
	InputActive:     lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("99")),
	InputInactive:   lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62")),
	SpinnerText:     lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
	HelpText:        lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
	ApprovalBorder:  lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).BorderForeground(lipgloss.Color("208")).Padding(1),
	ApprovalHeader:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208")),
	WelcomeTitle:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99")),
	WelcomeInfo:     lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
	WelcomeTip:      lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true),
}

// LightTheme is a light background theme.
var LightTheme = Theme{
	StatusBar:       lipgloss.NewStyle().Background(lipgloss.Color("252")),
	StatusBarText:   lipgloss.NewStyle().Foreground(lipgloss.Color("236")),
	UserLabel:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("26")),
	UserContent:     lipgloss.NewStyle().Foreground(lipgloss.Color("236")),
	AssistantText:   lipgloss.NewStyle().Foreground(lipgloss.Color("236")),
	SystemText:      lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Italic(true),
	ToolHeader:      lipgloss.NewStyle().Foreground(lipgloss.Color("244")).BorderLeft(true).BorderForeground(lipgloss.Color("244")).PaddingLeft(1),
	ToolSuccess:     lipgloss.NewStyle().Foreground(lipgloss.Color("244")).BorderLeft(true).BorderForeground(lipgloss.Color("28")).PaddingLeft(1),
	ToolError:       lipgloss.NewStyle().Foreground(lipgloss.Color("160")).BorderLeft(true).BorderForeground(lipgloss.Color("160")).PaddingLeft(1),
	ToolOutput:      lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
	InputActive:     lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("26")),
	InputInactive:   lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("244")),
	SpinnerText:     lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
	HelpText:        lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
	ApprovalBorder:  lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).BorderForeground(lipgloss.Color("160")).Padding(1),
	ApprovalHeader:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("160")),
	WelcomeTitle:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("26")),
	WelcomeInfo:     lipgloss.NewStyle().Foreground(lipgloss.Color("236")),
	WelcomeTip:      lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Italic(true),
}
