# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is an **Event Trigger Platform** - a production-quality, containerized system for creating, managing, and running triggers. The platform supports three trigger types:

1. **Webhook Triggers**: Event-driven triggers that fire when external systems POST to the webhook URL with valid JSON payload
2. **Time-Scheduled Triggers**: One-time triggers that fire at a specific absolute time (ISO 8601 format)
3. **CRON-Scheduled Triggers**: Recurring triggers that fire based on CRON expressions (e.g., daily, hourly, every 5 minutes)

## Technology Stack

- **Language**: Golang
- **Database**: MySQL (for triggers and event logs)
- **Queue**: Kafka (for reliable job processing)
- **CRON**: Go library for recurring schedules
- **Deployment**: Docker Compose (local infrastructure only)

## Architecture Overview

**Platform Components** (what we build):

```plain
┌─────────────┐
│  API Server │ (Gin framework)
│  (Golang)   │
└──────┬──────┘
       │
       ├─ CRUD → MySQL (triggers + trigger_schedules tables)
       │
       ├─ Fire Trigger → Kafka Topic ("trigger-events")
       │
       └─ Query Events → MySQL (event_logs table)
┌──────────────┐
│  Scheduler   │ (Goroutine with ticker)
│  (Golang)    │
└──────┬───────┘
       │
       ├─ Poll MySQL every 5s (trigger_schedules table)
       │    WHERE fire_at <= NOW() AND status = 'pending'
       │
       ├─ Fire trigger → Publish to Kafka
       │
       ├─ Create next schedule entry (for CRON triggers)
       │
       └─ Update schedule status → 'completed'
┌─────────────────┐
│   MySQL Event   │ (Built-in Event Scheduler)
│   Scheduler     │
└──────┬──────────┘
       │
       ├─ Archive events after 2 hours
       │
       └─ Delete events after 48 hours
```

**Consumer Integration** (external, horizontally scalable):

```plain
┌───────────────────────────────────────────────────┐
│            Kafka Topic: "trigger-events"          │
│  (Event logs published by API Server + Scheduler) │
└────────────────────┬──────────────────────────────┘
                     │
        ┌────────────┼────────────┐
        │            │            │
   ┌────▼────┐  ┌───▼─────┐  ┌──▼──────┐
   │Consumer │  │Consumer │  │Consumer │  (User-managed)
   │   #1    │  │   #2    │  │   #N    │  (Any language/framework)
   └────┬────┘  └────┬────┘  └────┬────┘
        │            │            │
        └────────────┼────────────┘
                     │
              ┌──────▼──────┐
              │ Execute:    │
              │ • HTTP POST │
              │ • Email     │
              │ • Webhook   │
              │ • Any logic │
              └─────────────┘
```

**Key Architectural Decision**: This platform is an **event publisher**, not an executor. We publish trigger events to Kafka, and external consumers (owned by users) subscribe and execute business logic. This enables:

- **Horizontal Scaling**: Users can run N consumers in any language
- **Flexibility**: Each consumer can implement different execution logic
- **Separation of Concerns**: Platform manages scheduling, users manage execution
- **Multi-tenancy**: Different consumers can handle different trigger types

## Database Schema

### 1. `triggers` Table

Stores trigger definitions and configuration. This table holds **context only** - no scheduling information.

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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

**Config JSON Schema** (varies by type):

- **Webhook**: `schema` (JSON schema for validation), `endpoint`, `http_method`, `headers`
- **Time-Scheduled**: `run_at` (ISO 8601 timestamp), `endpoint`, `http_method`, `headers`, `payload`, `timezone`
- **CRON-Scheduled**: `cron` (CRON expression), `endpoint`, `http_method`, `headers`, `payload`, `timezone`

### 2. `trigger_schedules` Table

