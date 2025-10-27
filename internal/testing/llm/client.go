package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// truncateForError truncates response bodies in error messages to prevent
// logging sensitive information like API keys.
func truncateForError(body []byte) string {
	s := string(body)
	if len(s) > 200 {
		return s[:200] + "... (truncated)"
	}
	return s
}

// Client is a unified interface for calling different LLM providers.
type Client interface {
	// Generate sends a prompt to the LLM and returns the response.
	Generate(ctx context.Context, prompt string) (string, error)

	// Provider returns the provider configuration for this client.
	Provider() ProviderConfig
}

// NewClient creates a new LLM client for the given provider configuration.
func NewClient(config ProviderConfig) (Client, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid provider config: %w", err)
	}

	switch config.Type {
	case ProviderClaude:
		return &claudeClient{
			config:     config,
			httpClient: &http.Client{Timeout: config.Timeout},
			baseURL:    "https://api.anthropic.com/v1/messages",
		}, nil
	case ProviderOpenAI:
		return &openAIClient{
			config:     config,
			httpClient: &http.Client{Timeout: config.Timeout},
			baseURL:    "https://api.openai.com/v1/chat/completions",
		}, nil
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", config.Type)
	}
}

// claudeClient implements the Client interface for Anthropic's Claude API.
type claudeClient struct {
	config     ProviderConfig
	httpClient *http.Client
	baseURL    string
}

// claudeRequest represents the request body for Claude's messages API.
type claudeRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Messages  []claudeMessage `json:"messages"`
}

// claudeMessage represents a message in the Claude API.
type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// claudeResponse represents the response from Claude's API.
type claudeResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model       string `json:"model"`
	StopReason  string `json:"stop_reason"`
	Usage       struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// Generate sends a prompt to Claude and returns the response.
func (c *claudeClient) Generate(ctx context.Context, prompt string) (string, error) {
	req := claudeRequest{
		Model:     c.config.Model,
		MaxTokens: 4096,
		Messages: []claudeMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(backoff):
			}
		}

		response, err := c.makeRequest(ctx, req)
		if err == nil {
			return response, nil
		}

		lastErr = err

		// Don't retry on context cancellation
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
	}

	return "", fmt.Errorf("failed after %d attempts: %w", c.config.MaxRetries+1, lastErr)
}

// makeRequest performs a single API request to Claude.
func (c *claudeClient) makeRequest(ctx context.Context, req claudeRequest) (string, error) {
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.config.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, truncateForError(body))
	}

	var claudeResp claudeResponse
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(claudeResp.Content) == 0 {
		return "", fmt.Errorf("no content in response")
	}

	return claudeResp.Content[0].Text, nil
}

// Provider returns the provider configuration.
func (c *claudeClient) Provider() ProviderConfig {
	return c.config
}

// openAIClient implements the Client interface for OpenAI's API.
type openAIClient struct {
	config     ProviderConfig
	httpClient *http.Client
	baseURL    string
}

// openAIRequest represents the request body for OpenAI's chat completions API.
type openAIRequest struct {
	Model    string            `json:"model"`
	Messages []openAIMessage   `json:"messages"`
	MaxTokens int              `json:"max_tokens,omitempty"`
}

// openAIMessage represents a message in the OpenAI API.
type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openAIResponse represents the response from OpenAI's API.
type openAIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// Generate sends a prompt to OpenAI and returns the response.
func (c *openAIClient) Generate(ctx context.Context, prompt string) (string, error) {
	req := openAIRequest{
		Model: c.config.Model,
		Messages: []openAIMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens: 4096,
	}

	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(backoff):
			}
		}

		response, err := c.makeRequest(ctx, req)
		if err == nil {
			return response, nil
		}

		lastErr = err

		// Don't retry on context cancellation
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
	}

	return "", fmt.Errorf("failed after %d attempts: %w", c.config.MaxRetries+1, lastErr)
}

// makeRequest performs a single API request to OpenAI.
func (c *openAIClient) makeRequest(ctx context.Context, req openAIRequest) (string, error) {
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, truncateForError(body))
	}

	var openAIResp openAIResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return openAIResp.Choices[0].Message.Content, nil
}

// Provider returns the provider configuration.
func (c *openAIClient) Provider() ProviderConfig {
	return c.config
}
