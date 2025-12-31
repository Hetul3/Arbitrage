package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hetulpatel/Arbitrage/internal/arb"
	"github.com/hetulpatel/Arbitrage/internal/matches"
)

// InsertArbOpportunity stores the outcome of an arb evaluation (profitable or not).
func (s *Store) InsertArbOpportunity(ctx context.Context, payload *matches.Payload, result arb.Result) error {
	if s == nil || s.db == nil || payload == nil {
		return fmt.Errorf("sqlite store not initialized or payload nil")
	}
	best := result.Best
	if best == nil {
		best = &matches.Opportunity{}
	}

	legsJSON, err := json.Marshal(best.Legs)
	if err != nil {
		return fmt.Errorf("marshal legs: %w", err)
	}
	rawJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	query := `
INSERT INTO arb_opportunities (
	pair_id, source_venue, source_market_id, source_question,
	source_yes_price, source_no_price,
	target_venue, target_market_id, target_question,
	target_yes_price, target_no_price,
	similarity, distance, matched_at, processed_at,
	direction, qty_contracts, total_cost_usd, profit_usd,
	budget_usd, kalshi_fees_usd, polymarket_fees_usd,
	legs_json, raw_payload_json
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

	processedAt := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = s.db.ExecContext(
		ctx,
		query,
		payload.PairID,
		payload.Source.Venue,
		payload.Source.Market.MarketID,
		payload.Source.Market.Question,
		payload.Source.Market.Price.YesAsk,
		payload.Source.Market.Price.NoAsk,
		payload.Target.Venue,
		payload.Target.Market.MarketID,
		payload.Target.Market.Question,
		payload.Target.Market.Price.YesAsk,
		payload.Target.Market.Price.NoAsk,
		payload.Similarity,
		payload.Distance,
		payload.MatchedAt.Format(time.RFC3339Nano),
		processedAt,
		best.Direction,
		best.Quantity,
		best.TotalCostUSD,
		best.ProfitUSD,
		best.BudgetUSD,
		best.KalshiFeesUSD,
		best.PolymarketFeesUSD,
		string(legsJSON),
		string(rawJSON),
	)
	return err
}
