# Event Trigger Platform

![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)
![MySQL](https://img.shields.io/badge/MySQL-8.0-4479A1?logo=mysql)
![Kafka](https://img.shields.io/badge/Kafka-3.8.1-231F20?logo=apache-kafka)
![License](https://img.shields.io/badge/license-MIT-green)

A production-ready, horizontally scalable event trigger management platform built with Go. Supports three trigger types: **webhook-based**, **time-scheduled**, and **CRON-scheduled** triggers with reliable Kafka event publishing.

## Features

- **Three Trigger Types**:
  - **Webhook Triggers**: Event-driven triggers with JSON schema validation
  - **Time-Scheduled Triggers**: One-time execution at specific ISO 8601 timestamps
  - **CRON-Scheduled Triggers**: Recurring execution based on CRON expressions

- **Production-Ready Architecture**:
  - RESTful API with comprehensive CRUD operations
  - Reliable event publishing via Kafka
  - MySQL persistence with proper indexing
  - Timezone-aware schedule calculation
  - Graceful shutdown and health checks
  - Structured logging with request tracing
  - Interactive Swagger/OpenAPI documentation

- **Event Management**:
  - Event log history with retention lifecycle (active → archived → deleted)
  - Automatic schedule creation and management
  - Manual test execution for triggers
  - Advanced filtering and pagination

- **Scalability & Reliability**:
  - Horizontal scaling ready (stateless API, external consumers)
  - Atomic transactions for data consistency
  - Optimized database queries with composite indexes
  - Metrics and observability endpoints
  - At-least-once scheduling with retry on publish failure

## Architecture

### System Components

```plain
┌─────────────────────────────────────────────────────────────────┐
│                         API Server (Gin)                         │
│  • CRUD Triggers      • Event Logs      • Webhook Receiver       │
│  • Health & Metrics   • Swagger Docs    • Graceful Shutdown      │
└────────────┬─────────────────────────────────────────┬───────────┘
             │                                         │
             ├─ MySQL (triggers + schedules)           │
             └─ Kafka Topic (trigger-events) ──────────┘
                         │
        ┌────────────────┼────────────────┐
        │                │                │
   ┌────▼────┐      ┌───▼─────┐     ┌───▼──────┐
   │Consumer │      │Consumer │     │Consumer  │  (External)
   │   #1    │      │   #2    │     │   #N     │  (User-owned)
   └────┬────┘      └────┬────┘     └────┬─────┘
        │                │                │
        └────────────────┼────────────────┘
                         │
                  Execute Business Logic
                  (HTTP POST, Email, etc.)

┌─────────────────────────────────────────────────────────────────┐
│                    Scheduler Service (Go)                        │
│  • Polls every 5s     • Fires triggers    • Creates next runs   │
│  • Status validation  • Kafka publish     • Marks completed     │
└─────────────────────────────────────────────────────────────────┘
```

### Design Philosophy

**Event Publisher, Not Executor**: This platform publishes trigger events to Kafka. External consumers (owned by users) subscribe and execute business logic. This enables:

- **Horizontal Scaling**: Run N consumers in any language
- **Flexibility**: Each consumer implements different execution logic
- **Separation of Concerns**: Platform manages scheduling, users manage execution
- **Multi-tenancy**: Different consumers for different trigger types

## Reliability Guarantees

- At-least-once scheduling semantics for time/cron triggers.
- Schedules move `pending → processing → completed` only after a successful Kafka publish.
- On publish failure (e.g., Kafka unavailable):
  - The schedule is reverted to `pending`, `attempt_count` is incremented, and it is retried on the next poll.
  - After max retries, the schedule is marked `cancelled` for operator visibility; the event is not lost silently.
- Webhook endpoint validates payloads against stored JSON Schema and publishes to Kafka on success.
- Webhook requests for unknown trigger IDs return 404 (not 500).

Kafka topic used: `trigger-events` (auto-created in local Compose).

### Kafka Message Schema

Messages published to Kafka topic `trigger-events` use this JSON shape:

```json
{
  "event_id": "<uuid>",
  "trigger_id": "<uuid>",
  "type": "webhook|time_scheduled|cron_scheduled",
  "payload": {"...": "..."},
  "fired_at": "2025-11-06T10:30:00Z",
  "source": "webhook|scheduler|manual-test"
}
```

Note: Endpoint/headers are stored in the trigger config and are not embedded in the Kafka message. Consumers that need these details should call the API (`GET /api/v1/triggers/:id`) to fetch the trigger configuration.

## External Consumer Guide

Below is a minimal Go consumer using `segmentio/kafka-go`:

```go
package main

import (
  "context"
  "encoding/json"
  "log"
  "time"
  "github.com/segmentio/kafka-go"
)

type TriggerEvent struct {
  EventID   string                 `json:"event_id"`
  TriggerID string                 `json:"trigger_id"`
  Type      string                 `json:"type"`
  Payload   map[string]interface{} `json:"payload"`
  FiredAt   time.Time              `json:"fired_at"`
  Source    string                 `json:"source"`
}

func main() {
  r := kafka.NewReader(kafka.ReaderConfig{
    Brokers:  []string{"localhost:9092"},
    Topic:    "trigger-events",
    GroupID:  "example-consumer",
    MinBytes: 1,
    MaxBytes: 10e6,
  })
  defer r.Close()

  for {
    msg, err := r.ReadMessage(context.Background())
    if err != nil { log.Fatal(err) }
    var ev TriggerEvent
    if err := json.Unmarshal(msg.Value, &ev); err != nil { log.Println("bad message:", err); continue }
    log.Printf("event %s for trigger %s type=%s source=%s", ev.EventID, ev.TriggerID, ev.Type, ev.Source)
    // If needed, fetch trigger config from API: GET /api/v1/triggers/{ev.TriggerID}
  }
}
```

Any language can be used; rely on consumer groups for horizontal scaling. Implement your own retry/deduplication at the consumer if needed.

## Quick Start

### Prerequisites

- **Docker** and **Docker Compose**
- **Go** 1.22+ (for local development)

### Run with Docker Compose (Recommended)

```bash
# Clone the repository
git clone https://github.com/Dhi13man/event-trigger-platform.git
cd event-trigger-platform

# Start all services (MySQL, Kafka, API, Scheduler)
cd deploy
docker-compose up -d

# Check health
curl http://localhost:8080/health

# View Swagger documentation
open http://localhost:8080/swagger/index.html
```

**That's it!** The platform is now running with:

- API Server: <http://localhost:8080>
- MySQL: localhost:3306
- Kafka: localhost:9092
- Swagger UI: <http://localhost:8080/swagger/index.html>

### Quick Test

```bash
# Create a webhook trigger
curl -X POST http://localhost:8080/api/v1/triggers \
  -H "Content-Type: application/json" \
  -d '{
    "name": "User Action Webhook",
    "type": "webhook",
    "config": {
      "endpoint": "https://webhook.site/your-unique-id",
      "http_method": "POST",
      "schema": {
        "type": "object",
        "properties": {
          "user_id": {"type": "string"},
          "action": {"type": "string"}
        },
        "required": ["user_id", "action"]
      }
    }
  }'

# Response will include webhook_url:
# {
#   "id": "550e8400-...",
#   "webhook_url": "http://localhost:8080/api/v1/webhook/550e8400-...",
#   ...
# }
```

## API Documentation

### Interactive Documentation

Visit **<http://localhost:8080/swagger/index.html>** for interactive API documentation with try-it-out functionality.

### Core Endpoints

#### Trigger Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/triggers` | Create a new trigger |
| GET | `/api/v1/triggers` | List all triggers (with pagination & filters) |
| GET | `/api/v1/triggers/:id` | Get trigger details |
| PUT | `/api/v1/triggers/:id` | Update trigger |
| DELETE | `/api/v1/triggers/:id` | Delete trigger |
| POST | `/api/v1/triggers/:id/test` | Manual test execution |

#### Event Logs

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/events` | List event logs (filter by status, source, trigger) |
| GET | `/api/v1/events/:id` | Get event log details |

#### Webhook Receiver

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/webhook/:trigger_id` | Receive webhook payload |

Status codes:

- 202 Accepted: payload validated and enqueued
- 400 Bad Request: invalid JSON or schema validation errors
- 404 Not Found: unknown or deleted trigger ID
- 500 Internal Error: server/DB issues

#### System

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check (DB, Kafka connectivity) |
| GET | `/metrics` | Prometheus-compatible metrics |

### API Examples

#### 1. Create a Time-Scheduled Trigger

Fire once at a specific time (e.g., Christmas greeting):

```bash
curl -X POST http://localhost:8080/api/v1/triggers \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Christmas Greeting 2025",
    "type": "time_scheduled",
    "config": {
      "run_at": "2025-12-25T00:00:00Z",
      "timezone": "America/New_York",
      "endpoint": "https://api.example.com/greetings",
      "http_method": "POST",
      "headers": {
        "Authorization": "Bearer token123"
      },
      "payload": {
        "message": "Merry Christmas!",
        "year": 2025
      }
    }
  }'
```

**Response:**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Christmas Greeting 2025",
  "type": "time_scheduled",
  "status": "active",
  "config": { ... },
  "next_scheduled_run": "2025-12-25T00:00:00Z",
  "created_at": "2025-11-06T10:00:00Z",
  "updated_at": "2025-11-06T10:00:00Z"
}
```

#### 2. Create a CRON-Scheduled Trigger

Recurring trigger (e.g., daily report at 9 AM):

```bash
curl -X POST http://localhost:8080/api/v1/triggers \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Daily Morning Report",
    "type": "cron_scheduled",
    "config": {
      "cron": "0 9 * * *",
      "timezone": "America/New_York",
      "endpoint": "https://api.example.com/reports/daily",
      "http_method": "POST",
      "headers": {
        "Authorization": "Bearer token123",
        "X-Report-Type": "daily"
      },
      "payload": {
        "report_type": "daily",
        "format": "json"
      }
    }
  }'
```

**Common CRON Expressions:**

- `* * * * *` - Every minute
- `0 * * * *` - Every hour
- `0 9 * * *` - Daily at 9:00 AM
- `0 9 * * 1` - Every Monday at 9:00 AM
- `0 0 1 * *` - Monthly on the 1st at midnight
- `*/5 * * * *` - Every 5 minutes

#### 3. Create a Webhook Trigger

Event-driven trigger with JSON schema validation:

```bash
curl -X POST http://localhost:8080/api/v1/triggers \
  -H "Content-Type: application/json" \
  -d '{
    "name": "User Action Webhook",
    "type": "webhook",
    "config": {
      "schema": {
        "type": "object",
        "properties": {
          "user_id": {"type": "string"},
          "action": {"type": "string", "enum": ["create", "update", "delete"]},
          "timestamp": {"type": "string", "format": "date-time"}
        },
        "required": ["user_id", "action"]
      },
      "endpoint": "https://api.example.com/user-actions",
      "http_method": "POST",
      "headers": {
        "Authorization": "Bearer token123",
        "X-Custom-Header": "value"
      }
    }
  }'
