# internal/storage

Unified persistence layer for the Arbitrage project. Currently focused on SQLite for local storage, analytics, and serving as a historical data warehouse.

## Subpackages

- **`sqlite`** – Implementation of the storage interface using `modernc.org/sqlite`. Handles the unified `markets` table which stores normalized data from all venues.

Note: Runtime matching and detection logic primarily uses Chroma and Redis; the storage layer is intended for analytical queries and UI data retrieval.
