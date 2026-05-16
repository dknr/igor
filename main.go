package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	configPath := flag.String("config", "", "Config file path")
	flag.Parse()

	// Set up structured logging to stdout
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg, err := Load(*configPath)
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	if cfg.Grunt.APIKey == "" {
		slog.Error("No API key configured. Please set grunt.api_key in your config file.")
		os.Exit(1)
	}

	slog.Info("Igor starting up", "server", cfg.Grunt.ServerAddr, "user", cfg.Grunt.UserID)

	bot := NewBot(
		cfg.Grunt.ServerAddr,
		cfg.Grunt.UserID,
		cfg.Grunt.APIKey,
		cfg.Grunt.Mention,
		cfg.LLM.BaseURL,
		cfg.LLM.Model,
		cfg.LLM.APIKey,
		cfg.Igor.SystemPrompt,
		cfg.LLM.MaxHistory,
		cfg.LLM.HistoryTimeout,
	)

	if err := bot.Run(); err != nil {
		slog.Error("Bot error", "error", err)
		os.Exit(1)
	}

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nShutting down...")
	bot.Stop()
}