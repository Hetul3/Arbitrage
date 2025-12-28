# internal/queue

Kafka publishing helpers. `PublishSnapshots` takes a batch of events, builds `MarketSnapshot` payloads, and writes them to the configured topic (using the same Go struct consumed by workers). Keeps collector code thin.
