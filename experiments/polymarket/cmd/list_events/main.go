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
	ClobTokenIds   string  `json:"clobTokenIds"` // JSON string array
	MinTickSize    float64 `json:"orderPriceMinTickSize"`
}

type ClobLevel struct {
	Price string `json:"price"`
	Size  string `json:"size"`
}

type ClobBook struct {
	Bids []ClobLevel `json:"bids"`
	Asks []ClobLevel `json:"asks"`
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

		fmt.Println("\nMarkets & Execution Details:")
		for _, m := range ev.Markets {
			fmt.Printf("\nMarket: %s (ID: %s)\n", m.Question, m.ID)
			fmt.Printf("Live Odds (Gamma Last Price): %.2f\n", m.LastTradePrice)
			fmt.Printf("Min Tick Size: %.4f\n", m.MinTickSize)

			// Parse CLOB Token IDs
			var tokenIds []string
			if err := json.Unmarshal([]byte(m.ClobTokenIds), &tokenIds); err == nil && len(tokenIds) >= 2 {
				fmt.Printf("CLOB Tokens: YES:%s, NO:%s\n", tokenIds[0], tokenIds[1])

				// Fetch depth for YES token (as example of execution data)
				clobURL := fmt.Sprintf("https://clob.polymarket.com/book?token_id=%s", tokenIds[0])
				cresp, err := client.Get(clobURL)
				if err == nil {
					defer cresp.Body.Close()
					var book ClobBook
					if err := json.NewDecoder(cresp.Body).Decode(&book); err == nil {
						fmt.Println("Real-time CLOB Order Book (YES Token):")
						if len(book.Bids) > 0 {
							fmt.Printf("  Top Bids: %v\n", book.Bids[:min(len(book.Bids), 3)])
						}
						if len(book.Asks) > 0 {
							fmt.Printf("  Top Asks: %v\n", book.Asks[:min(len(book.Asks), 3)])
						}
					}
				}
			}

			fmt.Printf("Rules/Resolution: %s\n", m.Description)
		}
	} else {
		fmt.Println("\nNo active Polymarket events found.")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
