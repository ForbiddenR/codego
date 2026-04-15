package main

import (
	"fmt"
	"os"

	"github.com/nice-code/codego/internal/config"
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
	configPathCmd := &cobra.Command{
		Use:   "path",
		Short: "Show config file path",
		RunE:  runConfigPath,
	}
	configCmd.AddCommand(configShowCmd, configPathCmd)

	// Sessions commands
	sessionsCmd := &cobra.Command{
		Use:   "sessions",
		Short: "Manage sessions",
	}
	sessionsListCmd := &cobra.Command{
		Use:   "list",
		Short: "List sessions",
		RunE:  runSessionsList,
	}
	sessionsCmd.AddCommand(sessionsListCmd)

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
	home, _ := os.UserHomeDir()
	fmt.Printf("Config: %s/.codego/config.yaml\n", home)
	fmt.Printf("Env:    %s/.codego/.env\n", home)
	return nil
}

func runSessionsList(cmd *cobra.Command, args []string) error {
	fmt.Println("(Session storage not implemented yet — coming in Phase 6)")
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
