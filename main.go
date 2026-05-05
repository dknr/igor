package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	configPath := flag.String("config", "", "Config file path")
	flag.Parse()

	cfg, err := Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	bot := NewBot(
		cfg.Grunt.ServerAddr,
		cfg.Grunt.UserID,
		cfg.Grunt.Mention,
		cfg.LLM.BaseURL,
		cfg.LLM.Model,
		cfg.LLM.APIKey,
		cfg.Igor.SystemPrompt,
	)

	if err := bot.Run(); err != nil {
		log.Fatalf("Bot error: %v", err)
	}

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nShutting down...")
	bot.Stop()
}