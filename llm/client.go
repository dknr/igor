package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Client is an LLM client for the llama.cpp API.
type Client struct {
	BaseURL      string
	Model        string
	APIKey       string
	SystemPrompt string
	HTTPClient   *http.Client
}

// NewClient creates a new LLM client.
func NewClient(baseURL, model, apiKey, systemPrompt string) *Client {
	return &Client{
		BaseURL:      baseURL,
		Model:        model,
		APIKey:       apiKey,
		SystemPrompt: systemPrompt,
		HTTPClient:   &http.Client{},
	}
}

// ChatMessage represents a message in the chat history.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest is the request body for the chat completions endpoint.
type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
}

// ChatResponse is the response from the chat completions endpoint.
type ChatResponse struct {
	Choices []Choice `json:"choices"`
}

// Choice represents a single choice in the response.
type Choice struct {
	Message Message `json:"message"`
}

// Message represents the message content in a choice.
type Message struct {
	Content string `json:"content"`
}

// Complete sends a prompt to the LLM and returns the generated text.
func (c *Client) Complete(ctx context.Context, prompt string) (string, error) {
	if c.HTTPClient == nil {
		c.HTTPClient = &http.Client{}
	}

	url := fmt.Sprintf("%s/chat/completions", c.BaseURL)

	messages := []ChatMessage{
		{Role: "system", Content: c.SystemPrompt},
		{Role: "user", Content: prompt},
	}

	req := ChatRequest{
		Model:    c.Model,
		Messages: messages,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	if c.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return chatResp.Choices[0].Message.Content, nil
}