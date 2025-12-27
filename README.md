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
2. Run Colima + Docker with working DNS (`colima start --dns 1.1.1.1 --dns 8.8.8.8`).
3. From repo root, explore the experiments to verify dependencies:
   - `make -C experiments run-polymarket-events`
   - `make -C experiments run-kalshi-events`
   - `make -C experiments run-chroma-create` (requires `experiments/.env` with `NEBIUS_API_KEY`)
4. When building new services, follow the architecture’s guidance for Kafka topics, Redis caches, Chroma schema, and SQLite warehouse tables. Implement work in small, testable increments so each hand-off can be verified before moving on (see `agents.md`).

## Status

- Architecture + experiments are finalized and in sync (SQLite instead of MySQL, Nebius instead of Gemini, taker-only execution for now).
- Implementation work will wire the collectors, snapshot worker, and CLI on top of the proven experiments.

Keep this README and `ARCHITECTURE.md` updated whenever design decisions change so future sessions can ramp up instantly.
