# Docker Deployment

## Quick Start

### Using Default Configuration

```bash
# From project root
docker compose -f deploy/docker-compose.yml up

# Or detached
docker compose -f deploy/docker-compose.yml up -d

# View logs
docker compose -f deploy/docker-compose.yml logs -f

# Stop
docker compose -f deploy/docker-compose.yml down
```

### Using Custom Configuration

For production use, credentials and configuration should be externalized:

1. Copy the example environment file:
```bash
cp deploy/.env.example deploy/.env
```

2. Edit `deploy/.env` with your custom values:
```bash
# MySQL Database Configuration
MYSQL_ROOT_PASSWORD=your_secure_root_password
MYSQL_DATABASE=event_trigger
MYSQL_USER=appuser
MYSQL_PASSWORD=your_secure_app_password
MYSQL_PORT=3306

# Kafka Configuration
KAFKA_PORT=9092
KAFKA_INTERNAL_PORT=29092

# API Configuration
API_PORT=8080
LOG_LEVEL=info

# Scheduler Configuration
SCHEDULER_INTERVAL=5s
```

3. Start the services with your custom configuration:
```bash
docker compose -f deploy/docker-compose.yml --env-file deploy/.env up -d
```

Alternatively, you can export environment variables directly:
```bash
export MYSQL_ROOT_PASSWORD=your_secure_root_password
export MYSQL_PASSWORD=your_secure_app_password
docker compose -f deploy/docker-compose.yml up -d
```

**Security Note**: Never commit the `.env` file with real credentials to version control. The `.env.example` file contains default values for development only.

## Services

- **MySQL 8.0** :3306 - Database with Event Scheduler for retention
- **Kafka 3.8.1** :9092 - Message queue (KRaft mode, no Zookeeper)
- **API Server** :8080 - REST API
- **Scheduler** - Polls triggers every 5s and publishes events to Kafka

## Architecture Notes

**Event-Driven Design**: This platform publishes trigger events to Kafka. External consumers (user-implemented) subscribe to the `trigger-events` topic to process triggers.

**Retention**: Handled by MySQL Event Scheduler (see `db/migrations/004_setup_retention_events.sql`)

- Archive events after 2 hours
- Delete events after 48 hours
- No separate service needed

**Kafka**: Uses KRaft mode (no Zookeeper dependency)

- Faster startup
- Lower memory footprint

## Consumer Integration

External consumers subscribe to Kafka topic `trigger-events` to process trigger events.

### Example Consumer (Go)

```go
package main

import (
    "context"
    "encoding/json"
    "github.com/segmentio/kafka-go"
    "log"
    "net/http"
)

type TriggerEvent struct {
    EventID   string          `json:"event_id"`
    TriggerID string          `json:"trigger_id"`
    Type      string          `json:"type"`
    Payload   json.RawMessage `json:"payload"`
    FiredAt   string          `json:"fired_at"`
}

func main() {
    reader := kafka.NewReader(kafka.ReaderConfig{
        Brokers: []string{"localhost:9092"},
        Topic:   "trigger-events",
        GroupID: "my-consumer-group",
    })
    defer reader.Close()

    for {
        msg, err := reader.ReadMessage(context.Background())
        if err != nil {
            log.Printf("Error reading message: %v", err)
            continue
        }

        var event TriggerEvent
        if err := json.Unmarshal(msg.Value, &event); err != nil {
            log.Printf("Error unmarshaling: %v", err)
            continue
        }

        // Execute business logic
        log.Printf("Processing event %s for trigger %s", event.EventID, event.TriggerID)

        // Example: HTTP POST to endpoint
        // http.Post(event.Payload.Endpoint, "application/json", bytes.NewBuffer(event.Payload.Body))
    }
}
```

### Example Consumer (Python)

```python
from kafka import KafkaConsumer
import json
import requests

consumer = KafkaConsumer(
    'trigger-events',
    bootstrap_servers=['localhost:9092'],
    group_id='my-consumer-group',
    value_deserializer=lambda m: json.loads(m.decode('utf-8'))
)

for message in consumer:
    event = message.value
    print(f"Processing event {event['event_id']} for trigger {event['trigger_id']}")

    # Execute business logic
    # requests.post(event['payload']['endpoint'], json=event['payload']['body'])
```

### Kafka Topic Schema

Events published to `trigger-events` have the following structure:

```json
{
  "event_id": "550e8400-e29b-41d4-a716-446655440000",
  "trigger_id": "123e4567-e89b-12d3-a456-426614174000",
  "type": "scheduled",
  "payload": {
    "endpoint": "https://webhook.site/xyz",
    "body": {"message": "hello"},
    "http_method": "POST"
  },
  "fired_at": "2025-11-05T10:30:00Z",
  "source": "scheduled"
}
```

### Scaling Consumers

Consumers can be horizontally scaled using Kafka consumer groups:

```bash
# Run multiple consumer instances (any language)
# Each instance joins the same consumer group
# Kafka automatically distributes partitions across instances
./my-consumer &
./my-consumer &
./my-consumer &
```

## Troubleshooting

```bash
# Check service status
docker compose -f deploy/docker-compose.yml ps

# Rebuild after code changes
docker compose -f deploy/docker-compose.yml build

# Clean start (removes volumes)
docker compose -f deploy/docker-compose.yml down -v
docker compose -f deploy/docker-compose.yml up
```
