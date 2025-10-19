// Package auth implements passwordless authentication logic for the auth service.
// It provides email-based verification code generation and validation.
package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math/big"
	"time"

	"instant/internal/database"
	"instant/internal/email"
	"instant/internal/session"

	"github.com/google/uuid"
)

const (
	// VerificationCodeTTL defines how long verification codes remain valid
	VerificationCodeTTL = 10 * time.Minute
)

var (
	// ErrInvalidCode is returned when verification code is invalid
	ErrInvalidCode = errors.New("invalid or expired verification code")
	// ErrUserNotFound is returned when user is not found
	ErrUserNotFound = errors.New("user not found")
	// ErrUsernameExists is returned when username is already taken
	ErrUsernameExists = errors.New("username already taken")
	// ErrEmailExists is returned when email is already registered
	ErrEmailExists = errors.New("email already registered")
	// ErrUnauthorized is returned when user is not authorized for the action
	ErrUnauthorized = errors.New("unauthorized action")
)

// Service defines the authentication service interface
type Service interface {
	RequestCode(ctx context.Context, email string) error
	VerifyCode(ctx context.Context, email, code, username string) (*User, error)
	VerifyCodeOnly(ctx context.Context, email, code string) error
	UpdateUser(ctx context.Context, userID string, updates UpdateUserRequest) (*User, error)
	DeleteUser(ctx context.Context, userID, email, code string) error
	GetUserByID(ctx context.Context, userID string) (*User, error)
}

// service implements the Service interface
type service struct {
	db          database.Service
	codeStore   session.Store
	emailSender email.Sender
}

// NewService creates a new authentication service
func NewService(db database.Service, codeStore session.Store, emailSender email.Sender) Service {
	return &service{
		db:          db,
		codeStore:   codeStore,
		emailSender: emailSender,
	}
}

// RequestCode generates and stores a verification code for the given email
func (s *service) RequestCode(ctx context.Context, email string) error {
	// Generate 6-digit verification code
	code := generateSixDigitCode()

	// Store code in Redis with TTL
	key := fmt.Sprintf("code:%s", email)
	err := s.codeStore.Set(ctx, key, code, VerificationCodeTTL)
	if err != nil {
		return fmt.Errorf("failed to store verification code: %w", err)
	}

	// Send verification code via email
	err = s.emailSender.SendVerificationCode(email, code)
	if err != nil {
		return fmt.Errorf("failed to send verification code: %w", err)
	}

	return nil
}

// VerifyCode verifies the provided code and returns the user
func (s *service) VerifyCode(ctx context.Context, email, code, username string) (*User, error) {
	// Get stored code from Redis
	key := fmt.Sprintf("code:%s", email)
	storedCode, err := s.codeStore.Get(ctx, key)
	if err != nil {
		return nil, ErrInvalidCode
	}

	// Compare codes
	if storedCode != code {
		return nil, ErrInvalidCode
	}

	// Delete used code immediately (best effort, log if fails)
	if err := s.codeStore.Delete(ctx, key); err != nil {
		log.Printf("Warning: failed to delete verification code for %s: %v", email, err)
	}

	// Get or create user with username
	user, err := s.getOrCreateUser(ctx, email, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get or create user: %w", err)
	}

	return user, nil
}

// VerifyCodeOnly verifies the provided code without creating or updating a user
func (s *service) VerifyCodeOnly(ctx context.Context, email, code string) error {
	// Get stored code from Redis
	key := fmt.Sprintf("code:%s", email)
	storedCode, err := s.codeStore.Get(ctx, key)
	if err != nil {
		return ErrInvalidCode
	}

	// Compare codes
	if storedCode != code {
		return ErrInvalidCode
	}

	// Delete used code immediately (best effort, log if fails)
	if err := s.codeStore.Delete(ctx, key); err != nil {
		log.Printf("Warning: failed to delete verification code for %s: %v", email, err)
	}

	return nil
}

// getOrCreateUser retrieves a user by email or creates a new one if not exists
func (s *service) getOrCreateUser(ctx context.Context, email, username string) (*User, error) {
	// Try to get existing user
	user, err := s.getUserByEmail(ctx, email)
	if err == nil {
		// If user exists but username is different, update it
		if user.Username != username {
			user.Username = username
			return s.updateUserUsername(ctx, user.ID, username)
		}
		return user, nil
	}

	// If user doesn't exist, create new one
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	return s.createUser(ctx, email, username)
}

