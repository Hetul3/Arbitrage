# internal/models

Types shared across services. Currently only `MarketSnapshot`, which carries the normalized event + specific market + `captured_at` timestamp. Kafka messages and Chroma documents both use this struct so downstream stages can deserialize the same payload.
