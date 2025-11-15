# Email Service with Kafka Integration

## Overview

The email functionality has been extracted from the Auth service into a dedicated Email service that communicates via Kafka. This provides:

- **Asynchronous email sending** - Auth service returns immediately without blocking on SMTP
- **Exactly-once delivery** - Redis-based idempotency ensures no duplicate emails
- **Resilience** - Automatic retries and Dead Letter Queue (DLQ) for failures
- **Scalability** - Multiple email service instances can process emails in parallel
- **Decoupling** - Email logic is completely separated from auth logic

## Architecture

### Before (Synchronous)
```
User → Gateway → Auth Service → SMTP Server → Response
                  (blocks here ⏳)
```

### After (Asynchronous with Kafka)
```
User → Gateway → Auth Service → Kafka → Email Service → SMTP → ✓ Email Sent
                  (returns immediately ⚡)      (with idempotency)
```

## Components

### 1. Auth Service (Producer)
- Generates verification codes
- Publishes email events to Kafka topic `email-events`
- Falls back to direct email if Kafka is unavailable

### 2. Email Service (Consumer)
- Consumes email events from Kafka
- Checks Redis for deduplication (idempotency)
- Sends emails via SMTP
- Retries failed emails (max 3 attempts)
- Sends failed emails to DLQ topic

### 3. Kafka Topics
- **`email-events`**: Main topic for email events (3 partitions)
- **`email-events-dlq`**: Dead Letter Queue for failed emails (1 partition)

### 4. Redis (Idempotency Store)
- Stores message IDs of sent emails
- Pattern: `email:sent:{messageID}`
- TTL: 24 hours
- Prevents duplicate email sends

## Setup Instructions

### Prerequisites

1. **External Kafka Broker** (already deployed at `13.48.120.205:32100`)
2. **Redis** (for idempotency store)
3. **SMTP Server** (for actual email sending)

### Step 1: Create Kafka Topics

You have three options to create the required Kafka topics:

#### Option A: Using the Shell Script
```bash
cd /home/noroot/Desktop/Highload_Backend/instant
./scripts/init-kafka-topics.sh
```

#### Option B: Using the Go Program
```bash
cd /home/noroot/Desktop/Highload_Backend/instant
go run scripts/create-topics/main.go
```

#### Option C: Using Docker with Kafka Image
```bash
# Create email-events topic
docker run --rm apache/kafka:latest kafka-topics.sh \
  --bootstrap-server 13.48.120.205:32100 \
  --create --topic email-events \
  --partitions 3 \
  --replication-factor 1

# Create DLQ topic
docker run --rm apache/kafka:latest kafka-topics.sh \
  --bootstrap-server 13.48.120.205:32100 \
  --create --topic email-events-dlq \
  --partitions 1 \
  --replication-factor 1
```

### Step 2: Configure Environment Variables

Update your `.env` file with Kafka configuration:

```bash
# Kafka Configuration
KAFKA_BROKERS=13.48.120.205:32100
KAFKA_TOPIC_EMAIL_EVENTS=email-events
KAFKA_TOPIC_EMAIL_DLQ=email-events-dlq
KAFKA_CONSUMER_GROUP=email-service-group

# Enable Kafka (set to "false" for legacy direct email mode)
ENABLE_KAFKA=true

# Email Service
EMAIL_SERVICE_PORT=8085
EMAIL_SERVICE_HOST=email-service

# SMTP Configuration (for email service)
EMAIL_MODE=smtp  # or "log" for development
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASSWORD=your-app-password
SMTP_FROM=noreply@yourapp.com
SMTP_FROM_NAME=Your App Name
```

### Step 3: Start the Services

```bash
# Build and start all services
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

# Check email service logs
docker-compose logs email-service

# You should see:
# - "Email event consumed"
# - "Email sent successfully"
```

#### Check Health Endpoints
```bash
# Email service health
curl http://localhost:8085/health

# Response:
# {
#   "status": "healthy",
#   "service": "email-service",
#   "redis": "connected",
#   "idempotency_records": 5
# }
```

#### Monitor Kafka Topics
```bash
# List topics
docker run --rm apache/kafka:latest kafka-topics.sh \
  --bootstrap-server 13.48.120.205:32100 \
  --list

# Consume messages (for debugging)
docker run --rm apache/kafka:latest kafka-console-consumer.sh \
  --bootstrap-server 13.48.120.205:32100 \
  --topic email-events \
  --from-beginning
```

