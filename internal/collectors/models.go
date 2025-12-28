package collectors

import (
	"context"
	"time"
)

// Venue identifies the platform a market/event belongs to.
type Venue string

const (
	VenuePolymarket Venue = "polymarket"
	VenueKalshi     Venue = "kalshi"
)

// FetchOptions control how many pages/items a collector should fetch per run.
type FetchOptions struct {
	Pages    int
	PageSize int
}

// Collector is implemented by venue-specific collectors (Polymarket, Kalshi, ...).
// Each collector is responsible for fetching, normalizing, and returning events
// that fit the FetchOptions constraints.
type Collector interface {
	Name() string
	Fetch(ctx context.Context, opts FetchOptions) ([]Event, error)
}

// Event represents a normalized event that may contain multiple markets/outcomes.
type Event struct {
	Venue             Venue
	EventID           string
	Title             string
	Description       string
	Category          string
	Status            string
	ResolutionSource  string
	ResolutionDetails string
	SettlementSources []ResolutionSource
	ContractTermsURL  string
	CloseTime         time.Time
	Markets           []Market
	Raw               map[string]any
}

// Market is a normalized market belonging to an event.
type Market struct {
	MarketID     string
	Question     string
	Subtitle     string
	TickSize     float64
	CloseTime    time.Time
	Volume       float64
	Volume24h    float64
	OpenInterest float64
	Price        PriceSnapshot
	Orderbooks   map[string]Orderbook // keyed by outcome/token label (e.g., YES, NO)
	ClobTokenIDs []string             // Polymarket-specific
	ReferenceURL string               // optional human-facing URL
}

// PriceSnapshot captures top-of-book values for YES/NO.
type PriceSnapshot struct {
	YesBid float64
	YesAsk float64
	NoBid  float64
	NoAsk  float64
}

// Orderbook stores top depth levels for an outcome.
type Orderbook struct {
	Bids []OrderbookLevel
	Asks []OrderbookLevel
}

// OrderbookLevel is a single price/quantity pair.
type OrderbookLevel struct {
	Price     float64
	Quantity  float64
	RawPrice  float64 // some venues report ints/cents; RawPrice preserves the original value
	RawAmount float64 // same for size/quantity
}

// ResolutionSource describes a named source (e.g., source + URL).
type ResolutionSource struct {
	Name string
	URL  string
}
