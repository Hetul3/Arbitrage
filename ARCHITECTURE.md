# Arbitrage Detector Architecture

This document captures the full problem definition, pipeline, and concrete schemas for the Polymarket/Kalshi cross-venue arbitrage project. Every new contributor/agent can rely on this file to understand the current plan end-to-end.

## Problem Space

Goal: Continuously detect risk-free (or near risk-free) arbitrage opportunities between **Polymarket** and **Kalshi** binary markets.

Requirements:

- Fetch live markets plus orderbooks from both venues within their documented rate limits.
- Match semantically equivalent markets while ensuring their resolution criteria and sources are compatible.
- Compute arbitrage feasibility using executable taker prices, depth-based slippage, and Kalshi fee formulas (rounded up to the nearest cent). Polymarket fees are currently 0 unless their API indicates otherwise.
- Enforce per-market tick sizes when planning any hypothetical execution.
- Refresh prices immediately before publishing an opportunity to guard against stale data.
- Produce CLI output for now **and** persist all normalized data/opportunities in SQLite for later analytics/frontends.

Key references:
- Polymarket Gamma + CLOB docs (rate limits, resolution fields, per-market tick sizes).
- Kalshi Trade API docs (events, series, market orderbooks, fee schedule, rate limits).
- Nebius OpenAI-compatible API for embeddings + LLM validation.

## High-Level Pipeline

```
Collectors -> Kafka (snapshots) -> Snapshot Worker
        Snapshot Worker -> Redis (embeddings) -> Chroma upsert -> Chroma match
        Snapshot Worker -> Arb precheck -> LLM (if needed) -> final orderbooks -> arb calc -> opportunities topic/CLI
```

1. **Collectors**: Separate Go processes for Polymarket and Kalshi.
   - Paginate each venue's event lists/search endpoints, fetch per-market detail, normalize everything, store in SQLite (warehouse only), then publish to Kafka (`snapshots.polymarket`, `snapshots.kalshi`).
   - Normalized snapshot includes IDs, text fields, resolution info, close time, tick size, token/orderbook IDs, best bid/ask, and any batch orderbook summary collected inline.

2. **Kafka Workers (current implementation)**
   - Venue-specific workers consume the snapshot topics, build an embedding string (`event_title`, `question`, settle date, trimmed description + subtitle), call Nebius, and upsert vectors + metadata (venue, IDs, `captured_at`, `captured_at_unix`, `close_time`, `text_hash`, `resolution_hash`) into Chroma. No Redis cache yet—every snapshot is embedded on the fly.
   - Each Chroma entry uses the deterministic ID `venue:market_id` (so new snapshots overwrite the same vector) and stores the full `MarketSnapshot` JSON in the `document` field for later re-use.

3. **Snapshot Worker**
   - Current implementation consumes `matches.live`, runs the taker-only arb pre-check with the captured orderbooks, and **only** forwards profitable + tradable pairs into the Nebius LLM validator (GPT-OSS 120B, temperature 0). The validator receives a structured JSON blob (both venues’ questions, rules text, settlement sources, cutoff times, Kalshi contract PDF excerpt) and returns `{ValidResolution, ResolutionReason}`; the worker logs the verdict for each pair.
   - Next iterations will ensure an embedding exists via Redis cache (`emb:<platform>:<market_id>:<text_hash>`). Misses call Nebius embeddings, store result in Redis (multi-day TTL), and immediately upsert to Chroma.
   - Will query Chroma for opposite-venue markets: topK=3 within last-hour freshness, cosine similarity ≥ threshold (currently 0.95), and optional category match.
   - For each candidate result:
     - Construct a `pair_id` (e.g., `sha256("poly:<id>|kalshi:<id>")`).
     - Look in Redis `pair_bundle:<poly_id>:<kalshi_id>:<poly_hash>:<kalshi_hash>` to see if a SAFE verdict + opportunity metrics already exist. A cache hit means text/resolution data are unchanged.
     - Acquire `pair_inflight:<pair_id>` (short TTL ~60s) to avoid duplicate concurrent work.
     - Two paths:
       1. **Cached SAFE pair**: Skip the first arb pre-check and LLM. Go directly to fresh orderbook fetch + final arb engine.
       2. **New/changed pair**: Run quick arb pre-check using snapshot prices (YES-on-one + NO-on-other, taker assumption). If both directions fail `p_yes + p_no + fees >= 1`, stop. Otherwise gather resolution info and call Nebius LLM to judge equivalence. Store verdict + summary metrics in SQLite (analytics) and Redis bundle cache.

