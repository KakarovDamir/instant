package follow

import (
	"context"
	_ "encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Service struct {
	repo  *Repository
	cache *redis.Client
}

func NewService(repo *Repository, redisAddr, redisPassword string, redisDB int) *Service {
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis connection failed: %v", err)
		rdb = nil
	} else {
		log.Println("Redis connected for follow service")
	}

	return &Service{
		repo:  repo,
		cache: rdb,
	}
}

func (s *Service) Follow(ctx context.Context, followerID, followingID uuid.UUID) error {
	if followerID == followingID {
		return fmt.Errorf("cannot follow yourself")
	}

	err := s.repo.Follow(ctx, followerID, followingID)
	if err != nil {
		return err
	}

	s.invalidateUserCache(ctx, followerID)
	s.invalidateUserCache(ctx, followingID)

	// ---- NOTIFICATION HOOK ----
	s.sendFollowNotification(followerID, followingID)

	return nil
}

func (s *Service) Unfollow(ctx context.Context, followerID, followingID uuid.UUID) error {
	err := s.repo.Unfollow(ctx, followerID, followingID)
	if err != nil {
		return err
	}

	s.invalidateUserCache(ctx, followerID)
	s.invalidateUserCache(ctx, followingID)

	return nil
}

// ---- Cache invalidation ----

func (s *Service) invalidateUserCache(ctx context.Context, userID uuid.UUID) {
	if s.cache != nil {
		pattern := fmt.Sprintf("follows:user:%s:*", userID.String())
		s.deleteByPattern(ctx, pattern)
	}
}

func (s *Service) deleteByPattern(ctx context.Context, pattern string) {
	iter := s.cache.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		s.cache.Del(ctx, iter.Val())
	}
}

// ---- Notification (simple placeholder, you can integrate Kafka/SQS later) ----

func (s *Service) sendFollowNotification(followerID, followingID uuid.UUID) {
	log.Printf("Notification: %s followed %s", followerID, followingID)
}

func (s *Service) ListFollowers(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	// Optional: implement Redis caching
	return s.repo.ListFollowers(ctx, userID)
}

func (s *Service) ListFollowing(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	// Optional: implement Redis caching
	return s.repo.ListFollowing(ctx, userID)
}
