package main

import (
	"context"
	"log"
	"os"
	"strconv"

	"github.com/hetulpatel/Arbitrage/internal/collectors"
	"github.com/hetulpatel/Arbitrage/internal/polymarket"
	sqlstore "github.com/hetulpatel/Arbitrage/internal/storage/sqlite"
)

func main() {
	ctx := context.Background()

	store, err := sqlstore.Open(os.Getenv("SQLITE_PATH"))
	if err != nil {
		log.Fatalf("open sqlite: %v", err)
	}
	defer store.Close()

	collector := polymarket.NewClient(polymarket.Config{})
	opts := collectors.FetchOptions{
		Pages:    envInt("POLYMARKET_PAGES", 1),
		PageSize: envInt("POLYMARKET_PAGE_SIZE", 20),
	}

	collectors.RunLoop(ctx, collector, opts, func(ctx context.Context, events []collectors.Event) error {
		log.Printf("[polymarket] fetched %d events", len(events))
		if err := store.UpsertPolymarketEvents(ctx, events); err != nil {
			return err
		}
		return nil
	})
}

func envInt(key string, def int) int {
	if val := os.Getenv(key); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			return parsed
		}
	}
	return def
}
