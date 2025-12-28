# Polymarket Worker (Dev)

Same as `polymarket_worker` but prints every consumed snapshot to stdout so you can verify the payload hitting Kafka.

Run as part of the dev pipeline:

```sh
make run-kafka-dev        # concise logs
make run-kafka-dev-verbose # dumps full JSON payloads
```

Env vars mirror the production worker (`POLYMARKET_WORKERS`, `POLYMARKET_WORKER_GROUP`, `NEBIUS_API_KEY`, `CHROMA_URL`, etc.). `POLYMARKET_WORKER_VERBOSE=1` forces verbose output even when running the non-verbose target.
