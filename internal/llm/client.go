package llm

import (
	"context"
	"fmt"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

const (
	defaultBaseURL = "https://api.tokenfactory.nebius.com/v1/"
	defaultModel   = "openai/gpt-oss-120b"
)

// Config holds client settings.
type Config struct {
	APIKey      string
	BaseURL     string
	Model       string
	Timeout     time.Duration
	Temperature float32
	MaxTokens   int
}

// Client wraps the Nebius/OpenAI-compatible API.
type Client struct {
	api         *openai.Client
	model       string
	temperature float32
	maxTokens   int
	timeout     time.Duration
}

// New creates a client from config.
func New(cfg Config) (*Client, error) {
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" {
		return nil, fmt.Errorf("llm: API key is required")
	}
	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = defaultModel
	}
	temp := cfg.Temperature
	if temp < 0 {
		temp = 0
	}
	maxTokens := cfg.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 800
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}

	openaiCfg := openai.DefaultConfig(apiKey)
	openaiCfg.BaseURL = baseURL

	return &Client{
		api:         openai.NewClientWithConfig(openaiCfg),
		model:       model,
		temperature: temp,
		maxTokens:   maxTokens,
		timeout:     timeout,
	}, nil
}

// Complete sends a single-shot prompt and returns the response text.
func (c *Client) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if c == nil {
		return "", fmt.Errorf("llm: client is nil")
	}
	if systemPrompt == "" || userPrompt == "" {
		return "", fmt.Errorf("llm: prompts must be provided")
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req := openai.ChatCompletionRequest{
		Model: c.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userPrompt},
		},
		MaxTokens:   c.maxTokens,
		Temperature: c.temperature,
	}

	resp, err := c.api.CreateChatCompletion(ctxWithTimeout, req)
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("llm: empty response")
	}
	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}
