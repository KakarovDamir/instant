// Package session provides session management functionality for all services.
// Sessions are stored in Redis with TTL-based expiration.
// This is a shared infrastructure package used by gateway and auth services.
package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrSessionNotFound is returned when a session is not found
	ErrSessionNotFound = errors.New("session not found")
	// ErrSessionExpired is returned when a session has expired
	ErrSessionExpired = errors.New("session expired")
	// ErrInvalidSession is returned when session data is invalid
	ErrInvalidSession = errors.New("invalid session")
)

// Manager defines the interface for session management operations
type Manager interface {
	Create(ctx context.Context, userID, email string, maxAge int) (string, error)
	Get(ctx context.Context, sessionID string) (*Session, error)
	Delete(ctx context.Context, sessionID string) error
	Validate(ctx context.Context, sessionID string) (bool, error)
}

// manager implements Manager interface
type manager struct {
	store Store
}

// NewManager creates a new session manager
func NewManager(store Store) Manager {
	return &manager{
		store: store,
	}
}

// Create creates a new session and returns the session ID
func (m *manager) Create(ctx context.Context, userID, email string, maxAge int) (string, error) {
	// Generate unique session ID
	sessionID := uuid.New().String()

	// Create session object
	now := time.Now()
	session := &Session{
		ID:        sessionID,
		UserID:    userID,
		Email:     email,
		CreatedAt: now,
		ExpiresAt: now.Add(time.Duration(maxAge) * time.Second),
	}

	// Serialize session to JSON
	sessionData, err := json.Marshal(session)
	if err != nil {
		return "", fmt.Errorf("failed to marshal session: %w", err)
	}

	// Store in Redis with TTL
	key := fmt.Sprintf("session:%s", sessionID)
	ttl := time.Duration(maxAge) * time.Second

	if err := m.store.Set(ctx, key, string(sessionData), ttl); err != nil {
		return "", fmt.Errorf("failed to store session: %w", err)
	}

	return sessionID, nil
}

// Get retrieves a session by ID
func (m *manager) Get(ctx context.Context, sessionID string) (*Session, error) {
	key := fmt.Sprintf("session:%s", sessionID)

	// Get from Redis
	sessionData, err := m.store.Get(ctx, key)
	if err != nil {
		return nil, ErrSessionNotFound
	}

	// Deserialize session
	var session Session
	if err := json.Unmarshal([]byte(sessionData), &session); err != nil {
		return nil, ErrInvalidSession
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		// Delete expired session
		m.store.Delete(ctx, key)
		return nil, ErrSessionExpired
	}

	return &session, nil
}

// Delete removes a session
func (m *manager) Delete(ctx context.Context, sessionID string) error {
	key := fmt.Sprintf("session:%s", sessionID)
	return m.store.Delete(ctx, key)
}

// Validate checks if a session exists and is valid
func (m *manager) Validate(ctx context.Context, sessionID string) (bool, error) {
	session, err := m.Get(ctx, sessionID)
	if err != nil {
		return false, err
	}

	return session != nil, nil
}
