## Experiments Environment

### Prerequisites

1. Install Colima and Docker CLI (e.g., `brew install colima docker`).
2. Start Colima with DNS resolvers so the VM can reach Docker Hub:
   ```
   colima stop
   colima start --dns 1.1.1.1 --dns 8.8.8.8
   ```
   Run those flags the first time (or whenever DNS breaks). You can bake them into `~/.colima/default/colima.yaml` if you always want them.
3. Switch the Docker CLI to the Colima socket once per machine:
   ```
   docker context create colima --docker "host=unix:///Users/hetulpatel/.colima/default/docker.sock"
   docker context use colima
   ```
4. Copy `experiments/.env.template` to `experiments/.env` and add the secrets needed by the LLM/Chroma experiments (e.g., `NEBIUS_API_KEY=...`). The Compose file (now inside `experiments/`) automatically loads that file, and `.env` is git-ignored.

### Layout

- `gohello/` – minimal Go sanity check that prints to stdout.
- `sqlite/` – contains shared helpers plus commands for creating, populating, querying, and deleting a SQLite database written to `sqlite/data/demo.db` (git-ignored).
- `redis/` – CLI demo that talks to a Redis instance to cache arbitrary strings.
- `kafka/` – combined CLI that produces messages and displays the consumer stream.
- `chroma/` – Nebius embedding helpers plus Chroma vector store commands (create/add/query/drop with persisted state).
- `llm/` – OpenAI-compatible CLI configured for Nebius Token Factory (terminal prompt + response).
- `polymarket/` – simple HTTP demo hitting Polymarket’s public search API (no key required).
- `kalshi/` – fetches the public events feed from Kalshi’s API (no key required for this endpoint).

### Commands

From the repo root:

- `make run-go-hello`
- `make run-sqlite-create`
- `make run-sqlite-populate`
- `make run-sqlite-query`
- `make run-sqlite-drop`
- `make run-redis-cli`
- `make run-chroma-create`
- `make run-chroma-add`
- `make run-chroma-query`
- `make run-chroma-drop`
- `make chroma-stop`
- `make run-llm`
- `make run-polymarket-search`
- `make run-kalshi-events`
- `make run-kafka`
- `make kafka-stop`
- `make down` – stop/remove any started containers.

Each target wraps `docker compose run --rm --build <service>`, so you always run inside the same container image. Typical SQLite flow:

1. `make run-sqlite-create`
2. `make run-sqlite-populate`
3. `make run-sqlite-query`
4. `make run-sqlite-drop` when you want a clean slate.

Add new experiments under `experiments/` and wire them into `docker-compose.yml` plus the Makefile with their own `run-…` targets.

Redis tips:

1. `make run-redis-cli` starts the `redis-server` container, then launches an interactive Go CLI. Type a word/phrase; if it isn't cached it will be stored (with a 24h TTL). Re-entering the same text shows a cache hit. Type `flush` to clear Redis or `exit` to quit.
2. After you exit, the Make target stops the Redis container automatically. Use `docker compose up -d redis-server` if you need it running separately.

LLM (Nebius) tips:

1. Create `experiments/.env` from `experiments/.env.template` and paste your `NEBIUS_API_KEY`. Run `make run-llm` to launch the CLI (no other services required). It targets the Nebius OpenAI-compatible endpoint (`https://api.tokenfactory.nebius.com/v1/`) and uses the `openai/gpt-oss-120b` model by default.
2. Every prompt is executed with a 60s timeout and responses are printed inline. Any API failure (auth, quota, etc.) is logged so you can see what went wrong.

Kafka tips:

1. `make run-kafka` spins up ZooKeeper and the Kafka broker, then launches a single interactive CLI. Type any message and it is published immediately; the embedded consumer prints each message every 10 seconds so you can see the stream lag. Exit with `exit`/`quit`, and the Make target automatically stops Kafka afterward.
2. If the CLI ever crashes or you need to force-stop Kafka, run `make kafka-stop` (or `make down`) to tear the containers down manually.
Chroma + Embeddings tips:

1. `make run-chroma-create` spins up the Chroma DB container, creates a fresh collection with a random name, and stores its identifiers under `experiments/chroma/data/state.json`.
2. `make run-chroma-add` lets you type documents; for each one the CLI calls the Nebius embedding endpoint and stores the resulting vector/doc pair in Chroma. `make run-chroma-query` accepts a free-form query, embeds it, and prints the closest stored entries with similarity scores. `make run-chroma-drop` deletes the collection and clears local state. Each command automatically starts/stops the Chroma container, but you can run `docker compose up -d chromadb` manually (and `make chroma-stop`) if you want the DB to stay up across commands.

Polymarket/Kalshi tips:

1. Both demos hit public GET endpoints. `make run-polymarket-search` prints the JSON from `https://gamma-api.polymarket.com/public-search`, while `make run-kalshi-events` fetches `https://api.elections.kalshi.com/trade-api/v2/events?limit=200`. Neither requires an API key, but you can extend the Go code later to add authentication headers or query params if the docs call for it.
