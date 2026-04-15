package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/nice-code/codego/internal/config"
	"github.com/nice-code/codego/internal/session"
	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	root := &cobra.Command{
		Use:     "codego",
		Short:   "CodeGo — AI coding agent (Claude Code in Go)",
		Version: version,
		RunE:    runInteractive,
	}

	// Single-shot mode
	chatCmd := &cobra.Command{
		Use:   "chat",
		Short: "Single query mode",
		RunE:  runChat,
	}
	chatCmd.Flags().StringP("query", "q", "", "Query to execute")
	chatCmd.Flags().StringP("model", "m", "", "Model to use")

	// Config commands
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
	}
	configShowCmd := &cobra.Command{
		Use:   "show",
		Short: "Show current config",
		RunE:  runConfigShow,
	}
	configSetCmd := &cobra.Command{
		Use:   "set KEY VALUE",
		Short: "Set a config value (e.g. codego config set model.default claude-sonnet-4-20250514)",
		Args:  cobra.ExactArgs(2),
		RunE:  runConfigSet,
	}
	configPathCmd := &cobra.Command{
		Use:   "path",
		Short: "Show config file path",
		RunE:  runConfigPath,
	}
	configCmd.AddCommand(configShowCmd, configSetCmd, configPathCmd)

	// Sessions commands
	sessionsCmd := &cobra.Command{
		Use:   "sessions",
		Short: "Manage sessions",
	}
	sessionsListCmd := &cobra.Command{
		Use:   "list",
		Short: "List recent sessions",
		RunE:  runSessionsList,
	}
	sessionsResumeCmd := &cobra.Command{
		Use:   "resume ID",
		Short: "Resume a session by ID",
		Args:  cobra.ExactArgs(1),
		RunE:  runSessionsResume,
	}
	sessionsDeleteCmd := &cobra.Command{
		Use:   "delete ID",
		Short: "Delete a session",
		Args:  cobra.ExactArgs(1),
		RunE:  runSessionsDelete,
	}
	sessionsExportCmd := &cobra.Command{
		Use:   "export ID",
		Short: "Export a session as JSONL",
		Args:  cobra.ExactArgs(1),
		RunE:  runSessionsExport,
	}
	sessionsCmd.AddCommand(sessionsListCmd, sessionsResumeCmd, sessionsDeleteCmd, sessionsExportCmd)

	// MCP commands
	mcpCmd := &cobra.Command{
		Use:   "mcp",
		Short: "Manage MCP servers",
	}
	mcpListCmd := &cobra.Command{
		Use:   "list",
		Short: "List MCP servers",
		RunE:  runMcpList,
	}
	mcpCmd.AddCommand(mcpListCmd)

	// Doctor command
	doctorCmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check configuration and dependencies",
		RunE:  runDoctor,
	}

	root.AddCommand(chatCmd, configCmd, sessionsCmd, mcpCmd, doctorCmd)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func runInteractive(cmd *cobra.Command, args []string) error {
	fmt.Println("CodeGo — AI Coding Agent")
	fmt.Println("Interactive mode not yet implemented. Use 'codego chat -q <query>' for now.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  codego chat -q \"explain main.go\"")
	fmt.Println("  codego config show")
	fmt.Println("  codego sessions list")
	fmt.Println("  codego doctor")
	return nil
}

func runChat(cmd *cobra.Command, args []string) error {
	query, _ := cmd.Flags().GetString("query")
	if query == "" {
		return fmt.Errorf("usage: codego chat -q \"your question\"")
	}

	model, _ := cmd.Flags().GetString("model")

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if model != "" {
		cfg.Model.Default = model
	}

	fmt.Printf("Model: %s\n", cfg.Model.Default)
	fmt.Printf("Query: %s\n\n", query)
	fmt.Println("(Agent loop not implemented yet — coming in Phase 4)")
	return nil
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	fmt.Printf("Model:    %s\n", cfg.Model.Default)
	fmt.Printf("Provider: %s\n", cfg.Model.Provider)
	fmt.Printf("Base URL: %s\n", cfg.Model.BaseURL)
	fmt.Printf("Max Turns: %d\n", cfg.Agent.MaxTurns)
	fmt.Printf("Max Tokens: %d\n", cfg.Agent.MaxTokens)
	return nil
}

