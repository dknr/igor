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
		var envelope struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(msgBytes, &envelope); err != nil {
			slog.Warn("Error unmarshaling message", "error", err)
			continue
		}

		switch envelope.Type {
		case "message":
			var broadcast grunt.Broadcast
			if err := json.Unmarshal(msgBytes, &broadcast); err != nil {
				slog.Warn("Error unmarshaling broadcast", "error", err)
				continue
			}

			slog.Info("Received message", "user", broadcast.UserID, "content", broadcast.Content)

			// Ignore own messages to prevent infinite loops
			if broadcast.UserID == b.client.UserID {
				continue
			}

			// Add ALL messages to history (not just @mentions)
			b.historyMgr.AddMessage(llm.ChatMessage{
				Role:    "user",
				Content: fmt.Sprintf("%s: %s", broadcast.UserID, broadcast.Content),
			})

			if strings.Contains(broadcast.Content, b.mention) {
				slog.Info("Mention detected", "prompt", broadcast.Content)

				// Generate response with timeout
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
				slog.Info("Calling LLM", "prompt", broadcast.Content)
				response, promptTokens, completionTokens, totalTokens, err := b.llmClient.Complete(ctx, broadcast.Content, b.historyMgr.Messages())
				cancel()

				if err != nil {
					slog.Error("LLM error", "error", err)
					continue
				}

				slog.Info("LLM response received", "response", response, "prompt_tokens", promptTokens, "completion_tokens", completionTokens, "total_tokens", totalTokens)

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
		case "event":
			var evt grunt.Event
			if err := json.Unmarshal(msgBytes, &evt); err == nil && evt.Event != "" {
				slog.Info("Event received", "event", evt.Event, "user", evt.UserID, "client_id", evt.ClientID)
			}
		case "error":
			var serr grunt.Error
			if err := json.Unmarshal(msgBytes, &serr); err == nil && serr.Message != "" {
				slog.Error("Server error", "message", serr.Message)
			}
		default:
			slog.Warn("Unknown message type", "type", envelope.Type)
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