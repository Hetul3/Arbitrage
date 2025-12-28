# internal/collectors

Shared abstractions + models for venue collectors.

- Defines the `Collector` interface (fetch normalized events/markets with pagination options).
- Exposes reusable data structures (`Event`, `Market`, `PriceSnapshot`, `Orderbook`, etc.) used across services, Kafka payloads, and SQLite warehouse rows.
- Future shared helpers (rate limiting, normalization, logging) will also live here.
