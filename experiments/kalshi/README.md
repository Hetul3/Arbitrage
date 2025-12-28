# Kalshi API Experiment

This directory contains experiments for interacting with the Kalshi Trade API (v2), specifically focusing on public elections and event data for the arbitrage engine.

## API Usage & Endpoints

Kalshi's Trade API provides public read-only access for many endpoints. The data follows a hierarchy: **Series > Event > Market**.

### 1. Listing Events
- **Endpoint**: `GET https://api.elections.kalshi.com/trade-api/v2/events`
- **Usage**: Use the `status=open` query parameter to find active markets.
- **Pagination**: Uses cursor-based pagination. The response includes a `cursor` string for the next page.

### 2. Event Details & Nested Markets
- **Endpoint**: `GET https://api.elections.kalshi.com/trade-api/v2/events/{event_ticker}?with_nested_markets=true`
- **Usage**: Fetches high-level metadata and all associated market tickers (e.g., individual candidates in an election).

### 3. Series Details (Matching Data)
- **Endpoint**: `GET https://api.elections.kalshi.com/trade-api/v2/series/{series_ticker}`
- **Usage**: Crucial for the **Matching/LLM Gate**. It provides comprehensive settlement sources and the official contract terms URL (PDF).

### 4. Market Orderbook (Execution Data)
- **Endpoint**: `GET https://api.elections.kalshi.com/trade-api/v2/markets/{market_ticker}/orderbook`
- **Usage**: Fetches depth levels.
- **Note**: Kalshi orderbooks return **bids only**. Because it is a binary market, a "YES Ask" is effectively `100 - NO Bid`. Our engine derives asks from the opposing side's bids.

---

## Data Grab List

### A) Matching & Resolution Fields (LLM Gate)
These fields are used to determine if a Kalshi market is equivalent to a Polymarket event.
- `ticker`: Stable identifier for the market.
- `title` / `sub_title`: Human-readable description.
- `rules_primary` / `rules_secondary`: Precise resolution criteria.
- `settlement_sources`: Names and URLs of official data sources (from Series).
- `contract_terms_url`: Legal definition of the event (from Series).

### B) Arbitrage & Execution Fields
These fields drive the pricing engine and slippage calculations.
- `yes_bid` / `yes_ask` / `no_bid` / `no_ask`: Top-of-book prices.
- `orderbook`: Depth levels (`[price, quantity]`) for YES and NO bids.
- `tick_size`: Minimum price increment (usually 1 cent/tick).
- `volume` / `open_interest`: Liquidity and activity signals.

## Running the Experiment

```sh
make run-kalshi-events
```
This runs the Go program in `cmd/events/main.go` which demonstrates fetching 3 pages of events and pulling a full "Matching + Arb" data snapshot for the first active event found.

## Multi-option Handling
Kalshi groups related outcomes under an `event_ticker`. For multi-option events (e.g., "Next President"):

1. **Nested Markets**: Use the `with_nested_markets=true` flag on the `/events/{event_ticker}` endpoint.
2. **Individual Tickers**: Each candidate or outcome is a separate **Market** with its own unique `ticker` (e.g., `KXELONMARS-99`).
3. **Orderbook Depth**: You **must** call `GET /markets/{ticker}/orderbook` for each specific market leg to get the valid bids and derive executable asks. 
4. **Deterministic Mapping**: This process is reliable as the nested market array contains all active outcomes for the event group.
