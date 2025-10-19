package session

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Store defines the interface for session storage operations
type Store interface {
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}

// redisStore implements Store interface using Redis
type redisStore struct {
	client *redis.Client
}

// NewRedisStore creates a new Redis-backed session store
func NewRedisStore(addr, password string, db int) Store {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	return &redisStore{
		client: client,
	}
}

// Set stores a key-value pair with TTL
func (s *redisStore) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return s.client.Set(ctx, key, value, ttl).Err()
}

// Get retrieves a value by key
func (s *redisStore) Get(ctx context.Context, key string) (string, error) {
	return s.client.Get(ctx, key).Result()
}

// Delete removes a key from the store
func (s *redisStore) Delete(ctx context.Context, key string) error {
	return s.client.Del(ctx, key).Err()
}

// Exists checks if a key exists in the store
func (s *redisStore) Exists(ctx context.Context, key string) (bool, error) {
	count, err := s.client.Exists(ctx, key).Result()
	return count > 0, err
}
