package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// EmbeddingCache defines the minimal interface our workers need.
type EmbeddingCache interface {
	Get(ctx context.Context, key string) ([]float32, bool, error)
	Set(ctx context.Context, key string, value []float32) error
	Close() error
}

type redisEmbeddingCache struct {
	client *redis.Client
	ttl    time.Duration
	prefix string
}

// NewRedisEmbeddingCache builds a cache with the given addr/password/db.
func NewRedisEmbeddingCache(addr, password string, db int, ttl time.Duration, prefix string) (EmbeddingCache, error) {
	if addr == "" {
		return nil, fmt.Errorf("redis addr is required")
	}
	if ttl <= 0 {
		ttl = 240 * time.Hour // 10 days
	}
	if prefix == "" {
		prefix = "emb"
	}
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	return &redisEmbeddingCache{
		client: client,
		ttl:    ttl,
		prefix: prefix,
	}, nil
}

func (c *redisEmbeddingCache) key(k string) string {
	return fmt.Sprintf("%s:%s", c.prefix, k)
}

func (c *redisEmbeddingCache) Get(ctx context.Context, key string) ([]float32, bool, error) {
	if c == nil || c.client == nil {
		return nil, false, nil
	}
	data, err := c.client.Get(ctx, c.key(key)).Bytes()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var out []float32
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, false, err
	}
	return out, true, nil
}

func (c *redisEmbeddingCache) Set(ctx context.Context, key string, value []float32) error {
	if c == nil || c.client == nil {
		return nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, c.key(key), data, c.ttl).Err()
}

func (c *redisEmbeddingCache) Close() error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Close()
}
