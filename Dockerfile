FROM golang:1.25-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build all services
RUN go build -o /app/gateway cmd/gateway/main.go && \
    go build -o /app/auth cmd/auth/main.go && \
    go build -o /app/posts cmd/posts/main.go && \
    go build -o /app/files cmd/files/main.go && \
    go build -o /app/likes cmd/likes/main.go

FROM alpine:3.20.1 AS prod
WORKDIR /app

# Copy all binaries
COPY --from=build /app/gateway /app/gateway
COPY --from=build /app/auth /app/auth
COPY --from=build /app/posts /app/posts
COPY --from=build /app/files /app/files
COPY --from=build /app/likes /app/likes

# Default command (can be overridden in docker-compose)
CMD ["./gateway"]


