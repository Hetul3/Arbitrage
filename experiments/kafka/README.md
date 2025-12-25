# Kafka Experiment

Validates that we can stand up ZooKeeper + Kafka under Colima, then interact with the broker using a single interactive CLI that both produces and consumes messages.

## Requirements

- Colima running with the Docker context selected.
- No external Kafka install; the Compose file uses `confluentinc/cp-zookeeper` and `confluentinc/cp-kafka`.

## Running the CLI

```sh
make run-kafka
```

This target:

1. Starts ZooKeeper and the Kafka broker.
2. Launches `./kafka/cmd/cli` inside the Go container.
3. Installs a trap that stops the containers automatically when you exit.

Inside the CLI:

- Type any message → it’s produced immediately.
- The embedded consumer prints each message after a 10-second delay so we can observe lag/ordering.
- `exit` or `quit` stops both producer and consumer, then the Make target stops Kafka.

## Tips

- If you need the broker to keep running for other experiments, start it manually (`docker compose up -d zookeeper kafka-broker`) and skip the Make target’s auto-stop, then use `make kafka-stop` when finished.
- The CLI waits for the broker to be ready and attempts to create the demo topic. If your broker auto-creates topics, a creation failure is logged but not fatal.