```

**Response includes webhook URL:**

```json
{
  "id": "abc123...",
  "name": "User Action Webhook",
  "type": "webhook",
  "status": "active",
  "webhook_url": "http://localhost:8080/api/v1/webhook/abc123...",
  ...
}
```

**Fire the webhook:**

```bash
curl -X POST http://localhost:8080/api/v1/webhook/abc123... \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_12345",
    "action": "create",
    "timestamp": "2025-11-06T10:30:00Z"
  }'
```

#### 4. List Triggers with Filters

```bash
# List all active CRON triggers (page 1, 20 items)
curl "http://localhost:8080/api/v1/triggers?type=cron_scheduled&status=active&page=1&limit=20"

# List all webhook triggers
curl "http://localhost:8080/api/v1/triggers?type=webhook"
```

**Response:**

```json
{
  "triggers": [
    {
      "id": "550e8400-...",
      "name": "Daily Report",
      "type": "cron_scheduled",
      "status": "active",
      "next_scheduled_run": "2025-11-07T09:00:00Z",
      ...
    }
  ],
  "pagination": {
    "current_page": 1,
    "page_size": 20,
    "total_pages": 3,
    "total_records": 55
  }
}
```

#### 5. Update Trigger

```bash
# Pause a trigger (keep schedules, won't fire while inactive)
curl -X PUT http://localhost:8080/api/v1/triggers/550e8400-... \
  -H "Content-Type: application/json" \
  -d '{
    "status": "inactive"
  }'

