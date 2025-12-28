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
	topic := kafka.TopicFromEnv("KALSHI_KAFKA_TOPIC", kafka.DefaultKalshiTopic)
	group := envString("KALSHI_WORKER_GROUP", "kalshi-workers")
	workerCount := envInt("KALSHI_WORKERS", 2)

	waitCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	if err := kafka.WaitForBroker(waitCtx, brokers); err != nil {
		log.Fatalf("[kalshi-worker] wait for broker: %v", err)
	}
	cancel()

	ensureCtx, cancelEnsure := context.WithTimeout(ctx, 30*time.Second)
	if err := kafka.EnsureTopic(ensureCtx, brokers, topic); err != nil {
		log.Printf("[kalshi-worker] ensure topic warning: %v", err)
	}
	cancelEnsure()

	log.Printf("[kalshi-worker] consuming %s with group %s (%d workers)", topic, group, workerCount)
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
