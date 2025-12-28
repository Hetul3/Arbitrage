# Polymarket Collector (Dev)

Developer-friendly entrypoint that continuously fetches Polymarket events, upserts them into SQLite, and dumps the normalized JSON payloads to stdout so you can inspect what will eventually be sent to Kafka. There is no polling delay; the loop runs as fast as the API allows (with built-in backoff on rate limits).

## Running

```sh
make run-polymarket-collector-dev
```

Use env vars (same as production command) to tune pagination/topic:
- `POLYMARKET_PAGES` (default 1)
- `POLYMARKET_PAGE_SIZE` (default 10)
- `POLYMARKET_KAFKA_TOPIC` (default `polymarket.snapshots`)
