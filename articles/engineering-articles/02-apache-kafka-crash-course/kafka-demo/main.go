package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/scram"
)

const (
	brokerAddress = "10.1.101.231:9094"
	topic         = "order-events"
)

// OrderEvent represents something that happened that happened in our system
type OrderEvent struct {
	OrderID    string  `json:"order_id"`
	CustomerID string  `json:"customer_id"`
	Amount     float64 `json:"amount"`
	Status     string  `json:"status"`
}

func newKafkaDialer() (*kafka.Dialer, error) {
	username := os.Getenv("KAFKA_USERNAME")
	password := os.Getenv("KAFKA_PASSWORD")
	if username == "" || password == "" {
		return nil, fmt.Errorf("KAFKA_USERNAME and KAFKA_PASSWORD must be set")
	}

	mechanism, err := scram.Mechanism(scram.SHA512, username, password)
	if err != nil {
		return nil, fmt.Errorf("create SCRAM-SHA-512 mechanism: %w", err)
	}

	return &kafka.Dialer{
		Timeout:       10 * time.Second,
		DualStack:     true,
		SASLMechanism: mechanism,
	}, nil
}

func runProducer(ctx context.Context, dialer *kafka.Dialer) error {
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:      []string{brokerAddress},
		Topic:        topic,
		Dialer:       dialer,
		Balancer:     &kafka.Hash{}, // some key always goes to same partition
		BatchTimeout: 10 * time.Millisecond,
	})
	defer writer.Close()

	events := []kafka.Message{
		{
			Key:   []byte("order-1"),
			Value: []byte(`{"order_id": "order-1", "customer_id": "customer-1", "amount": 100.0, "status": "placed"}`),
		},
		{
			Key:   []byte("order-2"),
			Value: []byte(`{"order_id": "order-2", "customer_id": "customer-2", "amount": 200.0, "status": "placed"}`),
		},
		{
			Key:   []byte("order-3"),
			Value: []byte(`{"order_id": "order-3", "customer_id": "customer-3", "amount": 300.0, "status": "payment_confirmed"}`),
		},
	}
	err := writer.WriteMessages(ctx, events...)
	if err != nil {
		return err
	}

	fmt.Printf("Produced %d order events to topic '%s'\n", len(events), topic)
	return nil
}

func runConsumer(ctx context.Context, dialer *kafka.Dialer, groupID string) error {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{brokerAddress},
		GroupID:  groupID, // consumer group, Kafka tracks offset per group
		Topic:    topic,
		Dialer:   dialer,
		MinBytes: 1,
		MaxBytes: 10e6, // 10MB
		MaxWait:  1 * time.Second,
	})
	defer reader.Close()

	fmt.Printf("Consumer group '%s' is listening for order events...\n", groupID)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	for {
		m, err := reader.FetchMessage(ctx)
		if err != nil {
			if err == context.DeadlineExceeded {
				fmt.Println("Consumer timeout reached, exiting...")
				return nil
			}
			return err
		}

		fmt.Printf(
			"[%s] partition:%d offset:%d key:%s value:%s\n",
			groupID, m.Partition, m.Offset, string(m.Key), string(m.Value),
		)

		// Commit offset only after successful processing
		// This is important: if processing fails, the offset is not advanced
		// and the message will be redeliverd
		if err := reader.CommitMessages(ctx, m); err != nil {
			log.Printf("failed to commit offset: %v", err)
		}
	}
}

func main() {
	// In a real system these run in separate services
	ctx := context.Background()
	dialer, err := newKafkaDialer()
	if err != nil {
		log.Fatalf("failed to configure Kafka client: %v", err)
	}

	err = runProducer(ctx, dialer)
	if err != nil {
		log.Fatalf("failed to run producer: %v", err)
	}

	// Two different consumer groups both reading the same topic independently
	// e.g., one for writing to DB, one for sending notifications
	consumerErrors := make(chan error, 2)
	go func() {
		consumerErrors <- runConsumer(ctx, dialer, "orders-db-writer")
	}()
	go func() {
		consumerErrors <- runConsumer(ctx, dialer, "orders-notification")
	}()

	for range 2 {
		if err := <-consumerErrors; err != nil {
			log.Printf("consumer failed: %v", err)
		}
	}
}
