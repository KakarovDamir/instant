# Kafka Email Service - Quick Start Guide

## Overview

Email functionality has been successfully extracted from the Auth service into a dedicated Email service using Kafka for asynchronous, exactly-once delivery.

## What Was Done

### Architecture Changes
✅ **Auth Service** now publishes email events to Kafka instead of sending emails directly
✅ **Email Service** (new) consumes events from Kafka and sends emails via SMTP
✅ **Redis-based idempotency** ensures exactly-once email delivery
✅ **Dead Letter Queue** for handling failed emails
✅ **Fallback mode** for direct email if Kafka is unavailable

### Files Created/Modified

#### New Files
- `internal/kafka/config.go` - Kafka configuration
- `internal/kafka/producer.go` - Kafka producer wrapper
- `internal/email/models.go` - Email event models
- `internal/email/idempotency.go` - Redis-based deduplication
- `internal/email/consumer.go` - Kafka consumer with retry logic
- `internal/email/handler.go` - HTTP handlers for health checks
- `cmd/email/main.go` - Email service entry point
- `scripts/init-kafka-topics.sh` - Shell script to create topics
- `scripts/create-topics/main.go` - Go program to create topics
- `docs/EMAIL_SERVICE_KAFKA.md` - Detailed documentation
- `docs/KAFKA_EMAIL_SETUP.md` - This file

#### Modified Files
- `internal/auth/service.go` - Added Kafka producer support
- `cmd/auth/main.go` - Initialize Kafka producer
- `internal/email/sender.go` - Added `SendEmailEvent()` method
- `.env.example` - Added Kafka configuration
- `docker-compose.yml` - Added email-service
- `Dockerfile` - Added email service build
- `go.mod` - Added confluent-kafka-go dependency

## Prerequisites

1. **Kafka Broker** running at `13.48.120.205:32100` ✅ (already deployed)
2. **Redis** for session storage and idempotency
3. **PostgreSQL** for user data
4. **SMTP Server** (optional, can use log mode for testing)

## Quick Start

### Step 1: Create Kafka Topics

Choose one of these methods:

**Option A: Using Go Script (Recommended)**
```bash
cd /home/noroot/Desktop/Highload_Backend/instant
go run scripts/create-topics/main.go
```

**Option B: Using Shell Script**
```bash
cd /home/noroot/Desktop/Highload_Backend/instant
chmod +x scripts/init-kafka-topics.sh
./scripts/init-kafka-topics.sh
```

**Expected Output:**
```
============================================================
Kafka Topic Initialization Script (Go)
============================================================
Kafka Brokers: 13.48.120.205:32100
Email Events Topic: email-events
DLQ Topic: email-events-dlq
Partitions: 3
Replication Factor: 1
============================================================

Connecting to Kafka...
✓ Connected to Kafka

Creating topics...
✓ Topic 'email-events' created successfully
✓ Topic 'email-events-dlq' created successfully
```

### Step 2: Configure Environment

Create a `.env` file from the example:
```bash
cp .env.example .env
```

Update these key variables in `.env`:
```bash
# Kafka Configuration (REQUIRED)
KAFKA_BROKERS=13.48.120.205:32100
KAFKA_TOPIC_EMAIL_EVENTS=email-events
KAFKA_TOPIC_EMAIL_DLQ=email-events-dlq
KAFKA_CONSUMER_GROUP=email-service-group

# Enable Kafka for Auth Service
ENABLE_KAFKA=true

# Email Service
EMAIL_SERVICE_PORT=8085
EMAIL_SERVICE_HOST=email-service

# Email Mode (use 'log' for testing, 'smtp' for production)
EMAIL_MODE=log

# SMTP Settings (only needed when EMAIL_MODE=smtp)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASSWORD=your-app-password
SMTP_FROM=noreply@yourapp.com
SMTP_FROM_NAME=Your App Name
```

### Step 3: Build and Start Services

```bash
# Build all services
docker-compose build

# Start all services
docker-compose up -d

# Check logs
docker-compose logs -f email-service
docker-compose logs -f auth-service
```

### Step 4: Verify Everything Works

#### Test the Email Flow
```bash
# Request a verification code
curl -X POST http://localhost:8080/auth/request-code \
  -H "Content-Type: application/json" \
  -d '{"email": "test@example.com"}'

# Expected response:
# {"message":"verification code sent to your email"}
```

#### Check Email Service Logs
```bash
docker-compose logs email-service | tail -20
```

You should see:
```
email-service  | {"level":"info","msg":"Email event consumed","messageID":"..."}
email-service  | {"level":"info","msg":"Email sent successfully","messageID":"..."}
email-service  | {"level":"info","msg":"Marked email as processed","messageID":"..."}
```

#### Verify Health Endpoints
```bash
# Email service health
curl http://localhost:8085/health

# Expected response:
{
  "status": "healthy",
  "service": "email-service",
  "redis": "connected",
  "idempotency_records": 1,
  "timestamp": "..."
}

# Email service stats
curl http://localhost:8085/stats

# Expected response:
{
  "idempotency_records": 1,
  "ttl_hours": 24
}
```

## Testing Idempotency

To verify exactly-once delivery, simulate duplicate messages:

```bash
# Send the same request twice quickly
curl -X POST http://localhost:8080/auth/request-code \
  -H "Content-Type: application/json" \
  -d '{"email": "duplicate@test.com"}'

curl -X POST http://localhost:8080/auth/request-code \
  -H "Content-Type: application/json" \
  -d '{"email": "duplicate@test.com"}'
```

Check logs for deduplication:
```bash
docker-compose logs email-service | grep "Duplicate"
```

Expected output:
```
email-service  | {"level":"warn","msg":"Duplicate email event detected, skipping","messageID":"..."}
```

