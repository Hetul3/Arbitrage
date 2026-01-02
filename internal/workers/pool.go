package workers

import (
	"context"
	"encoding/json"
	"sync"

	kafkago "github.com/segmentio/kafka-go"

	"github.com/hetulpatel/Arbitrage/internal/kafka"
	"github.com/hetulpatel/Arbitrage/internal/logging"
	"github.com/hetulpatel/Arbitrage/internal/models"
)

type Handler func(context.Context, *models.MarketSnapshot) error

func Run(ctx context.Context, brokers []string, topic, group string, workerCount int, handler Handler) {
	if workerCount <= 0 {
		workerCount = 1
	}

	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			reader := kafka.NewReader(brokers, topic, group)
			defer reader.Close()
			consume(ctx, reader, handler)
		}(i)
	}

	<-ctx.Done()
	wg.Wait()
}

func consume(ctx context.Context, reader *kafkago.Reader, handler Handler) {
	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			logging.Errorf("worker read error: %v", err)
			continue
		}

		var snapshot models.MarketSnapshot
		if err := json.Unmarshal(msg.Value, &snapshot); err != nil {
			logging.Errorf("worker unmarshal error: %v", err)
			continue
		}

		if handler != nil {
			if err := handler(ctx, &snapshot); err != nil {
				logging.Errorf("worker handler error: %v", err)
			}
		}
	}
}
