package models

import (
	"time"

	"github.com/hetulpatel/Arbitrage/internal/collectors"
)

// MarketSnapshot is the payload placed on Kafka topics.
type MarketSnapshot struct {
	Venue      collectors.Venue  `json:"venue"`
	Event      collectors.Event  `json:"event"`
	Market     collectors.Market `json:"market"`
	CapturedAt time.Time         `json:"captured_at"`
}

// Prepare clones the event to avoid embedding all markets in each snapshot.
func NewSnapshot(venue collectors.Venue, ev collectors.Event, market collectors.Market, capturedAt time.Time) MarketSnapshot {
	cloneEvent := ev
	cloneEvent.Markets = nil
	return MarketSnapshot{
		Venue:      venue,
		Event:      cloneEvent,
		Market:     market,
		CapturedAt: capturedAt,
	}
}
