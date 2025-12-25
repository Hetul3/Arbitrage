package kafkautil

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
	DefaultTopic = "demo-events"
	DefaultGroup = "demo-consumer"
)

func BrokersFromEnv() []string {
	raw := os.Getenv("KAFKA_BROKERS")
	if raw == "" {
		raw = "kafka-broker:9092"
	}
	parts := strings.Split(raw, ",")
	clean := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			clean = append(clean, trimmed)
		}
	}
	return clean
}

func TopicFromEnv() string {
	if val := os.Getenv("KAFKA_TOPIC"); val != "" {
		return val
	}
	return DefaultTopic
}

func ConsumerGroupFromEnv() string {
	if val := os.Getenv("KAFKA_CONSUMER_GROUP"); val != "" {
		return val
	}
	return DefaultGroup
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

	err = ctrlConn.CreateTopics(kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	})
	if err != nil && !strings.Contains(err.Error(), "Topic with this name already exists") {
		return fmt.Errorf("create topic: %w", err)
	}
	return nil
}

func WaitForBroker(ctx context.Context, brokers []string) error {
	if len(brokers) == 0 {
		return fmt.Errorf("no brokers configured")
	}

	var lastErr error
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

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

func NewWriter(brokers []string, topic string) *kafka.Writer {
	return &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		Async:        false,
		BatchTimeout: 100 * time.Millisecond,
	}
}

func NewReader(brokers []string, topic, group string) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:           brokers,
		Topic:             topic,
		GroupID:           group,
		StartOffset:       kafka.FirstOffset,
		MinBytes:          1,
		MaxBytes:          10e6,
		CommitInterval:    time.Second,
		HeartbeatInterval: 3 * time.Second,
		SessionTimeout:    30 * time.Second,
	})
}
