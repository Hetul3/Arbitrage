# Agent Workflow Guide
This file ensures every LLM agent knows how we collaborate on the Arbitrage project. Read it before making changes.

## Environment & Tooling

1. **Docker + Colima only** – all services run inside containers so results are deterministic. Follow `experiments/README.md` for setup (Colima with DNS flags, Docker context switching).
2. **No direct host installs** – if a dependency is missing, extend the Dockerfiles/Compose setup rather than installing locally.
3. **Use existing experiments** – each technology (SQLite, Redis, Kafka, Chroma, Nebius, Polymarket, Kalshi) already has a validated example under `experiments/`. Reuse their patterns when wiring production services.
4. **After schema changes** – run `make sqlite-migrate` (inside Docker) to rebuild `data/arb.db` so it matches the unified `markets` table. This wipes the DB; rerun the collectors afterward.

## Development Process

1. **One component at a time**
   - Pick a single service or feature (e.g., Polymarket collector) and implement it end-to-end inside Docker.
   - Provide simple `make`/Compose commands (mirroring `experiments/`) so the maintainer can run and verify quickly.
2. **Small, testable increments**
   - After each feature, document how to verify it (commands, expected output). Wait for validation before moving on.
   - Prefer unit tests/integration checks where practical; otherwise provide CLI instructions.
3. **No long speculative branches**
   - Avoid “big bang” changes. Ensure every commit/PR all the way up the stack is shippable.

## Knowledge Sources

- `ARCHITECTURE.md` – canonical problem statement, pipeline, schemas, Kafka topics, cache keys, DB layouts.
- `README.md` – quick repo overview and entry points.
- `experiments/README.md` – Docker/Colima setup + commands for each tech experiment.

## Hand-off Expectations

1. Update docs (`ARCHITECTURE.md`, `README.md`, this file) whenever architecture decisions change.
2. Provide running instructions + validation steps for each feature.
3. Highlight assumptions or open questions explicitly so the next agent/maintainer can address them.

## Directory Documentation

- Each major directory (collectors, workers, CLI, etc.) must contain an up-to-date `README.md` explaining its purpose, commands, and current status.
- When adding a new subdirectory, create its README immediately.
- Whenever code or behavior changes, update the relevant README so future agents can reconstruct context without digging through history.

## Toolchain Requirements

- Local Go commands must run with Go **1.24+** because the SQLite dependency (`modernc.org/sqlite v1.41.0`) requires it. If your host Go is ≥1.21, run `go env -w GOTOOLCHAIN=go1.24.11` or install Go 1.24 directly. Otherwise rely on the Docker/Compose commands (which already use Go 1.24 inside the container).
- Set `NEBIUS_API_KEY` (plus optional `NEBIUS_BASE_URL`/`NEBIUS_EMBED_MODEL`) and Chroma settings (`CHROMA_URL`, `CHROMA_COLLECTION`) before running the Kafka + embeddings pipeline.

Following this process keeps the project coherent across multiple AI sessions while giving the human overseer a clear verification path after each step.
