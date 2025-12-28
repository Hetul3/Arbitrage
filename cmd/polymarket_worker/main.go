package main

import (
	"context"
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
	topic := kafka.TopicFromEnv("POLYMARKET_KAFKA_TOPIC", kafka.DefaultPolyTopic)
	group := envString("POLYMARKET_WORKER_GROUP", "polymarket-workers")
	workerCount := envInt("POLYMARKET_WORKERS", 2)

	waitCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	if err := kafka.WaitForBroker(waitCtx, brokers); err != nil {
		log.Fatalf("[polymarket-worker] wait for broker: %v", err)
	}
	cancel()

	ensureCtx, cancelEnsure := context.WithTimeout(ctx, 30*time.Second)
	if err := kafka.EnsureTopic(ensureCtx, brokers, topic); err != nil {
		log.Printf("[polymarket-worker] ensure topic warning: %v", err)
	}
	cancelEnsure()

	embedClient := mustEmbedClient()
	chromaClient, collectionID := mustChromaClient(ctx)
	processor := workers.NewProcessor(embedClient, chromaClient, collectionID, "polymarket")

	log.Printf("[polymarket-worker] consuming %s with group %s (%d workers)", topic, group, workerCount)
	workers.Run(ctx, brokers, topic, group, workerCount, func(ctx context.Context, snap *models.MarketSnapshot) error {
		if err := processor.Handle(ctx, snap); err != nil {
			return err
		}
		log.Printf("[polymarket-worker] upserted market=%s event=%s", snap.Market.MarketID, snap.Event.EventID)
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
		log.Fatalf("[polymarket-worker] embed client: %v", err)
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
		log.Fatalf("[polymarket-worker] ensure chroma collection: %v", err)
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
