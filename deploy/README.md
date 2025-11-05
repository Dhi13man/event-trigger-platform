# Docker Deployment

This directory contains Docker Compose configuration for running the Event Trigger Platform locally.

## Quick Start

1. **Copy the example environment file:**
   ```bash
   cd deploy
   cp .env.example .env
   ```

2. **Edit `.env` with your desired configuration** (optional - defaults work out of the box)

3. **Start all services:**
   ```bash
   # From deploy directory
   docker compose up --build

   # Or from project root
   docker compose -f deploy/docker-compose.yml up --build
   ```

4. **Stop services:**
   ```bash
   docker compose down
   ```

5. **Stop and remove volumes (clean slate):**
   ```bash
   docker compose down -v
   ```

## Environment Configuration

The `.env` file controls all configuration. Key variables:

### MySQL
- `MYSQL_ROOT_PASSWORD` - Root password (default: `rootpassword`)
- `MYSQL_DATABASE` - Database name (default: `event_trigger`)
- `MYSQL_USER` - Application user (default: `appuser`)
- `MYSQL_PASSWORD` - Application password (default: `apppassword`)
- `MYSQL_PORT` - External port (default: `3306`)

### Kafka
- `KAFKA_NODE_ID` - Kafka node ID for KRaft (default: `1`)
- `KAFKA_EXTERNAL_PORT` - External access port (default: `9092`)
- `KAFKA_INTERNAL_PORT` - Internal Docker network port (default: `29092`)
- `CLUSTER_ID` - KRaft cluster ID (default: `event-trigger-kafka-cluster-v1`)

### API Server
- `API_PORT` - API server port (default: `8080`)
- `LOG_LEVEL` - Logging level (default: `info`)

### Scheduler
- `SCHEDULER_INTERVAL` - Polling interval (default: `5s`)

### Constructed Values
- `DATABASE_URL` - Full MySQL connection string
- `KAFKA_BROKERS` - Kafka broker addresses

All values must be defined in `.env` file (no defaults in docker-compose.yml).

## Services

The compose file starts these services:

1. **mysql** - MySQL 8.0 database
   - Port: 3306 (configurable via `MYSQL_PORT`)
   - Auto-runs migrations from `../db/migrations/`
   - Health check enabled

2. **kafka** - Apache Kafka 3.8.1 (KRaft mode, no Zookeeper)
   - External port: 9092 (host access)
   - Internal port: 29092 (Docker network)
   - Auto-creates topics
   - Health check enabled

3. **api** - REST API server
   - Port: 8080 (configurable via `API_PORT`)
   - Waits for MySQL and Kafka to be healthy
   - Health endpoint: `http://localhost:8080/health`

4. **scheduler** - Background scheduler
   - Polls MySQL every 5 seconds for due triggers
   - Publishes events to Kafka
   - Waits for MySQL and Kafka to be healthy

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

## Health Checks

Check service health:

```bash
# API health
curl http://localhost:8080/health

# MySQL
docker exec event-trigger-mysql mysqladmin ping -h localhost -u root -prootpassword

# Kafka
docker exec event-trigger-kafka kafka-broker-api-versions.sh --bootstrap-server localhost:9092
```

## Volumes

Persistent data stored in Docker volumes:
- `mysql_data` - MySQL database files
- `kafka_data` - Kafka logs and topics

## Network

All services communicate via `event-trigger-network` bridge network.

## Customization

### Using Different Ports

Edit `.env`:
```env
API_PORT=9000
MYSQL_PORT=3307
```

### Changing Passwords

Edit `.env`:
```env
MYSQL_ROOT_PASSWORD=my_secure_root_password
MYSQL_PASSWORD=my_secure_app_password
```

**Important:** If you change MySQL credentials, update `DATABASE_URL` to match:
```env
DATABASE_URL=appuser:my_secure_app_password@tcp(mysql:3306)/event_trigger?parseTime=true&loc=UTC
```

### Production Deployment

For production:
1. Use strong passwords in `.env`
2. Never commit `.env` to version control (it's already in `.gitignore`)
3. Consider using Docker secrets or external secret management
4. Enable TLS for Kafka and MySQL
5. Use a managed Kafka/MySQL service if possible
6. Adjust retention and replication settings for Kafka

## Troubleshooting

### Services won't start
```bash
# Check logs
docker compose logs -f

# Check specific service
docker compose logs -f api
docker compose logs -f mysql
```

### Database connection errors
```bash
# Verify MySQL is healthy
docker compose ps

# Check MySQL logs
docker compose logs mysql

# Test connection manually
docker exec -it event-trigger-mysql mysql -u appuser -papppassword event_trigger
```

### Kafka connection errors
```bash
# Verify Kafka is healthy
docker compose ps

# Check Kafka logs
docker compose logs kafka

# List topics
docker exec event-trigger-kafka kafka-topics.sh --list --bootstrap-server localhost:9092
```

### Reset everything
```bash
# Stop and remove all data
docker compose down -v

# Rebuild and start fresh
docker compose up --build
```
