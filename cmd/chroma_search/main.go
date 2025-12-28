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
	"github.com/hetulpatel/Arbitrage/internal/embed"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	queryText := flag.String("text", "", "The text to search for")
	limit := flag.Int("k", 3, "Number of top results to return")
	flag.Parse()

	if *queryText == "" {
		log.Fatal("Please provide search text using -text")
	}

	// 1. Setup Clients
	embedClient := mustEmbedClient()
	chromaClient := mustChromaClient()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// 2. Embed the query string
	fmt.Printf("Embedding query: \"%s\"...\n", *queryText)
	embedding, err := embedClient.Embed(ctx, *queryText)
	if err != nil {
		log.Fatalf("Failed to embed query: %v", err)
	}

	// 3. Find Collection
	collectionName := os.Getenv("CHROMA_COLLECTION")
	if collectionName == "" {
		collectionName = "market_snapshots"
	}
	collections, err := chromaClient.ListCollections(ctx)
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

	// 4. Query Chroma
	fmt.Printf("Searching for top %d matches in Chroma...\n\n", *limit)
	queryResp, err := chromaClient.Query(ctx, targetCol.ID, chroma.QueryRequest{
		QueryEmbeddings: [][]float32{embedding},
		NResults:        *limit,
		Include:         []string{"metadatas", "distances", "embeddings", "documents"},
	})
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	// 5. Output Results
	for i := range queryResp.IDs[0] {
		id := queryResp.IDs[0][i]
		dist := queryResp.Distances[0][i]
		meta := queryResp.Metadatas[0][i]
		emb := queryResp.Embeddings[0][i]
		doc := queryResp.Documents[0][i]

		// Extract Title from Document
		title := "Unknown"
		var snapshot map[string]any
		if err := json.Unmarshal([]byte(doc), &snapshot); err == nil {
			if market, ok := snapshot["market"].(map[string]any); ok && market["Question"] != nil {
				title = fmt.Sprintf("%v", market["Question"])
			} else if event, ok := snapshot["event"].(map[string]any); ok && event["Title"] != nil {
				title = fmt.Sprintf("%v", event["Title"])
			}
		}

		fmt.Printf("--- Match #%d ---\n", i+1)
		fmt.Printf("ID:       %s\n", id)
		fmt.Printf("Title:    %s\n", title)
		fmt.Printf("Venue:    %v\n", meta["venue"])
		fmt.Printf("Distance: %.4f\n", dist)

		fmt.Println("Metadata:")
		metaJSON, _ := json.MarshalIndent(meta, "  ", "  ")
		fmt.Printf("  %s\n", string(metaJSON))

		fmt.Printf("Embedding (first 5 dims): %v...\n", emb[:5])
		fmt.Println()
	}

	if len(queryResp.IDs[0]) == 0 {
		fmt.Println("No matches found.")
	}
}

func mustEmbedClient() *embed.Client {
	apiKey := os.Getenv("NEBIUS_API_KEY")
	if apiKey == "" {
		log.Fatal("NEBIUS_API_KEY environment variable is required")
	}
	cfg := embed.Config{
		APIKey:  apiKey,
		BaseURL: os.Getenv("NEBIUS_BASE_URL"),
		Model:   os.Getenv("NEBIUS_EMBED_MODEL"),
	}
	client, err := embed.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create embed client: %v", err)
	}
	return client
}

func mustChromaClient() *chroma.Client {
	chromaURL := os.Getenv("CHROMA_URL")
	if chromaURL == "" {
		chromaURL = "http://localhost:8000"
	}
	return chroma.NewClient(chromaURL)
}
