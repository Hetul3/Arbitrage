# Kalshi API Experiment

Validates the public elections feed from Kalshi’s trade API.

## Endpoint

- `GET https://api.elections.kalshi.com/trade-api/v2/events?limit=200`
- This endpoint is public; no API key or session token is needed for read-only access.

## Running

```sh
make run-kalshi-events
```

The Make target runs `./kalshi/cmd/events` in the Go container. The program:

1. Sends the GET request with a 20-second timeout.
2. Fails fast if the status code ≥ 300 (the error message prints the response body to aid debugging).
3. Prints the JSON payload.

## Notes

- Future authenticated endpoints will likely require API keys + session cookies. When we need them, extend this client to read credentials from `.env` and set the headers documented by Kalshi.
- The response data includes pagination info; update the request with `cursor` parameters if we need additional pages later.
