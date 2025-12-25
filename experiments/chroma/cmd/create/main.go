package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"llm/chroma/internal/chromaapi"
	"llm/chroma/state"
)

const (
	defaultChromaURL = "http://chromadb:8000"
	stateDir         = "chroma/data"
)

func main() {
	baseURL := strings.TrimSpace(getenv("CHROMA_URL", defaultChromaURL))
	client := chromaapi.NewClient(baseURL)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	name := fmt.Sprintf("demo-%s", uuid.NewString())
	collection, err := client.CreateCollection(ctx, name)
	if err != nil {
		log.Fatalf("create collection: %v", err)
	}

	if err := state.Save(stateDir, state.Snapshot{CollectionID: collection.ID, CollectionName: collection.Name}); err != nil {
		log.Fatalf("save state: %v", err)
	}

	fmt.Printf("Created collection %q (ID: %s)\n", collection.Name, collection.ID)
	fmt.Println("State saved; use the add/query commands next.")
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
