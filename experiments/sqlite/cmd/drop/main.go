package main

import (
	"log"

	"llm/sqlite/sqliteutil"
)

func main() {
	if err := sqliteutil.Drop(); err != nil {
		log.Fatalf("drop database: %v", err)
	}
	log.Printf("removed %s", sqliteutil.DBPath)
}
