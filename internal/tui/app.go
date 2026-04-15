package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nice-code/codego/internal/agent"
	"github.com/nice-code/codego/internal/types"
)

// ─── States ───

type AppState int

const (
	StateInput AppState = iota
	StateThinking
	StateToolRunning
	StateResponse
	StateError
)

func (s AppState) String() string {
	switch s {
	case StateInput:
		return "input"
	case StateThinking:
		return "thinking"
	case StateToolRunning:
		return "tool_running"
	case StateResponse:
		return "response"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

// ─── Messages (tea.Msg types) ───

type TextDeltaMsg struct{ Text string }
type ThinkingDeltaMsg struct{ Text string }
type ToolStartMsg struct {
	Name  string
	Input map[string]interface{}
}
type ToolEndMsg struct {
	Name   string
	Result *types.ToolResult
}
type AgentDoneMsg struct{ Text string }
type AgentErrorMsg struct{ Err error }
type tickMsg struct{}

// ─── Model ───

type AppModel struct {
	state       AppState
	agent       *agent.Agent
	messages    []MessageView
	input       textarea.Model
	viewport    viewport.Model
	width       int
	height      int
	err         error
	activeTool  string
	initialized bool
}

// NewAppModel creates a new TUI app model.
func NewAppModel(a *agent.Agent) AppModel {
	in := textarea.New()
	in.Placeholder = "Send a message..."
	in.Focus()
	in.SetHeight(3)
	in.ShowLineNumbers = false
	in.CharLimit = 0

	vp := viewport.New(80, 20)

	return AppModel{
		state:    StateInput,
		agent:    a,
		input:    in,
		viewport: vp,
	}
}

// Init implements tea.Model.
func (m AppModel) Init() tea.Cmd {
	return textarea.Blink
}

// Update implements tea.Model.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		return m.handleResize(msg)

	case tea.KeyMsg:
		return m.handleKey(msg)

	case TextDeltaMsg:
		if len(m.messages) > 0 {
			last := &m.messages[len(m.messages)-1]
			last.AppendText(msg.Text)
			m.viewport.SetContent(m.renderMessages())
			m.viewport.GotoBottom()
		}
		return m, nil

	case ThinkingDeltaMsg:
		return m, nil // thinking handled differently

	case ToolStartMsg:
		m.state = StateToolRunning
		m.activeTool = msg.Name
		m.messages = append(m.messages, NewToolCallView(msg.Name, msg.Input))
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, nil

	case ToolEndMsg:
		if len(m.messages) > 0 {
			m.messages[len(m.messages)-1].SetResult(msg.Result)
		}
		m.state = StateResponse
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, listenForAgentEvents(m)

	case AgentDoneMsg:
		m.state = StateInput
		m.activeTool = ""
		m.input.Focus()
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, textarea.Blink

	case AgentErrorMsg:
		m.state = StateError
		m.err = msg.Err
		m.messages = append(m.messages, NewSystemMessageView(fmt.Sprintf("Error: %v", msg.Err)))
		m.state = StateInput
		m.input.Focus()
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, textarea.Blink

	case tickMsg:
		if m.state == StateToolRunning || m.state == StateThinking {
			return m, tick()
		}
		return m, nil
	}

	// Pass to input when in input state
	if m.state == StateInput {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	return m, nil
}

// View implements tea.Model.
func (m AppModel) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	var sections []string

	// 1. Status bar
	sections = append(sections, m.renderStatusBar())

	// 2. Messages viewport
	sections = append(sections, m.viewport.View())

	// 3. Spinner bar (when thinking or running tool)
	if m.state == StateThinking || m.state == StateToolRunning {
		sections = append(sections, m.renderSpinner())
	}

	// 4. Input area
	sections = append(sections, m.renderInput())

	// 5. Help hint
	sections = append(sections, m.renderHelpHint())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// ─── Key handling ───

func (m AppModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "enter":
		if m.state == StateInput {
			input := strings.TrimSpace(m.input.Value())
			if input == "" {
				return m, nil
			}

			m.input.Reset()
			m.messages = append(m.messages, NewUserMessageView(input))
			m.messages = append(m.messages, NewAssistantMessageView())
			m.state = StateThinking
			m.viewport.SetContent(m.renderMessages())
			m.viewport.GotoBottom()
			return m, runAgent(m, input)
		}

	case "ctrl+l":
		m.messages = nil
		m.viewport.SetContent("")
		m.viewport.GotoTop()
		return m, nil

	case "pgup":
		m.viewport.HalfViewUp()
		return m, nil

	case "pgdown":
		m.viewport.HalfViewDown()
		return m, nil
	}

	// Pass to input
	if m.state == StateInput {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m AppModel) handleResize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height

	// Calculate viewport height: total - status(1) - spinner(1) - input(5) - help(1)
	vpHeight := m.height - 8
	if vpHeight < 3 {
		vpHeight = 3
	}

	m.viewport.Width = m.width
	m.viewport.Height = vpHeight
	m.input.SetWidth(m.width - 4)

	m.viewport.SetContent(m.renderMessages())
	return m, nil
}

