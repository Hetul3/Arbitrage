package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/hetulpatel/Arbitrage/internal/chroma"
	"github.com/hetulpatel/Arbitrage/internal/embed"
	"github.com/hetulpatel/Arbitrage/internal/kafka"
	"github.com/hetulpatel/Arbitrage/internal/models"
	"github.com/hetulpatel/Arbitrage/internal/workers"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	brokers := kafka.Brokers()
	topic := kafka.TopicFromEnv("KALSHI_KAFKA_TOPIC", kafka.DefaultKalshiTopic)
	group := envString("KALSHI_WORKER_GROUP", "kalshi-workers-dev")
	workerCount := envInt("KALSHI_WORKERS", 1)
	verbose := envBool("KALSHI_WORKER_VERBOSE", false)

	waitCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	if err := kafka.WaitForBroker(waitCtx, brokers); err != nil {
		log.Fatalf("[kalshi-worker-dev] wait for broker: %v", err)
	}
	cancel()

	ensureCtx, cancelEnsure := context.WithTimeout(ctx, 30*time.Second)
	if err := kafka.EnsureTopic(ensureCtx, brokers, topic); err != nil {
		log.Printf("[kalshi-worker-dev] ensure topic warning: %v", err)
	}
	cancelEnsure()

	embedClient := mustEmbedClient()
	chromaClient, collectionID := mustChromaClient(ctx)
	processor := workers.NewProcessor(embedClient, chromaClient, collectionID, "kalshi")

	log.Printf("[kalshi-worker-dev] consuming %s with group %s (%d workers, verbose=%t)", topic, group, workerCount, verbose)
	workers.Run(ctx, brokers, topic, group, workerCount, func(_ context.Context, snap *models.MarketSnapshot) error {
		if err := processor.Handle(ctx, snap); err != nil {
			return err
		}
		if verbose {
			b, err := json.MarshalIndent(snap, "", "  ")
			if err != nil {
				return err
			}
			fmt.Printf("[kalshi-worker-dev] %s\n", string(b))
		} else {
			fmt.Printf("[kalshi-worker-dev] upserted market=%s event=%s\n", snap.Market.MarketID, snap.Event.EventID)
		}
		return nil
	})
}

func mustEmbedClient() *embed.Client {
	cfg := embed.Config{
		APIKey:  os.Getenv("NEBIUS_API_KEY"),
		BaseURL: envString("NEBIUS_BASE_URL", ""),
		Model:   envString("NEBIUS_EMBED_MODEL", ""),
	}
	client, err := embed.New(cfg)
	if err != nil {
		log.Fatalf("[kalshi-worker-dev] embed client: %v", err)
	}
	return client
}

func mustChromaClient(ctx context.Context) (*chroma.Client, string) {
	chromaURL := envString("CHROMA_URL", "http://chromadb:8000")
	collectionName := envString("CHROMA_COLLECTION", "market_snapshots")
	client := chroma.NewClient(chromaURL)
	ensureCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	collection, err := client.EnsureCollection(ensureCtx, collectionName)
	if err != nil {
		log.Fatalf("[kalshi-worker-dev] ensure chroma collection: %v", err)
	}
	return client, collection.ID
}

func envInt(key string, def int) int {
	if val := os.Getenv(key); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			return parsed
		}
	}
	return def
}

func envString(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func envBool(key string, def bool) bool {
	if val := os.Getenv(key); val != "" {
		if parsed, err := strconv.ParseBool(val); err == nil {
			return parsed
		}
	}
	return def
}
