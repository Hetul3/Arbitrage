# Polymarket API Experiment

Quick smoke test that hits Polymarket’s public search endpoint from inside our Dockerized Go image.

## Endpoint

- `GET https://gamma-api.polymarket.com/public-search`
- No authentication required for this public route (expect JSON describing currently listed markets).

## Running

```sh
make run-polymarket-search
```

This builds the Go container and runs `./polymarket/cmd/search`, which:

1. Issues the GET request with a 20-second timeout.
2. Checks the HTTP status and logs an error if it is ≥ 300.
3. Prints the raw response body to stdout.

## Notes

- If Polymarket introduces rate limits, the API may start returning 429 responses. The current code logs the full response so we can inspect it and extend the client with headers, query parameters, or authentication as needed.
- Since the response can be large, consider piping it to `jq` or redirecting to a file when you run the real project.
