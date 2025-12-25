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
	Volume         int64  `json:"volume"`
	Volume24h      int64  `json:"volume_24h"`
	OpenInterest   int64  `json:"open_interest"`
	RulesPrimary   string `json:"rules_primary"`
	RulesSecondary string `json:"rules_secondary"`
}

type KalshiEvent struct {
	Ticker            string         `json:"event_ticker"`
	SeriesTicker      string         `json:"series_ticker"`
	Title             string         `json:"title"`
	SubTitle          string         `json:"sub_title"`
	Description       string         `json:"description"`
	Status            string         `json:"status"`
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
		var settlementSources []SettlementSource
		if err == nil {
			defer sresp.Body.Close()
			var sres KalshiSeriesResponse
			if err := json.NewDecoder(sresp.Body).Decode(&sres); err == nil {
				settlementSources = sres.Series.SettlementSources
			}
		}

		if len(settlementSources) > 0 {
			fmt.Println("Settlement Sources (from Series):")
			for _, s := range settlementSources {
				fmt.Printf("- %s (%s)\n", s.Name, s.URL)
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

		fmt.Println("\nMarkets & Details:")
		for _, m := range markets {
			fmt.Printf("\nMarket: %s\n", m.Title)
			fmt.Printf("Live Odds (Yes Ask): %d (Ticks)\n", m.YesAsk)
			fmt.Printf("Volume (Total): %d\n", m.Volume)
			fmt.Printf("Volume (24h): %d\n", m.Volume24h)
			fmt.Printf("Open Interest: %d\n", m.OpenInterest)
			fmt.Printf("Rules (Primary): %s\n", m.RulesPrimary)
			if m.RulesSecondary != "" {
				fmt.Printf("Rules (Secondary): %s\n", m.RulesSecondary)
			}
		}
	} else {
		fmt.Println("\nNo active Kalshi events found in the first 3 pages.")
	}
}
