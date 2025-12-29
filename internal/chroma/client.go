package chroma

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

const defaultBaseURL = "http://chromadb:8000"

type Client struct {
	baseURL string
	*http.Client
}

type Collection struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type createCollectionRequest struct {
	Name     string         `json:"name"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type AddRequest struct {
	IDs        []string         `json:"ids"`
	Documents  []string         `json:"documents,omitempty"`
	Metadatas  []map[string]any `json:"metadatas,omitempty"`
	Embeddings [][]float32      `json:"embeddings,omitempty"`
}
type UpsertRequest AddRequest

type QueryRequest struct {
	QueryEmbeddings [][]float32    `json:"query_embeddings"`
	NResults        int            `json:"n_results"`
	Where           map[string]any `json:"where,omitempty"`
	WhereDocument   map[string]any `json:"where_document,omitempty"`
	Include         []string       `json:"include,omitempty"`
}

type QueryResponse struct {
	IDs        [][]string         `json:"ids"`
	Documents  [][]string         `json:"documents"`
	Distances  [][]float32        `json:"distances"`
	Metadatas  [][]map[string]any `json:"metadatas"`
	Embeddings [][][]float32      `json:"embeddings"`
}

type GetRequest struct {
	IDs           []string       `json:"ids,omitempty"`
	Where         map[string]any `json:"where,omitempty"`
	WhereDocument map[string]any `json:"where_document,omitempty"`
	Sort          string         `json:"sort,omitempty"`
	Limit         int            `json:"limit,omitempty"`
	Offset        int            `json:"offset,omitempty"`
	Include       []string       `json:"include,omitempty"`
}

type GetResponse struct {
	IDs        []string         `json:"ids"`
	Documents  []string         `json:"documents"`
	Metadatas  []map[string]any `json:"metadatas"`
	Embeddings [][]float32      `json:"embeddings"`
}

func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		baseURL: baseURL,
		Client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) EnsureCollection(ctx context.Context, name string) (*Collection, error) {
	return c.EnsureCollectionWithMetadata(ctx, name, map[string]any{"hnsw:space": "cosine"})
}

func (c *Client) EnsureCollectionWithMetadata(ctx context.Context, name string, metadata map[string]any) (*Collection, error) {
	col, err := c.GetCollection(ctx, name)
	if err == nil {
		return col, nil
	}
	col, err = c.CreateCollection(ctx, name, metadata)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return nil, err
	}
	if err == nil {
		return col, nil
	}
	// collection existed but we need its ID
	return c.GetCollection(ctx, name)
}

func (c *Client) CreateCollection(ctx context.Context, name string, metadata map[string]any) (*Collection, error) {
	req := createCollectionRequest{Name: name, Metadata: metadata}
	var out Collection
	if err := c.do(ctx, http.MethodPost, "/api/v1/collections", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetCollection(ctx context.Context, name string) (*Collection, error) {
	var out Collection
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/collections/%s", name), nil, &out); err != nil {
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

func (c *Client) Upsert(ctx context.Context, collectionID string, req UpsertRequest) error {
	return c.do(ctx, http.MethodPost, fmt.Sprintf("/api/v1/collections/%s/upsert", collectionID), req, nil)
}

func (c *Client) Query(ctx context.Context, collectionID string, req QueryRequest) (*QueryResponse, error) {
	var out QueryResponse
	if err := c.do(ctx, http.MethodPost, fmt.Sprintf("/api/v1/collections/%s/query", collectionID), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Count(ctx context.Context, collectionID string) (int, error) {
	var count int
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/collections/%s/count", collectionID), nil, &count); err != nil {
		return 0, err
	}
	return count, nil
}

func (c *Client) Get(ctx context.Context, collectionID string, req GetRequest) (*GetResponse, error) {
	var out GetResponse
	if err := c.do(ctx, http.MethodPost, fmt.Sprintf("/api/v1/collections/%s/get", collectionID), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListCollections(ctx context.Context) ([]Collection, error) {
	var out []Collection
	if err := c.do(ctx, http.MethodGet, "/api/v1/collections", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) do(ctx context.Context, method, path string, in interface{}, out interface{}) error {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return fmt.Errorf("invalid base url: %w", err)
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	u.Path = strings.TrimSuffix(u.Path, "/") + path

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

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("chroma %s %s: %s", method, path, strings.TrimSpace(string(b)))
	}

	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}