## Monitoring

### View Kafka Messages

```bash
# Using kafka-console-consumer (if Kafka CLI tools installed)
docker run --rm apache/kafka:latest kafka-console-consumer.sh \
  --bootstrap-server 13.48.120.205:32100 \
  --topic email-events \
  --from-beginning \
  --max-messages 5
```

### Check Redis Idempotency Keys

```bash
# Connect to Redis
docker exec -it instant-redis-1 redis-cli

# List all email sent keys
KEYS email:sent:*

# Get details of a specific key
GET email:sent:{messageID}

# Check TTL
TTL email:sent:{messageID}
```

### Monitor Service Health

```bash
# Watch email service logs in real-time
docker-compose logs -f email-service

# Check all service statuses
docker-compose ps

# View resource usage
docker stats
```

## Troubleshooting

### Issue: "Failed to connect to Kafka"

**Solution:**
```bash
# Verify Kafka is accessible
telnet 13.48.120.205 32100

# If connection fails, check network/firewall
ping 13.48.120.205
```

### Issue: "Topic does not exist"

**Solution:**
```bash
# Create topics manually
go run scripts/create-topics/main.go

# Verify topics exist
docker run --rm apache/kafka:latest kafka-topics.sh \
  --bootstrap-server 13.48.120.205:32100 \
  --list
```

### Issue: Emails not being sent

**Checklist:**
1. ✅ Is email-service running? `docker-compose ps email-service`
2. ✅ Is Kafka accessible? `telnet 13.48.120.205 32100`
3. ✅ Are topics created? (see above)
4. ✅ Is Redis running? `docker-compose ps redis`
5. ✅ Check logs: `docker-compose logs email-service`

### Issue: Duplicate emails being sent

**Solution:**
```bash
# Check Redis connection
curl http://localhost:8085/health | jq .redis

# Should return "connected"
# If "disconnected", restart Redis:
docker-compose restart redis
```

### Issue: Messages going to DLQ

**View DLQ messages:**
```bash
docker run --rm apache/kafka:latest kafka-console-consumer.sh \
  --bootstrap-server 13.48.120.205:32100 \
  --topic email-events-dlq \
  --from-beginning
```

**Common causes:**
- SMTP server unreachable (check EMAIL_MODE, SMTP_HOST settings)
- Invalid email format
- SMTP authentication failure

## Configuration Options

### Auth Service

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `ENABLE_KAFKA` | Enable Kafka integration | `true` | No |
| `KAFKA_BROKERS` | Kafka broker addresses | `13.48.120.205:32100` | Yes (if Kafka enabled) |
| `KAFKA_TOPIC_EMAIL_EVENTS` | Email events topic | `email-events` | Yes (if Kafka enabled) |

### Email Service

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `KAFKA_BROKERS` | Kafka broker addresses | `13.48.120.205:32100` | Yes |
| `KAFKA_TOPIC_EMAIL_EVENTS` | Email events topic | `email-events` | Yes |
| `KAFKA_TOPIC_EMAIL_DLQ` | DLQ topic | `email-events-dlq` | Yes |
| `KAFKA_CONSUMER_GROUP` | Consumer group ID | `email-service-group` | Yes |
| `EMAIL_MODE` | `log` or `smtp` | `log` | Yes |
| `SMTP_HOST` | SMTP server host | - | Yes (if EMAIL_MODE=smtp) |
| `SMTP_PORT` | SMTP server port | `587` | Yes (if EMAIL_MODE=smtp) |
| `SMTP_USER` | SMTP username | - | Yes (if EMAIL_MODE=smtp) |
| `SMTP_PASSWORD` | SMTP password | - | Yes (if EMAIL_MODE=smtp) |

## Rollback / Disable Kafka

To switch back to direct email mode:

```bash
# Edit .env
ENABLE_KAFKA=false

# Restart auth service
docker-compose restart auth-service

# Email service can be stopped
docker-compose stop email-service
```

## Performance Tuning

### Scale Email Service Horizontally

```yaml
# docker-compose.yml
email-service:
  # ... existing config
  deploy:
    replicas: 3  # Run 3 instances
```

### Adjust Consumer Group Settings

In `.env`:
```bash
# Process more messages in parallel
KAFKA_CONSUMER_GROUP=email-service-group-1
```

### Increase Topic Partitions

```bash
# Create topics with more partitions
KAFKA_TOPIC_PARTITIONS=10 go run scripts/create-topics/main.go
```

## Next Steps

1. **Production SMTP Setup** - Configure real SMTP credentials
2. **Add Monitoring** - Set up Prometheus + Grafana for metrics
3. **Add More Email Types** - Welcome emails, password reset, etc.
4. **Email Templates** - Create HTML email templates
5. **Email Preferences** - Implement unsubscribe functionality

## Reference Documentation

- [Detailed Architecture Guide](./EMAIL_SERVICE_KAFKA.md)
- [Kafka Documentation](https://kafka.apache.org/documentation/)
- [confluent-kafka-go](https://github.com/confluentinc/confluent-kafka-go)
- [Idempotent Consumer Pattern](https://microservices.io/patterns/communication-style/idempotent-consumer.html)

## Support

For issues or questions:
1. Check logs: `docker-compose logs [service-name]`
2. Review [EMAIL_SERVICE_KAFKA.md](./EMAIL_SERVICE_KAFKA.md)
3. Test with `EMAIL_MODE=log` first before enabling SMTP
4. Verify Kafka connectivity with `telnet 13.48.120.205 32100`

---

**Implementation completed successfully! 🎉**

All 15 tasks completed:
- ✅ Kafka producer and consumer implementation
- ✅ Email service with idempotency
- ✅ Docker configuration
- ✅ Topic creation scripts
- ✅ Complete documentation
