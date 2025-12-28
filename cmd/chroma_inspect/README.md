# Chroma Inspect
The `chroma_inspect` tool allows you to peek into the ChromaDB vector store. It lists all available collections, shows the total item count for each, and displays the most recent 2 documents added across both venues (Kalshi and Polymarket).

## Running
```sh
make chroma-inspect
```
*(Or `go run ./cmd/chroma_inspect/main.go`)*

Environment Variables:
- `CHROMA_URL`: The URL of the ChromaDB server (default: `http://localhost:8000`)
