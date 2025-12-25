package embedding

import (
	"context"
	"fmt"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

const defaultModel = "Qwen/Qwen3-Embedding-8B"

type Client struct {
	api *openai.Client
}

func NewClient(apiKey, baseURL string) (*Client, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("NEBIUS_API_KEY not set")
	}
	cfg := openai.DefaultConfig(apiKey)
	if baseURL != "" {
		cfg.BaseURL = baseURL
	}
	return &Client{api: openai.NewClientWithConfig(cfg)}, nil
}

func (c *Client) Embed(ctx context.Context, input string) ([]float32, error) {
	req := openai.EmbeddingRequest{
		Model: defaultModel,
		Input: []string{input},
	}
	resp, err := c.api.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("embedding response empty")
	}
	return resp.Data[0].Embedding, nil
}
