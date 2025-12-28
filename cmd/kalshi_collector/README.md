# Kalshi Collector

Production-style entrypoint that continuously fetches Kalshi events/markets (events → nested markets → series metadata → orderbooks), normalizes them, upserts into SQLite, and publishes per-market snapshots into Kafka (`kalshi.snapshots`). HTTP calls automatically retry with exponential backoff when Kalshi rate-limits or returns transient errors. The dev variant prints full JSON for inspection.

## Running

```sh
make run-kalshi-collector
```

Environment variables:
- `KALSHI_PAGES` (default `1`)
- `KALSHI_PAGE_SIZE` (default `20`)
- `KALSHI_KAFKA_TOPIC` (default `kalshi.snapshots`)

Use `../kalshi_collector_dev` for verbose JSON dumps while debugging.
