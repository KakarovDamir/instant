# Godoc Documentation Template

This file provides templates for writing Go documentation following official Go conventions and best practices. Use these templates when adding documentation to new or existing packages.

## Quick Reference

### Key Principles

1. **Complete Sentences**: Comments are full sentences, properly capitalized and punctuated
2. **Name First**: Comments start with the name of the thing being described
3. **No Blank Lines**: Comments directly precede declarations with no blank lines
4. **Package Overview**: Every package has a package-level comment
5. **Exported Only**: Only document exported (capitalized) identifiers
6. **Examples**: Add testable examples for common use cases

### Documentation Checklist

- [ ] Package has package-level doc comment (in `doc.go` if lengthy)
- [ ] All exported types have doc comments
- [ ] All exported functions have doc comments
- [ ] All exported constants/variables have doc comments
- [ ] Complex interfaces have method doc comments
- [ ] Examples added for main use cases
- [ ] Verified with `godoc -http=:6060`

## Package-Level Documentation

### Template: Simple Package Comment

For straightforward packages, add a comment directly above the package declaration:

```go
// Package <name> provides <brief description of purpose>.
// <Additional context, scope, or usage notes if needed>.
package name
```

### Example: Simple Package

```go
// Package session provides session management functionality with Redis-backed storage.
// It handles session creation, validation, retrieval, and deletion for authenticated users.
package session
```

### Template: Complex Package (doc.go)

For packages requiring extensive documentation, create a separate `doc.go` file:

```go
/*
Package <name> provides <brief description>.

<Detailed package description - explain what the package does, its purpose,
and its primary use cases. This can be multiple paragraphs.>

# Overview

<High-level architecture, key concepts, or design decisions>

# Usage

Basic usage example:

	// Code example here
	svc := pkgname.NewService()
	result, err := svc.DoSomething()

# Authentication

<Explain authentication/authorization if relevant>

# Error Handling

<Describe error patterns and sentinel errors>

# Performance Considerations

<Note any performance characteristics users should know>
*/
package name
```

### Example: Complex Package

```go
/*
Package consul provides service discovery and registration using HashiCorp Consul.

This package wraps the Consul API client to provide simplified service registration,
deregistration, health checks, and service discovery with load balancing.

# Overview

Services register themselves with Consul on startup, providing their address,
port, and health check configuration. The registry automatically deregisters
services during graceful shutdown.

Service discovery allows clients to find healthy instances of registered services.
The discovery client implements random load balancing by default, but can be
extended for other strategies.

# Usage

Basic service registration:

	client := consul.NewClientWithToken("localhost:8500", "")
	registrar := consul.NewServiceRegistrar(client)

	config := &consul.ServiceConfig{
		ID:      "my-service-1",
		Name:    "my-service",
		Port:    8080,
		Address: "localhost",
		Check: &consul.HealthCheck{
			HTTP:     "http://localhost:8080/health",
			Interval: "10s",
		},
	}

	err := registrar.Register(config)

Service discovery:

	discovery := consul.NewServiceDiscovery(client)
	instance, err := discovery.DiscoverOne("my-service")

# Health Checks

Consul performs periodic health checks on registered services. Services must
implement a /health endpoint that returns 200 OK when healthy.

# Load Balancing

The DiscoverOne method returns a random healthy instance. For custom load
balancing, use Discover to get all healthy instances and implement your
own selection strategy.
*/
package consul
```

## Type Documentation

### Template: Struct

```go
// <TypeName> represents <what it represents>.
// <Additional details about the type's purpose, behavior, or constraints>.
type TypeName struct {
	// <FieldName> is <description of what this field represents>.
	// <Additional context if needed, like valid values or relationships>.
	FieldName type `json:"field_name"`

	// <AnotherField> specifies <purpose>.
	AnotherField type `json:"another_field"`
}
```

### Example: Struct

```go
// Session represents an authenticated user session with expiration information.
// Sessions are stored in Redis and automatically expire based on the TTL.
type Session struct {
	// ID is the unique session identifier (UUIDv4).
	ID string `json:"id"`

	// UserID is the authenticated user's unique identifier.
	UserID string `json:"user_id"`

	// Email is the authenticated user's email address.
	Email string `json:"email"`

	// CreatedAt is the timestamp when the session was created.
	CreatedAt time.Time `json:"created_at"`

	// ExpiresAt is the timestamp when the session expires.
	// After this time, the session is no longer valid.
	ExpiresAt time.Time `json:"expires_at"`
}
```