## Email Event Schema

### Event Structure
```json
{
  "message_id": "550e8400-e29b-41d4-a716-446655440000",
  "event_type": "verification_code",
  "timestamp": "2025-11-15T10:30:00Z",
  "recipient": "user@example.com",
  "data": {
    "code": "123456",
    "expires_in": "10m"
  }
}
```

### Event Types
- `verification_code` - Authentication verification codes (currently implemented)
- `welcome` - Welcome emails (future)
- `password_reset` - Password reset emails (future)

## Idempotency: Exactly-Once Email Delivery

### How It Works

1. **Message ID Generation**: Auth service generates a unique UUID for each email event
2. **Kafka Idempotence**: Producer has `enable.idempotence=true` to prevent duplicates in Kafka
3. **Consumer Deduplication**: Email service checks Redis before sending
4. **Atomic Check-and-Set**: Uses Redis `SET NX` to ensure only one email is sent

### Flow Diagram
```
┌─────────────────────────────────────────────────┐
│ Email Event Consumed from Kafka                 │
└──────────────────┬──────────────────────────────┘
                   │
                   ▼
          ┌────────────────────┐
          │ Check Redis:       │
          │ EXISTS email:sent: │
          │   {messageID}      │
          └─────────┬──────────┘
                    │
        ┌───────────┴───────────┐
        │                       │
        ▼                       ▼
   ┌─────────┐            ┌──────────┐
   │ EXISTS  │            │ NOT      │
   │ (True)  │            │ EXISTS   │
   └────┬────┘            └─────┬────┘
        │                       │
        ▼                       ▼
   ┌─────────────┐      ┌──────────────┐
   │ Skip        │      │ Send Email   │
   │ (Duplicate) │      │ via SMTP     │
   └─────────────┘      └───────┬──────┘
                                │
                                ▼
                        ┌───────────────┐
                        │ Mark as Sent: │
                        │ SET NX key    │
                        │ TTL: 24h      │
                        └───────────────┘
```

### Edge Cases Handled

| Scenario | Without Idempotency | With Idempotency |
|----------|-------------------|------------------|
| Consumer crashes after SMTP, before commit | Email sent 2x | Email sent 1x ✓ |
| Kafka redelivers message | Email sent 2x | Email sent 1x ✓ |
| Network timeout during SMTP | Retry sends 2nd email | Only 1 email ✓ |
| Multiple consumers (race) | Both send | First wins ✓ |

## Monitoring & Observability

### Logs
All email events are logged with structured logging:

```bash
# Email sent successfully
{"level":"info","msg":"Email sent successfully","messageID":"...","duration":"245ms"}

# Duplicate detected
{"level":"warn","msg":"Duplicate email event detected","messageID":"..."}

# Retry attempts
{"level":"warn","msg":"Failed to send email, will retry","attempt":1,"error":"..."}

# Sent to DLQ
{"level":"warn","msg":"Email event sent to DLQ","messageID":"..."}
```

### Metrics (Future Enhancement)
- Email send rate
- Duplicate rate (% of messages skipped)
- Failure rate
- DLQ message count
- Processing latency

### Health Checks

**Email Service**:
```bash
GET /health
```
Response:
```json
{
  "status": "healthy",
  "service": "email-service",
  "redis": "connected",
  "idempotency_records": 42,
  "timestamp": "2025-11-15T10:30:00Z"
}
```

**Idempotency Stats**:
```bash
GET /stats
```
Response:
```json
{
  "idempotency_records": 42,
  "ttl_hours": 24
}
```

## Failure Handling

### SMTP Failures
1. **Retry Logic**: Up to 3 attempts with exponential backoff (1s, 2s, 4s)
2. **Dead Letter Queue**: After 3 failures, send to `email-events-dlq`
3. **Manual Reprocessing**: DLQ messages can be manually replayed

### Kafka Broker Failures
- **Fallback Mode**: Auth service falls back to direct email sending
- **Logging**: Fallback events are logged for monitoring

### Redis Failures
- **Degraded Mode**: Email service continues but accepts at-least-once delivery
- **Logging**: Redis connection errors are logged

## Configuration Options

### Auth Service