# Update CRON expression (cancels old schedules, creates new ones)
curl -X PUT http://localhost:8080/api/v1/triggers/550e8400-... \
  -H "Content-Type: application/json" \
  -d '{
    "config": {
      "cron": "0 */2 * * *",
      "timezone": "America/New_York",
      "endpoint": "https://api.example.com/reports/daily",
      "http_method": "POST"
    }
  }'
```

#### 6. Query Event Logs

```bash
# List active event logs for a specific trigger
curl "http://localhost:8080/api/v1/events?trigger_id=550e8400-...&retention_status=active"

# List all successful events from scheduler
curl "http://localhost:8080/api/v1/events?source=scheduler&execution_status=success"

# List archived events (2-48 hours old)
curl "http://localhost:8080/api/v1/events?retention_status=archived&page=1&limit=50"
```

#### 7. Manual Test Run

```bash
# Fire a trigger immediately for testing
curl -X POST http://localhost:8080/api/v1/triggers/550e8400-.../test

# Event log will have is_test_run=true
```

#### 8. Health Check & Metrics

```bash
# Check system health
curl http://localhost:8080/health

# Response:
# {
#   "status": "healthy",
#   "timestamp": "2025-11-06T10:30:00Z",
#   "checks": {
#     "database": "ok",
#     "kafka": "ok"
#   }
# }

