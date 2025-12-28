# cmd/

Entrypoints for runnable services and CLIs. Each subdirectory is a standalone Go command that can be executed inside Docker.

Current commands:
- `polymarket_collector` – production-style Polymarket ingestion loop (quiet logs).
- `polymarket_collector_dev` – same logic but dumps normalized JSON for debugging.
- `kalshi_collector` – production-style Kalshi ingestion loop.
- `kalshi_collector_dev` – verbose JSON output for Kalshi.
- `polymarket_worker` / `_dev` – Kafka consumers for Polymarket snapshots (prod vs dev logging).
- `kalshi_worker` / `_dev` – Kafka consumers for Kalshi snapshots (prod vs dev logging).
- `sqlite_create_tables` – creates required SQLite tables.
- `sqlite_clear_tables` – deletes all rows while keeping the schema.
- `sqlite_drop_tables` – drops the SQLite tables.
- `sqlite_migrate` – drops legacy per-venue tables and recreates the unified `markets` table.
- `chroma_inspect` – quick peek into vector store counts and recent documents.
- `chroma_query` – cross-venue similarity search starting from a market ID.
- `chroma_search` – natural language vector search across all venues.

Each command has its own README with usage instructions and docker-compose targets.
