package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

// Config holds the application configuration.
type Config struct {
	Grunt GruntConfig `mapstructure:"grunt"`
	LLM   LLMConfig   `mapstructure:"llm"`
	Igor  IgorConfig  `mapstructure:"igor"`
}

// GruntConfig holds grunt server configuration.
type GruntConfig struct {
	ServerAddr string `mapstructure:"server_addr"`
	UserID     string `mapstructure:"user_id"`
	Mention    string `mapstructure:"mention"`
}

// LLMConfig holds LLM provider configuration.
type LLMConfig struct {
	BaseURL        string        `mapstructure:"base_url"`
	APIKey         string        `mapstructure:"api_key"`
	Model          string        `mapstructure:"model"`
	MaxHistory     int           `mapstructure:"max_history"`
	HistoryTimeout time.Duration `mapstructure:"history_timeout"`
}

// IgorConfig holds igor-specific configuration.
type IgorConfig struct {
	SystemPrompt string `mapstructure:"system_prompt"`
}

// Load reads configuration from file and environment variables.
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("grunt.server_addr", "http://localhost:54765")
	v.SetDefault("grunt.user_id", "igor")
	v.SetDefault("grunt.mention", "@igor")
	v.SetDefault("llm.base_url", "http://localhost:8080")
	v.SetDefault("llm.api_key", "")
	v.SetDefault("llm.model", "llama-3.2-3b-instruct")
	v.SetDefault("llm.max_history", 100)
	v.SetDefault("llm.history_timeout", "15m")
	v.SetDefault("igor.system_prompt", "You are igor, a simple LLM agent. Respond succinctly, in a gruff and terse manner.")

	// Expand $HOME for the default config path
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}

	// Configure viper to read config file
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(filepath.Join(home, ".config", "igor"))
	}

	// Read config file (optional - continues with defaults if not found)
	if err := v.ReadInConfig(); err != nil {
		fmt.Printf("DEBUG: No config file found, using defaults: %v\n", err)
	} else {
		fmt.Printf("DEBUG: Config file used: %s\n", v.ConfigFileUsed())
	}

	// Allow environment variables to override (IGOR_ prefix)
	v.SetEnvPrefix("igor")
	v.AutomaticEnv()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return &cfg, nil
}