# Polymarket Collector

Production-style entrypoint that continuously fetches Polymarket events/markets (Gamma + CLOB), normalizes them, upserts into SQLite, and publishes per-market snapshots onto Kafka (`polymarket.snapshots`). Requests automatically retry with exponential backoff when Polymarket rate-limits or briefly errors.

## Running

```sh
make run-polymarket-collector
```

Environment variables:
- `POLYMARKET_PAGES` (default `1`)
- `POLYMARKET_PAGE_SIZE` (default `20`)
- `POLYMARKET_KAFKA_TOPIC` (default `polymarket.snapshots`)

See `../polymarket_collector_dev` for a version that dumps the full JSON payload each interval.
