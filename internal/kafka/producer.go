package kafka

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// Producer wraps Kafka producer with helper methods
type Producer struct {
	producer *kafka.Producer
	config   *Config
	logger   *slog.Logger
}

// NewProducer creates a new Kafka producer
// Equivalent to Python:
// producer = KafkaProducer(
//     bootstrap_servers=['13.48.120.205:32100'],
//     value_serializer=lambda v: json.dumps(v).encode('utf-8'))
func NewProducer(config *Config, logger *slog.Logger) (*Producer, error) {
	// Configure producer with idempotence enabled
	producerConfig := &kafka.ConfigMap{
		"bootstrap.servers": config.Brokers,
		"enable.idempotence": config.EnableIdempotence, // Prevents duplicates in Kafka
		"acks":               config.Acks,              // Wait for all replicas
		"max.in.flight.requests.per.connection": 5,    // Required for idempotence
		"retries":                                2147483647, // Max retries
	}

	p, err := kafka.NewProducer(producerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	producer := &Producer{
		producer: p,
		config:   config,
		logger:   logger,
	}

	// Start delivery report handler in background
	go producer.handleDeliveryReports()

	logger.Info("Kafka producer initialized",
		"brokers", config.Brokers,
		"idempotence", config.EnableIdempotence)

	return producer, nil
}

// PublishEmailEvent publishes an email event to Kafka
// Equivalent to Python: producer.send('email-events', event_data)
func (p *Producer) PublishEmailEvent(topic string, event interface{}) error {
	// Serialize to JSON (like Python's json.dumps)
	jsonData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Create Kafka message
	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: kafka.PartitionAny, // Let Kafka choose partition
		},
		Value: jsonData,
	}

	// Produce message (non-blocking, uses delivery reports)
	err = p.producer.Produce(msg, nil)
	if err != nil {
		return fmt.Errorf("failed to produce message: %w", err)
	}

	p.logger.Debug("Email event published to Kafka",
		"topic", topic,
		"size", len(jsonData))

	return nil
}

// PublishEmailEventSync publishes an email event and waits for confirmation
// Use this for critical events where you need immediate feedback
func (p *Producer) PublishEmailEventSync(topic string, event interface{}) error {
	// Serialize to JSON
	jsonData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Create Kafka message
	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: kafka.PartitionAny,
		},
		Value: jsonData,
	}

	// Create delivery channel for this message
	deliveryChan := make(chan kafka.Event)

	// Produce message
	err = p.producer.Produce(msg, deliveryChan)
	if err != nil {
		close(deliveryChan)
		return fmt.Errorf("failed to produce message: %w", err)
	}

	// Wait for delivery report
	e := <-deliveryChan
	close(deliveryChan)

	m := e.(*kafka.Message)
	if m.TopicPartition.Error != nil {
		return fmt.Errorf("delivery failed: %w", m.TopicPartition.Error)
	}

	p.logger.Info("Email event published to Kafka (sync)",
		"topic", *m.TopicPartition.Topic,
		"partition", m.TopicPartition.Partition,
		"offset", m.TopicPartition.Offset)

	return nil
}

// handleDeliveryReports processes asynchronous delivery reports
func (p *Producer) handleDeliveryReports() {
	for e := range p.producer.Events() {
		switch ev := e.(type) {
		case *kafka.Message:
			if ev.TopicPartition.Error != nil {
				p.logger.Error("Delivery failed",
					"topic", *ev.TopicPartition.Topic,
					"error", ev.TopicPartition.Error)
			} else {
				p.logger.Debug("Message delivered",
					"topic", *ev.TopicPartition.Topic,
					"partition", ev.TopicPartition.Partition,
					"offset", ev.TopicPartition.Offset)
			}
		}
	}
}

// Flush waits for all messages to be delivered
// Equivalent to Python: producer.flush()
func (p *Producer) Flush(timeoutMs int) int {
	remaining := p.producer.Flush(timeoutMs)
	if remaining > 0 {
		p.logger.Warn("Failed to flush all messages",
			"remaining", remaining)
	}
	return remaining
}

// Close closes the producer
// Equivalent to Python: producer.close()
func (p *Producer) Close() {
	p.logger.Info("Closing Kafka producer...")

	// Flush remaining messages (10 second timeout)
	remaining := p.Flush(10000)
	if remaining > 0 {
		p.logger.Error("Some messages were not delivered",
			"count", remaining)
	}

	p.producer.Close()
	p.logger.Info("Kafka producer closed")
}
