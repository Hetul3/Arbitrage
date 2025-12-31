package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"

	"github.com/hetulpatel/Arbitrage/internal/arb"
	"github.com/hetulpatel/Arbitrage/internal/collectors"
	"github.com/hetulpatel/Arbitrage/internal/kafka"
	"github.com/hetulpatel/Arbitrage/internal/llm"
	"github.com/hetulpatel/Arbitrage/internal/matches"
	"github.com/hetulpatel/Arbitrage/internal/validator"
)

const (
	profitEpsilon = 1e-9
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	brokers := kafka.Brokers()
	topic := kafka.TopicFromEnv("MATCHES_KAFKA_TOPIC", kafka.DefaultMatchTopic)
	group := envString("SNAPSHOT_WORKER_GROUP", "snapshot-worker")
	workerCount := envInt("SNAPSHOT_WORKER_CONCURRENCY", 1)
	budget := envFloat("SNAPSHOT_WORKER_BUDGET_USD", 100)

	waitCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	if err := kafka.WaitForBroker(waitCtx, brokers); err != nil {
		log.Fatalf("[snapshot-worker] wait for broker: %v", err)
	}
	cancel()

	llmClient := mustLLMClient()
	valSvc := mustValidatorService(llmClient)

	log.Printf("[snapshot-worker] consuming %s with group %s (%d workers, budget=%.2f)", topic, group, workerCount, budget)
	runWorkers(ctx, brokers, topic, group, workerCount, budget, valSvc)
}

func runWorkers(ctx context.Context, brokers []string, topic, group string, workerCount int, budget float64, validatorSvc *validator.Service) {
	if workerCount <= 0 {
		workerCount = 1
	}
	if validatorSvc == nil {
		log.Printf("[snapshot-worker] validator not configured; exiting")
		return
	}
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			consume(ctx, brokers, topic, group, budget, validatorSvc)
		}(i)
	}
	<-ctx.Done()
	wg.Wait()
}

func consume(ctx context.Context, brokers []string, topic, group string, budget float64, validatorSvc *validator.Service) {
	reader := kafka.NewReader(brokers, topic, group)
	defer reader.Close()

	cfg := arb.Config{
		BudgetUSD:    budget,
		ForceVerdict: envBool("SNAPSHOT_WORKER_FORCE_VALIDATION", false),
	}
	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("[snapshot-worker] read error: %v", err)
			continue
		}

		var payload matches.Payload
		if err := json.Unmarshal(msg.Value, &payload); err != nil {
			log.Printf("[snapshot-worker] unmarshal error: %v", err)
			continue
		}

		result := arb.Evaluate(&payload, cfg)
		if result.Untradable {
			log.Printf("[snapshot-worker] pair=%s skipped (untradable: %s)", payload.PairID, result.Reason)
			continue
		}
		if result.Best == nil || result.Best.ProfitUSD <= profitEpsilon || result.Best.Quantity <= profitEpsilon {
			log.Printf("[snapshot-worker] pair=%s skipped (no profitable direction)", payload.PairID)
			continue
		}

		payload.Arbitrage = result.Best

		verdict, err := validatorSvc.Validate(ctx, &payload)
		if err != nil {
			log.Printf("[snapshot-worker] validator error pair=%s: %v", payload.PairID, err)
			continue
		}
		payload.ResolutionVerdict = matches.NewResolutionVerdict(verdict.ValidResolution, verdict.ResolutionReason)
		logLLMResult(&payload)
		appendValidationLog(&payload)
	}
}

func logLLMResult(payload *matches.Payload) {
	if payload == nil || payload.ResolutionVerdict == nil {
		return
	}
	pm := questionForVenue(payload, collectors.VenuePolymarket)
	kx := questionForVenue(payload, collectors.VenueKalshi)
	log.Printf("[snapshot-worker] LLM pair=%s polymarket=\"%s\" kalshi=\"%s\" valid=%t reason=%s",
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
		log.Printf("[snapshot-worker] validator log marshal error: %v", err)
		return
	}
	f, err := os.OpenFile("validator.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		log.Printf("[snapshot-worker] validator log open error: %v", err)
		return
	}
	defer f.Close()
	if _, err := f.Write(append(data, '\n')); err != nil {
		log.Printf("[snapshot-worker] validator log write error: %v", err)
	}
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
		log.Fatalf("[snapshot-worker] llm client: %v", err)
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
		log.Fatalf("[snapshot-worker] validator init: %v", err)
	}
	return svc
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
