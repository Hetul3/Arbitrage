# internal/kafka

Utility package for Kafka connectivity and common operations. Wraps `github.com/segmentio/kafka-go` to provide consistent behavior across collectors and workers.

## Features

- **Configuration**: Standardized helpers to read broker addresses and topic names from environment variables.
- **Connection**: `WaitForBroker` ensures the Kafka service is available before starting services.
- **Topic Management**: `EnsureTopic` handles the creation of topics with uniform partitions and replication factors.
- **Writers/Readers**: Pre-configured `NewWriter` and `NewReader` functions with optimized settings (least-bytes balancing, batching timeouts, etc.).
