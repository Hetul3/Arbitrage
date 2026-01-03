# snapshot_worker

Consumes the `matches.live` Kafka topic, runs the taker-only arbitrage pre-check
using the embedded orderbooks, and (when profitable + tradable) calls the Nebius
LLM validator to ensure the Polymarket/Kalshi markets truly represent the same
resolution criteria. The worker logs every validator verdict (pair ID, both
questions, verdict) so arbitrage candidates can be vetted before execution.

## Flags & Environment

| Variable | Default | Description |
| --- | --- | --- |
| `KAFKA_BROKERS` | `kafka-broker:9092` | Kafka bootstrap servers. |
| `MATCHES_KAFKA_TOPIC` | `matches.live` | Topic to consume match payloads from. |
| `SNAPSHOT_WORKER_GROUP` | `snapshot-worker` | Consumer group for this command. |
| `SNAPSHOT_WORKER_CONCURRENCY` | `1` | Number of concurrent consumer goroutines. |
| `SNAPSHOT_WORKER_BUDGET_USD` | `100` | Budget used during the pre-check simulation (taker assumption). |
| `NEBIUS_API_KEY` | _(required)_ | API key for the Nebius GPT-OSS 120B endpoint. |
| `NEBIUS_BASE_URL` | `https://api.tokenfactory.nebius.com/v1/` | Override for Nebius API base URL. |
| `VALIDATOR_MODEL` | `openai/gpt-oss-120b` | Model used for the resolution validator. |
| `VALIDATOR_TIMEOUT_SECONDS` | `45` | Timeout for each LLM request. |
| `VALIDATOR_MAX_TOKENS` | `800` | Max tokens for the response. |
| `VALIDATOR_SYSTEM_PROMPT` | _(built-in)_ | Optional override for the system prompt. |
| `PDFTOTEXT_BIN` | `pdftotext` | Path to the CLI used to extract Kalshi contract text. |
| `OPPORTUNITY_CACHE_TTL_HOURS` | `72` | TTL for the Redis cache that tracks the best profit per pair to suppress duplicate alerts. |

## Status

The worker now reaches the LLM stage: each profitable pair is enriched with the
taker arb simulation and full resolution metadata (including Kalshi contract PDF
text via `pdftotext`). The validator emits a structured JSON verdict
(`ValidResolution`, `ResolutionReason`) which is logged for now; Redis caching
and downstream publishing will follow in a later iteration. After the fresh
orderbook arb check, the worker also writes the best observed profit per pair to
Redis so subsequent runs only emit if the opportunity improves.
