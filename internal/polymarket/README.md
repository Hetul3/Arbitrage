# internal/polymarket

Client + collector implementation for Polymarket Gamma/CLOB APIs.

Responsibilities:
- Paginate `/events` to find active markets.
- Fetch per-event detail (`/events/{id}`) with nested markets.
- Parse `clobTokenIds`, tick sizes, and other metadata.
- Optionally fetch sample CLOB orderbooks to populate depth data.
- Return normalized `collectors.Event` records.
