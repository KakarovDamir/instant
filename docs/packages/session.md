# Package session

Package session provides session management functionality with Redis-backed storage and automatic expiration. It handles session creation, validation, retrieval, and deletion for authenticated users.

## Overview

The session package implements:
- Session lifecycle management (create, read, validate, delete)
- Redis-based session storage with TTL expiration
- Session model definitions and validation
- Store abstraction for flexible storage backends

## Installation

```go
import "instant/internal/session"
```

## Index

### Variables
- `ErrSessionNotFound` - Session does not exist or has expired
- `ErrSessionExpired` - Session exists but has expired
- `ErrInvalidSession` - Session data is corrupted or invalid

### Types
- `Manager` - Session lifecycle management interface
- `Store` - Storage backend interface
- `Session` - Session data model
- `manager` - Concrete manager implementation (unexported)
- `RedisStore` - Redis-backed store implementation (unexported)

### Functions
- `NewManager(store Store, secret string) Manager` - Creates session manager
- `NewRedisStore(addr, password string, db int) Store` - Creates Redis store

## Variables

### Error Definitions

```go
var (
    // ErrSessionNotFound indicates the session ID does not exist in storage
    ErrSessionNotFound = errors.New("session not found")

    // ErrSessionExpired indicates the session has passed its expiration time
    ErrSessionExpired = errors.New("session expired")

    // ErrInvalidSession indicates session data is corrupted or cannot be parsed
    ErrInvalidSession = errors.New("invalid session data")
)
```

## Types

### type Manager

```go
type Manager interface {
    // Create generates a new session for the authenticated user.
    // It stores the session in Redis with the specified max age (TTL).
    // Returns the generated session ID and error if creation fails.
    Create(ctx context.Context, userID, email string, maxAge int) (string, error)

    // Get retrieves an existing session by ID.
    // Returns ErrSessionNotFound if the session does not exist or has expired.
    // Returns ErrInvalidSession if session data is corrupted.
    Get(ctx context.Context, sessionID string) (*Session, error)

    // Delete removes a session from storage (logout).
    // Returns ErrSessionNotFound if the session does not exist.
    Delete(ctx context.Context, sessionID string) error

    // Validate checks if a session exists and has not expired.
    // This is a convenience method combining Get with expiration checking.
    // Returns ErrSessionNotFound, ErrSessionExpired, or ErrInvalidSession as appropriate.
    Validate(ctx context.Context, sessionID string) (*Session, error)
}
```

Manager defines the interface for session lifecycle management operations.

### type Store

```go
type Store interface {
    // Set stores a key-value pair with the specified TTL.
    // The value expires automatically after the TTL duration.
    Set(ctx context.Context, key, value string, ttl time.Duration) error

    // Get retrieves the value for the given key.
    // Returns empty string and no error if the key does not exist.
    Get(ctx context.Context, key string) (string, error)

    // Delete removes a key from storage.
    // No error is returned if the key does not exist.
    Delete(ctx context.Context, key string) error
}
```

Store defines the storage backend interface. This abstraction allows for different storage implementations (Redis, Memcached, in-memory, etc.).

### type Session

```go
type Session struct {
    // ID is the unique session identifier (UUIDv4)
    ID string `json:"id"`

    // UserID is the authenticated user's unique identifier
    UserID string `json:"user_id"`

    // Email is the authenticated user's email address
    Email string `json:"email"`

    // CreatedAt is the timestamp when the session was created
    CreatedAt time.Time `json:"created_at"`

    // ExpiresAt is the timestamp when the session expires
    ExpiresAt time.Time `json:"expires_at"`
}
```

Session represents an authenticated user session with expiration information.

## Functions

### func NewManager

```go
func NewManager(store Store, secret string) Manager
```

NewManager creates a new session manager with the provided storage backend and secret key. The secret is used for session ID generation and should be cryptographically random.

**Parameters:**
- `store` - Storage backend implementing the Store interface (typically Redis)
- `secret` - Secret key for session security (minimum 32 bytes recommended)

**Returns:**
- `Manager` - Configured session manager

**Example:**
```go
store := session.NewRedisStore("localhost:6379", "", 0)
manager := session.NewManager(store, "your-secret-key-min-32-chars")
```

### func NewRedisStore

```go
func NewRedisStore(addr, password string, db int) Store
```

NewRedisStore creates a Redis-backed session store using go-redis client.

