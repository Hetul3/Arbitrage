# internal/

Internal packages shared across services and commands.

## Packages

- **`chroma`** – Lightweight REST client for the Chroma vector store. Handles collection management and document/embedding upserts.
- **`collectors`** – Core interfaces and normalized models (`Event`, `Market`) used by all venue-specific collectors and the shared runner logic.
- **`embed`** – Client for turning market text into vectors using the Nebius OpenAI-compatible embedding API.
- **`hashutil`** – Deterministic SHA-256 hashing for deduplication and change detection (`text_hash`, `resolution_hash`).
- **`kafka`** – Low-level connectivity helpers, topic management, and pre-configured producers/consumers using `kafka-go`.
- **`kalshi`** – Kalshi-specific API client and collector implementation.
- **`models`** – Higher-level types used for cross-service communication, primarily the `MarketSnapshot` payload used in Kafka and Chroma.
- **`polymarket`** – Polymarket-specific API client and collector implementation.
- **`queue`** – High-level Kafka publishing logic that transforms raw collector events into snapshots for workers.
- **`storage`** – Persistence layer for SQLite, handling the unified `markets` table and analytics data.
- **`workers`** – Orchestration logic for Kafka consumers, including the background `Processor` that handles embedding and Chroma integration.
