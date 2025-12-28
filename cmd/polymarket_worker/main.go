package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/hetulpatel/Arbitrage/internal/kafka"
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

	log.Printf("[polymarket-worker] consuming %s with group %s (%d workers)", topic, group, workerCount)
	workers.Run(ctx, brokers, topic, group, workerCount, nil)
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
