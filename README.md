# Arbitrage

This repository builds a **Polymarket ↔ Kalshi cross-venue arbitrage detector**. It continuously ingests markets from both venues, finds equivalent events via embeddings + LLM validation, simulates executable taker trades (fees + slippage), and surfaces profitable opportunities via CLI while writing all normalized data into SQLite for future analytics/frontends. Every component is containerized (Docker + Colima) so agents get deterministic environments; see `experiments/README.md` for Colima/Docker prerequisites.

The latest end-to-end architecture, schemas, and pipeline details live in [ARCHITECTURE.md](ARCHITECTURE.md). Use that file to bootstrap any new agent/session—it contains the full problem space, component breakdown, Kafka topics, cache keys, and database layouts.

## Repository Layout

- `experiments/` – proven setups for every technology used in the main system (Go in Docker, SQLite, Redis, Kafka, Chroma+Nebius embeddings, Nebius LLM CLI, Polymarket/Kalshi API demos). Refer to its README for per-experiment commands.
- `ARCHITECTURE.md` – comprehensive design/requirements document (problem statement, pipeline, schemas).
- `agents.md` – workflow/hand-off guidelines for LLM agents (how we build/test iteratively, container usage expectations).
- `cmd/`, `internal/`, etc. – will host the production services mirroring the architecture (collectors, snapshot worker, CLI consumer, etc.).

## Quick Start

1. Review [ARCHITECTURE.md](ARCHITECTURE.md) for the canonical plan.
2. Ensure your local Go toolchain is **1.24+** (run `go env -w GOTOOLCHAIN=go1.24.11` if you only have Go ≥1.21 installed) or rely on the Docker targets below.
3. Copy `example.env` → `.env` (or export the listed variables manually) so collectors/workers share the same configuration (`SQLITE_PATH`, `KAFKA_BROKERS`, worker counts, etc.).
4. Run Colima + Docker with working DNS (`colima start --dns 1.1.1.1 --dns 8.8.8.8`).
4. From repo root, explore the experiments to verify dependencies:
   - `make -C experiments run-polymarket-events`
   - `make -C experiments run-kalshi-events`
   - `make -C experiments run-chroma-create` (requires `experiments/.env` with `NEBIUS_API_KEY`)
5. Initialize/migrate SQLite (runs inside Docker, default path `data/arb.db` mounted from the repo). **All migration commands drop data, so re-run collectors afterwards.**
   - `make sqlite-create` – creates the unified `markets` table (run once for new environments).
   - `make sqlite-migrate` – drops legacy `polymarket_markets`/`kalshi_markets` tables and recreates the unified schema (destroys data; run after pulling this change).
   - `make sqlite-clear` – optional reset of all rows.
   - `make sqlite-drop` – removes the table entirely.
6. Run the dockerized collectors (all containerized via `docker-compose.yml`):
   - `make run-polymarket-collector` – production loop (quiet logs) that polls continuously and relies on built-in exponential backoff when rate-limited.
   - `make run-polymarket-collector-dev` – same logic but dumps every normalized JSON payload in real time for debugging.
   - `make run-kalshi-collector` / `make run-kalshi-collector-dev` – Kalshi equivalents.
   - `make run-collectors` / `make run-collectors-dev` – run both production or both dev collectors together; `make collectors-down` stops/removes containers.
7. Run the Kafka-backed pipeline when needed:
   - `make run-kafka` – brings up ZooKeeper/Kafka, both collectors, and the production worker pools (silent consumption).
   - `make run-kafka-dev` – same stack but the dev workers log a concise line for every snapshot they pull (`consumed market=... event=...`).
   - `make run-kafka-dev-verbose` – identical to `run-kafka-dev` but the workers dump the full JSON payloads as they consume them.
   - Collectors in these pipeline commands always run in production mode (no stdout spam). Adjust topics/broker via env vars (`KAFKA_BROKERS`, `POLYMARKET_KAFKA_TOPIC`, `KALSHI_KAFKA_TOPIC`).
8. When building new services, follow the architecture’s guidance for Kafka topics, Redis caches, Chroma schema, and SQLite warehouse tables. Implement work in small, testable increments so each hand-off can be verified before moving on (see `agents.md`).

### SQLite schema summary

The unified `markets` table (backed by `data/arb.db` by default) mirrors the normalized structs we publish downstream. Key columns:

- `venue` (`polymarket` | `kalshi`), `market_id`, `event_id`
- Event metadata: `event_title`, `event_description`, `event_category`, `event_status`, `resolution_source`, `resolution_details`, `settlement_sources_json`, `contract_terms_url`
- Market metadata: `question`, `subtitle`, `reference_url`, `close_time`, `tick_size`, `yes_bid/ask`, `no_bid/ask`, `volume`, `volume_24h`, `open_interest`, `clob_token_yes/no`
- Orderbook depth + metadata (used for slippage modeling): `yes_bids_json`, `yes_asks_json`, `no_bids_json`, `no_asks_json`, `book_captured_at`, `book_hash`
- Hashes/timestamps/raw payload: `text_hash`, `resolution_hash`, `last_seen_at`, `raw_json`
- `yes_bids_json` / `yes_asks_json`: JSON arrays of `[price, quantity, rawPrice, rawAmount]` storing the entire ladder captured at `book_captured_at`. Slippage calculations will walk these levels (and the corresponding `no_*` ladders) to compute executable weighted-average fill prices.
- `book_hash`: SHA-256 of the ladder JSON blobs; useful for deduping collector snapshots or cache keys when feeding downstream services.

Use `sqlite3 data/arb.db 'SELECT * FROM markets LIMIT 5'` (or any GUI) to inspect exactly what will be sent to Kafka later.

## Status

- Architecture + experiments are finalized and in sync (SQLite instead of MySQL, Nebius instead of Gemini, taker-only execution for now).
- Implementation work will wire the collectors, snapshot worker, and CLI on top of the proven experiments.

Keep this README and `ARCHITECTURE.md` updated whenever design decisions change so future sessions can ramp up instantly.
