package main

import (
	"context"
	"os"
	"os/signal"
	"strconv"
	"time"

	kafkago "github.com/segmentio/kafka-go"

	"github.com/hetulpatel/Arbitrage/internal/collectors"
	kafkautil "github.com/hetulpatel/Arbitrage/internal/kafka"
	"github.com/hetulpatel/Arbitrage/internal/kalshi"
	"github.com/hetulpatel/Arbitrage/internal/logging"
	"github.com/hetulpatel/Arbitrage/internal/queue"
	sqlstore "github.com/hetulpatel/Arbitrage/internal/storage/sqlite"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	logging.InitFromEnv()

	store, err := sqlstore.Open(os.Getenv("SQLITE_PATH"))
	if err != nil {
		logging.Fatalf("open sqlite: %v", err)
	}
	defer store.Close()

	writer := setupWriter(ctx, "KALSHI_KAFKA_TOPIC", kafkautil.DefaultKalshiTopic)
	defer func() {
		if writer != nil {
			writer.Close()
		}
	}()

	collector := kalshi.NewClient(kalshi.Config{})
	opts := collectors.FetchOptions{
		PageSize: envInt("KALSHI_PAGE_SIZE", 100),
	}

	collectors.RunLoop(ctx, collector, opts, func(ctx context.Context, events []collectors.Event) error {
		logging.Infof("[kalshi] fetched %d events", len(events))
		if err := store.UpsertKalshiEvents(ctx, events); err != nil {
			return err
		}
		if err := queue.PublishSnapshots(ctx, writer, collectors.VenueKalshi, events); err != nil {
			logging.Errorf("[kalshi] publish error: %v", err)
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
		logging.Errorf("[kalshi] kafka unavailable: %v", err)
		return nil
	}
	ensureCtx, cancelEnsure := context.WithTimeout(ctx, 30*time.Second)
	if err := kafkautil.EnsureTopic(ensureCtx, brokers, topic); err != nil {
		logging.Errorf("[kalshi] ensure topic warning: %v", err)
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
