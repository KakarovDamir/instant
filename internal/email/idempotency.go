package email

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// IdempotencyStore handles deduplication of email events
type IdempotencyStore struct {
	redis  *redis.Client
	ttl    time.Duration
	logger *slog.Logger
}

// NewIdempotencyStore creates a new idempotency store
func NewIdempotencyStore(redisClient *redis.Client, logger *slog.Logger) *IdempotencyStore {
	return &IdempotencyStore{
		redis:  redisClient,
		ttl:    24 * time.Hour, // Keep deduplication records for 24 hours
		logger: logger,
	}
}

// keyPrefix returns the Redis key prefix for email deduplication
func (s *IdempotencyStore) keyPrefix() string {
	return "email:sent:"
}

// buildKey builds the Redis key for a given message ID
func (s *IdempotencyStore) buildKey(messageID string) string {
	return fmt.Sprintf("%s%s", s.keyPrefix(), messageID)
}

// IsProcessed checks if an email event has already been processed
func (s *IdempotencyStore) IsProcessed(ctx context.Context, messageID string) (bool, error) {
	key := s.buildKey(messageID)

	exists, err := s.redis.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check if message is processed: %w", err)
	}

	return exists > 0, nil
}

// MarkAsProcessed marks an email event as processed
// Returns true if successfully marked (first time), false if already exists (duplicate)
// Uses Redis SET NX (set if not exists) for atomic check-and-set
func (s *IdempotencyStore) MarkAsProcessed(ctx context.Context, event EmailEvent) (bool, error) {
	key := s.buildKey(event.MessageID)

	// Create metadata to store
	metadata := EmailMetadata{
		SentAt:    time.Now(),
		Recipient: event.Recipient,
		EventType: event.EventType,
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return false, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Use SET NX (set if not exists) for atomic operation
	// This ensures only one consumer can mark the message as processed
	success, err := s.redis.SetNX(ctx, key, metadataJSON, s.ttl).Result()
	if err != nil {
		return false, fmt.Errorf("failed to mark message as processed: %w", err)
	}

	if success {
		s.logger.Info("Marked email as processed",
			"messageID", event.MessageID,
			"recipient", event.Recipient,
			"type", event.EventType)
	} else {
		s.logger.Warn("Email already processed (duplicate detected)",
			"messageID", event.MessageID,
			"recipient", event.Recipient,
			"type", event.EventType)
	}

	return success, nil
}

// GetMetadata retrieves the metadata for a processed email
func (s *IdempotencyStore) GetMetadata(ctx context.Context, messageID string) (*EmailMetadata, error) {
	key := s.buildKey(messageID)

	data, err := s.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("message not found: %s", messageID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata: %w", err)
	}

	var metadata EmailMetadata
	err = json.Unmarshal([]byte(data), &metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &metadata, nil
}

// Clean removes old processed records (optional maintenance)
// This is not necessary as Redis TTL will auto-expire keys,
// but can be useful for manual cleanup if needed
func (s *IdempotencyStore) Clean(ctx context.Context) (int64, error) {
	// In our case, Redis auto-expires keys with TTL
	// So this is just for logging/monitoring purposes
	pattern := s.keyPrefix() + "*"

	var cursor uint64
	var count int64

	for {
		var keys []string
		var err error

		keys, cursor, err = s.redis.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return count, fmt.Errorf("failed to scan keys: %w", err)
		}

		count += int64(len(keys))

		if cursor == 0 {
			break
		}
	}

	s.logger.Info("Idempotency store stats",
		"active_records", count,
		"ttl", s.ttl)

	return count, nil
}
