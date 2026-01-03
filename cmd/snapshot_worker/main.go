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
	"github.com/hetulpatel/Arbitrage/internal/cache"
	"github.com/hetulpatel/Arbitrage/internal/collectors"
	"github.com/hetulpatel/Arbitrage/internal/kafka"
	"github.com/hetulpatel/Arbitrage/internal/kalshi"
	"github.com/hetulpatel/Arbitrage/internal/llm"
	"github.com/hetulpatel/Arbitrage/internal/logging"
	"github.com/hetulpatel/Arbitrage/internal/matches"
	"github.com/hetulpatel/Arbitrage/internal/models"
	"github.com/hetulpatel/Arbitrage/internal/polymarket"
	sqlstore "github.com/hetulpatel/Arbitrage/internal/storage/sqlite"
	"github.com/hetulpatel/Arbitrage/internal/validator"
)

const (
	profitEpsilon = 1e-9
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	logging.InitFromEnv()

	brokers := kafka.Brokers()
	topic := kafka.TopicFromEnv("MATCHES_KAFKA_TOPIC", kafka.DefaultMatchTopic)
	group := envString("SNAPSHOT_WORKER_GROUP", "snapshot-worker")
	workerCount := envInt("SNAPSHOT_WORKER_CONCURRENCY", 1)
	budget := envFloat("SNAPSHOT_WORKER_BUDGET_USD", 100)

	waitCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	if err := kafka.WaitForBroker(waitCtx, brokers); err != nil {
		logging.Fatalf("[snapshot-worker] wait for broker: %v", err)
	}
	cancel()

	llmClient := mustLLMClient()
	valSvc := mustValidatorService(llmClient)
	pmClient := mustPolymarketClient()
	kxClient := mustKalshiClient()
	verdictCache := mustVerdictCache()
	if verdictCache != nil {
		defer verdictCache.Close()
	}

	store := mustSQLiteStore()
	defer store.Close()

	logging.Infof("[snapshot-worker] consuming %s with group %s (%d workers, budget=%.2f)", topic, group, workerCount, budget)
	runWorkers(ctx, brokers, topic, group, workerCount, budget, workerDeps{
		validator:    valSvc,
		pmClient:     pmClient,
		kxClient:     kxClient,
		finalBudget:  budget,
		verdictCache: verdictCache,
		store:        store,
	})
}

type workerDeps struct {
	validator    *validator.Service
	pmClient     *polymarket.Client
	kxClient     *kalshi.Client
	finalBudget  float64
	verdictCache cache.VerdictCache
	store        *sqlstore.Store
}

func runWorkers(ctx context.Context, brokers []string, topic, group string, workerCount int, budget float64, deps workerDeps) {
	if workerCount <= 0 {
		workerCount = 1
	}
	if deps.validator == nil {
		logging.Errorf("[snapshot-worker] validator not configured; exiting")
		return
	}
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			consume(ctx, brokers, topic, group, budget, deps)
		}(i)
	}
	<-ctx.Done()
	wg.Wait()
}

func consume(ctx context.Context, brokers []string, topic, group string, budget float64, deps workerDeps) {
	reader := kafka.NewReader(brokers, topic, group)
	defer reader.Close()

	forceFirst := envBool("SNAPSHOT_WORKER_FORCE_VALIDATION", false)
	bypassLLM := envBool("SNAPSHOT_WORKER_BYPASS_LLM", false)
	cfg := arb.Config{
		BudgetUSD:    budget,
		ForceVerdict: forceFirst,
	}
	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			logging.Errorf("[snapshot-worker] read error: %v", err)
			continue
		}

		var payload matches.Payload
		if err := json.Unmarshal(msg.Value, &payload); err != nil {
			logging.Errorf("[snapshot-worker] unmarshal error: %v", err)
			continue
		}

		if payload.CachedVerdict && payload.ResolutionVerdict != nil && payload.ResolutionVerdict.ValidResolution {
			logging.Infof("[snapshot-worker] pair=%s using cached SAFE verdict", payload.PairID)
			logLLMResult(&payload)
			appendValidationLog(&payload)
			if err := deps.runFinalStage(ctx, &payload); err != nil {
				logging.Errorf("[snapshot-worker] final stage error pair=%s: %v", payload.PairID, err)
			}
			continue
		}

		result := arb.Evaluate(&payload, cfg)
		if result.Untradable {
			logging.Infof("[snapshot-worker] pair=%s skipped (untradable: %s)", payload.PairID, result.Reason)
			continue
		}
		if result.Best == nil || result.Best.ProfitUSD <= profitEpsilon || result.Best.Quantity <= profitEpsilon {
			logging.Infof("[snapshot-worker] pair=%s skipped (no profitable direction)", payload.PairID)
			continue
		}

		payload.Arbitrage = result.Best

		verdictKey := matches.VerdictCacheKey(&payload.Source, &payload.Target)
		if payload.CachedVerdict && payload.ResolutionVerdict != nil && payload.ResolutionVerdict.ValidResolution {
			logging.Infof("[snapshot-worker] pair=%s using cached SAFE verdict", payload.PairID)
			logLLMResult(&payload)
			appendValidationLog(&payload)
			if err := deps.runFinalStage(ctx, &payload); err != nil {
				logging.Errorf("[snapshot-worker] final stage error pair=%s: %v", payload.PairID, err)
			}
			continue
		}

		var verdict *matches.ResolutionVerdict
		if bypassLLM {
			verdict = matches.NewResolutionVerdict(true, "bypassed via SNAPSHOT_WORKER_BYPASS_LLM")
		} else {
			res, err := deps.validator.Validate(ctx, &payload)
			if err != nil {
				logging.Errorf("[snapshot-worker] validator error pair=%s: %v", payload.PairID, err)
				continue
			}
			verdict = matches.NewResolutionVerdict(res.ValidResolution, res.ResolutionReason)
		}

		payload.ResolutionVerdict = verdict
		logLLMResult(&payload)
		appendValidationLog(&payload)
		if deps.verdictCache != nil && verdictKey != "" {
			if err := deps.verdictCache.Set(ctx, verdictKey, verdict.ValidResolution); err != nil {
				logging.Errorf("[verdict-cache] set error key=%s: %v", verdictKey, err)
			} else {
				logging.Infof("[verdict-cache] stored key=%s valid=%t", verdictKey, verdict.ValidResolution)
			}
		}

		if verdict.ValidResolution {
			if err := deps.runFinalStage(ctx, &payload); err != nil {
				logging.Errorf("[snapshot-worker] final stage error pair=%s: %v", payload.PairID, err)
			}
		}
	}
}