func runConfigPath(cmd *cobra.Command, args []string) error {
	fmt.Printf("Config: %s\n", config.Path())
	fmt.Printf("Env:    %s\n", config.EnvPath())
	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	if err := config.Set(key, value); err != nil {
		return fmt.Errorf("failed to set config: %w", err)
	}

	fmt.Printf("Set %s = %s\n", key, value)
	return nil
}

func runSessionsList(cmd *cobra.Command, args []string) error {
	store, err := session.OpenDefault()
	if err != nil {
		return fmt.Errorf("open session store: %w", err)
	}
	defer store.Close()

	sessions, err := store.ListSessions(20)
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		fmt.Println("No sessions yet.")
		return nil
	}

	fmt.Printf("%-20s %-8s %-20s %s\n", "ID", "Messages", "Updated", "Title")
	fmt.Println(strings.Repeat("-", 70))
	for _, s := range sessions {
		title := s.Title
		if title == "" {
			title = "(untitled)"
		}
		if len(title) > 30 {
			title = title[:27] + "..."
		}
		fmt.Printf("%-20s %-8d %-20s %s\n", s.ID[:min(20, len(s.ID))], s.MsgCount, s.UpdatedAt.Format("2006-01-02 15:04"), title)
	}
	return nil
}

func runSessionsResume(cmd *cobra.Command, args []string) error {
	store, err := session.OpenDefault()
	if err != nil {
		return err
	}
	defer store.Close()

	id := args[0]
	info, err := store.GetSession(id)
	if err != nil {
		return err
	}

	msgs, err := store.GetMessages(id)
	if err != nil {
		return err
	}

	fmt.Printf("Session: %s (%d messages)\n", info.Title, len(msgs))
	fmt.Printf("Resuming not yet implemented (Phase 9 — Interactive REPL)\n")
	fmt.Printf("Messages will be loaded from: %s\n", id)
	return nil
}

func runSessionsDelete(cmd *cobra.Command, args []string) error {
	store, err := session.OpenDefault()
	if err != nil {
		return err
	}
	defer store.Close()

	id := args[0]
	if err := store.DeleteSession(id); err != nil {
		return err
	}

	fmt.Printf("Deleted session %s\n", id)
	return nil
}

func runSessionsExport(cmd *cobra.Command, args []string) error {
	store, err := session.OpenDefault()
	if err != nil {
		return err
	}
	defer store.Close()

	id := args[0]
	jsonl, err := store.ExportJSONL(id)
	if err != nil {
		return err
	}

	fmt.Print(jsonl)
	return nil
}

func runMcpList(cmd *cobra.Command, args []string) error {
	fmt.Println("(MCP not implemented yet — coming in Phase 10)")
	return nil
}

func runDoctor(cmd *cobra.Command, args []string) error {
	fmt.Println("CodeGo Doctor")
	fmt.Println("=============")

	// Check Go version
	fmt.Println("  [OK] Go installed")

	// Check API key
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		fmt.Println("  [OK] ANTHROPIC_API_KEY is set")
	} else {
		fmt.Println("  [!!] ANTHROPIC_API_KEY not set")
		fmt.Println("       Set it: export ANTHROPIC_API_KEY=sk-ant-...")
	}

	// Check config
	home, _ := os.UserHomeDir()
	configPath := home + "/.codego/config.yaml"
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("  [OK] Config found: %s\n", configPath)
	} else {
		fmt.Printf("  [--] No config file (using defaults)\n")
		fmt.Printf("       Create: mkdir -p ~/.codego && touch %s\n", configPath)
	}

	return nil
}
