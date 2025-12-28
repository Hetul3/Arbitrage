# Polymarket Collector

Production-style entrypoint that continuously fetches Polymarket events/markets (Gamma + CLOB), normalizes them, upserts into SQLite, and logs lightweight summaries (counts). Requests automatically retry with exponential backoff when Polymarket rate-limits or briefly errors. Future work will push the normalized payloads to Kafka, but the SQLite warehouse is already live.

## Running

```sh
make run-polymarket-collector
```

Environment variables:
- `POLYMARKET_PAGES` (default `1`)
- `POLYMARKET_PAGE_SIZE` (default `20`)
- `POLYMARKET_INTERVAL_SECONDS` (default `60`)

See `../polymarket_collector_dev` for a version that dumps the full JSON payload each interval.