**Parameters:**
- `addr` - Redis server address (e.g., "localhost:6379")
- `password` - Redis password (empty string for no auth)
- `db` - Redis database number (0-15, typically 0)

**Returns:**
- `Store` - Redis store implementation

**Example:**
```go
// Local development (no password)
store := session.NewRedisStore("localhost:6379", "", 0)

// Production (with password and specific DB)
store := session.NewRedisStore("redis.example.com:6379", "secure-password", 1)
```

## Implementation Details

### Session Storage Format

Sessions are stored in Redis with the following key pattern:
```
session:{session-id}
```

The value is JSON-encoded session data:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": "123e4567-e89b-12d3-a456-426614174000",
  "email": "user@example.com",
  "created_at": "2025-01-15T10:00:00Z",
  "expires_at": "2025-01-15T11:00:00Z"
}
```

### Session ID Generation

Session IDs are generated using `github.com/google/uuid` (UUIDv4), providing:
- Cryptographic randomness (122 bits)
- Collision resistance
- URL-safe format
- 36-character string representation

### TTL Management

Redis automatically deletes expired sessions using its built-in TTL mechanism:
- TTL is set when the session is created (`Create()`)
- No cleanup processes required
- Memory is reclaimed automatically

### Expiration Validation

The `Validate()` method performs two checks:
1. Session exists in Redis (not expired by TTL)
2. `ExpiresAt` timestamp has not passed (application-level check)

This dual-layer approach ensures consistency even if clocks drift slightly.

## Usage Examples

### Basic Session Management

```go
// Initialize store and manager
store := session.NewRedisStore("localhost:6379", "", 0)
manager := session.NewManager(store, os.Getenv("SESSION_SECRET"))

// Create session (1 hour TTL)
sessionID, err := manager.Create(
    context.Background(),
    "user-uuid",
    "user@example.com",
    3600, // maxAge in seconds
)
if err != nil {
    log.Fatal(err)
}
fmt.Println("Session ID:", sessionID)

// Validate session
sess, err := manager.Validate(context.Background(), sessionID)
if err != nil {
    if errors.Is(err, session.ErrSessionNotFound) {
        // Session does not exist or expired
    }
    if errors.Is(err, session.ErrSessionExpired) {
        // Session past expiration time
    }
    log.Fatal(err)
}
fmt.Printf("Valid session for: %s\n", sess.Email)

// Delete session (logout)
err = manager.Delete(context.Background(), sessionID)
if err != nil {
    log.Fatal(err)
}
```

### HTTP Cookie Integration

```go
import "github.com/gin-gonic/gin"

func loginHandler(c *gin.Context, manager session.Manager) {
    // After successful authentication
    sessionID, err := manager.Create(
        c.Request.Context(),
        userID,
        email,
        3600,
    )
    if err != nil {
        c.JSON(500, gin.H{"error": "session creation failed"})
        return
    }

    // Set HTTP-only cookie
    c.SetCookie(
        "session_id",           // name
        sessionID,              // value
        3600,                   // maxAge
        "/",                    // path
        "",                     // domain
        false,                  // secure (set true in production)
        true,                   // httpOnly
    )

    c.JSON(200, gin.H{"message": "authenticated"})
}

func authMiddleware(manager session.Manager) gin.HandlerFunc {
    return func(c *gin.Context) {
        sessionID, err := c.Cookie("session_id")
        if err != nil {
            c.AbortWithStatus(401)
            return
        }

        sess, err := manager.Validate(c.Request.Context(), sessionID)
        if err != nil {
            c.AbortWithStatus(401)
            return
        }

        // Attach session to context
        c.Set("session", sess)
        c.Set("user_id", sess.UserID)
        c.Next()
    }
}
```

### Custom Store Implementation

```go
// Example: In-memory store for testing
type MemoryStore struct {
    mu   sync.RWMutex
    data map[string]string
}

func NewMemoryStore() Store {
    return &MemoryStore{
        data: make(map[string]string),
    }
}

func (s *MemoryStore) Set(ctx context.Context, key, value string, ttl time.Duration) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.data[key] = value
    // Note: TTL not implemented for simplicity
    return nil
}

func (s *MemoryStore) Get(ctx context.Context, key string) (string, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.data[key], nil
}

func (s *MemoryStore) Delete(ctx context.Context, key string) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    delete(s.data, key)
    return nil
}

