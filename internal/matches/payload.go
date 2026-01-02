package matches

import (
	"fmt"
	"sort"
	"time"

	"github.com/hetulpatel/Arbitrage/internal/hashutil"
	"github.com/hetulpatel/Arbitrage/internal/models"
)

// Payload is the envelope published by the matcher and consumed by the arb engine.
type Payload struct {
	Version           int                   `json:"version"`
	PairID            string                `json:"pair_id"`
	Similarity        float64               `json:"similarity"`
	Distance          float64               `json:"distance"`
	MatchedAt         time.Time             `json:"matched_at"`
	Source            models.MarketSnapshot `json:"source"`
	Target            models.MarketSnapshot `json:"target"`
	Arbitrage         *Opportunity          `json:"arbitrage,omitempty"`
	ResolutionVerdict *ResolutionVerdict    `json:"resolution_verdict,omitempty"`
	Fresh             *FreshSnapshots       `json:"fresh,omitempty"`
	FinalOpportunity  *Opportunity          `json:"final_opportunity,omitempty"`
}

const payloadVersion = 1

// NewPayload builds a match payload with canonical pair ID ordering.
func NewPayload(source, target models.MarketSnapshot, similarity, distance float64) Payload {
	return Payload{
		Version:    payloadVersion,
		PairID:     buildPairID(&source, &target),
		Similarity: similarity,
		Distance:   distance,
		MatchedAt:  time.Now().UTC(),
		Source:     source,
		Target:     target,
	}
}

func buildPairID(a, b *models.MarketSnapshot) string {
	left := fmt.Sprintf("%s:%s", a.Venue, a.Market.MarketID)
	right := fmt.Sprintf("%s:%s", b.Venue, b.Market.MarketID)
	parts := []string{left, right}
	sort.Strings(parts)
	return hashutil.HashStrings(parts...)
}
