package email

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// Consumer wraps Kafka consumer with email processing logic
type Consumer struct {
	consumer         *kafka.Consumer
	sender           Sender
	idempotencyStore *IdempotencyStore
	dlqProducer      *kafka.Producer
	config           *ConsumerConfig
	logger           *slog.Logger
}

// ConsumerConfig holds consumer configuration
type ConsumerConfig struct {
	Brokers       string
	Topic         string
	DLQTopic      string
	ConsumerGroup string
	MaxRetries    int
}

// NewConsumer creates a new Kafka consumer
// Equivalent to Python:
// consumer = KafkaConsumer(
//     'email-events',
//     bootstrap_servers=['13.48.120.205:32100'],
//     group_id='email-service-group')
func NewConsumer(
	config *ConsumerConfig,
	sender Sender,
	idempotencyStore *IdempotencyStore,
	logger *slog.Logger,
) (*Consumer, error) {
	// Configure Kafka consumer
	consumerConfig := &kafka.ConfigMap{
		"bootstrap.servers":  config.Brokers,
		"group.id":           config.ConsumerGroup,
		"auto.offset.reset":  "earliest", // Read from beginning if no offset
		"enable.auto.commit": false,      // Manual commit for exactly-once
	}

	c, err := kafka.NewConsumer(consumerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	// Create DLQ producer
	dlqProducerConfig := &kafka.ConfigMap{
		"bootstrap.servers": config.Brokers,
	}
	dlqProducer, err := kafka.NewProducer(dlqProducerConfig)
	if err != nil {
		c.Close()
		return nil, fmt.Errorf("failed to create DLQ producer: %w", err)
	}

	consumer := &Consumer{
		consumer:         c,
		sender:           sender,
		idempotencyStore: idempotencyStore,
		dlqProducer:      dlqProducer,
		config:           config,
		logger:           logger,
	}

	logger.Info("Kafka consumer initialized",
		"brokers", config.Brokers,
		"topic", config.Topic,
		"group", config.ConsumerGroup)

	return consumer, nil
}

// Start starts consuming messages
// Equivalent to Python:
// for message in consumer:
//     process(message.value)
func (c *Consumer) Start(ctx context.Context) error {
	// Subscribe to topic
	err := c.consumer.Subscribe(c.config.Topic, nil)
	if err != nil {
		return fmt.Errorf("failed to subscribe to topic: %w", err)
	}

	c.logger.Info("Starting to consume messages",
		"topic", c.config.Topic)

	// Consume messages
	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Consumer shutting down...")
			return nil

		default:
			// Poll for messages (1 second timeout)
			msg, err := c.consumer.ReadMessage(1 * time.Second)
			if err != nil {
				// Timeout is not an error
				if err.(kafka.Error).Code() == kafka.ErrTimedOut {
					continue
				}
				c.logger.Error("Error reading message", "error", err)
				continue
			}

			// Process the message
			c.processMessage(ctx, msg)
		}
	}
}