### Template: Interface

```go
// <InterfaceName> defines <what the interface represents>.
// <Additional context about the interface's purpose or typical implementations>.
type InterfaceName interface {
	// <MethodName> does <what it does>.
	// <Parameter explanations if not obvious>.
	// Returns <what it returns> and error if <failure conditions>.
	MethodName(ctx context.Context, param type) (result, error)

	// <AnotherMethod> performs <action>.
	AnotherMethod(param type) error
}
```

### Example: Interface

```go
// Manager defines the interface for session lifecycle management operations.
// Implementations typically use Redis or similar key-value stores for session persistence.
type Manager interface {
	// Create generates a new session for the authenticated user.
	// It stores the session with the specified max age (TTL in seconds).
	// Returns the generated session ID and error if creation fails.
	Create(ctx context.Context, userID, email string, maxAge int) (string, error)

	// Get retrieves an existing session by ID.
	// Returns ErrSessionNotFound if the session does not exist or has expired.
	// Returns ErrInvalidSession if session data is corrupted.
	Get(ctx context.Context, sessionID string) (*Session, error)

	// Delete removes a session from storage (logout operation).
	// Returns ErrSessionNotFound if the session does not exist.
	Delete(ctx context.Context, sessionID string) error

	// Validate checks if a session exists and has not expired.
	// This is a convenience method combining Get with expiration checking.
	Validate(ctx context.Context, sessionID string) (*Session, error)
}
```

## Function Documentation

### Template: Constructor Function

```go
// New<Name> creates a new <name> with the provided <parameters>.
// <Additional context about initialization, validation, or side effects>.
func New<Name>(param1 type, param2 type) *Name {
	// Implementation
}
```

### Example: Constructor

```go
// NewService creates a new authentication service with the provided dependencies.
// The store parameter is used for temporary verification code storage,
// while db provides persistent user data storage.
//
// Example:
//
//	store := session.NewRedisStore("localhost:6379", "", 0)
//	db := database.NewConnection(cfg)
//	authSvc := auth.NewService(store, db)
func NewService(store session.Store, db *sql.DB) Service {
	return &service{
		store: store,
		db:    db,
	}
}
```

### Template: Regular Function

```go
// <FunctionName> performs <action>.
// <Parameter descriptions if not obvious from names>.
// <Return value descriptions>.
// Returns error if <specific failure conditions>.
func FunctionName(ctx context.Context, param type) (result, error) {
	// Implementation
}
```

### Example: Regular Function

```go
// generateSixDigitCode creates a cryptographically random 6-digit verification code.
// The code is a string representation of a number between 000000 and 999999.
// Returns error if the random number generator fails.
func generateSixDigitCode() (string, error) {
	max := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
```

## Constant and Variable Documentation

### Template: Constants

```go
// <ConstantName> defines <what it defines>.
// <Additional context about usage, valid range, or related constants>.
const ConstantName = value

// Grouped constants with shared context
const (
	// <Const1> represents <meaning>.
	Const1 = value

	// <Const2> specifies <purpose>.
	Const2 = value
)
```

### Example: Constants

```go
// VerificationCodeTTL defines how long a verification code remains valid.
// Codes are stored in Redis and automatically expire after this duration.
const VerificationCodeTTL = 10 * time.Minute

// HTTP status codes for custom responses
const (
	// StatusTooManyRequests indicates rate limiting is active.
	StatusTooManyRequests = 429

	// StatusServiceUnavailable indicates temporary service outage.
	StatusServiceUnavailable = 503
)
```

### Template: Variables (Sentinel Errors)

```go
var (
	// Err<Name> indicates <what this error means>.
	// <When this error occurs and how to handle it>.
	Err<Name> = errors.New("<error message>")

	// Err<Another> signals <condition>.
	Err<Another> = errors.New("<error message>")
)
```

### Example: Sentinel Errors

