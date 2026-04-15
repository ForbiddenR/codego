package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/go-viper/mapstructure/v2"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config is the top-level configuration structure.
type Config struct {
	Model  ModelConfig  `mapstructure:"model"`
	Agent  AgentConfig  `mapstructure:"agent"`
	MCP    MCPConfig    `mapstructure:"mcp"`
	Memory MemoryConfig `mapstructure:"memory"`
}

// ModelConfig holds model-related configuration.
type ModelConfig struct {
	Default  string `mapstructure:"default"`
	Provider string `mapstructure:"provider"`
	BaseURL  string `mapstructure:"base_url"`
	APIKey   string `mapstructure:"api_key"`
}

// AgentConfig holds agent behavior configuration.
type AgentConfig struct {
	MaxTurns  int `mapstructure:"max_turns"`
	MaxTokens int `mapstructure:"max_tokens"`
}

// MCPConfig holds MCP server configuration.
type MCPConfig struct {
	Servers map[string]MCPServer `mapstructure:"servers"`
}

// MCPServer is a single MCP server definition.
type MCPServer struct {
	Command string   `mapstructure:"command"`
	Args    []string `mapstructure:"args"`
}

// MemoryConfig holds memory system configuration.
type MemoryConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

// Dir returns the config directory path.
func Dir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".codego")
}

// Path returns the config file path.
func Path() string {
	return filepath.Join(Dir(), "config.yaml")
}

// EnvPath returns the .env file path.
func EnvPath() string {
	return filepath.Join(Dir(), ".env")
}

// Load reads configuration from file, environment, and defaults.
func Load() (*Config, error) {
	configDir := Dir()

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configDir)

	// Defaults
	viper.SetDefault("model.default", "claude-sonnet-4-20250514")
	viper.SetDefault("model.provider", "anthropic")
	viper.SetDefault("model.base_url", "https://api.anthropic.com")
	viper.SetDefault("agent.max_turns", 50)
	viper.SetDefault("agent.max_tokens", 8192)
	viper.SetDefault("memory.enabled", true)

	// Load .env file (ignore error if missing)
	godotenv.Load(EnvPath())

	// Environment variable overrides
	viper.SetEnvPrefix("CODEGO")
	viper.AutomaticEnv()

	// Read config file (ignore error if missing)
	_ = viper.ReadInConfig()

	// API key from env
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		viper.Set("model.api_key", apiKey)
	}

	var cfg Config
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           &cfg,
		WeaklyTypedInput: true,
		TagName:          "mapstructure",
	})
	if err != nil {
		return nil, err
	}
	if err := decoder.Decode(viper.AllSettings()); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Set writes a single key-value pair to the config file.
// Creates the config file if it doesn't exist.
func Set(key, value string) error {
	configDir := Dir()
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	viper.SetConfigFile(Path())
	_ = viper.ReadInConfig() // ignore if missing

	// Auto-detect type
	typed := parseValue(value)
	viper.Set(key, typed)

	return viper.WriteConfig()
}

// SetAll writes multiple key-value pairs at once.
func SetAll(pairs map[string]string) error {
	configDir := Dir()
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	viper.SetConfigFile(Path())
	_ = viper.ReadInConfig()

	for key, value := range pairs {
		viper.Set(key, parseValue(value))
	}

	return viper.WriteConfig()
}

// Get returns the current value for a config key.
func Get(key string) interface{} {
	return viper.Get(key)
}

// GetAll returns all config values.
func GetAll() map[string]interface{} {
	return viper.AllSettings()
}

// parseValue attempts to parse a string value into its proper type.
func parseValue(value string) interface{} {
	// Bool
	if b, err := strconv.ParseBool(value); err == nil {
		return b
	}
	// Int
	if i, err := strconv.Atoi(value); err == nil {
		return i
	}
	// Float
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return f
	}
	// String
	return value
}
