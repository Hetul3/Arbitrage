package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hetulpatel/Arbitrage/internal/cache"
	"github.com/hetulpatel/Arbitrage/internal/chroma"
	"github.com/hetulpatel/Arbitrage/internal/embed"
	"github.com/hetulpatel/Arbitrage/internal/hashutil"
	"github.com/hetulpatel/Arbitrage/internal/logging"
	"github.com/hetulpatel/Arbitrage/internal/models"
)

type Processor struct {
	embedClient  *embed.Client
	chromaClient *chroma.Client
	collectionID string
	venue        string
	cache        cache.EmbeddingCache
	logCache     bool
}

func NewProcessor(embedClient *embed.Client, chromaClient *chroma.Client, collectionID string, venue string, cache cache.EmbeddingCache, logCache bool) *Processor {
	return &Processor{embedClient: embedClient, chromaClient: chromaClient, collectionID: collectionID, venue: venue, cache: cache, logCache: logCache}
}

func (p *Processor) Handle(ctx context.Context, snap *models.MarketSnapshot) ([]float32, error) {
	text := buildEmbeddingText(snap)
	if text == "" {
		return nil, fmt.Errorf("empty embedding text for market %s", snap.Market.MarketID)
	}

	key := buildEmbeddingKey(snap, text)
	var embedding []float32
	var err error

	cacheMiss := false
	if p.cache != nil && key != "" {
		if cached, ok, cacheErr := p.cache.Get(ctx, key); cacheErr == nil && ok {
			if p.logCache {
				logging.Infof("[embed-cache] hit key=%s", key)
			}
			embedding = cached
		} else if cacheErr != nil {
			return nil, fmt.Errorf("redis get: %w", cacheErr)
		} else {
			cacheMiss = true
			if p.logCache {
				logging.Infof("[embed-cache] miss key=%s", key)
			}
		}
	}

	if embedding == nil {
		embedding, err = p.embedClient.Embed(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("embed: %w", err)
		}
	}

	metadata := buildMetadata(snap, text)

	docBytes, err := json.Marshal(snap)
	if err != nil {
		return nil, fmt.Errorf("marshal snapshot: %w", err)
	}

	id := fmt.Sprintf("%s:%s", snap.Venue, snap.Market.MarketID)

	upsertReq := chroma.UpsertRequest{
		IDs:        []string{id},
		Documents:  []string{string(docBytes)},
		Metadatas:  []map[string]any{metadata},
		Embeddings: [][]float32{embedding},
	}

	if err := p.chromaClient.Upsert(ctx, p.collectionID, upsertReq); err != nil {
		return nil, fmt.Errorf("chroma upsert: %w", err)
	}

	if embedding != nil && p.cache != nil && key != "" && cacheMiss {
		if err := p.cache.Set(ctx, key, embedding); err != nil {
			return nil, fmt.Errorf("redis set: %w", err)
		}
		if p.logCache {
			logging.Infof("[embed-cache] stored key=%s", key)
		}
	}

	return embedding, nil
}

func buildMetadata(snap *models.MarketSnapshot, embeddingText string) map[string]any {
	metadata := map[string]any{
		"venue":            string(snap.Venue),
		"market_id":        snap.Market.MarketID,
		"event_id":         snap.Event.EventID,
		"captured_at":      snap.CapturedAt.Format(time.RFC3339Nano),
		"captured_at_unix": snap.CapturedAt.Unix(),
		"text_hash":        hashutil.HashStrings(embeddingText),
		"resolution_hash": hashutil.HashStrings(
			snap.Event.ResolutionSource,
			snap.Event.ResolutionDetails,
			snap.Event.ContractTermsURL,
		),
	}

	if !snap.Market.CloseTime.IsZero() {
		metadata["close_time"] = snap.Market.CloseTime.Format(time.RFC3339Nano)
	} else if !snap.Event.CloseTime.IsZero() {
		metadata["close_time"] = snap.Event.CloseTime.Format(time.RFC3339Nano)
	}

	return metadata
}

func buildEmbeddingKey(snap *models.MarketSnapshot, embeddingText string) string {
	if snap == nil {
		return ""
	}
	textHash := hashutil.HashStrings(embeddingText)
	return fmt.Sprintf("%s:%s:%s", snap.Venue, snap.Market.MarketID, textHash)
}
