# Chroma + Nebius Embeddings Experiment

Proves out the workflow for creating and querying a vector store using:

- Nebius’ OpenAI-compatible embedding endpoint (`Qwen/Qwen3-Embedding-8B`)
- The Chroma DB docker image (everything stored under `experiments/chroma/data/`)

## Prerequisites

1. `cp experiments/.env.template experiments/.env` and set `NEBIUS_API_KEY`. Optional overrides:
   - `NEBIUS_BASE_URL` (defaults to `https://api.tokenfactory.nebius.com/v1/`)
   - `CHROMA_URL` (defaults to `http://chromadb:8000`)
2. Colima/Docker running.

The Make targets automatically start/stop the `chromadb` container, but data persists in `experiments/chroma/data/chroma/`.

## Commands

| Command | Purpose |
| --- | --- |
| `make run-chroma-create` | Starts Chroma, creates a collection with a random name, and writes `experiments/chroma/data/state.json`. |
| `make run-chroma-add` | Prompts for text lines, calls Nebius to embed each one, and stores them (ID + doc + metadata + embedding) in Chroma. |
| `make run-chroma-query` | Embeds a query string and prints the closest matches with similarity scores. |
| `make run-chroma-drop` | Deletes the collection (if it still exists) and clears the state file. |
| `make chroma-stop` | Manual helper if you started `chromadb` yourself and want to stop it. |

## Implementation Notes

- State is tracked in `state/state.go` (JSON file). If the Chroma data directory is wiped or the collection disappears, the drop command now logs a warning and still removes local state.
- The Chroma volume lives inside the repo (`experiments/chroma/data/chroma`) so it is easy to inspect or back up, but it’s git-ignored.
- All embedding calls run with a 60-second timeout; errors (rate limits, auth issues) are surfaced directly.
