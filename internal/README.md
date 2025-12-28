# internal/

Internal packages shared across commands. Currently includes:
- `collectors` – shared models/interfaces for venue ingestion.
- `polymarket` – Polymarket-specific API client and collector.
- `kalshi` – Kalshi-specific API client and collector.

Add new packages here as we flesh out Kafka writers, Redis clients, etc.