// processMessage processes a single Kafka message
func (c *Consumer) processMessage(ctx context.Context, msg *kafka.Message) {
	c.logger.Info("Received email event",
		"topic", *msg.TopicPartition.Topic,
		"partition", msg.TopicPartition.Partition,
		"offset", msg.TopicPartition.Offset)

	// Parse the event
	var event EmailEvent
	err := json.Unmarshal(msg.Value, &event)
	if err != nil {
		c.logger.Error("Failed to parse email event",
			"error", err,
			"raw_value", string(msg.Value))
		c.commitMessage(msg) // Commit to skip bad message
		return
	}

	// Validate event
	if event.MessageID == "" {
		c.logger.Error("Email event missing message_id",
			"recipient", event.Recipient,
			"type", event.EventType)
		c.commitMessage(msg) // Commit to skip invalid message
		return
	}

	// Check if already processed (idempotency check)
	isProcessed, err := c.idempotencyStore.IsProcessed(ctx, event.MessageID)
	if err != nil {
		c.logger.Error("Failed to check idempotency",
			"messageID", event.MessageID,
			"error", err)
		// Don't commit - will retry
		return
	}

	if isProcessed {
		c.logger.Warn("Duplicate email event detected, skipping",
			"messageID", event.MessageID,
			"recipient", event.Recipient,
			"type", event.EventType)
		c.commitMessage(msg) // Commit - already processed
		return
	}

	// Process with retry logic
	err = c.processWithRetry(ctx, event)
	if err != nil {
		c.logger.Error("Failed to process email event after retries",
			"messageID", event.MessageID,
			"error", err)
		// Send to DLQ
		c.sendToDLQ(event, err)
		c.commitMessage(msg) // Commit to move past failed message
		return
	}

	// Mark as processed (idempotency barrier)
	success, err := c.idempotencyStore.MarkAsProcessed(ctx, event)
	if err != nil {
		c.logger.Error("Failed to mark as processed",
			"messageID", event.MessageID,
			"error", err)
		// Don't commit - will retry
		return
	}

	if !success {
		c.logger.Warn("Message was processed by another consumer (race condition)",
			"messageID", event.MessageID)
	}

	// Commit offset
	c.commitMessage(msg)

	c.logger.Info("Email event processed successfully",
		"messageID", event.MessageID,
		"recipient", event.Recipient,
		"type", event.EventType)
}

// processWithRetry attempts to send email with retries
func (c *Consumer) processWithRetry(ctx context.Context, event EmailEvent) error {
	maxRetries := c.config.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3 // Default
	}

	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := c.sender.SendEmailEvent(event)
		if err == nil {
			// Success!
			if attempt > 1 {
				c.logger.Info("Email sent successfully after retry",
					"messageID", event.MessageID,
					"attempt", attempt)
			}
			return nil
		}

		lastErr = err
		c.logger.Warn("Failed to send email, will retry",
			"messageID", event.MessageID,
			"attempt", attempt,
			"maxRetries", maxRetries,
			"error", err)

		// Exponential backoff (1s, 2s, 4s)
		if attempt < maxRetries {
			backoff := time.Duration(attempt) * time.Second
			time.Sleep(backoff)
		}
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// sendToDLQ sends a failed message to the Dead Letter Queue
func (c *Consumer) sendToDLQ(event EmailEvent, processingError error) {
	// Add error information to event
	dlqEvent := map[string]interface{}{
		"original_event": event,
		"error":          processingError.Error(),
		"failed_at":      time.Now(),
		"consumer_group": c.config.ConsumerGroup,
	}

	jsonData, err := json.Marshal(dlqEvent)
	if err != nil {
		c.logger.Error("Failed to marshal DLQ event",
			"messageID", event.MessageID,
			"error", err)
		return
	}

	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &c.config.DLQTopic,
			Partition: kafka.PartitionAny,
		},
		Value: jsonData,
	}

	err = c.dlqProducer.Produce(msg, nil)
	if err != nil {
		c.logger.Error("Failed to send to DLQ",
			"messageID", event.MessageID,
			"error", err)
		return
	}

	c.logger.Warn("Email event sent to DLQ",
		"messageID", event.MessageID,
		"recipient", event.Recipient,
		"dlq_topic", c.config.DLQTopic)
}

// commitMessage commits the Kafka offset
func (c *Consumer) commitMessage(msg *kafka.Message) {
	_, err := c.consumer.CommitMessage(msg)
	if err != nil {
		c.logger.Error("Failed to commit offset",
			"topic", *msg.TopicPartition.Topic,
			"partition", msg.TopicPartition.Partition,
			"offset", msg.TopicPartition.Offset,
			"error", err)
	}
}

// Close closes the consumer
func (c *Consumer) Close() {
	c.logger.Info("Closing Kafka consumer...")
	c.dlqProducer.Flush(5000)
	c.dlqProducer.Close()
	c.consumer.Close()
	c.logger.Info("Kafka consumer closed")
}
