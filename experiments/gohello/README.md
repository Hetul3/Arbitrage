# Go Hello Experiment

This directory holds the smoke test that ensures Go tooling works inside our Docker environment before we build anything substantial.

## Requirements

- Colima running with Docker context pointed to it (`colima start --dns 1.1.1.1 --dns 8.8.8.8`, `docker context use colima`).
- No local Go install required; the container builds from `experiments/Dockerfile` (Go 1.24).

## How to Run

```sh
make run-go-hello
```

`make run-go-hello` builds the container image (cached after the first run) and executes `go run ./gohello`. Expect to see the “Hello from Arbitrage!” message in the terminal.

## Notes

- All Go commands run inside the same image we use for the rest of the experiments, so this is a quick way to confirm dependency downloads are succeeding.
- If Docker Hub pulls fail, ensure Colima started with functional DNS (see the root `experiments/README.md`).
