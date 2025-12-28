# Polymarket Worker

Kafka consumer that processes Polymarket snapshot messages. In production mode it silently consumes (future logic will handle persistence/arb). Used by `make run-pipeline`.

## Running

```sh
make run-pipeline    # includes this worker
# or run individually:
docker compose run --rm --build polymarket-worker
```

Environment knobs:
- `POLYMARKET_WORKERS` – number of goroutines consuming (default 2).
- `POLYMARKET_WORKER_GROUP` – Kafka consumer group ID.
- `POLYMARKET_KAFKA_TOPIC` – topic name (defaults to `polymarket.snapshots`).
