package embed

import (
	"context"
	"fmt"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

const defaultModel = "Qwen/Qwen3-Embedding-8B"
const defaultBaseURL = "https://api.tokenfactory.nebius.com/v1/"

// Client wraps the Nebius OpenAI-compatible embedding API.
type Client struct {
	api   *openai.Client
	model string
}

// Config controls how the embedding client is constructed.
type Config struct {
	APIKey  string
	BaseURL string
	Model   string
}

func New(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, fmt.Errorf("NEBIUS_API_KEY not set")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	if cfg.Model == "" {
		cfg.Model = defaultModel
	}

	conf := openai.DefaultConfig(cfg.APIKey)
	conf.BaseURL = cfg.BaseURL

	return &Client{
		api:   openai.NewClientWithConfig(conf),
		model: cfg.Model,
	}, nil
}

func (c *Client) Embed(ctx context.Context, text string) ([]float32, error) {
	req := openai.EmbeddingRequest{
		Model: openai.EmbeddingModel(c.model),
		Input: []string{text},
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
