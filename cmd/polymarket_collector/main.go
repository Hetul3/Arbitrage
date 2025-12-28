package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"time"

	kafkago "github.com/segmentio/kafka-go"

	"github.com/hetulpatel/Arbitrage/internal/collectors"
	kafkautil "github.com/hetulpatel/Arbitrage/internal/kafka"
	"github.com/hetulpatel/Arbitrage/internal/polymarket"
	"github.com/hetulpatel/Arbitrage/internal/queue"
	sqlstore "github.com/hetulpatel/Arbitrage/internal/storage/sqlite"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	store, err := sqlstore.Open(os.Getenv("SQLITE_PATH"))
	if err != nil {
		log.Fatalf("open sqlite: %v", err)
	}
	defer store.Close()

	writer := setupWriter(ctx, "POLYMARKET_KAFKA_TOPIC", kafka.DefaultPolyTopic)
	defer func() {
		if writer != nil {
			writer.Close()
		}
	}()

	collector := polymarket.NewClient(polymarket.Config{})
	opts := collectors.FetchOptions{
		Pages:    envInt("POLYMARKET_PAGES", 1),
		PageSize: envInt("POLYMARKET_PAGE_SIZE", 20),
	}

	collectors.RunLoop(ctx, collector, opts, func(ctx context.Context, events []collectors.Event) error {
		log.Printf("[polymarket] fetched %d events", len(events))
		if err := store.UpsertPolymarketEvents(ctx, events); err != nil {
			return err
		}
		if err := queue.PublishSnapshots(ctx, writer, collectors.VenuePolymarket, events); err != nil {
			log.Printf("[polymarket] publish error: %v", err)
		}
		return nil
	})
}

func setupWriter(ctx context.Context, envKey, fallbackTopic string) *kafkago.Writer {
	brokers := kafkautil.Brokers()
	topic := kafkautil.TopicFromEnv(envKey, fallbackTopic)
	waitCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	if err := kafkautil.WaitForBroker(waitCtx, brokers); err != nil {
		log.Printf("[polymarket] kafka unavailable: %v", err)
		return nil
	}
	ensureCtx, cancelEnsure := context.WithTimeout(ctx, 30*time.Second)
	if err := kafkautil.EnsureTopic(ensureCtx, brokers, topic); err != nil {
		log.Printf("[polymarket] ensure topic warning: %v", err)
	}
	cancelEnsure()
	return kafkautil.NewWriter(brokers, topic)
}

func envInt(key string, def int) int {
	if val := os.Getenv(key); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			return parsed
		}
	}
	return def
}
