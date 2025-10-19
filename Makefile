# Simple Makefile for a Go project

# Build the application
all: build test

build:
	@echo "Building all services..."
	@go build -o bin/gateway cmd/gateway/main.go
	@go build -o bin/auth cmd/auth/main.go
	@go build -o bin/posts cmd/posts/main.go
	@go build -o bin/files cmd/files/main.go
	@echo "Build complete!"

build-gateway:
	@echo "Building gateway..."
	@go build -o bin/gateway cmd/gateway/main.go

build-auth:
	@echo "Building auth service..."
	@go build -o bin/auth cmd/auth/main.go

build-posts:
	@echo "Building posts service..."
	@go build -o bin/posts cmd/posts/main.go

build-files:
	@echo "Building files service..."
	@go build -o bin/files cmd/files/main.go

# Run services locally
run-gateway:
	@go run cmd/gateway/main.go

run-auth:
	@go run cmd/auth/main.go

run-posts:
	@go run cmd/posts/main.go

run-files:
	@go run cmd/files/main.go

# Legacy support
run:
	@go run cmd/posts/main.go
# Create DB container
docker-run:
	@if docker compose up --build 2>/dev/null; then \
		: ; \
	else \
		echo "Falling back to Docker Compose V1"; \
		docker-compose up --build; \
	fi

# Shutdown DB container
docker-down:
	@if docker compose down 2>/dev/null; then \
		: ; \
	else \
		echo "Falling back to Docker Compose V1"; \
		docker-compose down; \
	fi

# Test the application
test:
	@echo "Testing..."
	@go test ./... -v
# Integrations Tests for the application
itest:
	@echo "Running integration tests..."
	@go test ./internal/database -v

# Database migrations
migrate:
	@./scripts/migrate.sh up

migrate-down:
	@./scripts/migrate.sh down

migrate-status:
	@./scripts/migrate.sh status

# Clean the binary
clean:
	@echo "Cleaning..."
	@rm -f main
	@rm -rf bin/

# Live Reload
watch:
	@if command -v air > /dev/null; then \
            air; \
            echo "Watching...";\
        else \
            read -p "Go's 'air' is not installed on your machine. Do you want to install it? [Y/n] " choice; \
            if [ "$$choice" != "n" ] && [ "$$choice" != "N" ]; then \
                go install github.com/air-verse/air@latest; \
                air; \
                echo "Watching...";\
            else \
                echo "You chose not to install air. Exiting..."; \
                exit 1; \
            fi; \
        fi

.PHONY: all build run test clean watch docker-run docker-down itest migrate migrate-down migrate-status
