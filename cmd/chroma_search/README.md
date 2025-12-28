# Chroma Search
The `chroma_search` tool allows you to search for binary markets using natural language strings. It uses the Nebius API to embed your search query and then finds the most relevant markets across all venues in ChromaDB.

## Running
```sh
make chroma-search text="<SEARCH_STRING>" k=<RESULTS>
```

Arguments:
- `text`: The natural language string to search for.
- `k`: The number of top results to return (default: 3).

Environment Variables:
- `NEBIUS_API_KEY`: Required to generate embeddings for the search text.
- `CHROMA_URL`: The URL of the ChromaDB server (default: `http://localhost:8000`)
- `NEBIUS_BASE_URL`: (Optional) Custom base URL for Nebius API.
- `NEBIUS_EMBED_MODEL`: (Optional) Custom embedding model name.
