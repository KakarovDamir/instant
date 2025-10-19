# Getting Started

Quick start guide for setting up and running the Instant Platform locally.

## Prerequisites

### Required Software

- **Go** 1.25 or later ([installation guide](https://go.dev/doc/install))
- **Docker** and **Docker Compose** ([installation guide](https://docs.docker.com/get-docker/))
- **Git** ([installation guide](https://git-scm.com/downloads))
- **Make** (usually pre-installed on Linux/macOS)

### Verify Installation

```bash
go version        # Should show go1.25 or later
docker --version  # Should show Docker version
docker-compose --version
make --version
```

## Quick Start (5 Minutes)

### 1. Clone Repository

```bash
git clone <repository-url>
cd instant
```

### 2. Configure Environment

```bash
# Copy example environment file
cp .env.example .env

# Edit if needed (defaults work for local development)
nano .env
```

### 3. Start All Services

```bash
# Build and start all services with Docker Compose
make docker-run

# Or manually:
docker-compose up --build
```

This starts:
- **API Gateway** - http://localhost:8080
- **Auth Service** - http://localhost:8081
- **Posts Service** - http://localhost:8082
- **Consul UI** - http://localhost:8500/ui
- **PostgreSQL** - localhost:5432
- **Redis** - localhost:6379

### 4. Verify Services

```bash
# Check all services are healthy
curl http://localhost:8080/health  # Gateway
curl http://localhost:8081/health  # Auth
curl http://localhost:8082/health  # Posts

# Or use the provided script
./test_all_routes.sh
```

### 5. Test Authentication Flow

```bash
# 1. Request verification code
curl -X POST http://localhost:8080/auth/request-code \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com"}'

# 2. Check logs for code (in another terminal)
docker-compose logs -f auth-service | grep "Verification code"
# Look for: [DEV] Verification code for test@example.com: 123456

# 3. Verify code and get session
curl -X POST http://localhost:8080/auth/verify-code \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","code":"123456"}' \
  -c cookies.txt

# 4. Access protected endpoint
curl http://localhost:8080/api/posts \
  -b cookies.txt
```

**Success!** You're now running the complete microservices platform.

---

## Development Setup

### Option 1: Docker Compose (Recommended for Full Stack)

Best for testing the complete system with all services.

```bash
# Start all services
make docker-run

# View logs
docker-compose logs -f

# Stop all services
make docker-down

# Rebuild after code changes
docker-compose up --build
```

**Pros:**
- Closest to production environment
- All services running
- Easy service discovery testing

**Cons:**
- Slower rebuild cycle
- More resource intensive

### Option 2: Local Development (Recommended for Active Development)

Best for rapid iteration on a single service.

#### Start Infrastructure

```bash
# Start only infrastructure services
docker-compose up consul redis psql_bp
```

#### Build Services

```bash
# Build all services
make build

# Or build individually
make build-gateway
make build-auth
make build-posts
```

Binaries are created in `bin/` directory.

#### Run Services

Open 3 terminal windows:

**Terminal 1 - Gateway:**
```bash
make run-gateway
# Or: go run cmd/gateway/main.go
```

**Terminal 2 - Auth Service:**
```bash
make run-auth
# Or: go run cmd/auth/main.go
```

**Terminal 3 - Posts Service:**
```bash
make run-posts
# Or: go run cmd/posts/main.go
```

**Pros:**
- Fast rebuild cycle (Go compilation is fast)
- Easy debugging with breakpoints
- Direct log output

**Cons:**
- Must manage multiple terminals
- Manual service startup

### Option 3: Live Reload (Best Developer Experience)

Uses [Air](https://github.com/cosmtrek/air) for automatic reload on code changes.

```bash
# Install Air (automatically done by Makefile)
make watch

# Or manually:
go install github.com/cosmtrek/air@latest
air
```

Configuration is in `.air.toml`. Air watches for file changes and automatically rebuilds.

**Pros:**
- Automatic reload on save
- Fast iteration
- No manual rebuilds

**Cons:**
- Additional tool dependency
- Slightly more resource usage

---

## Environment Configuration

### .env File Structure

```bash
# Gateway Configuration
GATEWAY_PORT=8080
SESSION_SECRET=your-secret-key-minimum-32-characters
SESSION_MAX_AGE=3600

# Service Ports
AUTH_SERVICE_PORT=8081
POSTS_SERVICE_PORT=8082

# Consul Configuration
CONSUL_HTTP_ADDR=localhost:8500  # Use consul:8500 in Docker

# Redis Configuration
REDIS_ADDR=localhost:6379        # Use redis:6379 in Docker
REDIS_PASSWORD=
REDIS_DB=0

# Database Configuration
DB_HOST=localhost                # Use psql_bp in Docker
DB_PORT=5432
DB_DATABASE=blueprint
DB_USERNAME=melkey
DB_PASSWORD=password1234
DB_SSLMODE=disable

# Environment
APP_ENV=development
```

### Local vs Docker Differences

| Variable | Local Development | Docker Compose |
|----------|-------------------|----------------|
| CONSUL_HTTP_ADDR | localhost:8500 | consul:8500 |
| REDIS_ADDR | localhost:6379 | redis:6379 |
| DB_HOST | localhost | psql_bp |

---

## Database Setup

### Initialize Database

The database schema is in `migrations/`:

```sql
-- migrations/001_create_users_table.sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(255) UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

Apply migrations:

```bash
# Connect to PostgreSQL
docker exec -it instant-psql_bp-1 psql -U melkey -d blueprint

# Run migrations
\i migrations/001_create_users_table.sql
\i migrations/002_add_username_column_up.sql

# Verify
\dt
\d users
```

### Database Tools

**psql (Command Line):**
```bash
# Connect to database
docker exec -it instant-psql_bp-1 psql -U melkey -d blueprint

# List tables
\dt

# Describe table
\d users

# Query
SELECT * FROM users;

# Exit
\q
```

**pgAdmin (GUI):**
```bash
docker run -p 5050:80 \
  -e 'PGADMIN_DEFAULT_EMAIL=admin@example.com' \
  -e 'PGADMIN_DEFAULT_PASSWORD=admin' \
  dpage/pgadmin4
```
Visit http://localhost:5050 and connect to localhost:5432.

---

## Service Verification

### Check Consul Registration

Visit http://localhost:8500/ui or:

```bash
# List all registered services
curl http://localhost:8500/v1/catalog/services | jq

# Check specific service health
curl http://localhost:8500/v1/health/service/auth-service | jq
```

You should see:
- `auth-service`
- `posts-service`

### Inspect Redis

```bash
# Connect to Redis CLI
docker exec -it instant-redis-1 redis-cli

# List all sessions
KEYS session:*

# View specific session
GET session:550e8400-e29b-41d4-a716-446655440000

# List verification codes
KEYS code:*

# Check TTL
TTL code:user@example.com

# Exit
exit
```

### Monitor Logs

```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f auth-service
docker-compose logs -f gateway
docker-compose logs -f posts-service

# Filter logs
docker-compose logs -f auth-service | grep "Verification code"
```

---

## Testing the API

### Using cURL

See [test_all_routes.sh](../../test_all_routes.sh) for complete examples.

```bash
# Run all tests
./test_all_routes.sh

# Or manually:
# 1. Health checks
curl http://localhost:8080/health

# 2. Request code
curl -X POST http://localhost:8080/auth/request-code \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com"}'

# 3. Get code from logs
docker-compose logs auth-service | grep "Verification code" | tail -1

# 4. Verify code
curl -X POST http://localhost:8080/auth/verify-code \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","code":"123456"}' \
  -c cookies.txt

# 5. Access protected endpoint
curl http://localhost:8080/api/posts -b cookies.txt
```

### Using Postman

1. Import the collection (TODO: create postman_collection.json)
2. Set base URL: `http://localhost:8080`
3. Run authentication flow
4. Cookies are managed automatically

### Using HTTPie

```bash
# Install HTTPie
brew install httpie  # macOS
apt install httpie   # Ubuntu

# Request code
http POST localhost:8080/auth/request-code email=test@example.com

# Verify code (HTTPie manages cookies automatically)
http POST localhost:8080/auth/verify-code \
  email=test@example.com \
  code=123456 \
  --session=user

# Access protected endpoint
http localhost:8080/api/posts --session=user
```

---

## Development Workflow

### Typical Development Cycle

1. **Start Infrastructure**
   ```bash
   docker-compose up consul redis psql_bp
   ```

2. **Make Code Changes**
   Edit files in `internal/` or `cmd/`

3. **Run Tests**
   ```bash
   go test ./internal/auth -v
   go test ./... -short
   ```

4. **Build and Run**
   ```bash
   make build
   make run-auth  # Or run-gateway, run-posts
   ```

5. **Test Changes**
   ```bash
   curl http://localhost:8081/health
   ```

6. **Commit**
   ```bash
   git add .
   git commit -m "feat: add username update endpoint"
   ```

### Hot Reload Workflow

```bash
# Terminal 1 - Infrastructure
docker-compose up consul redis psql_bp

# Terminal 2 - Live reload
make watch

# Make code changes - services reload automatically
```

---

## IDE Setup

### VS Code

**Recommended Extensions:**
- Go (golang.go)
- Docker (ms-azuretools.vscode-docker)
- REST Client (humao.rest-client)

**Configuration (.vscode/settings.json):**
```json
{
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "package",
  "go.formatTool": "goimports",
  "editor.formatOnSave": true,
  "go.testFlags": ["-v"],
  "go.coverOnSave": true
}
```

**Launch Configuration (.vscode/launch.json):**
```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Launch Auth Service",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}/cmd/auth/main.go",
      "envFile": "${workspaceFolder}/.env"
    }
  ]
}
```

### GoLand / IntelliJ IDEA

1. Open project root
2. Go SDK should be auto-detected
3. Right-click `cmd/auth/main.go` → Run
4. Set environment variables in run configuration

---

## Troubleshooting

### Services Won't Start

**Issue:** Port already in use
```
Error: bind: address already in use
```

**Solution:**
```bash
# Find process using port
lsof -i :8080
sudo kill -9 <PID>

# Or change port in .env
GATEWAY_PORT=8090
```

**Issue:** Cannot connect to Consul
```
Error: dial tcp localhost:8500: connect: connection refused
```

**Solution:**
```bash
# Check Consul is running
docker-compose ps consul

# Restart Consul
docker-compose restart consul

# Check Consul logs
docker-compose logs consul
```

### Database Connection Failed

**Issue:** Authentication failed for user
```
Error: pq: password authentication failed for user "melkey"
```

**Solution:**
```bash
# Verify credentials in .env match docker-compose.yml
# Restart database
docker-compose restart psql_bp

# Recreate database
docker-compose down -v  # Warning: deletes data
docker-compose up psql_bp
```

### Redis Connection Failed

**Issue:** Connection refused
```
Error: dial tcp localhost:6379: connect: connection refused
```

**Solution:**
```bash
# Check Redis is running
docker-compose ps redis

# Test connection
docker exec -it instant-redis-1 redis-cli ping
# Should return: PONG

# Restart Redis
docker-compose restart redis
```

### Verification Code Not Appearing

**Issue:** Code not in logs

**Solution:**
```bash
# Check auth service is running
docker-compose ps auth-service

# View logs in real-time
docker-compose logs -f auth-service | grep "Verification"

# Check Redis for stored codes
docker exec -it instant-redis-1 redis-cli
KEYS code:*
```

### Session Cookie Not Working

**Issue:** Always getting 401 Unauthorized

**Solution:**
```bash
# Verify cookie was set
curl -v http://localhost:8080/auth/verify-code ...
# Look for Set-Cookie header

# Check Redis has session
docker exec -it instant-redis-1 redis-cli
KEYS session:*

# Verify cookie is being sent
curl -v http://localhost:8080/api/posts -b cookies.txt
# Look for Cookie header in request
```

---

## Next Steps

Now that you have the platform running:

1. **Explore the API** - Try all endpoints in [API Reference](../api/)
2. **Add a New Service** - Follow [Adding Services Guide](./adding-services.md)
3. **Write Tests** - See [Testing Guide](./testing.md)
4. **Deploy to Production** - See [Deployment Guide](./deployment.md)
5. **Contribute** - Read [Contributing Guide](./contributing.md)

---

## Useful Commands Reference

```bash
# Development
make build              # Build all services
make clean              # Remove binaries
make run-gateway        # Run gateway locally
make run-auth           # Run auth service locally
make run-posts          # Run posts service locally
make watch              # Live reload with Air

# Docker
make docker-run         # Start all services
make docker-down        # Stop all services
make docker-logs        # View all logs
docker-compose up <service>  # Start specific service
docker-compose restart <service>  # Restart service

# Testing
go test ./...           # Run all tests
go test ./internal/auth -v  # Run auth tests with verbose output
go test -short ./...    # Skip integration tests
go test -race ./...     # Run with race detector
go test -cover ./...    # Run with coverage

# Database
docker exec -it instant-psql_bp-1 psql -U melkey -d blueprint
\dt                     # List tables
\d users                # Describe users table
SELECT * FROM users;    # Query users

# Redis
docker exec -it instant-redis-1 redis-cli
KEYS *                  # List all keys
KEYS session:*          # List sessions
GET session:<id>        # View session
FLUSHDB                 # Clear database (dev only!)

# Consul
open http://localhost:8500/ui  # Consul UI
curl http://localhost:8500/v1/catalog/services | jq
curl http://localhost:8500/v1/health/service/auth-service | jq
```

---

## Getting Help

- **Documentation**: Browse [docs/](../)
- **Issues**: Check existing GitHub issues
- **Examples**: See [examples/](../examples/)
- **Architecture**: Review [ARCHITECTURE.md](../../ARCHITECTURE.md)
