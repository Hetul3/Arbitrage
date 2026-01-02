package collectors

import (
	"context"

	"github.com/hetulpatel/Arbitrage/internal/logging"
)

// RunLoop continuously fetches data from a collector and hands it to handleFn.
// It immediately polls again after each iteration; rate limiting/backoff is handled
// inside the collector's HTTP client.
func RunLoop(ctx context.Context, collector Collector, opts FetchOptions, handleFn func(context.Context, []Event) error) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		events, err := collector.Fetch(ctx, opts)
		if err != nil {
			logging.Errorf("[%s] fetch failed: %v", collector.Name(), err)
		} else if handleFn != nil && len(events) > 0 {
			if err := handleFn(ctx, events); err != nil {
				logging.Errorf("[%s] handler error: %v", collector.Name(), err)
			}
		}
	}
}
