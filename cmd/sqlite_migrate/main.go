package main

import (
	"context"
	"log"
	"os"

	"github.com/hetulpatel/Arbitrage/internal/storage/sqlite"
)

func main() {
	path := os.Getenv("SQLITE_PATH")
	store, err := sqlite.Open(path)
	if err != nil {
		log.Fatalf("open sqlite: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	if err := store.MigrateToUnifiedSchema(ctx); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	log.Printf("SQLite schema migrated to unified markets table at %s", store.Path())
}
