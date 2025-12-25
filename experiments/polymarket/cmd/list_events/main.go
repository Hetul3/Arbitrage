package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const baseURL = "https://gamma-api.polymarket.com/events"

type PolymarketMarket struct {
	ID             string  `json:"id"`
	Question       string  `json:"question"`
	Description    string  `json:"description"`
	LastTradePrice float64 `json:"lastTradePrice"`
	VolumeNum      float64 `json:"volumeNum"`
	Volume24h      float64 `json:"volume24hr"`
}

type PolymarketEvent struct {
	ID                string             `json:"id"`
	Title             string             `json:"title"`
	Description       string             `json:"description"`
	ResolutionSources string             `json:"resolutionSource"`
	Closed            bool               `json:"closed"`
	Liquidity         float64            `json:"liquidity"`
	Volume            float64            `json:"volume"`
	OpenInterest      float64            `json:"openInterest"`
	Volume24h         float64            `json:"volume24hr"`
	Markets           []PolymarketMarket `json:"markets"`
}

func main() {
	client := &http.Client{Timeout: 20 * time.Second}
	limit := 10
	offset := 0
	var firstActiveEvent *PolymarketEvent

	for i := 0; i < 3; i++ {
		u, _ := url.Parse(baseURL)
		q := u.Query()
		q.Set("limit", strconv.Itoa(limit))
		q.Set("offset", strconv.Itoa(offset))
		q.Set("closed", "false")
		u.RawQuery = q.Encode()

		fmt.Printf("\n--- Polymarket Page %d ---\n", i+1)
		resp, err := client.Get(u.String())
		if err != nil {
			log.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		var events []PolymarketEvent
		if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
			log.Fatalf("decode response: %v", err)
		}

		if len(events) == 0 {
			break
		}

		for _, event := range events {
			fmt.Printf("- %s (ID: %s)\n", event.Title, event.ID)
			if firstActiveEvent == nil && !event.Closed {
				temp := event
				firstActiveEvent = &temp
			}
		}

		offset += limit
	}

	if firstActiveEvent != nil {
		fmt.Printf("\n--- Details for Polymarket Event: %s (ID: %s) ---\n", firstActiveEvent.Title, firstActiveEvent.ID)
		detailURL := fmt.Sprintf("%s/%s", baseURL, firstActiveEvent.ID)
		resp, err := client.Get(detailURL)
		if err != nil {
			log.Fatalf("detail request failed: %v", err)
		}
		defer resp.Body.Close()

		var ev PolymarketEvent
		if err := json.NewDecoder(resp.Body).Decode(&ev); err != nil {
			log.Fatalf("decode detail response: %v", err)
		}

		fmt.Printf("Event Title: %s\n", ev.Title)
		fmt.Printf("Description: %s\n", ev.Description)
		fmt.Printf("Total Event Volume: %.2f\n", ev.Volume)
		fmt.Printf("Event Volume (24h): %.2f\n", ev.Volume24h)
		fmt.Printf("Total Event Liquidity: %.2f\n", ev.Liquidity)
		fmt.Printf("Total Event Open Interest: %.2f\n", ev.OpenInterest)

		fmt.Println("\nMarkets & Details:")
		for _, m := range ev.Markets {
			fmt.Printf("\nMarket: %s\n", m.Question)
			fmt.Printf("Live Odds (Last Trade Price): %.2f\n", m.LastTradePrice)
			fmt.Printf("Volume (Total): %.2f\n", m.VolumeNum)
			fmt.Printf("Volume (24h): %.2f\n", m.Volume24h)
			fmt.Printf("Rules/Resolution: %s\n", m.Description)
		}
	} else {
		fmt.Println("\nNo active Polymarket events found.")
	}
}
