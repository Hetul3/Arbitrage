# internal/storage/sqlite

SQLite helpers for the arbitrage project.

- Provides a shared `Store` that opens `data/arb.db` (configurable via `SQLITE_PATH`).
- Manages the unified `markets` table (shared schema for both venues) plus helpers to migrate from the legacy per-venue tables.
- Persists the full normalized payload, including orderbook depth (`yes_bids_json`, `yes_asks_json`, `no_bids_json`, `no_asks_json`) and metadata (`book_captured_at`, `book_hash`), so SQLite mirrors what we send to Kafka.
- Exposes `CreateTables`, `DropTables`, `ClearTables`, `MigrateToUnifiedSchema`, and venue-specific upsert helpers.
- Collectors call `UpsertPolymarketEvents` / `UpsertKalshiEvents` so every snapshot is persisted automatically using the shared schema.
- Command-line utilities under `cmd/` invoke these helpers (create/drop/clear) so new environments can prep the DB with a single Make target.
