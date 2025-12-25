package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"llm/chroma/internal/chromaapi"
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

	client := chromaapi.NewClient(getenv("CHROMA_URL", defaultChromaURL))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := client.DeleteCollection(ctx, snap.CollectionID); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "does not exist") ||
			strings.Contains(err.Error(), "404") {
			log.Printf("collection already gone in Chroma, removing local state...")
		} else {
			log.Fatalf("delete collection: %v", err)
		}
	}

	if err := state.Remove(stateDir); err != nil {
		log.Printf("warning: failed removing state file: %v", err)
	}

	fmt.Printf("Deleted collection %q (%s)\n", snap.CollectionName, snap.CollectionID)
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
