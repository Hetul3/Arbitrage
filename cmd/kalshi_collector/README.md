# Kalshi Collector

Production-style entrypoint that continuously fetches Kalshi events/markets (events → nested markets → series metadata → orderbooks), normalizes them, upserts into SQLite, and logs concise summaries. HTTP calls automatically retry with exponential backoff when Kalshi rate-limits or returns transient errors. The dev variant prints full JSON for inspection.

## Running

```sh
make run-kalshi-collector
```

Environment variables:
- `KALSHI_PAGES` (default `1`)
- `KALSHI_PAGE_SIZE` (default `20`)
- `KALSHI_INTERVAL_SECONDS` (default `60`)

Use `../kalshi_collector_dev` for verbose JSON dumps while debugging.
