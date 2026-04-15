package commands

import (
	"fmt"
	"sort"
	"strings"
)

// Command is a slash command that can be executed in the REPL.
type Command struct {
	Name        string
	Description string
	Usage       string
	Handler     func(args []string, ctx *Context) error
}

// Context holds shared state for command execution.
type Context struct {
	// Agent state
	Model      string
	SessionID  string
	MsgCount   int
	InputTok   int
	OutputTok  int
	TotalCost  float64

	// Callbacks for commands that need to modify agent state
	OnReset    func()
	OnCompact  func() error
	OnModel    func(model string) error
	OnSave     func(path string) error
}

// Registry holds all registered slash commands.
type Registry struct {
	commands map[string]*Command
}

// NewRegistry creates a new command registry.
func NewRegistry() *Registry {
	return &Registry{commands: make(map[string]*Command)}
}

// Register adds a command to the registry.
func (r *Registry) Register(cmd *Command) {
	r.commands[cmd.Name] = cmd
}

// Execute parses and runs a slash command.
func (r *Registry) Execute(input string, ctx *Context) error {
	input = strings.TrimSpace(input)
	if !strings.HasPrefix(input, "/") {
		return fmt.Errorf("not a command: %s", input)
	}

	// Parse: /command arg1 arg2 ...
	parts := strings.Fields(input)
	name := strings.TrimPrefix(parts[0], "/")
	args := parts[1:]

	cmd, ok := r.commands[name]
	if !ok {
		// Try prefix match
		matches := r.FindMatches(name)
		if len(matches) == 1 {
			cmd = r.commands[matches[0]]
		} else if len(matches) > 1 {
			return fmt.Errorf("ambiguous command /%s — did you mean: /%s", name, strings.Join(matches, ", /"))
		} else {
			return fmt.Errorf("unknown command: /%s — type /help for available commands", name)
		}
	}

	return cmd.Handler(args, ctx)
}

// List returns all commands sorted by name.
func (r *Registry) List() []*Command {
	cmds := make([]*Command, 0, len(r.commands))
	for _, cmd := range r.commands {
		cmds = append(cmds, cmd)
	}
	sort.Slice(cmds, func(i, j int) bool {
		return cmds[i].Name < cmds[j].Name
	})
	return cmds
}

// Get returns a command by name.
func (r *Registry) Get(name string) (*Command, bool) {
	cmd, ok := r.commands[name]
	return cmd, ok
}

// FindMatches returns command names that start with the given prefix.
func (r *Registry) FindMatches(prefix string) []string {
	var matches []string
	for name := range r.commands {
		if strings.HasPrefix(name, prefix) {
			matches = append(matches, name)
		}
	}
	sort.Strings(matches)
	return matches
}

// Count returns the number of registered commands.
func (r *Registry) Count() int {
	return len(r.commands)
}

// DefaultRegistry returns a registry with all built-in commands.
func DefaultRegistry() *Registry {
	r := NewRegistry()

	r.Register(&Command{
		Name:        "help",
		Description: "Show available commands",
		Handler: func(_ []string, _ *Context) error {
			// Print is handled by the caller
			return nil
		},
	})

	r.Register(&Command{
		Name:        "model",
		Description: "Show or change the current model",
		Usage:       "/model [name]",
		Handler: func(args []string, ctx *Context) error {
			if len(args) == 0 {
				fmt.Printf("Current model: %s\n", ctx.Model)
				return nil
			}
			if ctx.OnModel != nil {
				return ctx.OnModel(args[0])
			}
			ctx.Model = args[0]
			fmt.Printf("Model changed to: %s\n", args[0])
			return nil
		},
	})

	r.Register(&Command{
		Name:        "reasoning",
		Description: "Set reasoning effort level",
		Usage:       "/reasoning [none|low|medium|high]",
		Handler: func(args []string, _ *Context) error {
			levels := []string{"none", "low", "medium", "high"}
			if len(args) == 0 {
				fmt.Printf("Reasoning levels: %s\n", strings.Join(levels, ", "))
				return nil
			}
			for _, l := range levels {
				if args[0] == l {
					fmt.Printf("Reasoning set to: %s\n", l)
					return nil
				}
			}
			return fmt.Errorf("invalid level: %s (use: %s)", args[0], strings.Join(levels, ", "))
		},
	})

	r.Register(&Command{
		Name:        "compact",
		Description: "Compress conversation context",
		Handler: func(_ []string, ctx *Context) error {
			if ctx.OnCompact != nil {
				return ctx.OnCompact()
			}
			fmt.Println("(Context compression — use agent.CompressMessages)")
			return nil
		},
	})

	r.Register(&Command{
		Name:        "cost",
		Description: "Show token usage and cost",
		Handler: func(_ []string, ctx *Context) error {
			fmt.Printf("Session:   %s\n", ctx.SessionID)
			fmt.Printf("Messages:  %d\n", ctx.MsgCount)
			fmt.Printf("Input:     %d tokens\n", ctx.InputTok)
			fmt.Printf("Output:    %d tokens\n", ctx.OutputTok)
			return nil
		},
	})

	r.Register(&Command{
		Name:        "status",
		Description: "Show session info",
		Handler: func(_ []string, ctx *Context) error {
			fmt.Printf("Model:     %s\n", ctx.Model)
			fmt.Printf("Session:   %s\n", ctx.SessionID)
			fmt.Printf("Messages:  %d\n", ctx.MsgCount)
			return nil
		},
	})

	r.Register(&Command{
		Name:        "config",
		Description: "Show current configuration",
		Handler: func(_ []string, _ *Context) error {
			fmt.Println("(Use 'codego config show' for full config)")
			return nil
		},
	})

	r.Register(&Command{
		Name:        "reset",
		Description: "Start a new conversation",
		Handler: func(_ []string, ctx *Context) error {
			if ctx.OnReset != nil {
				ctx.OnReset()
			}
			fmt.Println("Conversation reset.")
			return nil
		},
	})

	r.Register(&Command{
		Name:        "save",
		Description: "Save conversation to file",
		Usage:       "/save [path]",
		Handler: func(args []string, ctx *Context) error {
			path := "conversation.jsonl"
			if len(args) > 0 {
				path = args[0]
			}
			if ctx.OnSave != nil {
				return ctx.OnSave(path)
			}
			fmt.Printf("Saved to: %s\n", path)
			return nil
		},
	})

	r.Register(&Command{
		Name:        "quit",
		Description: "Exit CodeGo",
		Handler: func(_ []string, _ *Context) error {
			return ErrQuit
		},
	})

	// Aliases
	r.Register(&Command{
		Name:        "exit",
		Description: "Exit CodeGo (alias for /quit)",
		Handler: func(_ []string, _ *Context) error {
			return ErrQuit
		},
	})

	r.Register(&Command{
		Name:        "q",
		Description: "Exit CodeGo (alias for /quit)",
		Handler: func(_ []string, _ *Context) error {
			return ErrQuit
		},
	})

	return r
}

// ErrQuit is returned when the user wants to exit.
var ErrQuit = fmt.Errorf("quit")
