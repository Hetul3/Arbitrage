package matcher

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hetulpatel/Arbitrage/internal/cache"
	"github.com/hetulpatel/Arbitrage/internal/chroma"
	"github.com/hetulpatel/Arbitrage/internal/collectors"
	"github.com/hetulpatel/Arbitrage/internal/logging"
	"github.com/hetulpatel/Arbitrage/internal/matches"
	"github.com/hetulpatel/Arbitrage/internal/models"
)

type Config struct {
	Client       *chroma.Client
	CollectionID string
	TopK         int
	Threshold    float64
	Freshness    time.Duration
	Debug        bool
	VerdictCache cache.VerdictCache
}

type Finder struct {
	client       *chroma.Client
	collectionID string
	topK         int
	threshold    float64
	freshness    time.Duration
	debug        bool
	verdictCache cache.VerdictCache
}

type Result struct {
	Target        *models.MarketSnapshot
	Similarity    float64
	Distance      float64
	CachedVerdict bool
}

func NewFinder(cfg Config) (*Finder, error) {
	if cfg.Client == nil {
		return nil, fmt.Errorf("matcher: chroma client is required")
	}
	if cfg.CollectionID == "" {
		return nil, fmt.Errorf("matcher: collection id is required")
	}
	topK := cfg.TopK
	if topK <= 0 {
		topK = 3
	}
	threshold := cfg.Threshold
	if threshold <= 0 || threshold > 1 {
		threshold = 0.95
	}
	freshness := cfg.Freshness
	if freshness <= 0 {
		freshness = 10 * time.Minute
	}
	return &Finder{
		client:       cfg.Client,
		collectionID: cfg.CollectionID,
		topK:         topK,
		threshold:    threshold,
		freshness:    freshness,
		debug:        cfg.Debug,
		verdictCache: cfg.VerdictCache,
	}, nil
}

func (f *Finder) FindBestMatch(ctx context.Context, snap *models.MarketSnapshot, embedding []float32) (*Result, error) {
	if snap == nil || embedding == nil {
		return nil, nil
	}

	targetVenue, err := oppositeVenue(snap.Venue)
	if err != nil {
		return nil, err
	}

	var where map[string]any
	var cutoff time.Time

	if f.freshness > 0 {
		cutoff = time.Now().UTC().Add(-f.freshness)
		cutoffUnix := cutoff.Unix()
		where = map[string]any{
			"$and": []map[string]any{
				{"venue": string(targetVenue)},
				{"captured_at_unix": map[string]any{"$gte": cutoffUnix}},
			},
		}
	} else {
		where = map[string]any{"venue": string(targetVenue)}
	}

	queryReq := chroma.QueryRequest{
		QueryEmbeddings: [][]float32{embedding},
		NResults:        f.topK,
		Where:           where,
		Include:         []string{"documents", "metadatas", "distances"},
	}
	if f.debug {
		printDebug(
			"[matcher] query",
			"venue", snap.Venue,
			"market", snap.Market.MarketID,
			"topK", f.topK,
			"threshold", f.threshold,
			"fresh_cutoff", cutoff.Unix(),
			"where", where,
		)
	}
	resp, err := f.client.Query(ctx, f.collectionID, queryReq)
	if err != nil {
		return nil, fmt.Errorf("matcher query: %w", err)
	}
	if len(resp.Documents) == 0 || len(resp.Documents[0]) == 0 {
		if f.debug {
			printDebug("[matcher] no documents from query",
				"venue", snap.Venue,
				"market", snap.Market.MarketID,
			)
		}
		return nil, nil
	}

	for i := range resp.Documents[0] {
		dist := distanceAt(resp, i)
		similarity := 1 - dist
		if similarity < f.threshold {
			break
		}

		var target models.MarketSnapshot
		if err := json.Unmarshal([]byte(resp.Documents[0][i]), &target); err != nil {
			return nil, fmt.Errorf("decode match snapshot: %w", err)
		}

		if !cutoff.IsZero() && target.CapturedAt.Before(cutoff) {
			if f.debug {
				printDebug("[matcher] skipped stale candidate",
					"candidate", target.Market.MarketID,
					"captured_at", target.CapturedAt,
					"cutoff", cutoff,
				)
			}
			continue
		}

		if f.debug {
			printDebug("[matcher] candidate",
				"source", snap.Market.MarketID,
				"candidate", target.Market.MarketID,
				"similarity", similarity,
				"distance", dist,
				"captured_at", target.CapturedAt,
			)
		}
		if result, skip := f.checkVerdictCache(ctx, snap, &target, similarity, dist); skip {
			if result != nil {
				return result, nil
			}
			continue
		}

		return &Result{
			Target:     &target,
			Similarity: similarity,
			Distance:   dist,
		}, nil
	}

	if f.debug {
		printDebug("[matcher] no candidates returned",
			"venue", snap.Venue,
			"market", snap.Market.MarketID,
		)
	}
	return nil, nil
}

func (f *Finder) Threshold() float64 {
	return f.threshold
}

func (f *Finder) checkVerdictCache(ctx context.Context, source, target *models.MarketSnapshot, similarity, distance float64) (*Result, bool) {
	if f.verdictCache == nil {
		return nil, false
	}
	key := matches.VerdictCacheKey(source, target)
	if key == "" {
		return nil, false
	}
	verdict, ok, err := f.verdictCache.Get(ctx, key)
	if err != nil {
		logging.Errorf("[verdict-cache] get error key=%s: %v", key, err)
		return nil, false
	}
	if !ok {
		logging.Infof("[verdict-cache] miss key=%s", key)
		return nil, false
	}

	if verdict {
		logging.Infof("[verdict-cache] hit SAFE key=%s similarity=%.4f", key, similarity)
		return &Result{
			Target:        target,
			Similarity:    similarity,
			Distance:      distance,
			CachedVerdict: true,
		}, true
	}

	logging.Infof("[verdict-cache] hit UNSAFE key=%s similarity=%.4f", key, similarity)
	return nil, true
}

func oppositeVenue(v collectors.Venue) (collectors.Venue, error) {
	switch v {
	case collectors.VenuePolymarket:
		return collectors.VenueKalshi, nil
	case collectors.VenueKalshi:
		return collectors.VenuePolymarket, nil
	default:
		return "", fmt.Errorf("unknown venue %q", v)
	}
}

func distanceAt(resp *chroma.QueryResponse, idx int) float64 {
	if resp == nil || len(resp.Distances) == 0 || len(resp.Distances[0]) <= idx {
		return 1.0
	}
	return float64(resp.Distances[0][idx])
}

func printDebug(msg string, fields ...interface{}) {
	logging.Debugf("%s: %v", msg, fields)
}
