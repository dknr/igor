package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"grunt"
	"igor/llm"
)

// Bot represents the igor chatbot.
type Bot struct {
	client     *grunt.Client
	mention    string
	llmClient  *llm.Client
}

// NewBot creates a new Bot instance.
func NewBot(serverAddr, userID, mention, llmBaseURL, llmModel, llmAPIKey, systemPrompt string) *Bot {
	return &Bot{
		client: grunt.NewClient(serverAddr, userID),
		mention: mention,
		llmClient: llm.NewClient(llmBaseURL, llmModel, llmAPIKey, systemPrompt),
	}
}

// Run starts the bot, connecting to the server and listening for messages.
func (b *Bot) Run() error {
	if err := b.client.Register(); err != nil {
		return fmt.Errorf("failed to register: %w", err)
	}

	if err := b.client.Connect(); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	messages := b.client.StartListening()
	if messages == nil {
		return fmt.Errorf("failed to start listening")
	}

	log.Printf("Igor is listening for %s...", b.mention)

	for msgBytes := range messages {
		var broadcast grunt.Broadcast
		if err := json.Unmarshal(msgBytes, &broadcast); err != nil {
			log.Printf("Error unmarshaling message: %v", err)
			continue
		}

		if strings.Contains(broadcast.Content, b.mention) {
			// Extract the message content after the mention
			prompt := extractPrompt(broadcast.Content, b.mention)
			if prompt == "" {
				continue
			}

			// Generate response with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			response, err := b.llmClient.Complete(ctx, prompt)
			cancel()

			if err != nil {
				log.Printf("LLM error: %v", err)
				continue
			}

			// Post response to server
			if err := b.client.SendMessage(response); err != nil {
				log.Printf("Failed to send response: %v", err)
			}
		}
	}

	return nil
}

// Stop shuts down the bot gracefully.
func (b *Bot) Stop() {
	b.client.StopListening()
	b.client.Close()
}

// extractPrompt removes the mention prefix from the message content.
func extractPrompt(content, mention string) string {
	idx := strings.Index(content, mention)
	if idx == -1 {
		return ""
	}
	// Return everything after the mention
	prompt := strings.TrimSpace(content[idx+len(mention):])
	return prompt
}