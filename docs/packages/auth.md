# Package auth

Package auth implements passwordless authentication using email verification codes. It provides services for code generation, validation, and user management with Redis-backed code storage and PostgreSQL user persistence.

## Overview

The auth package is responsible for:
- Generating and validating 6-digit verification codes
- Managing user lifecycle (create, read, update, delete)
- Integrating with session management for authenticated access
- Storing verification codes in Redis with TTL-based expiration

## Installation

```go
import "instant/internal/auth"
```

## Index

### Constants
- `VerificationCodeTTL` - Time-to-live for verification codes (10 minutes)

### Variables
- `ErrInvalidCode` - Verification code is invalid or expired
- `ErrUserNotFound` - User does not exist in database
- `ErrUsernameExists` - Username already taken
- `ErrEmailExists` - Email already registered
- `ErrUnauthorized` - Authentication failed

### Types
- `Service` - Authentication service interface
- `service` - Concrete implementation (unexported)

### Functions
- `NewService(store session.Store, db *sql.DB) Service` - Creates new auth service

## Constants

### VerificationCodeTTL
```go
const VerificationCodeTTL = 10 * time.Minute
```

VerificationCodeTTL defines how long a verification code remains valid in Redis before automatic expiration. Codes are single-use and deleted after successful verification.

## Variables

### Error Definitions

```go
var (
    // ErrInvalidCode indicates the verification code is incorrect, expired, or already used
    ErrInvalidCode = errors.New("invalid or expired verification code")

    // ErrUserNotFound indicates no user exists with the given identifier
    ErrUserNotFound = errors.New("user not found")

    // ErrUsernameExists indicates the username is already taken by another user
    ErrUsernameExists = errors.New("username already exists")

    // ErrEmailExists indicates the email is already registered to another user
    ErrEmailExists = errors.New("email already exists")

    // ErrUnauthorized indicates authentication credentials are invalid
    ErrUnauthorized = errors.New("unauthorized")
)
```

These sentinel errors allow callers to identify specific failure conditions and handle them appropriately.

## Types

### type Service

```go
type Service interface {
    // RequestCode generates a 6-digit verification code for the given email address.
    // The code is stored in Redis with a 10-minute TTL and sent via configured delivery method.
    // Returns error if code generation or storage fails.
    RequestCode(ctx context.Context, email string) error

    // VerifyCode validates the verification code for the given email address.
    // On success, it fetches or creates the user in the database and returns the user model.
    // The verification code is deleted from Redis after successful validation.
    // Returns ErrInvalidCode if the code is incorrect, expired, or already used.
    VerifyCode(ctx context.Context, email, code string) (*models.User, error)

    // GetUserByID retrieves a user by their unique identifier.
    // Returns ErrUserNotFound if no user exists with the given ID.
    GetUserByID(ctx context.Context, userID string) (*models.User, error)

    // UpdateUser modifies existing user fields (username, email).
    // Returns ErrUsernameExists or ErrEmailExists if the new values conflict with existing users.
    // Returns ErrUserNotFound if the user does not exist.
    UpdateUser(ctx context.Context, userID string, req *models.UpdateUserRequest) (*models.User, error)

    // DeleteUser permanently removes a user from the database.
    // Returns ErrUserNotFound if the user does not exist.
    DeleteUser(ctx context.Context, userID string) error
}
```

Service defines the authentication service interface for all user authentication and management operations.

## Functions

### func NewService

```go
func NewService(store session.Store, db *sql.DB) Service
```

NewService creates a new authentication service with the provided dependencies. The store parameter is used for temporary verification code storage, while db provides persistent user data storage.

**Parameters:**
- `store` - Redis-backed store implementing session.Store interface
- `db` - PostgreSQL database connection for user persistence

**Returns:**
- `Service` - Configured authentication service ready for use

**Example:**
```go
store := session.NewRedisStore("localhost:6379", "", 0)
db := database.NewConnection(config.DatabaseConfig{
    Host:     "localhost",
    Port:     5432,
    Database: "instant",
    Username: "user",
    Password: "pass",
})

authService := auth.NewService(store, db)
```

