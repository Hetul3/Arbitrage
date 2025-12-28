# internal/workers

Utilities shared by the Kafka worker processes:
- `Run` – spins up a pool of consumer goroutines.
- `Processor` – orchestrates embedding + Chroma upsert for a `MarketSnapshot`.
- `text.go` – builds the embedding string (title + question + settle date + trimmed description/subtitle) so both venues behave consistently.

Workers (prod + dev) import this package, configure the Nebius/Chroma clients via env vars, and hook in their own logging.
