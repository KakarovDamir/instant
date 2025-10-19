# Documentation

Welcome to the Instant Platform documentation. This microservices platform implements an API Gateway pattern with service discovery, passwordless authentication, and session management.

## Documentation Structure

### 📦 [Package Documentation](./packages/)
Detailed documentation for each Go package in the project following godoc conventions.

- [Auth Package](./packages/auth.md) - Passwordless authentication service
- [Gateway Package](./packages/gateway.md) - API Gateway and routing
- [Session Package](./packages/session.md) - Session management
- [Consul Package](./packages/consul.md) - Service discovery and registration
- [Database Package](./packages/database.md) - Database connection management
- [Email Package](./packages/email.md) - Email notification service

### 🔌 [API Reference](./api/)
Complete API endpoint documentation with examples.

- [Authentication API](./api/auth.md) - Auth endpoints and flows
- [Posts API](./api/posts.md) - Content management endpoints
- [Gateway API](./api/gateway.md) - Gateway routes and middleware

### 📚 [Developer Guides](./guides/)
Step-by-step guides for common development tasks.

- [Getting Started](./guides/getting-started.md) - Quick start guide
- [Adding New Services](./guides/adding-services.md) - Microservice development
- [Testing Guide](./guides/testing.md) - Testing strategies and examples
- [Deployment Guide](./guides/deployment.md) - Production deployment
- [Contributing](./guides/contributing.md) - Contribution guidelines

### 💡 [Examples](./examples/)
Practical code examples and usage patterns.

- [Authentication Flow](./examples/auth-flow.md) - Complete auth examples
- [Service Registration](./examples/service-registration.md) - Consul integration
- [Session Management](./examples/session-management.md) - Session handling
- [Testing Examples](./examples/testing.md) - Unit and integration tests

## Quick Links

- **Project Architecture**: See [ARCHITECTURE.md](../ARCHITECTURE.md)
- **Development Setup**: See [Getting Started Guide](./guides/getting-started.md)
- **API Testing**: See [API Reference](./api/)
- **Godoc**: Run `godoc -http=:6060` and visit http://localhost:6060

## Go Documentation Standards

This project follows Go documentation best practices:

### Package Comments
Every package has a package-level comment explaining its purpose:
```go
// Package auth implements passwordless authentication using email verification codes.
// It provides services for code generation, validation, and user management.
package auth
```

### Function Comments
Exported functions have comments starting with the function name:
```go
// NewService creates a new authentication service with the provided dependencies.
// It requires a Redis store for code storage and a database connection for user management.
func NewService(store session.Store, db *sql.DB) Service {
    // ...
}
```

### Type Comments
All exported types are documented:
```go
// Service defines the authentication service interface for user authentication operations.
// It handles verification code generation, validation, and user lifecycle management.
type Service interface {
    // RequestCode generates a 6-digit verification code for the given email.
    RequestCode(ctx context.Context, email string) error

    // VerifyCode validates the verification code and returns the authenticated user.
    VerifyCode(ctx context.Context, email, code string) (*models.User, error)
}
```

### Example Functions
Testable examples serve as both documentation and tests:
```go
func ExampleService_RequestCode() {
    svc := auth.NewService(store, db)
    err := svc.RequestCode(context.Background(), "user@example.com")
    if err != nil {
        log.Fatal(err)
    }
    // Output: Verification code sent to user@example.com
}
```

## Viewing Documentation

### Local Godoc Server
```bash
# Install godoc (if not already installed)
go install golang.org/x/tools/cmd/godoc@latest

# Start godoc server
godoc -http=:6060

# Visit in browser
open http://localhost:6060/pkg/instant/
```

### Command Line
```bash
# View package documentation
go doc internal/auth

# View specific function
go doc internal/auth.Service

# View all package details
go doc -all internal/auth
```

### IDE Integration
Most Go IDEs (VS Code, GoLand, etc.) automatically display godoc comments on hover.

## Documentation Principles

1. **Complete Sentences**: Comments are full sentences that format well when extracted
2. **Start with Name**: Comments begin with the name of the thing being described
3. **Package-Level First**: Every package has an overview comment
4. **Examples**: Testable examples demonstrate real usage
5. **No Blank Lines**: Comments directly precede declarations
6. **Links**: Use `[Name]` to link to other symbols in the same package

## Contributing to Documentation

When adding new code:
1. Add package-level comments in `doc.go` if extensive
2. Document all exported types, functions, and constants
3. Add testable examples for common use cases
4. Update relevant guides and API documentation
5. Run `godoc` locally to verify formatting

## Need Help?

- **General Questions**: See [Getting Started](./guides/getting-started.md)
- **API Issues**: Check [API Reference](./api/)
- **Contributing**: Read [Contributing Guide](./guides/contributing.md)
- **Architecture**: Review [ARCHITECTURE.md](../ARCHITECTURE.md)
