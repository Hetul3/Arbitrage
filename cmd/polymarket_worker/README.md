# Polymarket Worker

Kafka consumer that reads `polymarket.snapshots`, creates embeddings via Nebius, and upserts each snapshot into Chroma. Used by `make run-kafka`.

## Running

```sh
make run-kafka       # includes this worker
# or run individually:
docker compose run --rm --build polymarket-worker
```

Environment knobs:
- `POLYMARKET_WORKERS` – number of goroutines consuming (default 2).
- `POLYMARKET_WORKER_GROUP` – Kafka consumer group ID.
- `POLYMARKET_KAFKA_TOPIC` – topic name (defaults to `polymarket.snapshots`).
- `NEBIUS_API_KEY` / `NEBIUS_BASE_URL` / `NEBIUS_EMBED_MODEL` – embedding service settings (API key required).
- `CHROMA_URL` / `CHROMA_COLLECTION` – Chroma endpoint + collection name.
