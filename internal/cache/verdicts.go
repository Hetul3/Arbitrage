package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// VerdictCache stores SAFE/UNSAFE decisions by pair + resolution hash.
type VerdictCache interface {
	Get(ctx context.Context, key string) (bool, bool, error)
	Set(ctx context.Context, key string, verdict bool) error
	Close() error
}

type redisVerdictCache struct {
	client *redis.Client
	ttl    time.Duration
	prefix string
}

func NewRedisVerdictCache(addr, password string, db int, ttl time.Duration, prefix string) (VerdictCache, error) {
	if addr == "" {
		return nil, fmt.Errorf("redis addr is required")
	}
	if ttl <= 0 {
		ttl = 240 * time.Hour
	}
	if prefix == "" {
		prefix = "pair_verdict"
	}
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	return &redisVerdictCache{client: client, ttl: ttl, prefix: prefix}, nil
}

func (c *redisVerdictCache) key(k string) string {
	return fmt.Sprintf("%s:%s", c.prefix, k)
}

func (c *redisVerdictCache) Get(ctx context.Context, key string) (bool, bool, error) {
	if c == nil || c.client == nil {
		return false, false, nil
	}
	val, err := c.client.Get(ctx, c.key(key)).Result()
	if err == redis.Nil {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	return val == "1", true, nil
}

func (c *redisVerdictCache) Set(ctx context.Context, key string, verdict bool) error {
	if c == nil || c.client == nil {
		return nil
	}
	value := "0"
	if verdict {
		value = "1"
	}
	return c.client.Set(ctx, c.key(key), value, c.ttl).Err()
}

func (c *redisVerdictCache) Close() error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Close()
}
