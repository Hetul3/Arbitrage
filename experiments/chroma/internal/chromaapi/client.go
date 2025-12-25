package chromaapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

type Collection struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type createCollectionRequest struct {
	Name     string                 `json:"name"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type AddRequest struct {
	IDs        []string         `json:"ids"`
	Documents  []string         `json:"documents,omitempty"`
	Metadatas  []map[string]any `json:"metadatas,omitempty"`
	Embeddings [][]float32      `json:"embeddings,omitempty"`
}

type QueryRequest struct {
	QueryEmbeddings [][]float32 `json:"query_embeddings,omitempty"`
	NResults        int         `json:"n_results,omitempty"`
}

type QueryResponse struct {
	IDs       [][]string         `json:"ids"`
	Documents [][]string         `json:"documents"`
	Distances [][]float32        `json:"distances"`
	Metadatas [][]map[string]any `json:"metadatas"`
}

func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) CreateCollection(ctx context.Context, name string) (*Collection, error) {
	payload := createCollectionRequest{Name: name}
	var out Collection
	if err := c.do(ctx, http.MethodPost, "/api/v1/collections", payload, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteCollection(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/collections/%s", id), nil, nil)
}

func (c *Client) Add(ctx context.Context, collectionID string, req AddRequest) error {
	return c.do(ctx, http.MethodPost, fmt.Sprintf("/api/v1/collections/%s/add", collectionID), req, nil)
}

func (c *Client) Query(ctx context.Context, collectionID string, req QueryRequest) (*QueryResponse, error) {
	var out QueryResponse
	if err := c.do(ctx, http.MethodPost, fmt.Sprintf("/api/v1/collections/%s/query", collectionID), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) do(ctx context.Context, method, p string, in interface{}, out interface{}) error {
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return fmt.Errorf("invalid base url: %w", err)
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	u.Path = strings.TrimSuffix(u.Path, "/") + p

	var body io.Reader
	if in != nil {
		buf, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("chroma %s %s: %s", method, p, strings.TrimSpace(string(b)))
	}

	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}
