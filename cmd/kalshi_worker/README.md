# Kalshi Worker

Kafka consumer for `kalshi.snapshots`. Each snapshot is embedded via Nebius and upserted into Chroma. Included in `make run-kafka`.

Run it via `make run-kafka` (or individually with `docker compose run --rm --build kalshi-worker`). Configuration:
- `KALSHI_WORKERS` – number of consumer goroutines (default 2).
- `KALSHI_WORKER_GROUP` – Kafka consumer group.
- `KALSHI_KAFKA_TOPIC` – defaults to `kalshi.snapshots`.
- `NEBIUS_API_KEY` / `NEBIUS_BASE_URL` / `NEBIUS_EMBED_MODEL` – embedding service settings.
- `CHROMA_URL` / `CHROMA_COLLECTION` – Chroma endpoint + collection name.
