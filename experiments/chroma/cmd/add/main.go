package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

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

	fmt.Printf("Connected to collection %q (ID: %s)\n", snap.CollectionName, snap.CollectionID)
	fmt.Println("Type text to store (empty line to skip, 'exit' to quit).")

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

		id := uuid.NewString()
		addReq := chromaapi.AddRequest{
			IDs:        []string{id},
			Documents:  []string{text},
			Metadatas:  []map[string]any{{"source": "cli"}},
			Embeddings: [][]float32{embeddingVec},
		}

		ctxAdd, cancelAdd := context.WithTimeout(context.Background(), 30*time.Second)
		err = chromaClient.Add(ctxAdd, snap.CollectionID, addReq)
		cancelAdd()
		if err != nil {
			log.Printf("chroma add failed: %v", err)
			continue
		}

		fmt.Printf("Stored entry %s\n", id)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
