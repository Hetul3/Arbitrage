package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"llm/chroma/internal/chromaapi"
	"llm/chroma/internal/embedding"
	"llm/chroma/state"
)

const (
	defaultChromaURL = "http://chromadb:8000"
	stateDir         = "chroma/data"
)

func main() {
	snap, err := state.Load(stateDir)
	if err != nil {
		log.Fatal(err)
	}

	apiKey := strings.TrimSpace(os.Getenv("NEBIUS_API_KEY"))
	if apiKey == "" {
		log.Fatal("NEBIUS_API_KEY is not set")
	}
	baseURL := strings.TrimSpace(getenv("NEBIUS_BASE_URL", "https://api.tokenfactory.nebius.com/v1/"))

	embedClient, err := embedding.NewClient(apiKey, baseURL)
	if err != nil {
		log.Fatal(err)
	}
	chromaClient := chromaapi.NewClient(getenv("CHROMA_URL", defaultChromaURL))

	fmt.Printf("Querying collection %q (ID: %s)\n", snap.CollectionName, snap.CollectionID)
	fmt.Println("Type a query to search similar entries, or 'exit' to quit.")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			fmt.Println("\nGoodbye.")
			return
		}
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}
		if strings.EqualFold(text, "exit") || strings.EqualFold(text, "quit") {
			fmt.Println("Goodbye.")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		embeddingVec, err := embedClient.Embed(ctx, text)
		cancel()
		if err != nil {
			log.Printf("embedding error: %v", err)
			continue
		}

		ctxQ, cancelQ := context.WithTimeout(context.Background(), 30*time.Second)
		resp, err := chromaClient.Query(ctxQ, snap.CollectionID, chromaapi.QueryRequest{
			QueryEmbeddings: [][]float32{embeddingVec},
			NResults:        5,
		})
		cancelQ()
		if err != nil {
			log.Printf("query failed: %v", err)
			continue
		}
		if len(resp.Documents) == 0 {
			fmt.Println("No results.")
			continue
		}

		fmt.Println("Top matches:")
		for i := range resp.Documents[0] {
			doc := resp.Documents[0][i]
			id := resp.IDs[0][i]
			var dist float32
			if len(resp.Distances) > 0 && len(resp.Distances[0]) > i {
				dist = resp.Distances[0][i]
			}
			sim := similarity(dist)
			fmt.Printf(" - [%s] score=%.4f\n   %s\n", id, sim, doc)
		}
	}
}

func similarity(distance float32) float32 {
	return 1 / (1 + distance)
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
