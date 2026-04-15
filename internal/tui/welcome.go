package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const banner = `
   ██████╗ ██████╗ ██████╗ ███████╗ ██████╗  ██████╗
  ██╔════╝██╔═══██╗██╔══██╗██╔════╝██╔════╝ ██╔═══██╗
  ██║     ██║   ██║██║  ██║█████╗  ██║  ███╗██║   ██║
  ██║     ██║   ██║██║  ██║██╔══╝  ██║   ██║██║   ██║
  ╚██████╗╚██████╔╝██████╔╝███████╗╚██████╔╝╚██████╔╝
   ╚═════╝ ╚═════╝ ╚═════╝ ╚══════╝ ╚═════╝  ╚═════╝`

// WelcomeScreen returns the welcome/banner text.
func WelcomeScreen(model, version string, width int) string {
	if width <= 0 {
		width = 80
	}

	theme := DefaultTheme

	var sb strings.Builder

	// Banner
	bannerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Bold(true)
	sb.WriteString(bannerStyle.Render(banner))
	sb.WriteString("\n\n")

	// Title
	title := theme.WelcomeTitle.Render(fmt.Sprintf("CodeGo v%s", version))
	sb.WriteString(title)
	sb.WriteString("\n\n")

	// Info
	info := []string{
		fmt.Sprintf("Model: %s", model),
		"Config: ~/.codego/config.yaml",
		"Memory: ~/.codego/memory.md",
	}
	for _, line := range info {
		sb.WriteString(theme.WelcomeInfo.Render("  "+line))
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	// Tips
	tips := []string{
		"Type your message and press Enter to start",
		"/help for all commands",
		"/model to change the AI model",
		"/compact to compress context",
	}
	sb.WriteString(theme.WelcomeTip.Render("Quick start:"))
	sb.WriteString("\n")
	for _, tip := range tips {
		sb.WriteString(theme.WelcomeTip.Render("  • " + tip))
		sb.WriteString("\n")
	}

	return sb.String()
}
