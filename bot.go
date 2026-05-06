package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"grunt"
	"igor/llm"
)

// Bot represents the igor chatbot.
type Bot struct {
	client     *grunt.Client
	password   string
	mention    string
	llmClient  *llm.Client
	historyMgr *HistoryManager
}

// NewBot creates a new Bot instance.
func NewBot(serverAddr, userID, password, mention, llmBaseURL, llmModel, llmAPIKey, systemPrompt string, maxHistory int, historyTimeout time.Duration) *Bot {
	return &Bot{
		client: grunt.NewClient(serverAddr, userID),
		password: password,
		mention: mention,
		llmClient: llm.NewClient(llmBaseURL, llmModel, llmAPIKey, systemPrompt),
		historyMgr: NewHistoryManager(maxHistory, historyTimeout),
	}
}

// Run starts the bot, connecting to the server and listening for messages.
func (b *Bot) Run() error {
	// Register the user if not exists (handles 409 Conflict gracefully)
	slog.Info("Attempting registration", "user", b.client.UserID)
	if err := b.client.Register(b.password); err != nil {
		if strings.Contains(err.Error(), "409") {
			slog.Info("User already registered", "user", b.client.UserID)
		} else {
			slog.Error("Registration failed", "error", err)
		}
	} else {
		slog.Info("Registration successful", "user", b.client.UserID)
	}

	slog.Info("Attempting login", "user", b.client.UserID)
	if err := b.client.Login(b.password); err != nil {
		return fmt.Errorf("failed to login: %w", err)
	}
	slog.Info("Login successful", "user", b.client.UserID)

	slog.Info("Attempting WebSocket connection", "server", b.client.ServerAddr)
	if err := b.client.Connect(); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	slog.Info("WebSocket connection established")

	messages := b.client.StartListening()
	if messages == nil {
		return fmt.Errorf("failed to start listening")
	}

	slog.Info("Igor is listening", "mention", b.mention)

	for msgBytes := range messages {
		var broadcast grunt.Broadcast
		if err := json.Unmarshal(msgBytes, &broadcast); err != nil {
			slog.Warn("Error unmarshaling message", "error", err)
			continue
		}

		slog.Info("Received message", "user", broadcast.UserID, "content", broadcast.Content)

		// Add ALL messages to history (not just @mentions)
		b.historyMgr.AddMessage(llm.ChatMessage{
			Role:    "user",
			Content: fmt.Sprintf("%s: %s", broadcast.UserID, broadcast.Content),
		})

		if strings.Contains(broadcast.Content, b.mention) {
			// Extract the message content after the mention
			prompt := extractPrompt(broadcast.Content, b.mention)
			if prompt == "" {
				continue
			}

			slog.Info("Mention detected", "prompt", prompt)

			// Generate response with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			slog.Info("Calling LLM", "prompt", prompt)
			response, err := b.llmClient.Complete(ctx, prompt, b.historyMgr.Messages())
			cancel()

			if err != nil {
				slog.Error("LLM error", "error", err)
				continue
			}

			slog.Info("LLM response received", "response", response)

			// Add response to history
			b.historyMgr.AddMessage(llm.ChatMessage{
				Role:    "assistant",
				Content: fmt.Sprintf("%s: %s", b.client.UserID, response),
			})

			// Post response to server
			if err := b.client.SendMessage(response); err != nil {
				slog.Error("Failed to send response", "error", err)
			} else {
				slog.Info("Response sent to server")
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