# Get metrics
curl http://localhost:8080/metrics
```

## Configuration

### Environment Variables

All configuration is managed via environment variables (12-factor app pattern):

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `DATABASE_URL` | MySQL connection string | - | ✅ |
| `KAFKA_BROKERS` | Kafka broker addresses | `localhost:9092` | ✅ |
| `API_PORT` | API server port | `8080` | ❌ |
| `LOG_LEVEL` | Logging level (debug, info, warn, error) | `info` | ❌ |
| `ENVIRONMENT` | Environment (development, production) | `development` | ❌ |
| `SCHEDULER_INTERVAL` | Scheduler polling interval | `5s` | ❌ |
| `CORS_ORIGINS` | Allowed CORS origins (comma-separated) | `*` | ❌ |

See `deploy/.env.example` for a working Compose setup and defaults that run locally.
The Compose file exposes Kafka on `localhost:9092` and the API at `localhost:8080`.

### Example `.env` File

```bash
# MySQL Configuration
DATABASE_URL=user:password@tcp(mysql:3306)/event_trigger?parseTime=true&loc=UTC

# Kafka Configuration
KAFKA_BROKERS=kafka:29092

# API Server
API_PORT=8080
LOG_LEVEL=info
ENVIRONMENT=production

# Scheduler
SCHEDULER_INTERVAL=5s

