package kafka

import (
	"fmt"
	"os"
	"strings"
)

// Config holds Kafka configuration
type Config struct {
	Brokers           string
	EmailEventsTopic  string
	EmailDLQTopic     string
	ConsumerGroup     string
	EnableIdempotence bool
	Acks              string
}

// LoadConfig loads Kafka configuration from environment variables
func LoadConfig() (*Config, error) {
	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		return nil, fmt.Errorf("KAFKA_BROKERS environment variable is required")
	}

	emailEventsTopic := os.Getenv("KAFKA_TOPIC_EMAIL_EVENTS")
	if emailEventsTopic == "" {
		emailEventsTopic = "email-events" // Default
	}

	emailDLQTopic := os.Getenv("KAFKA_TOPIC_EMAIL_DLQ")
	if emailDLQTopic == "" {
		emailDLQTopic = "email-events-dlq" // Default
	}

	consumerGroup := os.Getenv("KAFKA_CONSUMER_GROUP")
	if consumerGroup == "" {
		consumerGroup = "email-service-group" // Default
	}

	return &Config{
		Brokers:           brokers,
		EmailEventsTopic:  emailEventsTopic,
		EmailDLQTopic:     emailDLQTopic,
		ConsumerGroup:     consumerGroup,
		EnableIdempotence: true, // Always enable for exactly-once
		Acks:              "all", // Wait for all replicas
	}, nil
}

// GetBrokersList returns brokers as a slice
func (c *Config) GetBrokersList() []string {
	return strings.Split(c.Brokers, ",")
}
