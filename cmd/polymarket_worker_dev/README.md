# Polymarket Worker (Dev)

Same as `polymarket_worker` but prints every consumed snapshot to stdout so you can verify the payload hitting Kafka.

Run as part of the dev pipeline:

```sh
make run-pipeline-dev
```

Env vars mirror the production worker (`POLYMARKET_WORKERS`, `POLYMARKET_WORKER_GROUP`, etc.).
