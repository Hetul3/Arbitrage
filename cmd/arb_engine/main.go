package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"

	"github.com/hetulpatel/Arbitrage/internal/arb"
	"github.com/hetulpatel/Arbitrage/internal/kafka"
	"github.com/hetulpatel/Arbitrage/internal/logging"
	"github.com/hetulpatel/Arbitrage/internal/matches"
	sqlstore "github.com/hetulpatel/Arbitrage/internal/storage/sqlite"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	logging.InitFromEnv()

	brokers := kafka.Brokers()
	topic := kafka.TopicFromEnv("MATCHES_KAFKA_TOPIC", kafka.DefaultMatchTopic)
	group := envString("ARB_ENGINE_GROUP", "arb-engine")
	workerCount := envInt("ARB_ENGINE_WORKERS", 1)
	budget := envFloat("ARB_ENGINE_BUDGET_USD", 100)

	waitCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	if err := kafka.WaitForBroker(waitCtx, brokers); err != nil {
		logging.Fatalf("[arb-engine] wait for broker: %v", err)
	}
	cancel()

	ensureCtx, cancelEnsure := context.WithTimeout(ctx, 30*time.Second)
	if err := kafka.EnsureTopic(ensureCtx, brokers, topic); err != nil {
		logging.Errorf("[arb-engine] ensure topic warning: %v", err)
	}
	cancelEnsure()

	store, err := sqlstore.Open(os.Getenv("SQLITE_PATH"))
	if err != nil {
		logging.Fatalf("[arb-engine] open sqlite: %v", err)
	}
	defer store.Close()

	logging.Infof("[arb-engine] consuming %s with group %s (%d workers, budget=%.2f)", topic, group, workerCount, budget)
	runWorkers(ctx, brokers, topic, group, workerCount, budget, store)
}

func runWorkers(ctx context.Context, brokers []string, topic, group string, workerCount int, budget float64, store *sqlstore.Store) {
	if workerCount <= 0 {
		workerCount = 1
	}
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			consume(ctx, brokers, topic, group, budget, store)
		}(i)
	}
	<-ctx.Done()
	wg.Wait()
}

func consume(ctx context.Context, brokers []string, topic, group string, budget float64, store *sqlstore.Store) {
	reader := kafka.NewReader(brokers, topic, group)
	defer reader.Close()

	cfg := arb.Config{BudgetUSD: budget}
	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			logging.Errorf("[arb-engine] read error: %v", err)
			continue
		}
		var payload matches.Payload
		if err := json.Unmarshal(msg.Value, &payload); err != nil {
			logging.Errorf("[arb-engine] unmarshal error: %v", err)
			continue
		}
		result := arb.Evaluate(&payload, cfg)
		if result.Best != nil {
			payload.Arbitrage = result.Best
		}
		logOpportunity(&payload, result)
		if err := store.InsertArbOpportunity(ctx, &payload, result); err != nil {
			logging.Errorf("[arb-engine] sqlite error: %v", err)
		}
	}
}

func logOpportunity(payload *matches.Payload, result arb.Result) {
	pairID := payload.PairID
	if pairID == "" {
		pairID = "unknown"
	}

	if result.Untradable {
		logging.Errorf("[arb-engine] pair=%s UNTRADABLE reason=%s", pairID, result.Reason)
		return
	}

	if result.Best == nil || result.Best.Quantity <= 0 {
		logging.Infof("[arb-engine] pair=%s no opportunity found", pairID)
		return
	}
	fmt.Printf("[arb-opportunity] pair=%s dir=%s qty=%.2f cost=%.4f profit=%.4f fees=%.4f\n",
		pairID, result.Best.Direction, result.Best.Quantity, result.Best.TotalCostUSD, result.Best.ProfitUSD, result.Best.KalshiFeesUSD+result.Best.PolymarketFeesUSD)
}

func envInt(key string, def int) int {
	if val := os.Getenv(key); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			return parsed
		}
	}
	return def
}

func envFloat(key string, def float64) float64 {
	if val := os.Getenv(key); val != "" {
		if parsed, err := strconv.ParseFloat(val, 64); err == nil {
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
