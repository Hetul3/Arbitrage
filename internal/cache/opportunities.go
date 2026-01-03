package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// OpportunityRecord captures the best profitable result for a pair.
type OpportunityRecord struct {
	ProfitUSD float64   `json:"profit_usd"`
	Direction string    `json:"direction"`
	Quantity  float64   `json:"quantity"`
	UpdatedAt time.Time `json:"updated_at"`
}

// OpportunityCache stores the best opportunity per pair so we can suppress duplicates.
type OpportunityCache interface {
	Get(ctx context.Context, pairID string) (*OpportunityRecord, bool, error)
	Set(ctx context.Context, pairID string, record OpportunityRecord) error
	Close() error
}

type redisOpportunityCache struct {
	client *redis.Client
	ttl    time.Duration
	prefix string
}

// NewRedisOpportunityCache builds a cache keyed by the canonical pair ID.
func NewRedisOpportunityCache(addr, password string, db int, ttl time.Duration, prefix string) (OpportunityCache, error) {
	if addr == "" {
		return nil, fmt.Errorf("redis addr is required")
	}
	if ttl <= 0 {
		ttl = 240 * time.Hour
	}
	if prefix == "" {
		prefix = "pair_best"
	}
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	return &redisOpportunityCache{client: client, ttl: ttl, prefix: prefix}, nil
}

func (c *redisOpportunityCache) key(pairID string) string {
	return fmt.Sprintf("%s:%s", c.prefix, pairID)
}

func (c *redisOpportunityCache) Get(ctx context.Context, pairID string) (*OpportunityRecord, bool, error) {
	if c == nil || c.client == nil {
		return nil, false, nil
	}
	raw, err := c.client.Get(ctx, c.key(pairID)).Bytes()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var record OpportunityRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return nil, false, err
	}
	return &record, true, nil
}

func (c *redisOpportunityCache) Set(ctx context.Context, pairID string, record OpportunityRecord) error {
	if c == nil || c.client == nil {
		return nil
	}
	payload, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, c.key(pairID), payload, c.ttl).Err()
}

func (c *redisOpportunityCache) Close() error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Close()
}
