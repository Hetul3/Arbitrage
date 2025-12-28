# internal/chroma

Lightweight REST client for the Chroma vector store. Provides helpers to create/ensure collections and add/query documents. Used by worker processes to upsert embeddings.

Key APIs:
- `NewClient(baseURL)` – constructs the HTTP client (defaults to `http://chromadb:8000`).
- `EnsureCollection(ctx, name)` – fetch or create a collection and return its ID.
- `Add(ctx, collectionID, AddRequest)` – upsert embeddings/documents/metadata.
- `Query(ctx, collectionID, QueryRequest)` – utility for future matching stages.
