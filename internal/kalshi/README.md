# internal/kalshi

Client + collector implementation for Kalshi Trade API v2.

Responsibilities:
- Paginate `/events?status=open` with cursor support.
- Fetch detailed event payloads (`/events/{ticker}?with_nested_markets=true`).
- Fetch Series data (`/series/{series_ticker}`) to retrieve settlement sources and contract terms URLs.
- Fetch per-market orderbooks for sample depth.
- Produce normalized `collectors.Event` structs.
