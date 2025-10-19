# Architecture Documentation

## Overview

This project implements a microservices architecture with an API Gateway pattern, service discovery using Consul, and session-based passwordless authentication.

## System Architecture

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │
       ▼
┌─────────────────────────────────────────────┐
│           API Gateway (Port 8080)            │
│  - Session Validation Middleware            │
│  - Request Routing & Load Balancing         │
│  - Reverse Proxy                            │
└───────┬──────────────────┬──────────────────┘
        │                  │
        │                  ▼
        │         ┌────────────────┐
        │         │ Consul         │◄──────────┐
        │         │ Service        │           │
        │         │ Discovery      │           │
        │         └────────────────┘           │
        │                  │                   │
        │                  │                   │
        ▼                  ▼                   │
┌──────────────┐    ┌──────────────┐          │
│ Auth Service │    │ Posts Service│          │
│ (Port 8081)  │    │ (Port 8082)  │          │
│              │    │              │          │
│ Registers ───┼────┘              │          │
│ with Consul  │    └──────┬───────┘          │
└──────┬───────┘           │                  │
       │                   │                  │
       │                   │                  │
       ▼                   ▼                  │
┌─────────────┐      ┌─────────────┐         │
│   Redis     │      │ PostgreSQL  │         │
│ Session &   │      │  Database   │         │
│ Code Store  │      └─────────────┘         │
└─────────────┘                               │
                                              │
                     Registers ◄──────────────┘
