package main

import (
	"fmt"
	"log"

	"llm/sqlite/sqliteutil"
)

func main() {
	db, err := sqliteutil.Open()
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	users, err := sqliteutil.QueryUsers(db)
	if err != nil {
		log.Fatalf("query users: %v", err)
	}

	if len(users) == 0 {
		fmt.Println("No users found. Run populate first.")
		return
	}

	fmt.Printf("Listing %d users from %s:\n", len(users), sqliteutil.DBPath)
	for _, u := range users {
		fmt.Printf(" - #%d %s <%s>\n", u.ID, u.Name, u.Email)
	}
}
