# SQLite Create Tables

Utility command to initialize the unified `markets` table (shared by Polymarket + Kalshi).

## Running

```sh
make sqlite-create
```

Uses `SQLITE_PATH` (defaults to `data/arb.db`). Run this once before starting the collectors so upserts succeed.