# CORS
CORS_ORIGINS=https://app.example.com,https://admin.example.com
```

## Database Schema

### Core Tables

#### `triggers`

Stores trigger definitions and configuration.

```sql
CREATE TABLE triggers (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type ENUM('webhook', 'time_scheduled', 'cron_scheduled') NOT NULL,
    status ENUM('active', 'inactive') NOT NULL DEFAULT 'active',
    config JSON NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_status (status),
    INDEX idx_type (type)
);
```

#### `trigger_schedules`

Stores pending and completed schedule entries.

```sql
CREATE TABLE trigger_schedules (
    id VARCHAR(36) PRIMARY KEY,
    trigger_id VARCHAR(36) NOT NULL,
    fire_at DATETIME NOT NULL,
    status ENUM('pending', 'processing', 'completed', 'cancelled') NOT NULL DEFAULT 'pending',
    attempt_count INT NOT NULL DEFAULT 0,
    last_attempt_at DATETIME NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_fire_at_status (fire_at, status),
    INDEX idx_trigger_id (trigger_id),
    FOREIGN KEY (trigger_id) REFERENCES triggers(id) ON DELETE CASCADE
);
```

#### `event_logs`

Stores history of fired events with retention lifecycle.

```sql
CREATE TABLE event_logs (
    id VARCHAR(36) PRIMARY KEY,
    trigger_id VARCHAR(36) NULL,
    trigger_type ENUM('webhook', 'time_scheduled', 'cron_scheduled') NOT NULL,
    fired_at DATETIME NOT NULL,
    payload JSON NULL,
    source ENUM('webhook', 'scheduler', 'manual-test') NOT NULL,
    execution_status ENUM('success', 'failure') NOT NULL DEFAULT 'success',
    error_message TEXT NULL,
    retention_status ENUM('active', 'archived', 'deleted') NOT NULL DEFAULT 'active',
    is_test_run BOOLEAN NOT NULL DEFAULT FALSE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_fired_at (fired_at),
    INDEX idx_trigger_id (trigger_id),
    INDEX idx_retention_status (retention_status),
    FOREIGN KEY (trigger_id) REFERENCES triggers(id) ON DELETE SET NULL
);
```

### Retention Lifecycle

Event logs automatically transition through states:

- **0-2 hours**: `active` (default view)
- **2-48 hours**: `archived` (retrievable via `?retention_status=archived`)
- **48+ hours**: Permanently deleted

Managed by MySQL Event Scheduler (see `db/migrations/004_setup_retention_events.sql`).

## Troubleshooting

- Kafka unavailable during publish
  - Symptom: scheduler logs publish errors; schedules stay in `pending` with growing `attempt_count`.
  - Action: restore Kafka; the scheduler will retry automatically on the next tick.
- Webhook returns 404
  - Symptom: calling `/api/v1/webhook/:trigger_id` with an unknown ID.
  - Action: verify the trigger exists and is `webhook` type and `active`.
- Webhook returns 400
  - Symptom: invalid JSON or schema validation errors.
  - Action: correct payload per the stored JSON Schema in the trigger config.
- Retention not running
  - Symptom: old events never archive/delete.
  - Action: ensure MySQL Event Scheduler is ON and the retention events exist; see `db/migrations/004_setup_retention_events.sql`.
- Port collisions
  - Symptom: Compose fails to start.
  - Action: adjust `API_PORT`, `KAFKA_EXTERNAL_PORT`, or `MYSQL_PORT` in `deploy/.env`.

## Development

### Local Setup (Without Docker)

```bash
# Install dependencies
go mod download

# Start MySQL and Kafka (via Docker Compose)
cd deploy
docker-compose up -d mysql kafka

