# Polymarket API Experiment

This directory contains experiments for interacting with Polymarket's Gamma (Metadata) and CLOB (Execution) APIs.

## API Usage & Endpoints

Polymarket uses two distinct APIs for data collection: **Gamma** for high-level event metadata and the **CLOB (Central Limit Order Book)** for real-time pricing and depth.

### 1. Gamma API (Listing & Metadata)
- **Listing**: `GET https://gamma-api.polymarket.com/events`
- **Discovery**: Use `closed=false` and `active=true` to find tradable markets.
- **Details**: `GET https://gamma-api.polymarket.com/events/{id}`
- **Usage**: Provides the hierarchy of markets within an event and the critical `clobTokenIds` required to query the orderbook.

### 2. CLOB API (Real-time Execution Data)
- **Endpoint**: `GET https://clob.polymarket.com/book?token_id={id}`
- **Usage**: Fetches a live snapshot of the orderbook (bids and asks) for a specific token (YES or NO).

---

## Data Grab List

### A) Matching & Resolution Fields (LLM Gate)
These fields are used by the LLM to validate market equivalence.
- `id`: Unique event identifier.
- `question` / `description`: The core text defining the outcome.
- `resolutionSource`: Text describing where Polymarket will look for the result.
- `endDate`: When the market closes.
- `outcomes`: YES/NO labels or candidate names.

### B) Arbitrage & Execution Fields
These fields are used for price comparison and fill simulation.
- `clobTokenIds`: A JSON array of strings mapping outcomes to trading tokens.
- `lastTradePrice`: The last price recorded on Gamma (useful as a baseline).
- `minimum_tick_size`: The price granularity for CLOB orders.
- `volume` / `liquidity` / `openInterest`: Aggregate metrics for the whole event.
- **CLOB Book**: Real-time `bids` and `asks` (`[price, size]`) for precise slippage modeling.

## Running the Experiment

```sh
make run-polymarket-events
```
This runs `cmd/list_events/main.go`, which:
1. Paginates through 3 pages of events.
2. Identifies the first active event.
3. Fetches the detailed event payload.
4. Parses `clobTokenIds` for nested markets and performs live CLOB book lookups to demonstrate the "Arb Data" grab.

## Multi-option Handling
Polymarket events often have multiple markets (e.g. "Who will win the Oscar?").
1. The **Gamma Event** contains an array of `markets`.
2. Each **Market** has its own `clobTokenIds` (usually two: one for YES, one for NO).
3. We treat each candidate/outcome as an individual market leg, fetching its specific token books for execution analysis.