Stores **pending and completed schedule entries** for time-scheduled and CRON-scheduled triggers. This table is queried by the Scheduler service.

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
    INDEX idx_status (status),
    FOREIGN KEY (trigger_id) REFERENCES triggers(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

**Key Design Decisions**:

- **Webhook triggers**: Do NOT create entries in `trigger_schedules` (event-driven, not time-based)
- **Time-scheduled triggers**: Create **one** entry on trigger creation, status becomes 'completed' after firing
- **CRON-scheduled triggers**: Create **first** entry on trigger creation, create **next** entry after each fire while trigger is active
- **Scheduler queries**: `SELECT * FROM trigger_schedules WHERE fire_at <= NOW() AND status = 'pending' ORDER BY fire_at ASC`
- **Status flow**: `pending` → `processing` → `completed` (or `cancelled` if trigger is deactivated)

### 3. `event_logs` Table

Stores **history of fired events**. Every time a trigger fires (scheduled or webhook), an event log entry is created.

```sql
CREATE TABLE event_logs (
    id VARCHAR(36) PRIMARY KEY,
    trigger_id VARCHAR(36) NULL,  -- NULL for manual/test runs without persisted trigger
    trigger_type ENUM('webhook', 'time_scheduled', 'cron_scheduled') NOT NULL,
    fired_at DATETIME NOT NULL,
    payload JSON NULL,
    source ENUM('webhook', 'scheduler', 'manual-test') NOT NULL,
    execution_status ENUM('success', 'failure') NOT NULL DEFAULT 'success',
    error_message TEXT NULL,  -- Populated on failure
    retention_status ENUM('active', 'archived', 'deleted') NOT NULL DEFAULT 'active',
    is_test_run BOOLEAN NOT NULL DEFAULT FALSE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_fired_at (fired_at),
    INDEX idx_trigger_id (trigger_id),
    INDEX idx_retention_status (retention_status),
    INDEX idx_execution_status (execution_status),
    FOREIGN KEY (trigger_id) REFERENCES triggers(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

**Important Schema Notes**:

- `trigger_id` is **NULL** for pure manual/test runs that aren't persisted as triggers
- `execution_status` tracks whether the trigger execution succeeded or failed (platform-level: did we publish to Kafka?)
- `retention_status` tracks the lifecycle state (active/archived/deleted) - separate from execution status
- `error_message` stores failure details for debugging
- Foreign key uses `ON DELETE SET NULL` so deleting triggers doesn't delete event logs (history preserved)
- **Source values**:
  - `webhook` - Fired via webhook POST
  - `scheduler` - Fired by scheduler service (time or CRON)
  - `manual-test` - Fired via test API endpoint

### 4. `idempotency_keys` Table (Optional)

**Note**: This table is primarily for external consumer implementations. The platform itself writes event logs when publishing to Kafka, so platform-level idempotency is handled by transaction atomicity. However, if consumers need to track processed events, they can use this table pattern.

```sql
CREATE TABLE idempotency_keys (
    job_id VARCHAR(36) PRIMARY KEY,
    event_id VARCHAR(36) NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

External consumers can use this table to prevent duplicate processing when Kafka redelivers messages.

## Trigger Lifecycle Flows

### Webhook Trigger Flow

```plain
User creates webhook trigger
     ↓
1. INSERT INTO triggers (name, type='webhook', config, status='active')
2. Generate webhook_url: http://api.domain.com/api/v1/webhook/{trigger_id}
3. Return webhook_url to user
4. NO entries in trigger_schedules (event-driven, not scheduled)
     ↓
External system POSTs to webhook_url with JSON payload
     ↓
5. Validate payload against trigger.config.schema (JSON schema validation)
6. If valid:
   - INSERT INTO event_logs (trigger_id, source='webhook', ...)
   - Publish event to Kafka topic "trigger-events"
   - Return 202 Accepted
7. If invalid:
   - Return 400 Bad Request with validation errors
   - NO event log, NO Kafka publish
```

### Time-Scheduled Trigger Flow

```plain
User creates time_scheduled trigger (run_at: 2025-12-25T00:00:00Z)
     ↓
1. Parse config.run_at timestamp
2. INSERT INTO triggers (name, type='time_scheduled', config, status='active')
3. INSERT INTO trigger_schedules (trigger_id, fire_at=run_at, status='pending')
     ↓
Scheduler polls trigger_schedules every 5 seconds
     ↓
4. Finds schedule entry with fire_at <= NOW() AND status='pending'
5. UPDATE trigger_schedules SET status='processing' WHERE id=?
6. Fetch trigger context: SELECT * FROM triggers WHERE id=trigger_id
7. INSERT INTO event_logs (trigger_id, source='scheduler', ...)
8. Publish event to Kafka topic "trigger-events"
9. UPDATE trigger_schedules SET status='completed' WHERE id=?
10. UPDATE triggers SET status='inactive' WHERE id=? (one-time trigger is done)
```

### CRON-Scheduled Trigger Flow

```plain
User creates cron_scheduled trigger (cron: "0 9 * * *" - daily at 9am)
     ↓
1. Parse config.cron expression
2. Calculate first fire time using CRON library
3. INSERT INTO triggers (name, type='cron_scheduled', config, status='active')
4. INSERT INTO trigger_schedules (trigger_id, fire_at=first_fire_time, status='pending')
     ↓
Scheduler polls trigger_schedules every 5 seconds
     ↓
5. Finds schedule entry with fire_at <= NOW() AND status='pending'
6. UPDATE trigger_schedules SET status='processing' WHERE id=?
7. Fetch trigger context: SELECT * FROM triggers WHERE id=trigger_id
8. INSERT INTO event_logs (trigger_id, source='scheduler', ...)
9. Publish event to Kafka topic "trigger-events"
10. UPDATE trigger_schedules SET status='completed' WHERE id=?
11. IF trigger.status == 'active':
    - Calculate next fire time using CRON library
    - INSERT INTO trigger_schedules (trigger_id, fire_at=next_fire_time, status='pending')
12. ELSE (trigger deactivated):
    - Do NOT create next schedule entry
```

## Config JSON Examples

### Webhook Trigger Config

```json
{
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
```

### Time-Scheduled Trigger Config

```json
{
  "run_at": "2025-12-25T00:00:00Z",
  "endpoint": "https://api.example.com/christmas-greeting",
  "http_method": "POST",
  "headers": {
    "Authorization": "Bearer token123"
  },
  "payload": {
    "message": "Merry Christmas!",
    "year": 2025
  },
  "timezone": "America/New_York"
}
```

### CRON-Scheduled Trigger Config

```json
{
  "cron": "0 9 * * *",
  "timezone": "America/New_York",
  "endpoint": "https://api.example.com/daily-report",
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
```

**Common CRON Expressions:**

- `* * * * *` - Every minute
- `0 * * * *` - Every hour
- `0 9 * * *` - Daily at 9:00 AM
- `0 9 * * 1` - Every Monday at 9:00 AM
- `0 0 1 * *` - Monthly on the 1st at midnight
- `*/5 * * * *` - Every 5 minutes

## Core Components

### 1. API Server (Web API Layer)

**Responsibilities**:

- RESTful CRUD for triggers (all types: webhook, time_scheduled, cron_scheduled)
- Webhook receiver endpoint for webhook-type triggers
- Event log queries with filtering
- Test execution for manual trigger fires
**Key Endpoints**:

```plain
POST   /api/v1/triggers             - Create trigger (any type)
GET    /api/v1/triggers             - List all triggers (with type filter)
GET    /api/v1/triggers/{id}        - Get trigger details
PUT    /api/v1/triggers/{id}        - Update trigger (name, status, config)
DELETE /api/v1/triggers/{id}        - Delete trigger (CASCADE deletes schedules)
POST   /api/v1/triggers/{id}/test   - Test trigger (manual fire, is_test_run=true)
GET    /api/v1/events               - List event logs (filter by active/archived)
GET    /api/v1/events/{id}          - Get event log details
POST   /api/v1/webhook/{trigger_id} - Webhook endpoint for webhook-type triggers
```

**Critical Implementation Details**:

- **JSON schema validation** for webhook triggers using `github.com/xeipuuv/gojsonschema`
- **Schedule creation**: When creating time_scheduled or cron_scheduled triggers, create first entry in `trigger_schedules`
- **Webhook URL generation**: Return webhook URL in response for webhook-type triggers
- **Test mode**: `POST /triggers/{id}/test` creates event_log with `is_test_run=true`, publishes to Kafka, but doesn't affect schedules
- **Update handling**: If CRON expression changes, cancel pending schedules and create new ones

### 2. Scheduler

**Responsibilities**:

- Poll `trigger_schedules` table every 5 seconds for due schedule entries
- Fetch trigger context from `triggers` table
- Publish events to Kafka
- Create event logs
- Update schedule status and create next schedule for CRON triggers
- Deactivate one-time triggers after firing
**Key Logic**:

```go
// Pseudo-code
for {
    // Query due schedules
    schedules := db.Query(`
        SELECT ts.*, t.*
        FROM trigger_schedules ts
        JOIN triggers t ON ts.trigger_id = t.id
        WHERE ts.fire_at <= NOW()
        AND ts.status = 'pending'
        AND t.status = 'active'
        ORDER BY ts.fire_at ASC
        LIMIT 100
    `)
    for schedule := range schedules {
        // Start transaction
        tx.Begin()
        // Lock schedule row
        tx.Exec("UPDATE trigger_schedules SET status='processing' WHERE id=? AND status='pending'", schedule.ID)
        // Create event log
        eventLog := EventLog{
            ID: generateUUID(),
            TriggerID: schedule.TriggerID,
            TriggerType: schedule.Trigger.Type,
            FiredAt: time.Now(),
            Payload: schedule.Trigger.Config,
            Source: "scheduler",
            ExecutionStatus: "success",
        }
        tx.Exec("INSERT INTO event_logs (...) VALUES (...)", eventLog)
        // Publish to Kafka (outside transaction for at-least-once semantics)
        kafka.Publish("trigger-events", eventLog)
        // Mark schedule as completed
        tx.Exec("UPDATE trigger_schedules SET status='completed' WHERE id=?", schedule.ID)
        // Handle trigger type-specific logic
        if schedule.Trigger.Type == "time_scheduled" {
            // One-time trigger - deactivate
            tx.Exec("UPDATE triggers SET status='inactive' WHERE id=?", schedule.TriggerID)
        } else if schedule.Trigger.Type == "cron_scheduled" && schedule.Trigger.Status == "active" {
            // Recurring trigger - create next schedule
            nextFireTime := calculateNextCRONFireTime(schedule.Trigger.Config.Cron, schedule.Trigger.Config.Timezone)
            tx.Exec("INSERT INTO trigger_schedules (id, trigger_id, fire_at, status) VALUES (?, ?, ?, 'pending')",
                generateUUID(), schedule.TriggerID, nextFireTime)
        }
        tx.Commit()
    }
    time.Sleep(5 * time.Second)
}
```

**Timing Accuracy**: Must fire within ±10 seconds of scheduled time.

- 5-second polling interval ensures max 5s delay
- Database locking with `status='processing'` prevents duplicate fires
- Composite index on `(fire_at, status)` ensures fast queries

### 3. External Consumers (User-Implemented)

**IMPORTANT**: The platform does NOT include a worker service. Users implement their own consumers to process trigger events.
**Why External Consumers?**

- **Flexibility**: Users can implement in any language (Go, Python, Node.js, Java, etc.)
- **Horizontal Scaling**: Users control scaling independently of the platform
- **Separation of Concerns**: Platform = scheduling/publishing, Consumers = execution
- **Multi-tenancy**: Different consumers for different trigger types or business logic
**Consumer Implementation Pattern**:
Users subscribe to the Kafka topic `trigger-events` and consume messages. Example in Go:

```go
// Example Consumer (user-implemented, NOT part of platform)
package main
import (
    "github.com/segmentio/kafka-go"
    "encoding/json"
)
type TriggerEvent struct {
    EventID    string    `json:"event_id"`
    TriggerID  string    `json:"trigger_id"`
    Type       string    `json:"type"`
    Payload    json.RawMessage `json:"payload"`
    FiredAt    time.Time `json:"fired_at"`
}
func main() {
    reader := kafka.NewReader(kafka.ReaderConfig{
        Brokers: []string{"localhost:9092"},
        Topic:   "trigger-events",
        GroupID: "my-consumer-group",
    })
    for {
        msg, err := reader.ReadMessage(context.Background())
        if err != nil {
            log.Fatal(err)
        }
        var event TriggerEvent
        json.Unmarshal(msg.Value, &event)
        // Execute business logic
        executeBusinessLogic(event)
        // Optional: Write custom tracking, metrics, etc.
    }
}
func executeBusinessLogic(event TriggerEvent) {
    // HTTP POST, send email, invoke webhook, etc.
    http.Post(event.Payload.Endpoint, event.Payload.Body)
}
```

**Kafka Topic Schema**:
The platform publishes events to `trigger-events` topic with the following JSON structure:

```json
{
  "event_id": "550e8400-e29b-41d4-a716-446655440000",
  "trigger_id": "123e4567-e89b-12d3-a456-426614174000",
  "type": "scheduled",
  "payload": {
    "endpoint": "https://user-service.com/webhook",
    "body": {"message": "hello"},
    "http_method": "POST"
  },
  "fired_at": "2025-11-05T10:30:00Z",
  "source": "scheduled"
}
```

**Consumer Best Practices**:

- Use Kafka consumer groups for load balancing
- Implement idempotency using `idempotency_keys` table or your own tracking
- Handle retries with exponential backoff
- Monitor consumer lag
- Scale consumers independently (e.g., 5-10 instances)

### 4. Retention Manager

**Responsibilities**:

- Run periodic cleanup every 5-10 minutes
- **Active → Archived**: Move events to 'archived' after 2 hours from fired_at
- **Archived → Deleted**: Permanently delete events 48 hours total from fired_at (not 48 hours after archival!)
**Retention Timeline**:
- **0-2 hours**: Active (readable in default view)
- **2-48 hours**: Archived (retrievable via "archived" view)
- **48+ hours**: Permanently deleted
**Key Logic**:

```go
// Pseudo-code
for {
    // Archive active events older than 2 hours
    db.Exec(`
        UPDATE event_logs
        SET retention_status = 'archived'
        WHERE retention_status = 'active'
        AND fired_at < DATE_SUB(NOW(), INTERVAL 2 HOUR)
    `)
    // Delete events older than 48 hours TOTAL (from fired_at, not from archival!)
    db.Exec(`
        DELETE FROM event_logs
        WHERE fired_at < DATE_SUB(NOW(), INTERVAL 48 HOUR)
    `)
    // Clean up old idempotency keys (optional, e.g., older than 7 days)
    db.Exec(`
        DELETE FROM idempotency_keys
        WHERE created_at < DATE_SUB(NOW(), INTERVAL 7 DAY)
    `)
    time.Sleep(10 * time.Minute)
}
```

## Critical Design Considerations

### Idempotency & Deduplication

**Platform-Level Idempotency**:

- Each trigger fires once per scheduled time
- Event logs are written in atomic transaction when publishing to Kafka
- Duplicate prevention is handled at scheduler level (database locking)
**Consumer-Level Idempotency** (user responsibility):
- Consumers may receive duplicate Kafka messages due to redelivery
- Consumers should implement idempotency using:
  1. `idempotency_keys` table (query before processing)
  2. Unique event_id tracking in their own database
  3. Kafka offset management with consumer groups
Example consumer idempotency check:

```sql
START TRANSACTION;
-- Check if already processed
SELECT 1 FROM idempotency_keys WHERE job_id = ?;
-- If not exists, process and store
INSERT INTO idempotency_keys (job_id, event_id) VALUES (?, ?);
-- Execute business logic
COMMIT;
```

### Timing Accuracy (±10 seconds)

**Requirements**: Triggers must fire within 10 seconds of scheduled time.
**Implementation**:

- Scheduler polls every 5 seconds (half of tolerance window)
- Use `next_fire_time` index for fast queries
- Use optimistic locking or row-level locks to prevent double-firing:

  ```sql
  SELECT * FROM triggers WHERE id = ? FOR UPDATE
  ```

### Durability Guarantees

**Requirements**: No data loss, even if components crash.
**Platform Implementation**:

1. **Kafka**: Set `acks=all` for producer, use replication factor ≥ 2
2. **MySQL**: Use InnoDB with transactions, proper indexes
3. **Atomic Publishing**: Write event_logs and publish to Kafka in transaction
4. **Graceful Shutdown**: Handle SIGTERM in API and Scheduler
**Consumer Implementation** (user responsibility):

- Acknowledge Kafka messages only after successful processing
- Use consumer group with manual offset commits
- Implement error handling and retry logic

### Concurrency Handling

**Platform Concurrency**:

- **Scheduler**: Single instance (multiple instances may cause duplicate fires)
- **API Server**: Horizontally scalable (stateless, behind load balancer)
- **Database Locking**: Use `FOR UPDATE` when scheduler updates trigger state
**Consumer Concurrency** (user managed):
- **Kafka Consumer Groups**: Run N consumer instances in same consumer group
- **Partition-based**: Kafka distributes partitions across consumers
- **Independent Scaling**: Scale consumers based on message throughput
- **Load Balancing**: Kafka handles rebalancing automatically

## API Requirements

### Trigger Management

- `POST /triggers` - Create new trigger (validate schedule/schema)
  - **For API triggers**: Response must include webhook URL (e.g., `https://api.domain.com/webhook/{trigger_id}`)
  - **For scheduled triggers**: Response includes trigger details with next_fire_time
- `GET /triggers` - List all triggers with pagination
- `GET /triggers/{id}` - Get trigger details
- `PUT /triggers/{id}` - Update trigger (revalidate config, affects future firings only)
- `DELETE /triggers/{id}` - Delete trigger (does NOT delete event logs - they follow retention lifecycle)
- `POST /triggers/{id}/test` - Manual/test run (fires once, creates event log with `is_test_run=true`)

### Event Log Queries

- `GET /events` - List events (default: active only; `?status=archived` for archived view)
  - Filter by: `retention_status` (active/archived), `trigger_id`, `execution_status` (success/failure), time range
- `GET /events/{id}` - Get event details with full payload and error_message if failed

### Webhook Receiver

- `POST /webhook/{trigger_id}` - Receive payload, validate against schema, fire trigger
  - Valid payload → queues job, returns 202 Accepted
  - Invalid payload → returns 400 Bad Request with validation errors
  - Unknown trigger → returns 404 Not Found

### Health & Metrics

- `GET /health` - Health check for API server, database, Kafka connectivity
- `GET /metrics` - Expose metrics:
  - `pending_events_count` - Events published to Kafka topic (consumer lag can be tracked externally)
  - `published_events_count` - Total events published to Kafka
  - `events_active_count` - Event logs in active state (last 2 hours)
  - `events_archived_count` - Event logs in archived state (2-48 hours)
  - `trigger_counts_by_type` - Count of scheduled vs API triggers
  - `trigger_fire_latency_p50/p95/p99` - Timing accuracy metrics (optional bonus)

## Testing Requirements

**Framework**: Use any testing framework (e.g., `testing`, `testify`, `ginkgo`)
**Required Automated Tests** (must demonstrate correctness):

### 1. Create + Fire (API Trigger)

- Create API trigger with expected payload structure (JSON schema)
- POST valid payload to webhook URL → assert event log created with correct payload
- POST invalid payload → assert validation error and NO event log created

### 2. Scheduled Trigger Accuracy

- Create one-shot scheduled trigger that fires soon (may use accelerated timing in tests)
- Assert firing occurred within **10 seconds** of scheduled time
- Verify event log created with `execution_status='success'`

### 3. Manual/Test Runs

- Trigger a manual/test scheduled run via `POST /triggers/{id}/test`
- Assert it fires once and is NOT persisted as a new trigger
- Assert event log has `is_test_run=true` and can have `trigger_id=null` for pure test runs

### 4. Retention Lifecycle

- Simulate or accelerate time to verify **active → archived → deleted** transitions
- Option 1: Use test configuration to shorten retention periods (e.g., 10s active, 30s total)
- Option 2: Manipulate `fired_at` timestamps in test database
- Assert event transitions: active (0-2h) → archived (2-48h) → deleted (48h+)

### 5. Idempotency / Duplicate Protection

- Fire the same scheduled trigger multiple times (e.g., parallel scheduler instances)
- Assert **at-most-once** event publishing (only ONE event log created per fire time)
- Verify database locking prevents duplicate trigger fires
- Note: Consumer-side idempotency is not tested (external responsibility)

### 6. Concurrent Firing

- Fire multiple triggers simultaneously (e.g., 10 triggers at same timestamp)
- Assert all events logged correctly without race conditions or double-logging
- Verify all firings occur within timing SLA (±10 seconds)
**Test Execution**:

```bash
go test ./... -v                    # Run all tests
go test ./internal/api/... -v       # API tests
go test ./internal/scheduler/... -v # Scheduler tests
go test -tags=integration ./...     # Integration tests (if separated)
```

**Note**: Tests don't need to simulate high production load, but must demonstrate correctness of logic.

## Common Development Patterns

### Adding a New Trigger Type

1. Update `type` enum in `triggers` table
2. Define config JSON schema
3. Add validation logic in API server
4. Update Kafka event publishing schema
5. Document for external consumer implementations
6. Add tests

### Debugging Timing Issues

- Check scheduler polling frequency (should be ≤ 5 seconds)
- Verify `next_fire_time` index exists
- Check for database lock contention
- Monitor Kafka consumer lag

### Consumer Error Handling

**Platform Responsibility**:

- Platform publishes events reliably to Kafka
- Event logs track that publishing occurred
**Consumer Responsibility** (external):
- Consumers should retry with exponential backoff (e.g., 1s, 2s, 4s, 8s, 16s)
- After max retries (e.g., 5), log error in consumer's own error tracking
- Consumers can optionally write execution results back to platform via API

## Bonus Features (Optional - Extra Credit)

These features are NOT required but can earn up to **6 bonus points**:

### 1. Simple Authentication (2 pts)

- Implement API key authentication for management APIs
- `X-API-Key` header or equivalent mechanism
- Protect trigger CRUD operations (not needed for webhook endpoints)

### 2. Aggregated Event Logs (2 pts)

- `GET /triggers/{id}/events/summary` - Count events by trigger in last 48 hours
- Toggle between success/failure counts
- Useful for monitoring trigger health

### 3. Metrics Endpoint + Dashboard (2 pts)

- Enhanced `/metrics` endpoint with counts AND latencies
- JSON response with clear schema OR basic dashboard
- Include p50/p95/p99 latencies for trigger firing accuracy

## Delivery Checklist

**Must Have** (40 pts):

- [ ] Docker Compose file with all services (API, Scheduler, Kafka, MySQL) - **runnable with single command**
- [ ] All 6 automated tests passing and documented
- [ ] README with setup instructions, API docs, consumer integration guide, design notes
- [ ] API documentation (OpenAPI/Swagger or clear curl examples)
- [ ] Consumer integration documentation (Kafka topic schema, example consumer code)
- [ ] Database schema with proper indexes (next_fire_time, fired_at, retention_status)
- [ ] Retention lifecycle implemented and tested (active → archived → deleted)
- [ ] Platform-level idempotency mechanism (scheduler locking)
- [ ] Timing accuracy verified (±10 seconds) in tests
- [ ] Graceful shutdown handling (SIGTERM for API and Scheduler)
- [ ] Error logging sufficient to debug missed triggers or duplicates
- [ ] Health checks for web app, database, Kafka
- [ ] Metrics endpoint with required counts
**Bonus** (6 pts):
- [ ] API authentication with keys
- [ ] Aggregated event logs by trigger
- [ ] Enhanced metrics with latencies
**Deployment**:
- [ ] Deployed instance on free public cloud (AWS, GCP, Azure free tier)
- [ ] Deployment link added to README
- [ ] **IMPORTANT**: No hosted DB/Redis - must use local Docker containers only

## README Requirements (CRITICAL)

The README.md must contain the following sections (per assignment PDF):

### 1. Setup Instructions

```bash
# Clone and run locally
git clone <repo>
cd event-trigger-platform
docker compose up
```

### 2. Running Tests

```bash
# How to run all tests
go test ./... -v
# How to run with coverage
go test -cover ./...
```

### 3. Deployed Instance

- Link to deployed instance (e.g., `https://your-app.fly.io`)
- How to access the API
- Any required headers or authentication

### 4. API Usage Examples

Provide curl/HTTPie examples for:

- Creating scheduled trigger
- Creating API trigger (show webhook URL in response)
- Firing API trigger via webhook
- Manual/test run
- Listing events (active and archived)
- Health and metrics endpoints
Example:

```bash
# Create scheduled trigger
curl -X POST http://localhost:8080/triggers \
  -H "Content-Type: application/json" \
  -d '{
    "type": "scheduled",
    "config": {
      "schedule": "2025-11-05T15:00:00Z",
      "endpoint": "https://webhook.site/xyz",
      "payload": {"message": "Hello"},
      "http_method": "POST"
    }
  }'
# Create API trigger
curl -X POST http://localhost:8080/triggers \
  -H "Content-Type: application/json" \
  -d '{
    "type": "api",
    "config": {
      "webhook_url": "auto-generated",
      "schema": {"type": "object", "properties": {"name": {"type": "string"}}},
      "endpoint": "https://webhook.site/xyz",
      "http_method": "POST"
    }
  }'
# Response: {"id": "abc", "webhook_url": "http://localhost:8080/webhook/abc"}
```

### 5. Design Decisions & Trade-offs

Document why you chose:

- **Queue**: Kafka (reliable, supports delayed delivery, durable)
- **Database**: MySQL (ACID, good for read-heavy workloads with indexes, JSON support)
- **Scheduling**: Polling every 5s (simple, meets ±10s SLA)
- **Idempotency**: Separate table for job_id tracking (prevents race conditions)

### 6. Assumptions Made

Example:

- Webhook endpoints are assumed to be publicly accessible
- Scheduled triggers use UTC timezone
- Test configuration allows accelerated retention for testing (10s active, 30s total)

### 7. Limitations & Future Improvements

Example:

- Current implementation: Single-threaded scheduler (could use distributed locking for HA)
- Improvement: Add priority queues for urgent triggers
- Improvement: Implement circuit breaker for failing endpoints
- Improvement: Add trigger tagging/categorization

**Performance & Scalability:**

- **Redis caching for scheduler trigger reads**: Current implementation uses JOIN queries (`trigger_schedules` + `triggers` table) which is fine for monolithic/small-scale deployments. At scale (thousands of triggers), scheduler polling every 5s with JOINs could become a bottleneck. Consider Redis cache-aside pattern: cache trigger config by `trigger_id` with 5-10min TTL, invalidate on UPDATE/DELETE. Monitor database query latencies (p95/p99) to determine when caching is needed. Alternative: Use read replicas before adding caching layer complexity.

### 8. Tools/Resources Used

Give credit to:

- ChatGPT/Claude for assistance
- Open source libraries (Gin, segmentio/kafka-go, etc.)
- Reference architectures or tutorials followed
**Reviewer Expectations**:
- Instructions will be followed exactly as written
- Must work on fresh machine with only Docker installed
- If instructions don't work, submission will be rejected

## Development Commands

### Building

```bash
# Build all platform services
go build -o bin/api ./cmd/api
go build -o bin/scheduler ./cmd/scheduler
# Or build specific service
go build -o bin/api ./cmd/api
go build -o bin/scheduler ./cmd/scheduler
```

### Running Services Locally

```bash
# Set environment variables
export DATABASE_URL="user:pass@tcp(localhost:3306)/event_trigger"
export KAFKA_BROKERS="localhost:9092"
# Run platform services
go run ./cmd/api
go run ./cmd/scheduler
```

### Testing

```bash
# Run all tests
go test ./...
# Run tests with coverage
go test -cover ./...
# Run specific package tests
go test ./internal/scheduler/...
# Run integration tests (when implemented)
go test -tags=integration ./...
```

### Database Migrations

```bash
# Migrations stored in: db/migrations/
# Fixtures stored in: db/fixtures/
# TODO: Add migration tool commands when implemented
```

### Docker Compose

```bash
# Start infrastructure (MySQL + Kafka)
cd deploy && docker-compose up -d
# Stop infrastructure
cd deploy && docker-compose down
# View logs
cd deploy && docker-compose logs -f
```

## Project Structure

Current Golang project layout:

```plain
event-trigger-platform/
├── cmd/
│   ├── api/          # API server entrypoint (stub implemented)
│   └── scheduler/    # Scheduler entrypoint (5s ticker implemented)
├── internal/
│   ├── api/          # HTTP handlers (stub Server type)
│   ├── events/       # Event log repository (stub)
│   ├── scheduler/    # Scheduling Engine with ticker
│   └── storage/      # MySQLClient wrapper
├── platform/
│   └── events/       # Kafka Publisher (stub)
├── pkg/
│   └── config/       # App config from env vars
├── db/
│   ├── migrations/   # SQL migration files (5 migrations)
│   └── fixtures/     # Test data fixtures (empty)
├── deploy/
│   ├── docker-compose.yml  # Infrastructure: MySQL, Kafka, API, Scheduler
│   └── README.md           # Docker deployment guide
├── Dockerfile        # Multi-stage build for api and scheduler
├── start.sh/.bat     # Helper scripts for easy startup
├── go.mod
└── CLAUDE.md
```

### Package Organization Philosophy

- **cmd/**: Binary entrypoints - minimal main.go files that wire dependencies
- **internal/**: Application-specific code not importable by external projects
  - Core business logic organized by domain (api, scheduler)
  - Storage abstractions (MySQLClient)
- **platform/**: Infrastructure abstractions (Kafka publisher)
- **pkg/**: Reusable utilities that could be extracted (config loading)
- **db/**: Database-specific assets (migrations, fixtures)

## Implementation Status

### ✅ Bootstrap Completed

- Basic project structure with cmd/, internal/, platform/, pkg/ layout
- Entrypoint stubs for platform services (api, scheduler)
- Scheduler ticker loop (5 second polling)
- Configuration loading from environment variables
- Storage/repository type stubs
- **Docker Compose** with MySQL (8.0) + Kafka (3.8.1 KRaft mode)
- **Database migrations** (5 SQL files including retention events)
- **Multi-stage Dockerfile** for optimized builds
- **MySQL Event Scheduler** for automatic retention management
- Start scripts (start.sh, start.bat) for easy setup

### 🚧 Critical Path to Completion

**High Priority (Functional Completeness - 12pts):**

- [ ] Kafka publisher integration (`platform/events/publisher.go`) - Use segmentio/kafka-go
- [ ] Event log repository (`internal/storage/event_logs.go`) - Create, List, Get operations
- [ ] EventService (`internal/events/service.go`) - FireTrigger + QueryEvents
- [ ] Scheduler polling logic (`cmd/scheduler/main.go`) - Query due schedules, fire via EventService
- [ ] Retention pipeline - Background job for active→archived→deleted transitions
- [ ] Webhook JSON schema validation - xeipuuv/gojsonschema integration
- [ ] Test trigger endpoint implementation (POST /triggers/:id/test)
**Medium Priority (Testing - 6pts):**
- [ ] Automated test suite (all 6 required tests from PDF)
- [ ] Integration tests with testcontainers (MySQL + Kafka)
**Documentation:**
- [ ] README.md rewrite with API examples, setup instructions, design decisions
- [ ] Consumer integration guide with example code
- [ ] Resolve terminology conflict: "API triggers" (PDF) vs "webhook" (code)
- [ ] Decision on interval-based scheduling ("every 30 minutes" style triggers)

## Key Libraries

**Current Dependencies** (from go.mod):

- Go version: 1.22.3
- Module: github.com/dhima/event-trigger-platform
**Recommended Libraries to Add**:
- **Web Framework**: `github.com/gin-gonic/gin` or `github.com/labstack/echo`
- **MySQL Driver**: `github.com/go-sql-driver/mysql`
- **Kafka Client**: `github.com/segmentio/kafka-go` or `github.com/IBM/sarama`
- **CRON**: `github.com/robfig/cron/v3`
- **JSON Schema**: `github.com/xeipuuv/gojsonschema`
- **UUID**: `github.com/google/uuid`
- **Testing**: `github.com/stretchr/testify`
- **Migration Tool**: `github.com/golang-migrate/migrate/v4`

## Architecture Patterns in Use

### Dependency Injection

All services use constructor functions that accept dependencies:

```go
// cmd/api/main.go wires dependencies
srv := api.NewServer() // Will inject DB, Kafka, etc.
```

### Repository Pattern

Storage abstraction separates business logic from data access:

- `internal/storage/mysql.go` - MySQLClient wrapper for raw SQL access
- `internal/events/log_repository.go` - Domain-specific event log operations

### Publisher Pattern

Event-driven architecture with Kafka:

- `platform/events/publisher.go` - Publishes trigger events to Kafka topic
- Consumers are **external** - implemented by users in any language

### Configuration from Environment

12-factor app pattern:

- `pkg/config/config.go` loads DATABASE_URL, KAFKA_BROKERS from env
- No hardcoded connection strings

## Deployment Constraints (CRITICAL)

⚠️ **READ THIS CAREFULLY** - Violating these constraints will result in automatic rejection:

1. **No Hosted Services**: Do NOT use any hosted DB (RDS, Cloud SQL), Redis, or managed queue services
   - All infrastructure must run in **local Docker containers**
   - MySQL and Kafka must be in docker-compose.yml
2. **Single Command Setup**: Must be runnable with `docker compose up` (or `docker-compose up`)
   - Database migrations should auto-run on startup
   - No manual setup steps beyond running docker compose
3. **No External Dependencies**: Reviewer has ONLY Docker installed
   - Cannot assume: kubectl, terraform, npm, python, etc.
   - Everything needed must be in the Docker containers
4. **Free Deployment**: Public deployment must use free tier services
   - AWS Free Tier, GCP Free Tier, Fly.io, Railway, Render, etc.
   - No credit card required services unless explicitly free tier
5. **Language Constraint**: Must use **Golang** (Python also acceptable per PDF, but codebase uses Go)

## Notes for Claude Code

### Critical Non-Negotiables

- **Timing is Critical**: The ±10 second requirement is non-negotiable. Scheduler must poll every 5 seconds maximum.
- **Event Publishing Architecture**: Platform publishes events to Kafka, consumers are external (user-implemented)
- **Separate Execution from Retention Status**: Use `execution_status` (success/failure) and `retention_status` (active/archived) as separate fields
- **48 Hours Total, Not Cumulative**: Events delete 48 hours from fired_at, not 48 hours after archival (2+48=50)
- **Platform-Level Idempotency**: Scheduler uses database locking to prevent duplicate fires. Consumer-level idempotency is external responsibility.

### Database Design

- **Two-Table Architecture**:
  - `triggers` table: Context only (no scheduling info)
  - `trigger_schedules` table: Pending/completed schedule entries
  - Separation allows clean auditing and schedule history
- **Mandatory Indexes**:
  - `triggers`: `idx_status`, `idx_type`
  - `trigger_schedules`: `idx_fire_at_status` (composite), `idx_trigger_id`, `idx_status`
  - `event_logs`: `idx_fired_at`, `idx_trigger_id`, `idx_retention_status`, `idx_execution_status`
- **Foreign Key Behavior**:
  - `trigger_schedules.trigger_id` → `ON DELETE CASCADE` (delete schedules when trigger deleted)
  - `event_logs.trigger_id` → `ON DELETE SET NULL` (preserve history even if trigger deleted)
- **Null Trigger IDs**: Manual test runs should have `trigger_id=NULL` in event_logs
- **Schedule Status Flow**: `pending` → `processing` → `completed` (or `cancelled`)
- **Webhook Triggers**: Do NOT create entries in `trigger_schedules` (event-driven)

### Architecture Quality

- **Clean Separation**: Web API (cmd/api) → Queue (Kafka) → External Consumers (user-implemented)
- **Background Jobs**: Scheduler (cmd/scheduler) runs as separate process, Retention via MySQL Event Scheduler
- **Graceful Shutdown**: Handle SIGTERM in API and Scheduler, finish in-flight operations before exiting
- **Health Checks**: API service must have health endpoint checking database and Kafka connectivity

### Testing Strategy

- **Accelerate Time in Tests**: Use test config to shorten retention (e.g., 10s active, 30s total)
- **Integration Tests**: Spin up Docker containers for MySQL/Kafka using testcontainers or similar
- **Timing Tests**: Verify scheduled triggers fire within 10 seconds using time.After() assertions

### Current Implementation State

#### ✅ **Phase 1: Infrastructure & API Framework (COMPLETE)**

- Project structure with cmd/, internal/, platform/, pkg/ layout
- **Gin v1.11** HTTP framework integrated
- **Swagger/Swaggo v1.16** - Interactive API documentation at `/swagger/index.html`
- **Zap v1.27** - Production-ready structured logging (`internal/logging/logger.go`)
- Complete middleware stack: Recovery, Request ID, Logging, CORS (`internal/api/middleware/`)
- **Full CRUD API handlers** with service layer integration (`internal/api/handlers/triggers.go`):
  - POST /api/v1/triggers (create with config validation)
  - GET /api/v1/triggers (list with pagination)
  - GET /api/v1/triggers/:id (get with next_scheduled_run)
  - PUT /api/v1/triggers/:id (update)
  - DELETE /api/v1/triggers/:id (delete)
  - POST /api/v1/triggers/:id/test (stubbed - needs EventService)
- Webhook receiver endpoint stub (`internal/api/handlers/webhooks.go`)
- Event log query handlers stub (`internal/api/handlers/events.go`)
- Health and metrics endpoints (`internal/api/handlers/health.go`, `metrics.go`)
- Graceful shutdown support (SIGTERM handling in `internal/api/server.go`)
- Docker Compose with MySQL 8.0 + Kafka 3.8.1 (KRaft mode)
- Multi-stage Dockerfile for API and Scheduler
- Makefile for automation (build, swagger, test, docker commands)
- Configuration from environment variables (`pkg/config/config.go`)

#### ✅ **Phase 2: Data Models (COMPLETE)**

- `Trigger` model with three types: webhook, time_scheduled, cron_scheduled
- `TriggerSchedule` model for pending/completed schedule entries
- `EventLog` model for fired event history
- Swagger-annotated request/response DTOs
- Config structs for each trigger type

#### ✅ **Phase 3: Database Layer (COMPLETE)**

**Implemented:**

1. **Database migrations** with two-table architecture:
   - `db/migrations/001_create_triggers_table.sql` - triggers + trigger_schedules tables
   - `db/migrations/002_create_event_logs_table.sql` - event_logs with retention_status
   - Proper indexes: idx_fire_at_status, idx_trigger_id, idx_retention_status
   - Foreign keys with CASCADE/SET NULL behavior
2. **Repository layer** (`internal/storage/triggers.go`):
   - TriggerRepository with CRUD operations
   - Atomic CreateTrigger with transaction support (trigger + schedule in single tx)
   - ListTriggers with pagination and next_scheduled_run JOIN
   - GetTrigger with next fire time lookup
3. **MySQLClient** (`internal/storage/mysql.go`) with transaction wrapper

#### ✅ **Phase 4a: TriggerService (COMPLETE)** | ⏳ **Phase 4b: EventService (TODO)**

**TriggerService COMPLETE** (`internal/triggers/service.go`):

1. **Schedule calculation with timezone support**:
   - ISO 8601 timestamp parsing for time_scheduled triggers
   - CRON expression parsing using `github.com/robfig/cron/v3`
   - Timezone resolution (UTC default, configurable via config.timezone)
   - Next fire time calculation for recurring triggers
2. **Full CRUD implementation**:
   - CreateTrigger: Config validation + normalization, atomic trigger+schedule creation
   - UpdateTrigger: Handle config changes, cancel pending schedules, create new ones
   - DeleteTrigger: Cascade delete via FK constraint
   - GetTrigger: Return trigger with next_scheduled_run
   - ListTriggers: Pagination with type/status filters
3. **Config normalization for all trigger types**:
   - Webhook: Schema validation structure
   - Time-scheduled: run_at parsing with timezone
   - CRON-scheduled: cron expression validation, first fire time calculation

**EventService TODO**:

1. **FireTrigger**: Create event_log + publish to Kafka
2. **QueryEvents**: List with filters (retention_status, trigger_id, execution_status, time range)

#### ⏳ **Phase 5: Kafka Integration (TODO)**

**Need to implement:**

1. **Publisher interface** (`platform/events/publisher.go`)
2. **Kafka producer** using `github.com/segmentio/kafka-go`
3. Topic: "trigger-events"
4. Config: acks=all, retries, timeout

#### ⏳ **Phase 6: Scheduler Service (TODO)**

**Need to implement:**

1. Poll trigger_schedules every 5 seconds
2. JOIN with triggers table for context
3. Fire trigger via EventService
4. Update schedule status to 'completed'
5. Create next schedule for CRON triggers
6. Deactivate one-time triggers

#### ⏳ **Phase 7: Webhook Validation (TODO)**

**Need to implement:**

1. JSON schema validation using `github.com/xeipuuv/gojsonschema`
2. Validate webhook payloads against trigger config
3. Return 400 with validation errors on failure

#### ⏳ **Phase 8: Testing (TODO)**

**Need to implement all 6 required tests:**

1. Create + Fire Webhook Trigger
2. Time-Scheduled Trigger Accuracy (±10s)
3. CRON-Scheduled Recurring Triggers
4. Manual/Test Runs
5. Retention Lifecycle (active → archived → deleted)
6. Concurrent Firing + Idempotency

## Required Dependencies to Add

```bash
go get github.com/go-sql-driver/mysql
go get github.com/segmentio/kafka-go
go get github.com/robfig/cron/v3
go get github.com/xeipuuv/gojsonschema
```

## Evaluation Rubric (Total: 40 pts + 6 bonus)

This is how the submission will be scored:

### 1. Functional Completeness (12 pts)

- CRUD + trigger creation and API firing: 4 pts
- Manual/test runs implemented correctly: 2 pts
- Event logs + retention lifecycle: 6 pts

### 2. Correctness & Timing (8 pts)

- Scheduled events fire within 10s SLA in tests: 4 pts
- Idempotency/duplicate handling at platform level: 2 pts
- Concurrent firing correctness: 2 pts

### 3. Architecture & Implementation Quality (8 pts)

- Clean separation of platform/consumer, clear event publishing flow: 4 pts
- Persistence and data modeling, indexing choices: 2 pts
- Observability & health endpoints: 2 pts

### 4. Code Quality & Tests (6 pts)

- Readable, modular, documented code: 3 pts
- Automated tests that exercise key behavior: 3 pts

### 5. Deployment & Reproducibility (4 pts)

- Containerized + runs locally via single command: 2 pts
- Deployed to free public cloud + link in README: 2 pts

### 6. Bonus Features (6 pts)

- Simple authentication for management APIs: 2 pts
- Aggregated event logs by trigger: 2 pts
- Metrics endpoint with latencies + dashboard: 2 pts
-
