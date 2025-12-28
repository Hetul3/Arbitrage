package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/hetulpatel/Arbitrage/internal/chroma"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	marketID := flag.String("id", "", "The Market ID to find matches for")
	limit := flag.Int("k", 3, "Number of top results to return")
	flag.Parse()

	if *marketID == "" {
		log.Fatal("Please provide a market ID using -id")
	}

	chromaURL := os.Getenv("CHROMA_URL")
	if chromaURL == "" {
		chromaURL = "http://localhost:8000"
	}

	client := chroma.NewClient(chromaURL)
	collectionName := os.Getenv("CHROMA_COLLECTION")
	if collectionName == "" {
		collectionName = "market_snapshots"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Find the collection
	collections, err := client.ListCollections(ctx)
	if err != nil {
		log.Fatalf("Error listing collections: %v", err)
	}
	var targetCol *chroma.Collection
	for _, col := range collections {
		if col.Name == collectionName {
			targetCol = &col
			break
		}
	}
	if targetCol == nil {
		log.Fatalf("Collection %s not found", collectionName)
	}

	// 2. Resolve the market
	// We first try to get by the exact Chroma ID
	getResp, err := client.Get(ctx, targetCol.ID, chroma.GetRequest{
		IDs:     []string{*marketID},
		Limit:   1,
		Include: []string{"embeddings", "documents", "metadatas"},
	})

	// If not found by ID, try searching by the "market_id" metadata (the venue's ticker)
	if err == nil && len(getResp.IDs) == 0 {
		getResp, err = client.Get(ctx, targetCol.ID, chroma.GetRequest{
			Where:   map[string]any{"market_id": *marketID},
			Limit:   1,
			Include: []string{"embeddings", "documents", "metadatas"},
		})
	}

	if err != nil || len(getResp.IDs) == 0 {
		log.Fatalf("Could not find market with ID or market_id '%s' in Chroma. Try using the full ID or the ticker (e.g. KXCOLONIZEMARS-50)", *marketID)
	}

	sourceVenue := getResp.Metadatas[0]["venue"].(string)
	embedding := getResp.Embeddings[0]

	fmt.Printf("Found source market: %s (%s)\n", *marketID, sourceVenue)
	fmt.Printf("Searching for top %d matches in the OPPOSITE venue...\n\n", *limit)

	// 3. Query for similar items in the opposite venue
	targetVenue := "polymarket"
	if sourceVenue == "polymarket" {
		targetVenue = "kalshi"
	}

	queryResp, err := client.Query(ctx, targetCol.ID, chroma.QueryRequest{
		QueryEmbeddings: [][]float32{embedding},
		NResults:        *limit,
		Where:           map[string]any{"venue": targetVenue},
		Include:         []string{"documents", "metadatas", "distances"},
	})
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	fmt.Printf("Top %d matches in %s:\n\n", *limit, targetVenue)
	for i := range queryResp.IDs[0] {
		id := queryResp.IDs[0][i]
		dist := queryResp.Distances[0][i]
		doc := queryResp.Documents[0][i]

		fmt.Printf("[%d] Distance: %.4f | ID: %s\n", i+1, dist, id)

		// Parse doc to show title
		var snapshot map[string]any
		json.Unmarshal([]byte(doc), &snapshot)

		title := "Unknown"
		if market, ok := snapshot["market"].(map[string]any); ok && market["Question"] != nil {
			title = fmt.Sprintf("%v", market["Question"])
		} else if event, ok := snapshot["event"].(map[string]any); ok && event["Title"] != nil {
			title = fmt.Sprintf("%v", event["Title"])
		}

		fmt.Printf("    Title: %s\n", title)
		fmt.Println()
	}

	if len(queryResp.IDs[0]) == 0 {
		fmt.Printf("No matches found in %s.\n", targetVenue)
	}
}
