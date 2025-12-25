# Redis Experiment

Confirms that Colima + Docker can run a Redis server and that our Go client can talk to it from inside the same Compose stack.

## Workflow

1. `make run-redis-cli`
   - Starts the `redis-server` container.
   - Launches `./redis/cmd/cli` inside the Go image.
   - Stops Redis automatically when you exit the CLI.
2. In the CLI:
   - Type any value → it is cached with a 24h TTL.
   - Type the same value again → you’ll see a “CACHE HIT” message.
   - `flush` clears the DB (uses `FLUSHDB`).
   - `exit` or `quit` leaves the session.

## Tips

- Everything happens inside Docker; no local Redis install is necessary.
- If you want Redis to stay up between runs, start it with `docker compose up -d redis-server` and skip the auto-stop logic (Ctrl+C to leave the CLI, then `docker compose stop redis-server` when you’re done).
