# SQLite Clear Tables

Deletes all rows from the unified `markets` table without dropping the schema.

## Running

```sh
make sqlite-clear
```

Respects `SQLITE_PATH` (default `data/arb.db`).
