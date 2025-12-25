package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

func main() {
	addr := getenv("REDIS_ADDR", "redis-server:6379")
	ctx := context.Background()

	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
		DB:   0,
	})
	defer rdb.Close()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("connect to redis at %s: %v", addr, err)
	}

	fmt.Printf("Connected to Redis at %s.\n", addr)
	fmt.Println("Type any string to cache it, 'flush' to clear the DB, or 'exit' to quit.")

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

		switch strings.ToLower(text) {
		case "exit", "quit":
			fmt.Println("Goodbye.")
			return
		case "flush":
			if err := rdb.FlushDB(ctx).Err(); err != nil {
				log.Printf("failed flushing redis: %v", err)
				continue
			}
			fmt.Println("Cache cleared.")
			continue
		}

		val, err := rdb.Get(ctx, text).Result()
		switch {
		case err == nil:
			fmt.Printf("CACHE HIT: %q was stored with value %q\n", text, val)
		case err == redis.Nil:
			if err := rdb.Set(ctx, text, text, 24*time.Hour).Err(); err != nil {
				log.Printf("failed caching %q: %v", text, err)
				continue
			}
			fmt.Printf("CACHE MISS: added %q to Redis (expires in 24h)\n", text)
		default:
			log.Printf("redis error: %v", err)
		}
	}
}

func getenv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
