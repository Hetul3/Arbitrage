package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/hetulpatel/Arbitrage/internal/chroma"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	chromaURL := os.Getenv("CHROMA_URL")
	if chromaURL == "" {
		chromaURL = "http://localhost:8000"
	}

	client := chroma.NewClient(chromaURL)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collections, err := client.ListCollections(ctx)
	if err != nil {
		log.Fatalf("Error listing collections: %v\n(Make sure Chroma is running at %s and port 8000 is exposed)", err, chromaURL)
	}

	if len(collections) == 0 {
		fmt.Println("No collections found.")
		return
	}

	fmt.Printf("Found %d collections:\n", len(collections))
	for _, col := range collections {
		count, err := client.Count(ctx, col.ID)
		if err != nil {
			fmt.Printf("- %s (ID: %s): Error getting count: %v\n", col.Name, col.ID, err)
			continue
		}
		fmt.Printf("- %s (ID: %s): %d items\n", col.Name, col.ID, count)

		if count > 0 {
			fmt.Println("  Recent documents (limit 2):")
			peek, err := client.Get(ctx, col.ID, chroma.GetRequest{
				Limit:   2,
				Include: []string{"documents", "metadatas"},
			})
			if err != nil {
				fmt.Printf("    Error peeking: %v\n", err)
				continue
			}
			for i := range peek.IDs {
				fmt.Printf("    ID: %s\n", peek.IDs[i])
				if len(peek.Documents[i]) > 100 {
					fmt.Printf("    Doc: %s...\n", peek.Documents[i][:100])
				} else {
					fmt.Printf("    Doc: %s\n", peek.Documents[i])
				}
				meta, _ := json.Marshal(peek.Metadatas[i])
				fmt.Printf("    Meta: %s\n", string(meta))
				fmt.Println()
			}
		}
	}
}
