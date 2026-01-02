package main

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"strconv"
	"time"

	kafkago "github.com/segmentio/kafka-go"

	"github.com/hetulpatel/Arbitrage/internal/cache"
	"github.com/hetulpatel/Arbitrage/internal/chroma"
	"github.com/hetulpatel/Arbitrage/internal/embed"
	"github.com/hetulpatel/Arbitrage/internal/kafka"
	"github.com/hetulpatel/Arbitrage/internal/logging"
	"github.com/hetulpatel/Arbitrage/internal/matcher"
	"github.com/hetulpatel/Arbitrage/internal/matches"
	"github.com/hetulpatel/Arbitrage/internal/models"
	"github.com/hetulpatel/Arbitrage/internal/workers"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	logging.InitFromEnv()

	brokers := kafka.Brokers()
	topic := kafka.TopicFromEnv("POLYMARKET_KAFKA_TOPIC", kafka.DefaultPolyTopic)
	group := envString("POLYMARKET_WORKER_GROUP", "polymarket-workers")
	workerCount := envInt("POLYMARKET_WORKERS", 2)

	waitCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	if err := kafka.WaitForBroker(waitCtx, brokers); err != nil {
		logging.Fatalf("[polymarket-worker] wait for broker: %v", err)
	}
	cancel()

	ensureCtx, cancelEnsure := context.WithTimeout(ctx, 30*time.Second)
	if err := kafka.EnsureTopic(ensureCtx, brokers, topic); err != nil {
		logging.Errorf("[polymarket-worker] ensure topic warning: %v", err)
	}
	cancelEnsure()

	embedClient := mustEmbedClient()
	chromaClient, collectionID := mustChromaClient(ctx)
	embedCache := mustEmbeddingCache()
	defer func() {
		if embedCache != nil {
			embedCache.Close()
		}
	}()

	logCache := envBool("REDIS_EMBED_LOG_HITS", false)
	processor := workers.NewProcessor(embedClient, chromaClient, collectionID, "polymarket", embedCache, logCache)
	finder := mustFinder(chromaClient, collectionID, envBool("MATCH_DEBUG", false))
	matchWriter := setupMatchWriter(ctx, brokers)
	defer func() {
		if matchWriter != nil {
			matchWriter.Close()
		}
	}()

	matchLogger := matcher.NewLogger(matcher.LogModeQuiet)

	logging.Infof("[polymarket-worker] consuming %s with group %s (%d workers)", topic, group, workerCount)
	workers.Run(ctx, brokers, topic, group, workerCount, func(ctx context.Context, snap *models.MarketSnapshot) error {
		embedding, err := processor.Handle(ctx, snap)
		if err != nil {
			return err
		}
		matchCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		res, matchErr := finder.FindBestMatch(matchCtx, snap, embedding)
		cancel()
		if matchErr != nil {
			return matchErr
		}
		if res != nil {
			matchLogger.LogMatch(snap, res, finder.Threshold())
			publishMatch(ctx, matchWriter, snap, res)
		}
		logging.Infof("[polymarket-worker] upserted market=%s event=%s", snap.Market.MarketID, snap.Event.EventID)
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
		logging.Fatalf("[polymarket-worker] embed client: %v", err)
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
		logging.Fatalf("[polymarket-worker] ensure chroma collection: %v", err)
	}
	return client, collection.ID
}

func mustFinder(client *chroma.Client, collectionID string, debug bool) *matcher.Finder {
	cfg := matcher.Config{
		Client:       client,
		CollectionID: collectionID,
		TopK:         envInt("MATCH_TOP_K", 3),
		Threshold:    envFloat("MATCH_SIMILARITY_THRESHOLD", 0.95),
		Freshness:    time.Duration(envInt("MATCH_FRESH_WINDOW_SECONDS", 600)) * time.Second,
		Debug:        debug,
	}
	finder, err := matcher.NewFinder(cfg)
	if err != nil {
		logging.Fatalf("[polymarket-worker] matcher config: %v", err)
	}
	return finder
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

func setupMatchWriter(ctx context.Context, brokers []string) *kafkago.Writer {
	topic := kafka.TopicFromEnv("MATCHES_KAFKA_TOPIC", kafka.DefaultMatchTopic)
	if topic == "" {
		return nil
	}
	ensureCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := kafka.EnsureTopic(ensureCtx, brokers, topic); err != nil {
		logging.Errorf("[polymarket-worker] ensure matches topic warning: %v", err)
	}
	return kafka.NewWriter(brokers, topic)
}

func publishMatch(ctx context.Context, writer *kafkago.Writer, source *models.MarketSnapshot, res *matcher.Result) {
	if writer == nil || res == nil || res.Target == nil {
		return
	}
	sourceCopy := *source
	targetCopy := *res.Target
	payload := matches.NewPayload(sourceCopy, targetCopy, res.Similarity, res.Distance)
	data, err := json.Marshal(payload)
	if err != nil {
		logging.Errorf("[polymarket-worker] marshal match error: %v", err)
		return
	}
	msg := kafkago.Message{
		Key:   []byte(payload.PairID),
		Value: data,
		Time:  payload.MatchedAt,
	}
	if err := writer.WriteMessages(ctx, msg); err != nil {
		logging.Errorf("[polymarket-worker] publish match error: %v", err)
	}
}

func envFloat(key string, def float64) float64 {
	if val := os.Getenv(key); val != "" {
		if parsed, err := strconv.ParseFloat(val, 64); err == nil {
			return parsed
		}
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

func mustEmbeddingCache() cache.EmbeddingCache {
	addr := envString("REDIS_ADDR", "redis:6379")
	if addr == "" {
		return nil
	}
	db := envInt("REDIS_DB", 0)
	ttlHours := envInt("REDIS_EMBED_TTL_HOURS", 240)
	cacheClient, err := cache.NewRedisEmbeddingCache(addr, os.Getenv("REDIS_PASSWORD"), db, time.Duration(ttlHours)*time.Hour, "emb")
	if err != nil {
		logging.Fatalf("[polymarket-worker] redis cache: %v", err)
	}
	return cacheClient
}