func logLLMResult(payload *matches.Payload) {
	if payload == nil || payload.ResolutionVerdict == nil {
		return
	}
	pm := questionForVenue(payload, collectors.VenuePolymarket)
	kx := questionForVenue(payload, collectors.VenueKalshi)
	logging.Infof("[snapshot-worker] LLM pair=%s polymarket=\"%s\" kalshi=\"%s\" valid=%t reason=%s",
		payload.PairID, pm, kx, payload.ResolutionVerdict.ValidResolution, payload.ResolutionVerdict.ResolutionReason)
}

func questionForVenue(payload *matches.Payload, venue collectors.Venue) string {
	if payload == nil {
		return ""
	}
	if payload.Source.Venue == venue {
		if payload.Source.Market.Question != "" {
			return payload.Source.Market.Question
		}
		return payload.Source.Event.Title
	}
	if payload.Target.Venue == venue {
		if payload.Target.Market.Question != "" {
			return payload.Target.Market.Question
		}
		return payload.Target.Event.Title
	}
	return ""
}

func snapshotForVenue(payload *matches.Payload, venue collectors.Venue) *models.MarketSnapshot {
	if payload == nil {
		return nil
	}
	if payload.Source.Venue == venue {
		return &payload.Source
	}
	if payload.Target.Venue == venue {
		return &payload.Target
	}
	return nil
}

func appendValidationLog(payload *matches.Payload) {
	if payload == nil {
		return
	}
	entry := map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"payload":   payload,
	}
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		logging.Errorf("[snapshot-worker] validator log marshal error: %v", err)
		return
	}
	f, err := os.OpenFile("validator.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		logging.Errorf("[snapshot-worker] validator log open error: %v", err)
		return
	}
	defer f.Close()
	if _, err := f.Write(append(data, '\n')); err != nil {
		logging.Errorf("[snapshot-worker] validator log write error: %v", err)
	}
}

func (d workerDeps) runFinalStage(parentCtx context.Context, payload *matches.Payload) error {
	if payload == nil {
		return fmt.Errorf("nil payload")
	}
	if d.pmClient == nil || d.kxClient == nil {
		return fmt.Errorf("refresh clients not configured")
	}
	pmSnap := snapshotForVenue(payload, collectors.VenuePolymarket)
	kxSnap := snapshotForVenue(payload, collectors.VenueKalshi)
	if pmSnap == nil || kxSnap == nil {
		return fmt.Errorf("missing source snapshots")
	}

	timeout := time.Duration(envInt("FRESH_FETCH_TIMEOUT_SECONDS", 15)) * time.Second
	ctx, cancel := context.WithTimeout(parentCtx, timeout)
	defer cancel()

	freshPM, err := d.pmClient.MarketSnapshot(ctx, pmSnap.Event.EventID, pmSnap.Market.MarketID)
	if err != nil {
		return fmt.Errorf("refresh polymarket: %w", err)
	}
	freshKX, err := d.kxClient.MarketSnapshot(ctx, kxSnap.Event.EventID, kxSnap.Market.MarketID, "")
	if err != nil {
		return fmt.Errorf("refresh kalshi: %w", err)
	}

	payload.Fresh = &matches.FreshSnapshots{
		Polymarket: freshPM,
		Kalshi:     freshKX,
	}

	freshPayload := matches.Payload{
		PairID:    payload.PairID,
		Source:    *freshPM,
		Target:    *freshKX,
		MatchedAt: time.Now().UTC(),
	}
	result := arb.Evaluate(&freshPayload, arb.Config{BudgetUSD: d.finalBudget})
	payload.FinalOpportunity = result.Best

	if result.Best != nil {
		if err := d.store.InsertArbOpportunity(parentCtx, payload, result); err != nil {
			logging.Errorf("[snapshot-worker] sqlite insert error pair=%s: %v", payload.PairID, err)
		}
	}

	if result.Best == nil {
		fmt.Printf("[snapshot-worker] final pair=%s no profitable direction after refresh\n", payload.PairID)
		appendFinalLog(payload)
		return nil
	}
	fmt.Printf("[snapshot-worker] final pair=%s dir=%s qty=%.2f profit=%.4f\n", payload.PairID, result.Best.Direction, result.Best.Quantity, result.Best.ProfitUSD)
	appendFinalLog(payload)
	return nil
}

