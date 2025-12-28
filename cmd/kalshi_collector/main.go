package main

import (
	"context"
	"log"
	"os"
	"strconv"

	"github.com/hetulpatel/Arbitrage/internal/collectors"
	"github.com/hetulpatel/Arbitrage/internal/kalshi"
	sqlstore "github.com/hetulpatel/Arbitrage/internal/storage/sqlite"
)

func main() {
	ctx := context.Background()

	store, err := sqlstore.Open(os.Getenv("SQLITE_PATH"))
	if err != nil {
		log.Fatalf("open sqlite: %v", err)
	}
	defer store.Close()

	collector := kalshi.NewClient(kalshi.Config{})
	opts := collectors.FetchOptions{
		Pages:    envInt("KALSHI_PAGES", 1),
		PageSize: envInt("KALSHI_PAGE_SIZE", 20),
	}

	collectors.RunLoop(ctx, collector, opts, func(ctx context.Context, events []collectors.Event) error {
		log.Printf("[kalshi] fetched %d events", len(events))
		return store.UpsertKalshiEvents(ctx, events)
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
