# Chroma Query
The `chroma_query` tool performs cross-venue similarity searches. Given a specific Market ID (or Ticker), it retrieves that market's embedding and searches for the most similar entries in the **opposite** venue using vector similarity.

## Running
```sh
make chroma-query id=<MARKET_ID_OR_TICKER> k=<RESULTS>
```

Arguments:
- `id`: The unique Chroma ID (e.g., `kalshi:KX-123`) or the venue's ticker (`KX-123`).
- `k`: The number of top results to return (default: 3).

Environment Variables:
- `CHROMA_URL`: The URL of the ChromaDB server (default: `http://localhost:8000`)
- `CHROMA_COLLECTION`: The collection to query (default: `market_snapshots`)