// getUserByEmail retrieves a user by email
func (s *service) getUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `SELECT id, email, username, created_at, updated_at FROM users WHERE email = $1`

	var user User
	row := s.db.QueryRow(ctx, query, email)

	err := row.Scan(&user.ID, &user.Email, &user.Username, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// createUser creates a new user with the given email and username
func (s *service) createUser(ctx context.Context, email, username string) (*User, error) {
	user := &User{
		ID:        uuid.New().String(),
		Email:     email,
		Username:  username,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	query := `
		INSERT INTO users (id, email, username, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, email, username, created_at, updated_at
	`

	row := s.db.QueryRow(ctx, query, user.ID, user.Email, user.Username, user.CreatedAt, user.UpdatedAt)

	var createdUser User
	err := row.Scan(&createdUser.ID, &createdUser.Email, &createdUser.Username, &createdUser.CreatedAt, &createdUser.UpdatedAt)
	if err != nil {
		// Check for unique constraint violations
		if isUniqueViolation(err, "users_username_key") {
			return nil, ErrUsernameExists
		}
		if isUniqueViolation(err, "users_email_key") {
			return nil, ErrEmailExists
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	log.Printf("Created new user: %s (ID: %s, Username: %s)", createdUser.Email, createdUser.ID, createdUser.Username)

	return &createdUser, nil
}

// updateUserUsername updates the username for an existing user
func (s *service) updateUserUsername(ctx context.Context, userID, username string) (*User, error) {
	query := `
		UPDATE users
		SET username = $1, updated_at = $2
		WHERE id = $3
		RETURNING id, email, username, created_at, updated_at
	`

	row := s.db.QueryRow(ctx, query, username, time.Now(), userID)

	var user User
	err := row.Scan(&user.ID, &user.Email, &user.Username, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		// Check for unique constraint violations
		if isUniqueViolation(err, "users_username_key") {
			return nil, ErrUsernameExists
		}
		return nil, fmt.Errorf("failed to update username: %w", err)
	}

	log.Printf("Updated username for user: %s (ID: %s, New Username: %s)", user.Email, user.ID, user.Username)

	return &user, nil
}

// generateSixDigitCode generates a cryptographically secure random 6-digit verification code
func generateSixDigitCode() string {
	// Use crypto/rand for security-sensitive random generation
	// Generate random number between 100000 and 999999
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		// This should never fail in practice, but handle it defensively
		panic(fmt.Sprintf("failed to generate secure random number: %v", err))
	}
	code := int(n.Int64()) + 100000
	return fmt.Sprintf("%06d", code)
}

// GetUserByID retrieves a user by their ID
func (s *service) GetUserByID(ctx context.Context, userID string) (*User, error) {
	query := `SELECT id, email, username, created_at, updated_at FROM users WHERE id = $1`

	var user User
	row := s.db.QueryRow(ctx, query, userID)

	err := row.Scan(&user.ID, &user.Email, &user.Username, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

// UpdateUser updates user information (email and/or username)
func (s *service) UpdateUser(ctx context.Context, userID string, updates UpdateUserRequest) (*User, error) {
	// First verify user exists
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Build dynamic update query based on provided fields
	updateFields := []string{}
	args := []interface{}{}
	argCount := 1

	if updates.Username != nil {
		updateFields = append(updateFields, fmt.Sprintf("username = $%d", argCount))
		args = append(args, updates.Username)
		argCount++
	}

	if updates.Email != nil {
		updateFields = append(updateFields, fmt.Sprintf("email = $%d", argCount))
		args = append(args, updates.Email)
		argCount++
	}

	// If no fields to update, return current user
	if len(updateFields) == 0 {
		return user, nil
	}

	// Always update updated_at
	updateFields = append(updateFields, fmt.Sprintf("updated_at = $%d", argCount))
	args = append(args, time.Now())
	argCount++

	// Add user ID as final parameter
	args = append(args, userID)

	query := fmt.Sprintf(`
		UPDATE users
		SET %s
		WHERE id = $%d
		RETURNING id, email, username, created_at, updated_at
	`, joinStrings(updateFields, ", "), argCount)

	row := s.db.QueryRow(ctx, query, args...)

	var updatedUser User
	err = row.Scan(&updatedUser.ID, &updatedUser.Email, &updatedUser.Username, &updatedUser.CreatedAt, &updatedUser.UpdatedAt)
	if err != nil {
		// Check for unique constraint violations
		if isUniqueViolation(err, "users_username_key") {
			return nil, ErrUsernameExists
		}
		if isUniqueViolation(err, "users_email_key") {
			return nil, ErrEmailExists
		}
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	log.Printf("Updated user: %s (ID: %s)", updatedUser.Email, updatedUser.ID)

	return &updatedUser, nil
}

// DeleteUser deletes a user account after verifying the code
func (s *service) DeleteUser(ctx context.Context, userID, email, code string) error {
	// Verify the code first
	err := s.VerifyCodeOnly(ctx, email, code)
	if err != nil {
		return err
	}

	// Verify user exists and email matches
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	if user.Email != email {
		return ErrUnauthorized
	}

	// Delete user
	query := `DELETE FROM users WHERE id = $1`
	_, err = s.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	log.Printf("Deleted user: %s (ID: %s)", email, userID)

	return nil
}

// joinStrings joins a slice of strings with a separator
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// isUniqueViolation checks if the error is a unique constraint violation
func isUniqueViolation(err error, constraintName string) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	// Check for PostgreSQL unique constraint violation error messages
	return errMsg != "" &&
		(contains(errMsg, "duplicate key value violates unique constraint") &&
		 contains(errMsg, constraintName))
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr) >= 0
}

// findSubstring finds the index of a substring in a string
func findSubstring(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
