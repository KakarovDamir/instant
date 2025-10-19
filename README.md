# Project instant

A high-performance file operations service built with Go, featuring presigned URLs for direct client-to-storage uploads/downloads using MinIO (S3-compatible storage).

## Features

- **Presigned URLs**: Secure, time-limited URLs for direct file uploads/downloads
- **S3-Compatible Storage**: Uses MinIO for scalable object storage
- **Clean Architecture**: Separation of concerns (server, database, storage layers)
- **Health Checks**: Monitor database and storage service status
- **Docker Support**: Full Docker Compose setup for easy local development
- **RESTful API**: Built with Gin framework

## Quick Links

- **[Quick Start Guide](QUICKSTART.md)** - Get up and running in 5 minutes
- **[File Operations API](FILE_OPERATIONS_API.md)** - Detailed API documentation

## Getting Started

### Prerequisites

- Go 1.25+ ([installation guide](https://go.dev/doc/install))
- Docker & Docker Compose

### Installation

1. Clone the repository
2. Copy environment variables:
   ```bash
   cp .env.example .env
   ```
3. Start all services:
   ```bash
   make docker-run
   ```
4. Check health:
   ```bash
   curl http://localhost:8080/health
   ```

For detailed instructions, see [QUICKSTART.md](QUICKSTART.md).

## MakeFile

Run build make command with tests
```bash
make all
```

Build the application
```bash
make build
```

Run the application
```bash
make run
```
Create DB container
```bash
make docker-run
```

Shutdown DB Container
```bash
make docker-down
```

DB Integrations Test:
```bash
make itest
```

Live reload the application:
```bash
make watch
```

Run the test suite:
```bash
make test
```

Clean up binary from the last build:
```bash
make clean
```