// ─── Rendering ───

func (m AppModel) renderStatusBar() string {
	modelName := "unknown"
	if m.agent != nil {
		modelName = m.agent.Model()
	}

	left := lipgloss.NewStyle().Bold(true).Render("CodeGo")
	center := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(modelName)
	right := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(
		fmt.Sprintf("%d msgs", len(m.messages)))

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(center) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}

	return lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Width(m.width).
		Render(left + strings.Repeat(" ", gap) + center + " " + right)
}

func (m AppModel) renderMessages() string {
	var sb strings.Builder
	for _, msg := range m.messages {
		rendered := msg.Render(m.width - 4)
		sb.WriteString(rendered)
		sb.WriteString("\n\n")
	}
	return sb.String()
}

func (m AppModel) renderSpinner() string {
	spinnerChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	idx := 0
	// Use a simple approach based on message count as a tick proxy
	if len(m.messages) > 0 {
		idx = len(m.messages) % len(spinnerChars)
	}

	label := "Thinking"
	if m.state == StateToolRunning {
		label = fmt.Sprintf("Running %s", m.activeTool)
	}

	style := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	return style.Render(fmt.Sprintf("  %s %s...", spinnerChars[idx], label))
}

func (m AppModel) renderInput() string {
	borderColor := lipgloss.Color("62")
	if m.state == StateInput {
		borderColor = lipgloss.Color("99")
	}

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(m.width - 4)

	return border.Render(m.input.View())
}

func (m AppModel) renderHelpHint() string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render("  enter send · ctrl+l clear · pgup/pgdn scroll · ctrl+c quit")
}

// ─── Agent bridge ───

// runAgent starts the agent in a goroutine and bridges callbacks to tea.Msg.
func runAgent(model AppModel, input string) tea.Cmd {
	return func() tea.Msg {
		msgCh := make(chan tea.Msg, 32)

		model.agent.OnText = func(text string) {
			msgCh <- TextDeltaMsg{Text: text}
		}
		model.agent.OnThinking = func(text string) {
			msgCh <- ThinkingDeltaMsg{Text: text}
		}
		model.agent.OnToolStart = func(name string, input types.ToolInput) {
			msgCh <- ToolStartMsg{Name: name, Input: input}
		}
		model.agent.OnToolEnd = func(name string, result *types.ToolResult) {
			msgCh <- ToolEndMsg{Name: name, Result: result}
		}

		go func() {
			defer close(msgCh)
			result, err := model.agent.Run(context.Background(), input)
			if err != nil {
				msgCh <- AgentErrorMsg{Err: err}
				return
			}
			msgCh <- AgentDoneMsg{Text: result.Text}
		}()

		// Block until first message
		return <-msgCh
	}
}

// listenForAgentEvents returns a cmd that reads the next event from the agent.
// This is called after each ToolEnd to continue draining events.
func listenForAgentEvents(_ AppModel) tea.Cmd {
	// The channel-based approach already handles this.
	// After ToolEnd, the goroutine continues and will eventually send
	// another ToolStart, ToolEnd, or AgentDone.
	// We just need a tick to keep the TUI responsive.
	return tick()
}

func tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}