```

## Components

### 1. API Gateway (`cmd/gateway`)
**Purpose**: Entry point for all client requests
**Port**: 8080
**Responsibilities**:
- Session validation via middleware
- Service discovery and routing
- Reverse proxy to backend services
- Load balancing (random selection)

**Routes**:
- `/auth/*` → Auth Service (public)
- `/api/*` → Backend Services (requires session)
- `/health` → Gateway health check

### 2. Auth Service (`cmd/auth`)
**Purpose**: Passwordless authentication
**Port**: 8081
**Endpoints**:
- `POST /request-code` - Generate verification code
- `POST /verify-code` - Verify code and create session
- `POST /logout` - Invalidate session
- `GET /health` - Health check

**Flow**:
1. User requests code with email
2. 6-digit code generated and stored in Redis (10 min TTL)
3. Code logged to console (email integration TODO)
4. User submits code
5. Code validated, user created/retrieved from DB
6. Session created in Redis
7. Session cookie returned to client

### 3. Posts Service (`cmd/posts`)
**Purpose**: Content management (posts, etc.)
**Port**: 8082
**Features**:
- Existing functionality preserved
- Registers with Consul on startup
- Receives user context via `X-User-ID` header

### 4. Consul
**Purpose**: Service discovery and health checking
**Port**: 8500 (HTTP API), 8600 (DNS)
**Features**:
- Service registry
- Health checks every 10s
- Automatic instance discovery
- Web UI at http://localhost:8500/ui

### 5. Redis
**Purpose**: Session and verification code storage
**Port**: 6379
**Usage**:
- Session data: `session:{sessionID}`
- Verification codes: `code:{email}`
- TTL-based expiration

### 6. PostgreSQL
**Purpose**: Persistent data storage
**Port**: 5432
**Tables**: users, posts, etc.

## Data Flow

### Authentication Flow
```
1. Client → Gateway → Auth Service: POST /auth/request-code
   └─ Auth Service generates code
   └─ Code stored in Redis (10 min TTL)
   └─ Code logged to console

2. Client → Gateway → Auth Service: POST /auth/verify-code
   └─ Auth Service validates code from Redis
   └─ User fetched/created in PostgreSQL
   └─ Session created in Redis (1 hour TTL)
   └─ Session cookie returned

3. Client → Gateway (with session cookie)
   └─ Gateway validates session in Redis
   └─ If valid, adds X-User-ID header
   └─ Request proxied to backend service
```

### Protected Request Flow
```
Client → Gateway (with session_id cookie)
         ↓
    Validate session in Redis
         ↓
    [Valid?] ──No──► 401 Unauthorized
         ↓ Yes
    Discover service from Consul
         ↓
    Add X-User-ID header
         ↓
    Reverse proxy to service
         ↓
    Service processes request
         ↓
    Response ──► Client
```

## Session Management

### Session Structure
```json
{
  "id": "uuid-v4",
  "user_id": "user-uuid",
  "email": "user@example.com",
  "created_at": "2025-01-01T00:00:00Z",
  "expires_at": "2025-01-01T01:00:00Z"
}
```

### Session Validation
- **Location**: Gateway middleware
- **Method**: Cookie-based (`session_id`)
- **Storage**: Redis with TTL
- **Expiration**: 1 hour (configurable via `SESSION_MAX_AGE`)

### Why Gateway-Level Validation?
✅ **Centralized**: Single point of validation
✅ **Performance**: One Redis lookup per request
✅ **Security**: Backend services trust gateway headers
✅ **Simplicity**: No duplicate validation logic

## Service Discovery

### Registration
Each service registers with Consul on startup:
```go
consul.Register(&ServiceConfig{
    ID:      "service-name-hostname",
    Name:    "service-name",
    Port:    8081,
    Check: &HealthCheck{
        HTTP:     "http://host:port/health",
        Interval: "10s",
    },
})
```

### Discovery
Gateway discovers services dynamically:
```go
instance := consul.DiscoverOne("auth-service")
// Returns: {Address: "auth-service", Port: 8081}
```

### Load Balancing
- **Algorithm**: Random selection
- **Health Checks**: Only healthy instances returned
- **Future**: Can implement round-robin, least-connections, etc.

## Configuration

### Environment Variables

#### Gateway
```env
GATEWAY_PORT=8080
CONSUL_HTTP_ADDR=consul:8500
REDIS_ADDR=redis:6379
REDIS_PASSWORD=
REDIS_DB=0
SESSION_SECRET=your-secret-key
SESSION_MAX_AGE=3600
```

#### Auth Service
```env
AUTH_SERVICE_PORT=8081
AUTH_SERVICE_HOST=auth-service
CONSUL_HTTP_ADDR=consul:8500
REDIS_ADDR=redis:6379
DB_HOST=psql_bp
DB_PORT=5432
DB_DATABASE=blueprint
DB_USERNAME=melkey
DB_PASSWORD=password1234
```

#### Posts Service
```env
PORT=8082
POSTS_SERVICE_HOST=posts-service
CONSUL_HTTP_ADDR=consul:8500
DB_HOST=psql_bp
...
```

## Building and Running

### Local Development
```bash
# Copy environment file
cp .env.example .env

# Build all services
make build

# Run individual services
make run-gateway
make run-auth
make run-posts
```

### Docker Compose
```bash
# Start all services
docker-compose up --build

# Or
make docker-run

# Stop all services
make docker-down
```

### Services will be available at:
- Gateway: http://localhost:8080
- Auth Service: http://localhost:8081
- Posts Service: http://localhost:8082
- Consul UI: http://localhost:8500/ui
- PostgreSQL: localhost:5432
- Redis: localhost:6379

## Testing the System

### 1. Check Health
```bash
curl http://localhost:8080/health
# Should return: {"status":"healthy","service":"api-gateway"}
```

### 2. Request Verification Code
```bash
curl -X POST http://localhost:8080/auth/request-code \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com"}'

# Check logs for code: [DEV] Verification code for test@example.com: 123456
```

### 3. Verify Code and Get Session
```bash
curl -X POST http://localhost:8080/auth/verify-code \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","code":"123456"}' \
  -c cookies.txt

# Returns session cookie in cookies.txt
```

### 4. Access Protected Resource
```bash
curl http://localhost:8080/api/posts \
  -b cookies.txt

# Or with header:
curl http://localhost:8080/api/posts \
  -H "Cookie: session_id=your-session-id"
```

### 5. Check Consul Services
Visit http://localhost:8500/ui to see registered services.

## Project Structure

```
instant/
├── cmd/                 # Entry points for each service (Go convention)
│   ├── gateway/         # API Gateway entry point
│   ├── auth/            # Auth service entry point
│   └── posts/           # Posts service entry point
│
├── internal/            # Private packages (compiler-enforced)
│   ├── gateway/         # Gateway-specific logic
│   │   ├── middleware.go  # Session validation
│   │   ├── handler.go     # Reverse proxy
│   │   └── router.go      # Route setup
│   │
│   ├── auth/            # Auth-specific logic
│   │   ├── handler.go     # Auth endpoints
│   │   ├── service.go     # Business logic
│   │   └── models.go      # Auth models
│   │
│   ├── posts/           # Posts-specific logic
│   │   ├── server.go      # Server setup
│   │   └── routes.go      # Route handlers
│   │
│   ├── consul/          # Shared: Service discovery (infrastructure)
│   │   ├── client.go      # Consul client wrapper
│   │   ├── registry.go    # Service registration
│   │   └── discovery.go   # Service discovery
│   │
│   ├── session/         # Shared: Session management (infrastructure)
│   │   ├── manager.go     # Session CRUD
│   │   ├── store.go       # Redis store
│   │   └── models.go      # Session models
│   │
│   └── database/        # Shared: Database access (infrastructure)
│       └── database.go    # PostgreSQL connection
│
├── docker-compose.yml   # Multi-service orchestration
├── Dockerfile           # Multi-binary build
├── Makefile             # Build commands
└── .env.example         # Configuration template
```

### Structure Rationale (The Go Way)

This project follows **Go best practices** while avoiding over-engineering:

#### Why `internal/` (Not `pkg/`)?

**✅ Using `internal/`**:
- **Compiler-enforced privacy** (Go 1.4+): Prevents accidental external imports
- Officially documented in Go specs
- Clear intent: "This code is private to this project"
- Freedom to refactor without breaking external users

**❌ NOT using `pkg/`**:
- No special compiler treatment (just a convention)
- Criticized as "useless abstraction" by Go community
- Adds 4 characters to every import path with no benefit
- Implies "public API" when we're building an application, not a library
- **Not recommended for applications** (per Go community consensus 2024)

#### Why Service-Specific + Shared Structure?

This is a **monorepo with microservices pattern**:

**Service-Specific Packages** (`gateway/`, `auth/`, `posts/`):
- Each service owns its domain logic
- Independent, loosely coupled
- Can be extracted to separate repos if needed

**Shared Infrastructure** (`consul/`, `session/`, `database/`):
- Common infrastructure concerns
- Acceptable in a monorepo (same codebase, same team)
- **Not shared domain models** (maintains service independence)
- Avoids code duplication for technical concerns

#### Go Philosophy Applied

From official Go guidelines and community best practices:

1. **"Start simple, add complexity only when needed"**
   - We have 3 services in one repo (simple)
   - No premature abstraction layers
   - No unnecessary `pkg/` directory

2. **"Prefer composition over inheritance"**
   - Small, focused interfaces (e.g., `Store`, `Manager`, `ServiceDiscovery`)
   - No deep hierarchies
   - Clear dependencies

3. **"Package names matter more than directory structure"**
   - Package names match their purpose: `session`, `consul`, `gateway`
   - Single-word, lowercase packages
   - Descriptive without being generic (no `utils`, `helpers`)

4. **YAGNI (You Aren't Gonna Need It)**
   - No root domain package (services don't share domain types)
   - No elaborate layered architecture
   - No `pkg/` for "future external use"

#### When to Refactor?

This structure scales for monorepo applications. Consider refactoring if:

- ❌ **Don't split** unless you need true independent deployment
- ❌ **Don't add `pkg/`** unless publishing a library
- ✅ **Do extract shared code** if it becomes a genuine reusable library (then move to `pkg/` or separate repo)
- ✅ **Do split repos** if services need independent versioning/deployment

**Current verdict**: Structure is appropriate for a monorepo microservices application.

## Go Conventions Applied

### Naming
- **Packages**: lowercase, single-word (`consul`, `session`, `gateway`)
- **Files**: snake_case (`service_discovery.go`, `session_manager.go`)
- **Exported**: UpperCamelCase (`NewGateway`, `SessionManager`)
- **Unexported**: lowerCamelCase (`loadBalance`, `validateSession`)

### Design Patterns
- **Interfaces**: Small, focused (`Store`, `Manager`, `ServiceDiscovery`)
- **Composition**: No inheritance, only embedding
- **Explicit errors**: Always returned, never panicked (except startup)
- **Context**: Passed as first parameter

### Code Organization
- **internal/**: Private packages, not importable
- **cmd/**: Entry points for each service
- **No pkg/**: Currently no shared public libraries

## Future Enhancements

### Phase 2: Email Integration
- [ ] Add SMTP email sender (`internal/email/`)
- [ ] Use `github.com/wneessen/go-mail`
- [ ] HTML email templates

### Phase 3: Additional Services
- [ ] Comments service
- [ ] Likes service
- [ ] Follow service
- [ ] Feed service
- [ ] Files service

### Phase 4: Advanced Features
- [ ] JWT instead of sessions (optional)
- [ ] Rate limiting middleware
- [ ] Request/response logging
- [ ] Metrics and monitoring (Prometheus)
- [ ] Distributed tracing (Jaeger)
- [ ] Circuit breaker pattern
- [ ] API versioning
- [ ] GraphQL gateway

### Phase 5: Production Readiness
- [ ] TLS/HTTPS support
- [ ] Kubernetes deployment
- [ ] Horizontal pod autoscaling
- [ ] Database migrations (golang-migrate)
- [ ] Centralized logging (ELK stack)
- [ ] Secret management (Vault)
- [ ] Load testing
- [ ] Security audit

## Troubleshooting

### Service not registering with Consul
- Check Consul is running: `docker-compose ps consul`
- Check service logs for registration errors
- Verify `CONSUL_HTTP_ADDR` is correct

### Session validation failing
- Check Redis is running: `docker-compose ps redis`
- Verify session cookie is being sent
- Check Redis for session key: `redis-cli GET session:{id}`

### Service discovery failing
- Check service health in Consul UI
- Verify service registered: `curl http://localhost:8500/v1/catalog/services`
- Check health endpoint returns 200

### Cannot connect to database
- Check PostgreSQL is running
- Verify database credentials in `.env`
- Check database health: `docker-compose exec psql_bp pg_isready`

## Security Considerations

### Current Implementation
✅ HttpOnly cookies (prevents XSS)
✅ Session expiration
✅ Health check endpoints
✅ Service isolation

### TODO for Production
⚠️ Enable HTTPS/TLS
⚠️ Set Secure flag on cookies
⚠️ Add CSRF protection
⚠️ Rate limiting
⚠️ Input validation and sanitization
⚠️ Secret rotation
⚠️ Network policies

## Contributing

When adding new services:
1. Create entry point in `cmd/{service-name}/`
2. Implement business logic in `internal/{service-name}/`
3. Register with Consul on startup
4. Add health check endpoint
5. Add routes to gateway router
6. Update docker-compose.yml
7. Update this documentation

## License

[Your License Here]
