package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"
)

const baseURL = "https://api.elections.kalshi.com/trade-api/v2/events"
const baseSeriesURL = "https://api.elections.kalshi.com/trade-api/v2/series"

type KalshiMarket struct {
	Ticker         string `json:"ticker"`
	Title          string `json:"title"`
	YesAsk         int64  `json:"yes_ask"`
	YesBid         int64  `json:"yes_bid"`
	NoAsk          int64  `json:"no_ask"`
	NoBid          int64  `json:"no_bid"`
	Volume         int64  `json:"volume"`
	Volume24h      int64  `json:"volume_24h"`
	OpenInterest   int64  `json:"open_interest"`
	RulesPrimary   string `json:"rules_primary"`
	RulesSecondary string `json:"rules_secondary"`
	CloseTime      string `json:"close_time"`
	TickSize       int64  `json:"tick_size"`
}

type OrderbookLevel struct {
	Price    int64 `json:"price"`
	Quantity int64 `json:"quantity"`
}

type KalshiOrderbook struct {
	Yes [][]int64 `json:"yes"` // [price, quantity]
	No  [][]int64 `json:"no"`  // [price, quantity]
}

type KalshiEvent struct {
	Ticker            string         `json:"event_ticker"`
	SeriesTicker      string         `json:"series_ticker"`
	Title             string         `json:"title"`
	SubTitle          string         `json:"sub_title"`
	Description       string         `json:"description"`
	Status            string         `json:"status"`
	Category          string         `json:"category"`
	ResolutionSources []string       `json:"settlement_sources"`
	Markets           []KalshiMarket `json:"markets"`
}

type KalshiResponse struct {
	Events []KalshiEvent `json:"events"`
	Cursor string        `json:"cursor"`
}

type KalshiDetailResponse struct {
	Event   KalshiEvent    `json:"event"`
	Markets []KalshiMarket `json:"markets"`
}

type SettlementSource struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type KalshiSeries struct {
	Ticker            string             `json:"ticker"`
	SettlementSources []SettlementSource `json:"settlement_sources"`
	ContractTermsURL  string             `json:"contract_terms_url"`
	ContractURL       string             `json:"contract_url"`
}

type KalshiSeriesResponse struct {
	Series KalshiSeries `json:"series"`
}

func main() {
	client := &http.Client{Timeout: 20 * time.Second}
	cursor := ""
	var firstActiveEvent *KalshiEvent

	for i := 0; i < 3; i++ {
		u, _ := url.Parse(baseURL)
		q := u.Query()
		q.Set("limit", "10")
		q.Set("status", "open")
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		u.RawQuery = q.Encode()

		fmt.Printf("\n--- Kalshi Page %d (status=open) ---\n", i+1)
		resp, err := client.Get(u.String())
		if err != nil {
			log.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		var kalshiResp KalshiResponse
		if err := json.NewDecoder(resp.Body).Decode(&kalshiResp); err != nil {
			log.Fatalf("decode response: %v", err)
		}

		for _, event := range kalshiResp.Events {
			fmt.Printf("- %s (%s)\n", event.Title, event.Ticker)
			if firstActiveEvent == nil {
				temp := event
				firstActiveEvent = &temp
			}
		}

		cursor = kalshiResp.Cursor
		if cursor == "" {
			break
		}
	}

	if firstActiveEvent != nil {
		fmt.Printf("\n--- Details for Kalshi Event: %s (%s) ---\n", firstActiveEvent.Title, firstActiveEvent.Ticker)
		detailURL := fmt.Sprintf("%s/%s?with_nested_markets=true", baseURL, firstActiveEvent.Ticker)
		resp, err := client.Get(detailURL)
		if err != nil {
			log.Fatalf("detail request failed: %v", err)
		}
		defer resp.Body.Close()

		var detailResp KalshiDetailResponse
		if err := json.NewDecoder(resp.Body).Decode(&detailResp); err != nil {
			log.Fatalf("decode detail response: %v", err)
		}

		ev := detailResp.Event
		fmt.Printf("Event Title: %s\n", ev.Title)
		fmt.Printf("Sub-title: %s\n", ev.SubTitle)

		// Fetch Series info for settlement sources
		fmt.Printf("\n--- Fetching Series Information: %s ---\n", ev.SeriesTicker)
		seriesURL := fmt.Sprintf("%s/%s", baseSeriesURL, ev.SeriesTicker)
		sresp, err := client.Get(seriesURL)
		var series *KalshiSeries
		if err == nil {
			defer sresp.Body.Close()
			var sres KalshiSeriesResponse
			if err := json.NewDecoder(sresp.Body).Decode(&sres); err == nil {
				series = &sres.Series
			}
		}

		if series != nil {
			fmt.Println("Matching Data (Series Level):")
			fmt.Printf("- Contract Terms: %s\n", series.ContractTermsURL)
			if len(series.SettlementSources) > 0 {
				fmt.Println("- Settlement Sources:")
				for _, s := range series.SettlementSources {
					fmt.Printf("  * %s (%s)\n", s.Name, s.URL)
				}
			}
		}

		markets := detailResp.Markets
		if len(markets) == 0 {
			markets = ev.Markets
		}

		var totalVolume int64
		var totalOpenInterest int64
		for _, m := range markets {
			totalVolume += m.Volume
			totalOpenInterest += m.OpenInterest
		}

		fmt.Printf("Total Event Volume: %d\n", totalVolume)
		fmt.Printf("Total Event Open Interest: %d\n", totalOpenInterest)

		fmt.Println("\nMarkets & Execution Details:")
		for _, m := range markets {
			fmt.Printf("\nMarket: %s (%s)\n", m.Title, m.Ticker)
			fmt.Printf("Status: Active/Open (Exp: %s)\n", m.CloseTime)
			fmt.Printf("Live Odds (Top Bid/Ask): Yes:%d/%d, No:%d/%d\n", m.YesBid, m.YesAsk, m.NoBid, m.NoAsk)
			fmt.Printf("Tick Size: %d\n", m.TickSize)
			fmt.Printf("Volume (Total): %d, Volume (24h): %d, OI: %d\n", m.Volume, m.Volume24h, m.OpenInterest)

			// Fetch Orderbook Depth
			bookURL := fmt.Sprintf("https://api.elections.kalshi.com/trade-api/v2/markets/%s/orderbook?depth=5", m.Ticker)
			bresp, err := client.Get(bookURL)
			if err == nil {
				defer bresp.Body.Close()
				var book KalshiOrderbook
				if err := json.NewDecoder(bresp.Body).Decode(&book); err == nil {
					fmt.Println("Order Book Depth (Bids):")
					fmt.Printf("  YES Bids: %v\n", book.Yes)
					fmt.Printf("  NO Bids:  %v\n", book.No)
					fmt.Println("  (Note: Kalshi returns bids only. Asks are derived as 100-opposing_bid)")
				}
			}
			fmt.Printf("Rules (Primary): %s\n", m.RulesPrimary)
			if m.RulesSecondary != "" {
				fmt.Printf("Rules (Secondary): %s\n", m.RulesSecondary)
			}
		}
	} else {
		fmt.Println("\nNo active Kalshi events found in the first 3 pages.")
	}
}
