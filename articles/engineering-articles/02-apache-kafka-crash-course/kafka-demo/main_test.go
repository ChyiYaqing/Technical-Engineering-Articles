package main

import (
	"testing"

	"github.com/segmentio/kafka-go/sasl/scram"
)

func TestNewKafkaDialerRequiresCredentials(t *testing.T) {
	t.Setenv("KAFKA_USERNAME", "")
	t.Setenv("KAFKA_PASSWORD", "")

	if _, err := newKafkaDialer(); err == nil {
		t.Fatal("expected an error when Kafka credentials are missing")
	}
}

func TestNewKafkaDialerUsesSCRAMSHA512(t *testing.T) {
	t.Setenv("KAFKA_USERNAME", "kafka-demo")
	t.Setenv("KAFKA_PASSWORD", "test-password")

	dialer, err := newKafkaDialer()
	if err != nil {
		t.Fatalf("newKafkaDialer returned an error: %v", err)
	}
	if dialer.SASLMechanism == nil {
		t.Fatal("expected a SASL mechanism")
	}
	if got, want := dialer.SASLMechanism.Name(), scram.SHA512.Name(); got != want {
		t.Fatalf("SASL mechanism = %q, want %q", got, want)
	}
}
