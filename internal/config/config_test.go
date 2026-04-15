package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

// resetViper clears viper state between tests.
func resetViper() {
	viper.Reset()
}

func TestLoad_Defaults(t *testing.T) {
	resetViper()
	// Point config dir to a temp dir with no config file
	tmp := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmp)
	defer os.Setenv("HOME", origHome)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Model.Default != "claude-sonnet-4-20250514" {
		t.Errorf("model.default = %q, want %q", cfg.Model.Default, "claude-sonnet-4-20250514")
	}
	if cfg.Model.Provider != "anthropic" {
		t.Errorf("model.provider = %q, want %q", cfg.Model.Provider, "anthropic")
	}
	if cfg.Model.BaseURL != "https://api.anthropic.com" {
		t.Errorf("model.base_url = %q, want %q", cfg.Model.BaseURL, "https://api.anthropic.com")
	}
	if cfg.Agent.MaxTurns != 50 {
		t.Errorf("agent.max_turns = %d, want 50", cfg.Agent.MaxTurns)
	}
	if cfg.Agent.MaxTokens != 8192 {
		t.Errorf("agent.max_tokens = %d, want 8192", cfg.Agent.MaxTokens)
	}
	if !cfg.Memory.Enabled {
		t.Error("memory.enabled should be true")
	}
}

func TestLoad_FromFile(t *testing.T) {
	resetViper()
	tmp := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmp)
	defer os.Setenv("HOME", origHome)

	// Write a config file
	configDir := filepath.Join(tmp, ".codego")
	os.MkdirAll(configDir, 0o755)
	os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(`
model:
  default: claude-opus-4-20250514
  provider: custom
  base_url: http://localhost:8080
agent:
  max_turns: 100
  max_tokens: 16384
`), 0o644)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Model.Default != "claude-opus-4-20250514" {
		t.Errorf("model.default = %q", cfg.Model.Default)
	}
	if cfg.Model.Provider != "custom" {
		t.Errorf("model.provider = %q", cfg.Model.Provider)
	}
	if cfg.Agent.MaxTurns != 100 {
		t.Errorf("max_turns = %d", cfg.Agent.MaxTurns)
	}
	if cfg.Agent.MaxTokens != 16384 {
		t.Errorf("max_tokens = %d", cfg.Agent.MaxTokens)
	}
}

func TestSet_And_Load(t *testing.T) {
	resetViper()
	tmp := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmp)
	defer os.Setenv("HOME", origHome)

	// Set a value
	err := Set("model.default", "gpt-4o")
	if err != nil {
		t.Fatalf("Set() error: %v", err)
	}

	// Verify file was created
	configPath := filepath.Join(tmp, ".codego", "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config file should have been created")
	}

	// Read file content
	content, _ := os.ReadFile(configPath)
	if len(content) == 0 {
		t.Error("config file should not be empty")
	}
}

func TestSetAll(t *testing.T) {
	resetViper()
	tmp := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmp)
	defer os.Setenv("HOME", origHome)

	err := SetAll(map[string]string{
		"model.default":    "test-model",
		"agent.max_tokens": "4096",
		"memory.enabled":   "false",
	})
	if err != nil {
		t.Fatalf("SetAll() error: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(tmp, ".codego", "config.yaml"))
	s := string(content)
	if s == "" {
		t.Fatal("config file empty")
	}
}

func TestParseValue(t *testing.T) {
	tests := []struct {
		input string
		want  interface{}
	}{
		{"true", true},
		{"false", false},
		{"42", 42},
		{"3.14", 3.14},
		{"hello", "hello"},
		{"claude-sonnet-4-20250514", "claude-sonnet-4-20250514"},
	}
	for _, tt := range tests {
		got := parseValue(tt.input)
		if got != tt.want {
			t.Errorf("parseValue(%q) = %v (%T), want %v (%T)", tt.input, got, got, tt.want, tt.want)
		}
	}
}

func TestDir(t *testing.T) {
	d := Dir()
	if d == "" {
		t.Error("Dir() should not be empty")
	}
	if filepath.Base(d) != ".codego" {
		t.Errorf("Dir() base = %q, want .codego", filepath.Base(d))
	}
}

func TestPath(t *testing.T) {
	p := Path()
	if filepath.Base(p) != "config.yaml" {
		t.Errorf("Path() base = %q, want config.yaml", filepath.Base(p))
	}
}

func TestEnvPath(t *testing.T) {
	p := EnvPath()
	if filepath.Base(p) != ".env" {
		t.Errorf("EnvPath() base = %q, want .env", filepath.Base(p))
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	resetViper()
	tmp := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmp)
	defer os.Setenv("HOME", origHome)

	origKey := os.Getenv("ANTHROPIC_API_KEY")
	os.Setenv("ANTHROPIC_API_KEY", "sk-test-key-123")
	defer os.Setenv("ANTHROPIC_API_KEY", origKey)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Model.APIKey != "sk-test-key-123" {
		t.Errorf("api_key = %q, want %q", cfg.Model.APIKey, "sk-test-key-123")
	}
}
