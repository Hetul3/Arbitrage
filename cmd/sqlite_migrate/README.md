# SQLite Migrate

Drops the old `polymarket_markets` / `kalshi_markets` tables (if they exist) and recreates the unified `markets` schema used by both venues.

## Running

```sh
make sqlite-migrate
```

The command runs inside Docker, respects `SQLITE_PATH` (default `data/arb.db`), and will delete existing data. Run `make sqlite-create` if you need the original per-venue tables instead.