3. **Final Arb Stage** (runs inside the same worker regardless of path):
   - Fetch the freshest orderbooks via APIs (Polymarket `/books` batched; Kalshi `/orderbook`). Always hit APIs, no reuse cache.
   - Walk orderbooks to compute taker fills for configured budgets (e.g., $100). Include Kalshi fee rounding formula, Polymarket fee=0 (unless API changes). Output profit/ROI, contracts hedged, total fees.
   - Update Redis pair bundle with latest verdict + best opportunity metrics, even if profit ≤ 0 (records history for future comparisons).
   - Emit to Kafka `opportunities.live` and append to SQLite `opportunities` table **only if** profit > 0 and either no prior cached opportunity exists or the new profit/edge materially improves the cached one.

4. **CLI / Future API**: Small Go service consuming `opportunities.live`, caches current best per pair (Redis + in-memory), and prints structured summaries in real time. Future frontends can read from the same data.

## Components

| Component | Responsibility |
| --- | --- |
| `polymarket_collector` | Paginate Gamma events, fetch details & relevant CLOB snapshots, normalize, store in SQLite, publish to Kafka. |
| `kalshi_collector` | Paginate events/markets, fetch per-market detail + orderbook snippets, normalize, store, publish. |
| `polymarket_worker` / `_dev` | Consumes `polymarket.snapshots`, embeds each market via Nebius, and upserts vectors + metadata into Chroma (prod logs summaries, dev can dump payloads). |
| `kalshi_worker` / `_dev` | Same as above for the Kalshi topic. |
| `snapshot_worker` | Consumes matches, runs the taker-only arb pre-check, and forwards only profitable + tradable pairs to the Nebius LLM validator (includes Kalshi contract PDF extraction + structured verdict logging). |
| `chroma_maintainer` | Periodically deletes entries older than 1 hour to keep vector store fresh. |
| `cli_consumer` | Displays live opportunities; later replaced/augmented by HTTP API. |

All components run in Go using Docker Compose for local orchestration. Supporting services: Kafka, Redis, SQLite (file-based), ChromaDB, Nebius API access.

## Kafka Topics

1. `snapshots.polymarket` – key `polymarket:<market_id>`
2. `snapshots.kalshi` – key `kalshi:<market_ticker>`
3. `pairs.candidates` – key `pair:<poly_id>:<kalshi_id>`
4. `opportunities.live` – key `pair:<poly_id>:<kalshi_id>`

No third stage—matching + validation + final arb happen within the snapshot worker.

### `MarketSnapshot` payload (JSON)

```json
{
  "platform": "polymarket",
  "market_id": "string",
  "event_id": "string|null",
  "category": "politics",
  "title": "Will Candidate X win?",
  "description": "...",
  "resolution_source": "AP",
  "resolution_description": "...",
  "end_time": 1730000000,
  "tick_size": 0.01,
  "clob_tokens": {"yes": "tokenA", "no": "tokenB"},
  "prices": {"yes_bid": 0.48, "yes_ask": 0.50, "no_bid": 0.50, "no_ask": 0.52},
  "orderbook_snapshot": {
    "yes": [[0.50, 100], [0.51, 50]],
    "no": [[0.50, 120], [0.49, 40]]
  },
  "text_hash": "sha256(title+description)",
  "resolution_hash": "sha256(resolution_source+resolution_description+end_time)",
  "captured_at": 1730000005
}
```

Fields mirror Kalshi payloads (tick_size, settlement sources, etc.).

### `PairCandidate` payload

```json
{
  "pair_id": "sha256('poly:<id>|kalshi:<id>')",
  "polymarket": {
    "market_id": "...",
    "text_hash": "...",
    "resolution_hash": "...",
    "prices": {"yes_ask": 0.48, "no_ask": 0.52},
    "orderbook": {"yes": [...], "no": [...]},
    "tick_size": 0.01
  },
  "kalshi": {
    "market_ticker": "...",
    "text_hash": "...",
    "resolution_hash": "...",
    "prices": {"yes_ask": 0.49, "no_ask": 0.51},
    "orderbook": {"yes": [...], "no": [...]},
    "tick_size": 0.01
  },
  "cached_bundle": false,
  "matched_at": 1730000300
}
```

`cached_bundle=true` indicates the Redis cache already contains a SAFE verdict + previous opportunity metrics and the hashes still match.

### `Match` payload

Matcher results are appended to `matches.log` for debugging and published to Kafka (`matches.live`). Each payload contains the full source/target snapshots so downstream stages do not need to hit Chroma again.

```json
{
  "version": 1,
  "pair_id": "...",
  "similarity": 0.97,
  "distance": 0.03,
  "matched_at": 1730000300,
  "source": { "... MarketSnapshot ..." },
  "target": { "... MarketSnapshot ..." },
  "arbitrage": null
}
```

