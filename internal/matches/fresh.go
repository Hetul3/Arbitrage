package matches

import "github.com/hetulpatel/Arbitrage/internal/models"

// FreshSnapshots holds the live snapshots fetched right before the final arb pass.
type FreshSnapshots struct {
	Polymarket *models.MarketSnapshot `json:"polymarket,omitempty"`
	Kalshi     *models.MarketSnapshot `json:"kalshi,omitempty"`
}