## Implementation Details

### Verification Code Storage

Verification codes are stored in Redis with the following key pattern:
```
code:{email}
```

The value is the 6-digit numeric code, and the key expires after `VerificationCodeTTL` (10 minutes).

### User Database Schema

The auth service expects a `users` table with the following structure:

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(255) UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

### Code Generation

Codes are generated using the `generateSixDigitCode()` function, which produces cryptographically random 6-digit numbers (000000-999999). The implementation uses `crypto/rand` for security.

### Email Delivery (Development)

Currently, verification codes are logged to stdout with the format:
```
[DEV] Verification code for user@example.com: 123456
```

Production deployments should integrate with `internal/email` package for SMTP delivery.

## Usage Examples

### Basic Authentication Flow

```go
// Initialize service
authSvc := auth.NewService(store, db)

// 1. Request verification code
err := authSvc.RequestCode(context.Background(), "user@example.com")
if err != nil {
    log.Fatal(err)
}
// Check logs for code: [DEV] Verification code for user@example.com: 123456

// 2. Verify code and get user
user, err := authSvc.VerifyCode(context.Background(), "user@example.com", "123456")
if err != nil {
    if errors.Is(err, auth.ErrInvalidCode) {
        // Handle invalid/expired code
    }
    log.Fatal(err)
}

fmt.Printf("Authenticated user: %s (%s)\n", user.Email, user.ID)
```

### User Management

```go
// Get user by ID
user, err := authSvc.GetUserByID(ctx, userID)
if err != nil {
    if errors.Is(err, auth.ErrUserNotFound) {
        // Handle user not found
    }
}

// Update user
updateReq := &models.UpdateUserRequest{
    Username: strPtr("newusername"),
}
updated, err := authSvc.UpdateUser(ctx, userID, updateReq)
if err != nil {
    if errors.Is(err, auth.ErrUsernameExists) {
        // Handle duplicate username
    }
}

// Delete user
err = authSvc.DeleteUser(ctx, userID)
```

## Error Handling

All Service methods return errors that can be checked using `errors.Is()`:

```go
err := authSvc.VerifyCode(ctx, email, code)
if err != nil {
    switch {
    case errors.Is(err, auth.ErrInvalidCode):
        // Code is wrong, expired, or already used
        return 401, "Invalid verification code"
    case errors.Is(err, auth.ErrUserNotFound):
        // Unexpected state (should not happen in normal flow)
        return 500, "User creation failed"
    default:
        // Database or Redis error
        return 500, "Authentication service error"
    }
}
```

## Testing

The auth package includes comprehensive tests for all service methods. Tests use testcontainers for PostgreSQL and mockable interfaces for Redis.

Run tests:
```bash
go test ./internal/auth -v
go test ./internal/auth -run TestVerifyCode
```

## Security Considerations

1. **Code Generation**: Uses `crypto/rand` for cryptographically secure random codes
2. **Single-Use Codes**: Codes are deleted from Redis after successful verification
3. **TTL Expiration**: Codes expire automatically after 10 minutes
4. **Email Validation**: Basic email format validation (extend as needed)
5. **SQL Injection Protection**: Uses parameterized queries for all database operations
6. **Unique Constraints**: Database enforces email and username uniqueness

## Future Enhancements

- [ ] Rate limiting for code requests (prevent spam)
- [ ] SMTP integration for production email delivery
- [ ] Code attempt limits (lock after N failed attempts)
- [ ] Audit logging for authentication events
- [ ] Multi-factor authentication options
- [ ] Account recovery flows
- [ ] Email verification for account changes

## Related Packages

- [session](./session.md) - Session management for authenticated users
- [gateway](./gateway.md) - API Gateway routing and middleware
- [database](./database.md) - Database connection utilities
- [email](./email.md) - Email delivery service (TODO)

## Godoc

For complete API documentation with source code links, run:
```bash
godoc -http=:6060
```

Then visit: http://localhost:6060/pkg/instant/internal/auth
