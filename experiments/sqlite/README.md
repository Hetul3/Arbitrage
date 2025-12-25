# SQLite Experiment

Verifies that we can work with a local SQLite database without needing an external server. We use the CGO-free `modernc.org/sqlite` driver so it runs cleanly inside the Go Docker image.

## Requirements

- Docker/Colima running (same setup as the other experiments).
- No extra dependencies; the database file is stored under `experiments/sqlite/data/demo.db` and is git-ignored.

## Commands

Run each target from the repo root:

| Command | Description |
| --- | --- |
| `make run-sqlite-create` | Creates/initializes the schema and writes `state.json`. Safe to run multiple times. |
| `make run-sqlite-populate` | Inserts the demo user rows (clears existing rows first). |
| `make run-sqlite-query` | Reads and prints the stored users. |
| `make run-sqlite-drop` | Deletes the database file for a clean slate. |

## Tips & Troubleshooting

- The code makes sure the `data/` directory exists before opening SQLite, so permission issues generally mean the Docker volume cannot be mounted (make sure you run from the project root).
- Because the database file lives on the host, data persists between container runs unless you explicitly drop it.
