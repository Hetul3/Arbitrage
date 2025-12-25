# Nebius LLM CLI

This experiment wires the Nebius Token Factory (OpenAI-compatible) API into a terminal chat loop so we can confirm credentials, network access, and SDK usage.

## Setup

1. From the repo root, run `cp experiments/.env.template experiments/.env`.
2. Add `NEBIUS_API_KEY=...` (and `NEBIUS_BASE_URL` if you want to override the default `https://api.tokenfactory.nebius.com/v1/`).
3. Ensure Colima/Docker are running.

## Running

```sh
make run-llm
```

The Make target builds the Go image and runs `./llm/cmd/chat` with `.env` injected. In the CLI:

- Type a prompt → the app calls `openai/gpt-oss-120b`.
- Responses are printed immediately in the same terminal.
- `exit` or `quit` terminates the session.

## Notes for Future Work

- The CLI currently resets the conversation each prompt (no multi-turn history). When we extend this in the real project, we can persist messages in a slice and send them all in the `ChatCompletionRequest`.
- Timeouts are set to 60 seconds; failures (auth errors, quota issues) are logged with full messages to help debugging.
