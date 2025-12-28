# Kalshi Worker (Dev)

Verbose variant of the Kalshi worker. Every message is unmarshaled and printed, making it easy to confirm the collector → Kafka → worker flow. Used by `make run-pipeline-dev`.

Env vars are the same as the production worker (`KALSHI_WORKERS`, `KALSHI_WORKER_GROUP`, `KALSHI_KAFKA_TOPIC`).
