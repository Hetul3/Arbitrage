package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hetulpatel/Arbitrage/internal/chroma"
	"github.com/hetulpatel/Arbitrage/internal/embed"
	"github.com/hetulpatel/Arbitrage/internal/hashutil"
	"github.com/hetulpatel/Arbitrage/internal/models"
)

type Processor struct {
	embedClient  *embed.Client
	chromaClient *chroma.Client
	collectionID string
	venue        string
}

func NewProcessor(embedClient *embed.Client, chromaClient *chroma.Client, collectionID string, venue string) *Processor {
	return &Processor{embedClient: embedClient, chromaClient: chromaClient, collectionID: collectionID, venue: venue}
}

func (p *Processor) Handle(ctx context.Context, snap *models.MarketSnapshot) error {
	text := buildEmbeddingText(snap)
	if text == "" {
		return fmt.Errorf("empty embedding text for market %s", snap.Market.MarketID)
	}

	embedding, err := p.embedClient.Embed(ctx, text)
	if err != nil {
		return fmt.Errorf("embed: %w", err)
	}

	metadata := buildMetadata(snap, text)

	docBytes, err := json.Marshal(snap)
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}

	id := fmt.Sprintf("%s:%s", snap.Venue, snap.Market.MarketID)

	req := chroma.AddRequest{
		IDs:        []string{id},
		Documents:  []string{string(docBytes)},
		Metadatas:  []map[string]any{metadata},
		Embeddings: [][]float32{embedding},
	}

	if err := p.chromaClient.Add(ctx, p.collectionID, req); err != nil {
		return fmt.Errorf("chroma add: %w", err)
	}

	return nil
}

func buildMetadata(snap *models.MarketSnapshot, embeddingText string) map[string]any {
	metadata := map[string]any{
		"venue":       string(snap.Venue),
		"market_id":   snap.Market.MarketID,
		"event_id":    snap.Event.EventID,
		"captured_at": snap.CapturedAt.Format(time.RFC3339Nano),
		"text_hash":   hashutil.HashStrings(embeddingText),
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
