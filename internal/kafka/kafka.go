package kafka

import (
    "context"
    "fmt"
    "net"
    "os"
    "strings"
    "time"

    "github.com/segmentio/kafka-go"
)

const (
    DefaultBroker      = "kafka-broker:9092"
    DefaultPolyTopic   = "polymarket.snapshots"
    DefaultKalshiTopic = "kalshi.snapshots"
)

func Brokers() []string {
    raw := os.Getenv("KAFKA_BROKERS")
    if raw == "" {
        raw = DefaultBroker
    }
    parts := strings.Split(raw, ",")
    out := make([]string, 0, len(parts))
    for _, p := range parts {
        if trimmed := strings.TrimSpace(p); trimmed != "" {
            out = append(out, trimmed)
        }
    }
    return out
}

func TopicFromEnv(envKey, fallback string) string {
    if val := os.Getenv(envKey); val != "" {
        return val
    }
    return fallback
}

func WaitForBroker(ctx context.Context, brokers []string) error {
    if len(brokers) == 0 {
        return fmt.Errorf("no brokers configured")
    }

    ticker := time.NewTicker(time.Second)
    defer ticker.Stop()

    var lastErr error
    for {
        conn, err := kafka.DialContext(ctx, "tcp", brokers[0])
        if err == nil {
            conn.Close()
            return nil
        }
        lastErr = err

        select {
        case <-ctx.Done():
            return fmt.Errorf("waiting for broker: %w (last error: %v)", ctx.Err(), lastErr)
        case <-ticker.C:
        }
    }
}

func EnsureTopic(ctx context.Context, brokers []string, topic string) error {
    if len(brokers) == 0 {
        return fmt.Errorf("no brokers configured")
    }

    conn, err := kafka.DialContext(ctx, "tcp", brokers[0])
    if err != nil {
        return fmt.Errorf("dial broker %s: %w", brokers[0], err)
    }
    defer conn.Close()

    controller, err := conn.Controller()
    if err != nil {
        return fmt.Errorf("get controller: %w", err)
    }

    ctrlConn, err := kafka.DialContext(ctx, "tcp", net.JoinHostPort(controller.Host, fmt.Sprintf("%d", controller.Port)))
    if err != nil {
        return fmt.Errorf("dial controller: %w", err)
    }
    defer ctrlConn.Close()

    cfg := kafka.TopicConfig{
        Topic:             topic,
        NumPartitions:     3,
        ReplicationFactor: 1,
    }
    if err := ctrlConn.CreateTopics(cfg); err != nil && !strings.Contains(err.Error(), "already exists") {
        return fmt.Errorf("create topic: %w", err)
    }
    return nil
}

func NewWriter(brokers []string, topic string) *kafka.Writer {
    return &kafka.Writer{
        Addr:         kafka.TCP(brokers...),
        Topic:        topic,
        Balancer:     &kafka.LeastBytes{},
        BatchTimeout: 100 * time.Millisecond,
        RequiredAcks: kafka.RequireOne,
    }
}

func NewReader(brokers []string, topic, group string) *kafka.Reader {
    return kafka.NewReader(kafka.ReaderConfig{
        Brokers:           brokers,
        Topic:             topic,
        GroupID:           group,
        MinBytes:          1,
        MaxBytes:          10e6,
        HeartbeatInterval: 3 * time.Second,
        SessionTimeout:    30 * time.Second,
        CommitInterval:    time.Second,
        StartOffset:       kafka.FirstOffset,
    })
}