| Variable | Description | Default |
|----------|-------------|---------|
| `ENABLE_KAFKA` | Enable Kafka integration | `true` |
| `KAFKA_BROKERS` | Kafka broker addresses | `13.48.120.205:32100` |
| `KAFKA_TOPIC_EMAIL_EVENTS` | Email events topic | `email-events` |

### Email Service

| Variable | Description | Default |
|----------|-------------|---------|
| `KAFKA_BROKERS` | Kafka broker addresses | `13.48.120.205:32100` |
| `KAFKA_TOPIC_EMAIL_EVENTS` | Email events topic | `email-events` |
| `KAFKA_TOPIC_EMAIL_DLQ` | Dead letter queue topic | `email-events-dlq` |
| `KAFKA_CONSUMER_GROUP` | Consumer group ID | `email-service-group` |
| `EMAIL_MODE` | Email mode (`log` or `smtp`) | `log` |
| `SMTP_HOST` | SMTP server host | - |
| `SMTP_PORT` | SMTP server port | `587` |
| `SMTP_USER` | SMTP username | - |
| `SMTP_PASSWORD` | SMTP password | - |
| `SMTP_FROM` | From email address | `noreply@example.com` |
| `SMTP_FROM_NAME` | From name | `Your App` |

## Scaling

### Horizontal Scaling
Add more email service instances to increase throughput:

```yaml
# docker-compose.yml
email-service:
  # ... existing config
  deploy:
    replicas: 3  # Run 3 instances
```

Each instance will consume from different partitions.

### Partitioning Strategy
- **3 partitions** for `email-events` topic
- **Load balancing** across consumers
- **Order preservation** within partition

## Troubleshooting

### Issue: Emails not being sent

**Check 1**: Is Kafka topic created?
```bash
docker run --rm apache/kafka:latest kafka-topics.sh \
  --bootstrap-server 13.48.120.205:32100 \
  --list
```

**Check 2**: Is email service running?
```bash
docker-compose ps email-service
docker-compose logs email-service
```

**Check 3**: Are messages in Kafka?
```bash
docker run --rm apache/kafka:latest kafka-console-consumer.sh \
  --bootstrap-server 13.48.120.205:32100 \
  --topic email-events \
  --from-beginning
```

### Issue: Duplicate emails being sent

**Check 1**: Is Redis accessible?
```bash
curl http://localhost:8085/health
# Check "redis": "connected"
```

**Check 2**: Check idempotency records
```bash
docker exec -it instant-redis-1 redis-cli
> KEYS email:sent:*
> GET email:sent:{messageID}
```

### Issue: Messages going to DLQ

**Check 1**: View DLQ messages
```bash
docker run --rm apache/kafka:latest kafka-console-consumer.sh \
  --bootstrap-server 13.48.120.205:32100 \
  --topic email-events-dlq \
  --from-beginning
```

**Check 2**: Check email service logs for errors
```bash
docker-compose logs email-service | grep ERROR
```

## Migration from Direct Email

To switch back to direct email mode (legacy):

```bash
# In .env file
ENABLE_KAFKA=false
```

Then restart auth service:
```bash
docker-compose restart auth-service
```

## Best Practices

1. **Always use unique message IDs** - Generated by auth service
2. **Monitor DLQ topic** - Set up alerts for messages in DLQ
3. **Keep Redis healthy** - Idempotency depends on it
4. **Use SMTP mode in production** - `log` mode is for development only
5. **Scale consumers** - Add more instances for higher throughput
6. **Monitor Kafka lag** - Check consumer lag regularly

## Future Enhancements

- [ ] Add more email types (welcome, password reset, etc.)
- [ ] Implement email templates system
- [ ] Add metrics and dashboards (Prometheus + Grafana)
- [ ] Support email attachments
- [ ] Add email scheduling (send at specific time)
- [ ] Implement email preferences (unsubscribe, etc.)
- [ ] Add email analytics (open rate, click rate)

## References

- [Kafka Exactly-Once Semantics](https://www.confluent.io/blog/exactly-once-semantics-are-possible-heres-how-apache-kafka-does-it/)
- [Idempotent Consumer Pattern](https://microservices.io/patterns/communication-style/idempotent-consumer.html)
- [Event-Driven Architecture](https://martinfowler.com/articles/201701-event-driven.html)
