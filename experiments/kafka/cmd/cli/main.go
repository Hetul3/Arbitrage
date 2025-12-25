package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"llm/kafka/internal/kafkautil"

	"github.com/segmentio/kafka-go"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	brokers := kafkautil.BrokersFromEnv()
	topic := kafkautil.TopicFromEnv()
	group := kafkautil.ConsumerGroupFromEnv()

	waitCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	if err := kafkautil.WaitForBroker(waitCtx, brokers); err != nil {
		log.Fatalf("wait for broker: %v", err)
	}
	cancel()

	ensureCtx, cancelTopic := context.WithTimeout(ctx, 30*time.Second)
	if err := kafkautil.EnsureTopic(ensureCtx, brokers, topic); err != nil {
		log.Printf("warning: ensure topic failed: %v (broker will auto-create if enabled)", err)
	}
	cancelTopic()

	writer := kafkautil.NewWriter(brokers, topic)
	defer writer.Close()

	reader := kafkautil.NewReader(brokers, topic, group)
	defer reader.Close()

	fmt.Printf("Kafka CLI connected to %v, topic %q.\n", brokers, topic)
	fmt.Println("Type messages to produce. Consumer prints entries every 10 seconds.")
	fmt.Println("Commands: 'exit'/'quit'.")

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			m, err := reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("consumer error: %v", err)
				time.Sleep(time.Second)
				continue
			}

			time.Sleep(10 * time.Second)
			fmt.Printf("\n[consumer %s] %s\n> ", time.Now().Format(time.RFC3339), string(m.Value))
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			fmt.Println("\nGoodbye.")
			break
		}
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}

		if strings.EqualFold(text, "exit") || strings.EqualFold(text, "quit") {
			fmt.Println("Goodbye.")
			stop()
			wg.Wait()
			return
		}

		msg := kafka.Message{
			Key:   []byte(fmt.Sprintf("%d", time.Now().UnixNano())),
			Value: []byte(text),
		}

		if err := writer.WriteMessages(ctx, msg); err != nil {
			log.Printf("produce error: %v", err)
			continue
		}

		fmt.Println("sent.")
	}

	stop()
	wg.Wait()
}
