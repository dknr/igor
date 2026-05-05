package main

import (
	"time"

	"igor/llm"
)

// HistoryManager maintains conversation context.
type HistoryManager struct {
	messages     []llm.ChatMessage
	maxMessages  int
	lastActivity time.Time
	timeout      time.Duration
}

// NewHistoryManager creates a new history manager.
func NewHistoryManager(maxMessages int, timeout time.Duration) *HistoryManager {
	return &HistoryManager{
		messages:     make([]llm.ChatMessage, 0, maxMessages),
		maxMessages:  maxMessages,
		lastActivity: time.Now(),
		timeout:      timeout,
	}
}

// AddMessage adds a message to history, clearing if timeout exceeded.
func (h *HistoryManager) AddMessage(msg llm.ChatMessage) {
	// Check for timeout
	if time.Since(h.lastActivity) > h.timeout {
		h.messages = h.messages[:0] // clear history
	}
	h.lastActivity = time.Now()
	h.messages = append(h.messages, msg)

	// Trim to max
	if len(h.messages) > h.maxMessages {
		h.messages = h.messages[len(h.messages)-h.maxMessages:]
	}
}

// Messages returns the current history.
func (h *HistoryManager) Messages() []llm.ChatMessage {
	return h.messages
}