```go
var (
	// ErrInvalidCode indicates the verification code is incorrect, expired, or already used.
	// Clients should prompt the user to request a new code.
	ErrInvalidCode = errors.New("invalid or expired verification code")

	// ErrUserNotFound indicates no user exists with the given identifier.
	// This may occur when looking up users by ID or email.
	ErrUserNotFound = errors.New("user not found")

	// ErrUnauthorized indicates authentication credentials are invalid.
	// The client must re-authenticate with valid credentials.
	ErrUnauthorized = errors.New("unauthorized")
)
```

## Example Functions

### Template: Basic Example

```go
// Example<FunctionName> demonstrates basic usage of <FunctionName>.
func Example<FunctionName>() {
	// Setup (if needed)
	// Call function
	// Print result
	// Output:
	// expected output line 1
	// expected output line 2
}
```

### Example: Basic Example

```go
// ExampleNewManager demonstrates creating a session manager.
func ExampleNewManager() {
	store := session.NewRedisStore("localhost:6379", "", 0)
	manager := session.NewManager(store, "secret-key")

	sessionID, _ := manager.Create(context.Background(), "user-id", "user@example.com", 3600)
	fmt.Println("Session created:", sessionID != "")
	// Output:
	// Session created: true
}
```

### Template: Complex Example with Setup

```go
// Example<TypeName>_<method> demonstrates <usage scenario>.
func Example<TypeName>_<method>() {
	// Setup dependencies
	// Create instance
	// Demonstrate usage
	// Show results
	// Output:
	// expected output
}
```

### Example: Complex Example

```go
// ExampleService_VerifyCode demonstrates the verification flow.
func ExampleService_VerifyCode() {
	// Setup (normally from test fixtures)
	store := session.NewMemoryStore()
	db := setupTestDB()
	svc := auth.NewService(store, db)

	// Request code
	_ = svc.RequestCode(context.Background(), "user@example.com")

	// Verify code (using known test code)
	user, err := svc.VerifyCode(context.Background(), "user@example.com", "123456")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("User email:", user.Email)
	// Output:
	// User email: user@example.com
}
```

## Method Documentation

### Template: Method

```go
// <MethodName> performs <action> on <receiver type>.
// <Parameter descriptions>.
// <Return value descriptions>.
// Returns error if <failure conditions>.
func (r *ReceiverType) MethodName(ctx context.Context, param type) (result, error) {
	// Implementation
}
```

### Example: Method

```go
// Create generates a new session for the authenticated user.
// It stores the session in Redis with the specified max age (TTL).
// The session ID is a UUIDv4 string that clients should store as a cookie.
// Returns the session ID and error if creation or storage fails.
func (m *manager) Create(ctx context.Context, userID, email string, maxAge int) (string, error) {
	session := &Session{
		ID:        uuid.New().String(),
		UserID:    userID,
		Email:     email,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Duration(maxAge) * time.Second),
	}

	data, err := json.Marshal(session)
	if err != nil {
		return "", err
	}

	key := fmt.Sprintf("session:%s", session.ID)
	ttl := time.Duration(maxAge) * time.Second

	if err := m.store.Set(ctx, key, string(data), ttl); err != nil {
		return "", err
	}

	return session.ID, nil
}
```

## Special Documentation Patterns

### Deprecation Notice

```go
// RequestCodeLegacy generates a verification code.
//
// Deprecated: Use RequestCode instead. This method will be removed in v2.0.
func RequestCodeLegacy(email string) error {
	// Implementation
}
```

### Links to Other Symbols

```go
// Manager uses Store for persistence and Session for data structure.
// See Store interface for storage requirements.
// See Session type for session data fields.
type Manager interface {
	Create(ctx context.Context, userID, email string, maxAge int) (string, error)
}
```

### Code Blocks in Comments

```go
// CreateSession creates a new session with the following flow:
//
//	1. Generate UUIDv4 session ID
//	2. Create Session struct with user data
//	3. Serialize to JSON
//	4. Store in Redis with TTL
//	5. Return session ID
//
// Example usage:
//
//	sessionID, err := manager.Create(ctx, "user123", "user@example.com", 3600)
//	if err != nil {
//		return err
//	}
```

### Lists in Comments

