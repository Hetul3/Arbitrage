# Kalshi Collector (Dev)

Developer entrypoint that continuously fetches Kalshi events, upserts them into SQLite, and prints the normalized JSON payloads (exactly what production will push downstream). No artificial delay between runs; exponential backoff handles rate limits.

## Running

```sh
make run-kalshi-collector-dev
```

Environment overrides:
- `KALSHI_PAGES` (default 1)
- `KALSHI_PAGE_SIZE` (default 10)
- `KALSHI_KAFKA_TOPIC` (default `kalshi.snapshots`)
