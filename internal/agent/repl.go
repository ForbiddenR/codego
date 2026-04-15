package agent

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/nice-code/codego/internal/commands"
	"github.com/nice-code/codego/internal/config"
	"github.com/nice-code/codego/internal/session"
	"github.com/nice-code/codego/internal/types"
)

// REPLConfig holds configuration for the REPL.
type REPLConfig struct {
	Agent       *Agent
	Session     *session.Store
	Config      *config.Config
	Input       io.Reader
	Output      io.Writer
	ShowBanner bool
}

// REPL is a plain-text read-eval-print loop.
type REPL struct {
	agent   *Agent
	store   *session.Store
	cfg     *config.Config
	cmds    *commands.Registry
	input   *bufio.Reader
	output  io.Writer
	session *session.SessionInfo
	usage   types.Usage
}

// NewREPL creates a new REPL.
func NewREPL(cfg REPLConfig) (*REPL, error) {
	input := cfg.Input
	if input == nil {
		input = os.Stdin
	}
	output := cfg.Output
	if output == nil {
		output = os.Stdout
	}

	r := &REPL{
		agent:   cfg.Agent,
		store:   cfg.Session,
		cfg:     cfg.Config,
		cmds:    commands.DefaultRegistry(),
		input:   bufio.NewReader(input),
		output:  output,
	}

	// Wire up command callbacks
	r.cmds = commands.DefaultRegistry()

	return r, nil
}

// Run starts the REPL loop.
func (r *REPL) Run(ctx context.Context) error {
	r.printBanner()

	// Create session
	sessionID := time.Now().Format("20060102_150405")
	if r.store != nil {
		if err := r.store.CreateSession(sessionID, ""); err != nil {
			return fmt.Errorf("create session: %w", err)
		}
		r.session = &session.SessionInfo{ID: sessionID}
	}

	for {
		r.printf("\n> ")
		line, err := r.input.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				r.printf("\n")
				return nil
			}
			return err
		}

		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}

		// Slash command?
		if strings.HasPrefix(input, "/") {
			if err := r.handleCommand(input); err != nil {
				if err == commands.ErrQuit {
					r.printf("Bye!\n")
					return nil
				}
				r.printf("Error: %v\n", err)
			}
			continue
		}

		// Run agent
		if err := r.handleInput(ctx, input); err != nil {
			r.printf("Error: %v\n", err)
		}
	}
}

// RunSingle executes a single query (non-interactive mode).
func (r *REPL) RunSingle(ctx context.Context, query string) error {
	r.setupCallbacks()
	return r.handleInput(ctx, query)
}

func (r *REPL) handleInput(ctx context.Context, input string) error {
	// Save user message to session
	if r.store != nil && r.session != nil {
		r.store.AppendMessage(r.session.ID, types.NewUserMessage(input))
		// Set title from first message
		if r.session.Title == "" {
			title := input
			if len(title) > 50 {
				title = title[:47] + "..."
			}
			r.store.UpdateTitle(r.session.ID, title)
			r.session.Title = title
		}
	}

	// Setup callbacks for streaming
	r.setupCallbacks()

	// Run agent
	result, err := r.agent.Run(ctx, input)
	if err != nil {
		return err
	}

	// Update usage
	r.usage = mergeUsage(r.usage, result.Usage)

	// Save assistant messages to session
	if r.store != nil && r.session != nil {
		msgs := r.agent.Messages()
		if len(msgs) >= 2 {
			// Save the last assistant message (and any tool messages)
			for i := len(msgs) - 1; i >= 1; i-- {
				msg := msgs[i]
				if msg.Role == types.RoleUser && !msg.HasToolCalls() {
					break // reached the user message, stop
				}
				if msg.Role == types.RoleAssistant {
					r.store.AppendMessage(r.session.ID, msg)
				}
			}
		}
	}

	return nil
}

func (r *REPL) handleCommand(input string) error {
	ctx := &commands.Context{
		Model:     r.agent.Model(),
		SessionID: "",
		MsgCount:  len(r.agent.Messages()),
		InputTok:  r.usage.InputTokens,
		OutputTok: r.usage.OutputTokens,
	}

	if r.session != nil {
		ctx.SessionID = r.session.ID
	}

	// Wire callbacks
	ctx.OnReset = func() {
		r.agent.Reset()
		r.usage = types.Usage{}
		r.printf("Conversation reset.\n")
	}
	ctx.OnModel = func(model string) error {
		// Would need to recreate agent with new model
		r.printf("Model change requires restart. Use: codego chat -m %s\n", model)
		return nil
	}
	ctx.OnCompact = func() error {
		cfg := DefaultCompressionConfig()
		msgs := r.agent.Messages()
		compressed := CompressMessages(msgs, cfg)
		r.agent.SetMessages(compressed)
		r.printf("Context compressed: %d → %d messages\n", len(msgs), len(compressed))
		return nil
	}
	ctx.OnSave = func(path string) error {
		if r.store != nil && r.session != nil {
			jsonl, err := r.store.ExportJSONL(r.session.ID)
			if err != nil {
				return err
			}
			return os.WriteFile(path, []byte(jsonl), 0o644)
		}
		return fmt.Errorf("no session to save")
	}

	err := r.cmds.Execute(input, ctx)

	// Handle /help specially (it's a no-op in registry, we print here)
	if input == "/help" {
		r.printHelp()
		return nil
	}

	return err
}

func (r *REPL) setupCallbacks() {
	r.agent.OnText = func(text string) {
		fmt.Fprint(r.output, text)
	}
	r.agent.OnThinking = func(text string) {
		// Don't show thinking in plain text mode
	}
	r.agent.OnToolStart = func(name string, _ types.ToolInput) {
		r.printf("\n  ⠋ Running %s...\n", name)
	}
	r.agent.OnToolEnd = func(name string, result *types.ToolResult) {
		if result.IsError {
			r.printf("  ✗ %s failed\n", name)
		} else {
			r.printf("  ✓ %s done\n", name)
		}
	}
}

func (r *REPL) printBanner() {
	if r.cfg != nil {
		r.printf("CodeGo — AI Coding Agent\n")
		r.printf("Model: %s\n", r.cfg.Model.Default)
	} else {
		r.printf("CodeGo — AI Coding Agent\n")
	}
	r.printf("Type /help for commands, /quit to exit\n")
}

func (r *REPL) printHelp() {
	r.printf("\nCommands:\n")
	for _, cmd := range r.cmds.List() {
		desc := cmd.Description
		usage := cmd.Usage
		if usage == "" {
			usage = "/" + cmd.Name
		}
		r.printf("  %-20s %s\n", usage, desc)
	}
}

func (r *REPL) printf(format string, args ...any) {
	fmt.Fprintf(r.output, format, args...)
}
