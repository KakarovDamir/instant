package posts

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Service handles business logic for posts with caching
type Service struct {
	repo  *Repository
	cache *redis.Client
}

// NewService creates a new posts service with Redis caching
func NewService(repo *Repository, redisAddr, redisPassword string, redisDB int) *Service {
	// Initialize Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis connection failed: %v. Caching disabled.", err)
		rdb = nil
	} else {
		log.Println("Redis cache connected for posts service")
	}

	return &Service{
		repo:  repo,
		cache: rdb,
	}
}

// CreatePost creates a new post and invalidates relevant caches
func (s *Service) CreatePost(ctx context.Context, userID uuid.UUID, caption, imageURL string) (*Post, error) {
	post, err := s.repo.Create(ctx, userID, caption, imageURL)
	if err != nil {
		return nil, err
	}

	// Invalidate caches
	s.invalidateUserPostsCache(ctx, userID)
	s.invalidateAllPostsCache(ctx)

	return post, nil
}

// GetPost retrieves a post by ID with caching
func (s *Service) GetPost(ctx context.Context, postID int64) (*Post, error) {
	// Try cache first
	if s.cache != nil {
		cacheKey := fmt.Sprintf("post:%d", postID)
		cached, err := s.cache.Get(ctx, cacheKey).Result()
		if err == nil {
			var post Post
			if err := json.Unmarshal([]byte(cached), &post); err == nil {
				log.Printf("Cache hit for post %d", postID)
				return &post, nil
			}
		}
	}

	// Cache miss - fetch from database
	post, err := s.repo.GetByID(ctx, postID)
	if err != nil {
		return nil, err
	}

	// Store in cache (5 minute TTL)
	if s.cache != nil {
		cacheKey := fmt.Sprintf("post:%d", postID)
		data, _ := json.Marshal(post)
		s.cache.Set(ctx, cacheKey, data, 5*time.Minute)
	}

	return post, nil
}

// GetAllPosts retrieves all posts with pagination and caching
func (s *Service) GetAllPosts(ctx context.Context, page, pageSize int) (*PaginatedPostsResponse, error) {
	// Try cache first
	if s.cache != nil {
		cacheKey := fmt.Sprintf("posts:all:page:%d:size:%d", page, pageSize)
		cached, err := s.cache.Get(ctx, cacheKey).Result()
		if err == nil {
			var response PaginatedPostsResponse
			if err := json.Unmarshal([]byte(cached), &response); err == nil {
				log.Printf("Cache hit for posts page %d", page)
				return &response, nil
			}
		}
	}

	// Cache miss - fetch from database
	posts, totalCount, err := s.repo.GetAll(ctx, page, pageSize)
	if err != nil {
		return nil, err
	}

	totalPages := int(totalCount) / pageSize
	if int(totalCount)%pageSize != 0 {
		totalPages++
	}

	response := &PaginatedPostsResponse{
		Posts:      posts,
		Page:       page,
		PageSize:   pageSize,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}

	// Store in cache (2 minute TTL for lists)
	if s.cache != nil {
		cacheKey := fmt.Sprintf("posts:all:page:%d:size:%d", page, pageSize)
		data, _ := json.Marshal(response)
		s.cache.Set(ctx, cacheKey, data, 2*time.Minute)
	}

	return response, nil
}

// GetUserPosts retrieves posts by user ID with pagination and caching
func (s *Service) GetUserPosts(ctx context.Context, userID uuid.UUID, page, pageSize int) (*PaginatedPostsResponse, error) {
	// Try cache first
	if s.cache != nil {
		cacheKey := fmt.Sprintf("posts:user:%s:page:%d:size:%d", userID.String(), page, pageSize)
		cached, err := s.cache.Get(ctx, cacheKey).Result()
		if err == nil {
			var response PaginatedPostsResponse
			if err := json.Unmarshal([]byte(cached), &response); err == nil {
				log.Printf("Cache hit for user %s posts page %d", userID.String(), page)
				return &response, nil
			}
		}
	}

	// Cache miss - fetch from database
	posts, totalCount, err := s.repo.GetByUserID(ctx, userID, page, pageSize)
	if err != nil {
		return nil, err
	}

	totalPages := int(totalCount) / pageSize
	if int(totalCount)%pageSize != 0 {
		totalPages++
	}

	response := &PaginatedPostsResponse{
		Posts:      posts,
		Page:       page,
		PageSize:   pageSize,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}

	// Store in cache (2 minute TTL for lists)
	if s.cache != nil {
		cacheKey := fmt.Sprintf("posts:user:%s:page:%d:size:%d", userID.String(), page, pageSize)
		data, _ := json.Marshal(response)
		s.cache.Set(ctx, cacheKey, data, 2*time.Minute)
	}

	return response, nil
}

// UpdatePost updates a post and invalidates caches
func (s *Service) UpdatePost(ctx context.Context, postID int64, userID uuid.UUID, caption *string, imageURL *string) (*Post, error) {
	post, err := s.repo.Update(ctx, postID, userID, caption, imageURL)
	if err != nil {
		return nil, err
	}

	// Invalidate caches
	s.invalidatePostCache(ctx, postID)
	s.invalidateUserPostsCache(ctx, userID)
	s.invalidateAllPostsCache(ctx)

	return post, nil
}

// DeletePost deletes a post and invalidates caches
func (s *Service) DeletePost(ctx context.Context, postID int64, userID uuid.UUID) error {
	err := s.repo.Delete(ctx, postID, userID)
	if err != nil {
		return err
	}

	// Invalidate caches
	s.invalidatePostCache(ctx, postID)
	s.invalidateUserPostsCache(ctx, userID)
	s.invalidateAllPostsCache(ctx)

	return nil
}

// Cache invalidation helpers
func (s *Service) invalidatePostCache(ctx context.Context, postID int64) {
	if s.cache != nil {
		cacheKey := fmt.Sprintf("post:%d", postID)
		s.cache.Del(ctx, cacheKey)
	}
}

func (s *Service) invalidateUserPostsCache(ctx context.Context, userID uuid.UUID) {
	if s.cache != nil {
		// Delete all cached pages for this user
		pattern := fmt.Sprintf("posts:user:%s:*", userID.String())
		s.deleteByPattern(ctx, pattern)
	}
}

func (s *Service) invalidateAllPostsCache(ctx context.Context) {
	if s.cache != nil {
		// Delete all cached pages for all posts
		pattern := "posts:all:*"
		s.deleteByPattern(ctx, pattern)
	}
}

func (s *Service) deleteByPattern(ctx context.Context, pattern string) {
	iter := s.cache.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		s.cache.Del(ctx, iter.Val())
	}
	if err := iter.Err(); err != nil {
		log.Printf("Error scanning cache keys: %v", err)
	}
}
