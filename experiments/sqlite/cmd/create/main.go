package main

import (
	"log"

	"llm/sqlite/sqliteutil"
)

func main() {
	db, err := sqliteutil.Open()
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	if err := sqliteutil.EnsureSchema(db); err != nil {
		log.Fatalf("ensure schema: %v", err)
	}

	log.Printf("created database schema at %s", sqliteutil.DBPath)
}