func mustLLMClient() *llm.Client {
	cfg := llm.Config{
		APIKey:      os.Getenv("NEBIUS_API_KEY"),
		BaseURL:     envString("NEBIUS_BASE_URL", ""),
		Model:       envString("VALIDATOR_MODEL", ""),
		Temperature: 0,
		MaxTokens:   envInt("VALIDATOR_MAX_TOKENS", 800),
		Timeout:     time.Duration(envInt("VALIDATOR_TIMEOUT_SECONDS", 45)) * time.Second,
	}
	client, err := llm.New(cfg)
	if err != nil {
		logging.Fatalf("[snapshot-worker] llm client: %v", err)
	}
	return client
}

func mustValidatorService(llmClient *llm.Client) *validator.Service {
	pdfExtractor := validator.NewCommandPDFExtractor("")
	svc, err := validator.NewService(validator.Config{
		LLMClient:    llmClient,
		PDFExtractor: pdfExtractor,
		SystemPrompt: envString("VALIDATOR_SYSTEM_PROMPT", ""),
	})
	if err != nil {
		logging.Fatalf("[snapshot-worker] validator init: %v", err)
	}
	return svc
}

func mustVerdictCache() cache.VerdictCache {
	addr := envString("REDIS_ADDR", "redis:6379")
	if addr == "" {
		return nil
	}
	db := envInt("REDIS_DB", 0)
	ttlHours := envInt("REDIS_EMBED_TTL_HOURS", 240)
	cacheClient, err := cache.NewRedisVerdictCache(addr, os.Getenv("REDIS_PASSWORD"), db, time.Duration(ttlHours)*time.Hour, "pair_verdict")
	if err != nil {
		logging.Fatalf("[snapshot-worker] redis verdict cache: %v", err)
	}
	return cacheClient
}

func mustPolymarketClient() *polymarket.Client {
	cfg := polymarket.Config{
		BaseURL: envString("POLYMARKET_API_URL", ""),
		BookURL: envString("POLYMARKET_BOOK_URL", ""),
		Timeout: time.Duration(envInt("POLYMARKET_HTTP_TIMEOUT_SECONDS", 20)) * time.Second,
	}
	return polymarket.NewClient(cfg)
}

func mustKalshiClient() *kalshi.Client {
	cfg := kalshi.Config{
		BaseURL:   envString("KALSHI_API_URL", ""),
		SeriesURL: envString("KALSHI_SERIES_URL", ""),
		BookURL:   envString("KALSHI_MARKET_URL", ""),
		Timeout:   time.Duration(envInt("KALSHI_HTTP_TIMEOUT_SECONDS", 20)) * time.Second,
	}
	return kalshi.NewClient(cfg)
}

func appendFinalLog(payload *matches.Payload) {
	if payload == nil {
		return
	}
	entry := map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"payload":   payload,
	}
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		logging.Errorf("[snapshot-worker] final log marshal error: %v", err)
		return
	}
	f, err := os.OpenFile("final_arb.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		logging.Errorf("[snapshot-worker] final log open error: %v", err)
		return
	}
	defer f.Close()
	if _, err := f.Write(append(data, '\n')); err != nil {
		logging.Errorf("[snapshot-worker] final log write error: %v", err)
	}
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

func envBool(key string, def bool) bool {
	if val := os.Getenv(key); val != "" {
		if parsed, err := strconv.ParseBool(val); err == nil {
			return parsed
		}
	}
	return def
}

func mustSQLiteStore() *sqlstore.Store {
	path := os.Getenv("SQLITE_PATH")
	if path == "" {
		path = "data/arb.db"
	}
	store, err := sqlstore.Open(path)
	if err != nil {
		logging.Fatalf("[snapshot-worker] sqlite open: %v", err)
	}
	return store
}
