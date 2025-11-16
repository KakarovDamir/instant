FROM golang:1.25-bookworm AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build all services
# Note: Using vendored librdkafka (no -tags dynamic needed on Debian)
RUN go build -o /app/gateway cmd/gateway/main.go && \
    go build -o /app/auth cmd/auth/main.go && \
    go build -o /app/email cmd/email/main.go && \
    go build -o /app/posts cmd/posts/main.go && \
    go build -o /app/files cmd/files/main.go && \
    go build -o /app/likes cmd/likes/main.go && \
    go build -o /app/follow cmd/follow/main.go 

FROM debian:bookworm-slim AS prod

# Install ca-certificates for HTTPS connections
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy all binaries
COPY --from=build /app/gateway /app/gateway
COPY --from=build /app/auth /app/auth
COPY --from=build /app/email /app/email
COPY --from=build /app/posts /app/posts
COPY --from=build /app/files /app/files
COPY --from=build /app/likes /app/likes
COPY --from=build /app/follow /app/follow  

# Default command (can be overridden in docker-compose)
CMD ["./gateway"]


