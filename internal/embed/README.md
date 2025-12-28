# internal/embed

Wrapper around Nebius' OpenAI-compatible embedding API (`github.com/sashabaranov/go-openai`). The workers use this client to turn snapshot text (title + question + settle date + trimmed description/subtitle) into vectors before inserting into Chroma.

Environment-driven config:
- `NEBIUS_API_KEY` (required)
- `NEBIUS_BASE_URL` (defaults to `https://api.tokenfactory.nebius.com/v1/`)
- `NEBIUS_EMBED_MODEL` (defaults to `Qwen/Qwen3-Embedding-8B`)