# Run database migrations
mysql -h localhost -u appuser -p event_trigger < ../db/migrations/*.sql

# Start API server
cd ..
export DATABASE_URL="appuser:apppassword@tcp(localhost:3306)/event_trigger?parseTime=true"
export KAFKA_BROKERS="localhost:9092"
go run ./cmd/api

# In another terminal, start scheduler
go run ./cmd/scheduler
```

### Generate Swagger Docs

```bash
# Install swag
go install github.com/swaggo/swag/cmd/swag@latest

# Generate docs
swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal
```

### Build Binaries

```bash
# Build all services
make build

# Or build individually
go build -o bin/api ./cmd/api
go build -o bin/scheduler ./cmd/scheduler
```

### Project Structure

```plain
event-trigger-platform/
├── cmd/
│   ├── api/              # API server entrypoint
│   └── scheduler/        # Scheduler entrypoint
├── internal/
│   ├── api/              # HTTP handlers, middleware, server
│   ├── events/           # Event log repository
│   ├── logging/          # Structured logger
│   ├── models/           # Data models and DTOs
│   ├── scheduler/        # Scheduling engine
│   ├── storage/          # MySQL client and repositories
│   └── triggers/         # Trigger service and business logic
├── platform/
│   └── events/           # Kafka publisher
├── pkg/
│   └── config/           # Configuration loading
├── db/
│   ├── migrations/       # SQL migration files
│   └── fixtures/         # Test data (optional)
├── deploy/
│   ├── docker-compose.yml
│   ├── .env
│   └── README.md
├── docs/                 # Swagger generated docs
├── Dockerfile
├── Makefile
├── go.mod
├── CLAUDE.md            # AI agent instructions
└── README.md
```

## Testing

### Run Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./internal/triggers/...

# Run integration tests (when implemented)
go test -tags=integration ./...
```

### Test Categories (Planned)

1. **Create + Fire Webhook Trigger**: Validate payload, fire webhook
2. **Time-Scheduled Accuracy**: Fire within ±10 seconds
3. **CRON Recurring**: Create next schedule after fire
4. **Manual Test Runs**: Test execution without affecting schedules
5. **Retention Lifecycle**: Active → archived → deleted transitions
6. **Concurrent Firing**: Race condition and idempotency checks

## Production Deployment

### Docker Compose (Recommended for Small-Medium Scale)

```bash
cd deploy
docker-compose up -d

# Scale API servers
docker-compose up -d --scale api=3

# View logs
docker-compose logs -f api scheduler

# Stop all services
docker-compose down
```

### Kubernetes (Large Scale)

Coming soon: Helm charts and K8s manifests for production deployment.

### Environment Considerations

#### Dev

- Single API instance
- Single scheduler instance
- Local MySQL and Kafka
- Debug logging

#### Production

- Multiple API instances (load balanced)
- Single scheduler instance (use distributed lock for HA in future)
- Managed MySQL (RDS, Cloud SQL) with read replicas
- Managed Kafka (MSK, Confluent Cloud) with replication factor ≥ 2
- Structured logging (JSON format)
- Metrics collection (Prometheus/Grafana)

### Monitoring & Observability

**Metrics Exposed** (via `/metrics`):

- `trigger_counts_by_type` - Active triggers by type
- `events_active_count` - Events in active state
- `events_archived_count` - Events in archived state
- `published_events_count` - Total events published to Kafka
- `trigger_fire_latency_p50/p95/p99` - Timing accuracy (optional)

**Logging**:

- Structured JSON logs with request IDs
- Configurable log levels (debug, info, warn, error)
- Request/response logging via middleware

**Health Checks**:

- Database connectivity
- Kafka connectivity
- Service readiness

## Future Scope & Scalability

### Performance Optimizations

#### 1. Redis Caching for Scheduler Reads

**Current State**: Scheduler uses JOIN queries (`trigger_schedules` + `triggers` tables) every 5 seconds.

**Optimization**: Implement Redis cache-aside pattern for trigger configs.

```plain
┌─────────────┐
│  Scheduler  │
└──────┬──────┘
       │
       ├─ Check Redis cache (trigger:{id})
       │  └─ Hit: Use cached config
       │  └─ Miss: Query MySQL + cache result (TTL: 5-10min)
       │
       └─ On UPDATE/DELETE: Invalidate cache
```

**Benefits**:

- Reduced database load
- Faster scheduler polling
- Better horizontal scaling

**When to Implement**: When scheduler database queries become a bottleneck (monitor p95/p99 latencies).

**Alternative**: Use MySQL read replicas before adding cache complexity.

#### 2. Distributed Scheduler (High Availability)

**Current**: Single scheduler instance.

**Future**: Multiple scheduler instances with distributed locking (Redis, etcd, or database-based).

```go
// Pseudo-code
for {
    lock, err := acquireDistributedLock("scheduler-lock", 10*time.Second)
    if err != nil {
        continue
    }
    defer lock.Release()

    // Poll and process schedules
    processSchedules()
}
```

#### 3. Horizontal Scaling

**API Server**: ✅ Already stateless, ready for horizontal scaling behind load balancer.

**Scheduler**: ⚠️ Currently single instance. For HA, use distributed locking.

**External Consumers**: ✅ Kafka consumer groups enable automatic load balancing.

#### 4. Database Optimizations

- **Read Replicas**: Route scheduler queries to read replicas
- **Partitioning**: Partition `event_logs` by `fired_at` for faster archival
- **Connection Pooling**: Tune `max_open_conns` and `max_idle_conns` based on load
- **Query Optimization**: Add covering indexes for hot queries

### Feature Roadmap

#### Phase 1: Core Functionality (Current)

- CRUD API for triggers
- Three trigger types (webhook, time_scheduled, cron_scheduled)
- Atomic schedule management
- Swagger documentation
- Scheduler polling and event publishing
- Webhook receiver with JSON schema validation
- Event log querying

#### Phase 2: Reliability & Observability

- Retry mechanisms with exponential backoff
- Circuit breaker for failing endpoints
- Distributed tracing (OpenTelemetry)
- Prometheus metrics + Grafana dashboards
- Alerting rules (PagerDuty, Slack)
- Structured logging to centralized store (ELK, Datadog)

#### Phase 3: Advanced Features

- **Priority Queues**: Urgent triggers fire first
- **Trigger Tagging**: Organize triggers by team/project
- **Webhook Retries**: Configurable retry policy per trigger
- **Rate Limiting**: Prevent abuse (per-user, per-IP)
- **Authentication**: API key or JWT-based auth
- **RBAC**: Role-based access control for multi-tenant use
- **Audit Logs**: Track who created/modified triggers
- **Trigger Dependencies**: Chain triggers together

#### Phase 4: Enterprise Features

- **Multi-Region Deployment**: Active-active across regions
- **Data Residency**: Store data in specific regions (GDPR)
- **Advanced Monitoring**: Anomaly detection, SLO tracking
- **Cost Tracking**: Per-trigger execution costs
- **A/B Testing**: Test trigger configs before rollout
- **Trigger Marketplace**: Pre-built trigger templates

### Scalability Benchmarks (Target)

| Metric | Small Scale | Medium Scale | Large Scale |
|--------|-------------|--------------|-------------|
| Active Triggers | < 1,000 | 1,000 - 10,000 | 10,000+ |
| Events/Second | < 100 | 100 - 1,000 | 1,000+ |
| API Instances | 1-2 | 3-5 | 10+ (auto-scaled) |
| Scheduler Instances | 1 | 1-2 (with locking) | 3+ (with locking) |
| Database | Single MySQL | Read replicas | Sharding + replicas |
| Kafka Partitions | 3 | 6 | 12+ |

## Contributing

We welcome contributions from the community! Please see our [Contributing Guidelines](CONTRIBUTING.md) for detailed information on:

- Setting up your development environment
- Coding standards and best practices
- Testing requirements
- Pull request process
- Code review guidelines

For quick contributions:

1. **Fork** the repository
2. **Create** a feature branch (`git checkout -b feature/amazing-feature`)
3. **Commit** with clear messages (`git commit -m 'Add amazing feature'`)
4. **Push** to the branch (`git push origin feature/amazing-feature`)
5. **Open** a Pull Request

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for complete guidelines.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Gin](https://github.com/gin-gonic/gin) - HTTP web framework
- [robfig/cron](https://github.com/robfig/cron) - CRON expression parsing
- [segmentio/kafka-go](https://github.com/segmentio/kafka-go) - Kafka client
- [swaggo/swag](https://github.com/swaggo/swag) - Swagger documentation
- [zap](https://github.com/uber-go/zap) - Structured logging

## Support

- **Issues**: [GitHub Issues](https://github.com/Dhi13man/event-trigger-platform/issues)
- **Swagger UI**: <http://localhost:8080/swagger/index.html> (when running)

> Built with Go, MySQL, and Kafka