Use blank lines to separate items:

```go
// The service implements several security measures:
//
// - Codes expire after 10 minutes
// - Single-use codes (deleted after verification)
// - Cryptographically secure random generation
// - Rate limiting (TODO)
// - Email validation
```

## Testing Documentation

### Test Function Naming

```go
// TestServiceCreate verifies session creation with valid inputs.
func TestServiceCreate(t *testing.T) {
	// Test implementation
}

// TestServiceCreate_InvalidInput verifies error handling for invalid inputs.
func TestServiceCreate_InvalidInput(t *testing.T) {
	// Test implementation
}
```

### Benchmark Documentation

```go
// BenchmarkSessionCreate measures session creation performance.
func BenchmarkSessionCreate(b *testing.B) {
	// Benchmark implementation
}
```

## Complete Package Example

Here's a complete small package following all conventions:

```go
// Package counter provides a thread-safe counter with persistence.
package counter

import (
	"context"
	"errors"
	"sync"
)

var (
	// ErrNegativeValue indicates an attempt to set a negative counter value.
	ErrNegativeValue = errors.New("counter value cannot be negative")
)

// Counter represents a thread-safe counter that can be incremented and decremented.
// All operations are atomic and safe for concurrent use.
type Counter interface {
	// Increment increases the counter by the given amount.
	// Returns the new value after incrementing.
	Increment(amount int) int

	// Decrement decreases the counter by the given amount.
	// Returns ErrNegativeValue if the result would be negative.
	Decrement(amount int) (int, error)

	// Value returns the current counter value.
	Value() int

	// Reset sets the counter back to zero.
	Reset()
}

type counter struct {
	mu    sync.RWMutex
	value int
}

// NewCounter creates a new counter initialized to zero.
func NewCounter() Counter {
	return &counter{
		value: 0,
	}
}

// Increment increases the counter by the given amount.
// The amount must be non-negative.
func (c *counter) Increment(amount int) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.value += amount
	return c.value
}

// Decrement decreases the counter by the given amount.
// Returns ErrNegativeValue if the result would be negative.
func (c *counter) Decrement(amount int) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.value-amount < 0 {
		return c.value, ErrNegativeValue
	}

	c.value -= amount
	return c.value, nil
}

// Value returns the current counter value atomically.
func (c *counter) Value() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.value
}

// Reset sets the counter back to zero atomically.
func (c *counter) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value = 0
}
```

## Godoc Verification

After writing documentation, verify it renders correctly:

```bash
# Start godoc server
godoc -http=:6060

# Open in browser
open http://localhost:6060/pkg/instant/internal/<package>

# Check for:
# - Package overview appears
# - All exports are documented
# - Examples are shown
# - Links work correctly
# - Formatting is correct
```

## Common Mistakes to Avoid

### ❌ Wrong: Blank line between comment and declaration
```go
// Create creates a session.

func (m *manager) Create(ctx context.Context) {}
```

### ✅ Right: No blank line
```go
// Create creates a session.
func (m *manager) Create(ctx context.Context) {}
```

### ❌ Wrong: Comment doesn't start with symbol name
```go
// Creates a new session for the user.
func (m *manager) Create(ctx context.Context) {}
```

### ✅ Right: Starts with symbol name
```go
// Create creates a new session for the user.
func (m *manager) Create(ctx context.Context) {}
```

### ❌ Wrong: Incomplete sentences
```go
// session manager
type Manager interface {}
```

### ✅ Right: Complete sentences
```go
// Manager defines the interface for session management.
type Manager interface {}
```

### ❌ Wrong: Documenting unexported symbols
```go
// store implements the Store interface.
type store struct {}
```

### ✅ Right: Only exported symbols
```go
// Store defines the interface for session storage.
type Store interface {}

type store struct {} // unexported, no doc comment needed
```

## Resources

- **Official Guide**: https://go.dev/blog/godoc
- **Doc Comments**: https://tip.golang.org/doc/comment
- **Effective Go**: https://go.dev/doc/effective_go#commentary
- **Package Documentation**: https://yourbasic.org/golang/package-documentation/

---

Use this template as a reference when documenting new packages. Consistency in documentation style makes the codebase more maintainable and professional.