// Use in tests
manager := session.NewManager(NewMemoryStore(), "test-secret")
```

## Error Handling

All Manager methods return errors that should be checked:

```go
sess, err := manager.Validate(ctx, sessionID)
if err != nil {
    switch {
    case errors.Is(err, session.ErrSessionNotFound):
        // Session does not exist - user needs to login
        return 401, "session not found"
    case errors.Is(err, session.ErrSessionExpired):
        // Session expired - user needs to re-authenticate
        return 401, "session expired"
    case errors.Is(err, session.ErrInvalidSession):
        // Data corruption - delete and force re-login
        manager.Delete(ctx, sessionID)
        return 401, "invalid session"
    default:
        // Redis connection error or other infrastructure issue
        return 500, "session validation failed"
    }
}
```

## Testing

### Unit Tests

```go
func TestSessionLifecycle(t *testing.T) {
    store := session.NewRedisStore("localhost:6379", "", 0)
    manager := session.NewManager(store, "test-secret")
    ctx := context.Background()

    // Create session
    sessionID, err := manager.Create(ctx, "user123", "test@example.com", 3600)
    assert.NoError(t, err)
    assert.NotEmpty(t, sessionID)

    // Validate session
    sess, err := manager.Validate(ctx, sessionID)
    assert.NoError(t, err)
    assert.Equal(t, "user123", sess.UserID)

    // Delete session
    err = manager.Delete(ctx, sessionID)
    assert.NoError(t, err)

    // Verify deletion
    _, err = manager.Get(ctx, sessionID)
    assert.ErrorIs(t, err, session.ErrSessionNotFound)
}
```

Run tests:
```bash
go test ./internal/session -v
```

## Configuration

### Environment Variables

Typical configuration via environment variables:

```bash
# Redis connection
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# Session settings
SESSION_SECRET=your-secret-key-minimum-32-characters
SESSION_MAX_AGE=3600  # 1 hour in seconds
```

### Session Duration Recommendations

| Use Case | Recommended TTL | Reasoning |
|----------|-----------------|-----------|
| Web application | 1-24 hours | Balance between UX and security |
| API tokens | 15-60 minutes | Short-lived, refresh token pattern |
| Mobile apps | 7-30 days | Less frequent re-auth, better UX |
| High-security | 15-30 minutes | Minimize exposure window |

## Security Considerations

1. **HTTP-Only Cookies**: Always use `httpOnly` flag to prevent XSS attacks
2. **Secure Flag**: Enable `secure` flag in production (HTTPS only)
3. **Secret Management**: Store SESSION_SECRET in secure configuration (e.g., AWS Secrets Manager)
4. **Session Fixation**: Generate new session ID on privilege escalation
5. **CSRF Protection**: Implement CSRF tokens for state-changing operations
6. **Logout**: Always delete session on explicit logout
7. **Idle Timeout**: Consider implementing sliding expiration for inactive users

## Performance Considerations

### Redis Connection Pooling

The Redis client automatically manages connection pooling:
```go
import "github.com/go-redis/redis/v8"

client := redis.NewClient(&redis.Options{
    Addr:         "localhost:6379",
    PoolSize:     10,  // Default connection pool size
    MinIdleConns: 5,   // Minimum idle connections
})
```

### Session Validation Frequency

- **Every Request**: Current implementation (secure but higher Redis load)
- **Cached Validation**: Consider caching validation results for 30-60 seconds
- **Refresh Strategy**: Extend TTL on each validation (sliding expiration)

### Redis Optimization

```bash
# Redis configuration for session storage
maxmemory 256mb
maxmemory-policy allkeys-lru  # Evict least recently used keys
appendonly no                  # Sessions can be lost on restart
```

## Future Enhancements

- [ ] Sliding expiration (extend TTL on activity)
- [ ] Multi-device session management (list user's sessions)
- [ ] Session metadata (IP address, user agent)
- [ ] Rate limiting for session operations
- [ ] Session event logging (create, validate, delete)
- [ ] Distributed session store (Redis Cluster)
- [ ] Session encryption at rest
- [ ] Refresh token support

## Related Packages

- [auth](./auth.md) - Authentication service using sessions
- [gateway](./gateway.md) - Gateway middleware for session validation
- [consul](./consul.md) - Service discovery

## Godoc

For complete API documentation with source code links:
```bash
godoc -http=:6060
```

Visit: http://localhost:6060/pkg/instant/internal/session
