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

	"github.com/hetulpatel/Arbitrage/internal/kafka"
	"github.com/hetulpatel/Arbitrage/internal/models"
	"github.com/hetulpatel/Arbitrage/internal/workers"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	brokers := kafka.Brokers()
	topic := kafka.TopicFromEnv("POLYMARKET_KAFKA_TOPIC", kafka.DefaultPolyTopic)
	group := envString("POLYMARKET_WORKER_GROUP", "polymarket-workers-dev")
	workerCount := envInt("POLYMARKET_WORKERS", 1)
	verbose := envBool("POLYMARKET_WORKER_VERBOSE", false)

	waitCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	if err := kafka.WaitForBroker(waitCtx, brokers); err != nil {
		log.Fatalf("[polymarket-worker-dev] wait for broker: %v", err)
	}
	cancel()

	ensureCtx, cancelEnsure := context.WithTimeout(ctx, 30*time.Second)
	if err := kafka.EnsureTopic(ensureCtx, brokers, topic); err != nil {
		log.Printf("[polymarket-worker-dev] ensure topic warning: %v", err)
	}
	cancelEnsure()

	log.Printf("[polymarket-worker-dev] consuming %s with group %s (%d workers, verbose=%t)", topic, group, workerCount, verbose)
	workers.Run(ctx, brokers, topic, group, workerCount, func(_ context.Context, snap *models.MarketSnapshot) error {
		if verbose {
			b, err := json.MarshalIndent(snap, "", "  ")
			if err != nil {
				return err
			}
			fmt.Printf("[polymarket-worker-dev] %s\n", string(b))
		} else {
			fmt.Printf("[polymarket-worker-dev] consumed market=%s event=%s captured=%s\n",
				snap.Market.MarketID, snap.Event.EventID, snap.CapturedAt.Format(time.RFC3339))
		}
		return nil
	})
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
