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

	if err := store.CreateTables(context.Background()); err != nil {
		log.Fatalf("create tables: %v", err)
	}
	log.Printf("SQLite tables created at %s", path)
}
