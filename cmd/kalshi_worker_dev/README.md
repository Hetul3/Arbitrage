# Kalshi Worker (Dev)

Verbose variant of the Kalshi worker. Every message is embedded/upserted just like prod, and you can choose concise or verbose logging via the run target (`make run-kafka-dev` or `make run-kafka-dev-verbose`).

Env vars mirror the production worker (`KALSHI_WORKERS`, `KALSHI_WORKER_GROUP`, `NEBIUS_API_KEY`, `CHROMA_URL`, etc.). Set `KALSHI_WORKER_VERBOSE=1` to force verbose logging even when using the non-verbose dev target.
