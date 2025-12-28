# Kalshi Worker

Kafka consumer for Kalshi snapshot messages. Today it only acknowledges messages (future versions will trigger downstream processing). Included in `make run-pipeline`.

Run it via `make run-pipeline` (or individually with `docker compose run --rm --build kalshi-worker`). Configuration:
- `KALSHI_WORKERS` – number of consumer goroutines (default 2).
- `KALSHI_WORKER_GROUP` – Kafka consumer group.
- `KALSHI_KAFKA_TOPIC` – defaults to `kalshi.snapshots`.
