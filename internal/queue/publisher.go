package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/hetulpatel/Arbitrage/internal/collectors"
	"github.com/hetulpatel/Arbitrage/internal/models"
)

func PublishSnapshots(ctx context.Context, writer *kafka.Writer, venue collectors.Venue, events []collectors.Event) error {
	if writer == nil || len(events) == 0 {
		return nil
	}

	captured := time.Now().UTC()
	msgs := make([]kafka.Message, 0)

	for _, ev := range events {
		if len(ev.Markets) == 0 {
			continue
		}
		for _, m := range ev.Markets {
			snapshot := models.NewSnapshot(venue, ev, m, captured)
			payload, err := json.Marshal(snapshot)
			if err != nil {
				return fmt.Errorf("marshal snapshot %s: %w", m.MarketID, err)
			}
			key := fmt.Sprintf("%s-%s-%d", venue, m.MarketID, snapshot.CapturedAt.UnixNano())
			msgs = append(msgs, kafka.Message{Key: []byte(key), Value: payload})
		}
	}

	if len(msgs) == 0 {
		return nil
	}
	return writer.WriteMessages(ctx, msgs...)
}