The arb engine consumes this topic, simulates both legs with slippage + fees, and writes the best `arbitrage` object back on the payload (and logs the concise summary).

### `Opportunity` payload

```json
{
  "pair_id": "...",
  "direction": "BUY_YES_POLY_BUY_NO_KALSHI",
  "profit_usd": 2.15,
  "edge_bps": 130,
  "max_size_contracts": 180,
  "budget_usd": 100,
  "kalshi_fee_taker": 1.75,
  "freshness_seconds": 2,
  "books": {
    "polymarket": {...},
    "kalshi": {...}
  },
  "cached_bundle": true,
  "captured_at": 1730000320
}
```

## Redis Keys

| Key | Purpose | TTL |
| --- | --- | --- |
| `emb:<platform>:<market_id>:<text_hash>` | Cached Nebius embedding for title+description. | ≥5 days (LRU acceptable). |
| `pair_bundle:<poly_id>:<kalshi_id>:<poly_hash>:<kalshi_hash>` | Stores `{verdict, last_profit_usd, edge_bps, max_size, updated_at}` for combined match/LLM result. | Long TTL (days) with LRU eviction acceptable. |
| `pair_inflight:<pair_id>` | Short lock to prevent duplicate processing. | ~60 seconds. |

A cache hit on `pair_bundle` that matches the latest hashes allows us to skip the initial arb pre-check + LLM.

## SQLite Tables (analytics only)

Used as a warehouse; runtime logic never depends on these tables.

- `markets` (PRIMARY KEY `(venue, market_id)`): stores the normalized snapshot for every venue. Columns mirror the collector structs:
  - Venue + identifiers: `venue`, `market_id`, `event_id`.
  - Event metadata: `event_title`, `event_description`, `event_category`, `event_status`, `resolution_source`, `resolution_details`, `settlement_sources_json`, `contract_terms_url`.
  - Market metadata: `question`, `subtitle`, `reference_url`, `close_time`, `tick_size`, `yes_bid`, `yes_ask`, `no_bid`, `no_ask`, `volume`, `volume_24h`, `open_interest`, `clob_token_yes`, `clob_token_no`.
  - Orderbook depth + metadata (per market snapshot): `yes_bids_json`, `yes_asks_json`, `no_bids_json`, `no_asks_json`, `book_captured_at`, `book_hash`. These capture the ladder as arrays of `[price, quantity, rawPrice, rawAmount]` ready for downstream slippage simulations.
  - Hashes, bookkeeping, and debugging: `text_hash`, `resolution_hash`, `last_seen_at`, `raw_json`.
- (coming later) `pairs`, `pair_decisions`, and `opportunities` tables for downstream stages once implemented.

## LLM + Matching Details

- Embeddings: Nebius OpenAI-compatible model, focusing on title + description (and optional resolution description if it adds clarity). Resolution sources are **not** included to avoid punishing otherwise equivalent markets.
- Matching: topK=3; threshold 0.95 cosine similarity. Deterministic filters on timing, numeric thresholds, etc. Additional pairs can be tested if the top result was previously rejected.
- LLM validation (Nebius) triggers only when there is no cached verdict for the current resolution hashes. Prompt includes both market descriptions, resolution text, and Kalshi settlement sources/contract_terms references. Outputs SAFE/UNSAFE.
- **TODO:** once Redis + LLM verdict caching is wired in, the matcher must walk
  the sorted candidates and consult the cache: skip hits previously marked
  UNSAFE, fast-path SAFE verdicts straight into the arb stage, and only fall
  back to the LLM when no cached verdict exists for the current hash pair.

## Fees & Slippage

- Polymarket fees currently assumed 0 (watch for API updates).
- Kalshi taker fee: `roundUpToCent(0.07 * C * P * (1-P))`.
- Final arb engine always retrieves fresh orderbooks and walks depth for configured budgets (default $100, easily configurable). Maker scenarios deferred until later.

## Freshness + Cleanup

- Chroma: entries older than 1 hour deleted by maintainer job (ensures matches only consider recent markets).
- Matching window: only consider Chroma hits updated within last hour, but Redis caches can live longer because they’re guarded by text/resolution hashes.
- Final arb fetch: always hits live APIs; failure to fetch (market closed, etc.) should be handled gracefully and logged.

## CLI Output

- Consumes `opportunities.live` and prints structured summaries with pair IDs, venue directions, profit, budgets, freshness, and references to resolution info/LLM verdict.
- Acts as a reference “UI” until a dedicated frontend/API is built.

---

This architecture reflects the latest decisions (SQLite warehouse, Nebius embeddings/LLM, single worker pipeline, two Kafka stages, Redis-based caching). Update this file whenever design changes so future agents can bootstrap instantly.
