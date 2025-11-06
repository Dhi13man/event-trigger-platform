# Deployment (Docker Compose)

This directory contains Docker Compose configuration to run the Event Trigger Platform locally with MySQL and Kafka. For architecture, API usage, and consumer guidance, see the root README: `../README.md`.

## Quick Start

1. Copy env defaults

   ```bash
   cd deploy
   cp .env.example .env
   ```

2. Start stack

   ```bash
   docker compose up -d --build
   ```

3. Verify

   ```bash
   curl http://localhost:8080/health
   open http://localhost:8080/swagger/index.html
   ```

4. Stop / reset

   ```bash
   docker compose down          # stop
   docker compose down -v       # stop and remove volumes (clean slate)
   ```

## Environment (.env)

The `.env` file configures all services. See `.env.example` for working defaults.

- MySQL: `MYSQL_ROOT_PASSWORD`, `MYSQL_DATABASE`, `MYSQL_USER`, `MYSQL_PASSWORD`, `MYSQL_PORT`
- Kafka (KRaft): `KAFKA_NODE_ID`, `KAFKA_EXTERNAL_PORT`, `KAFKA_INTERNAL_PORT`, `KAFKA_CONTROLLER_PORT`, `CLUSTER_ID`
- API: `API_PORT`, `LOG_LEVEL`
- Scheduler: `SCHEDULER_INTERVAL`
- App wiring: `DATABASE_URL`, `KAFKA_BROKERS`

Example:

```env
# Database URL constructed from the above values
DATABASE_URL=appuser:apppassword@tcp(mysql:3306)/event_trigger?parseTime=true&loc=UTC
KAFKA_BROKERS=kafka:29092
API_PORT=8080
LOG_LEVEL=info
SCHEDULER_INTERVAL=5s
```

## Services

- mysql (MySQL 8.0)
  - Port: `3306` (host) → `MYSQL_PORT`
  - Runs migrations from `../db/migrations/`
  - Health check enabled

- kafka (Apache Kafka 3.8.1, KRaft)
  - External: `9092` (host), Internal: `29092` (Docker network)
  - Auto-creates topics
  - Health check enabled

- api (REST API server)
  - Port: `8080` (mapped from `API_PORT`)
  - Health: `GET /health`, Metrics: `GET /metrics`, Swagger: `/swagger/index.html`

- scheduler (background scheduler)
  - Polls for due schedules and publishes to Kafka

## Common Operations

```bash
# From this directory
docker compose up -d                 # start
docker compose logs -f api scheduler # tail logs
docker compose up -d --scale api=3   # scale API instances
docker compose down                  # stop
docker compose down -v               # stop and remove data
```

## Health Checks

```bash
# API
curl http://localhost:8080/health

# MySQL
docker exec event-trigger-mysql mysqladmin ping -h localhost -u root -p$MYSQL_ROOT_PASSWORD

# Kafka
docker exec event-trigger-kafka kafka-broker-api-versions.sh --bootstrap-server localhost:9092
docker exec event-trigger-kafka kafka-topics.sh --list --bootstrap-server localhost:9092
```

## Volumes & Network

- Volumes: `mysql_data` (MySQL), `kafka_data` (Kafka logs/topics)
- Network: `event-trigger-network` (bridge)

## Customization

- Ports: update `API_PORT`, `MYSQL_PORT`, `KAFKA_EXTERNAL_PORT` in `.env`
- Credentials: update `MYSQL_*` and keep `DATABASE_URL` in sync

```env
MYSQL_ROOT_PASSWORD=my_secure_root
MYSQL_PASSWORD=my_secure_app_password
DATABASE_URL=appuser:my_secure_app_password@tcp(mysql:3306)/event_trigger?parseTime=true&loc=UTC
```

## Production Notes (Compose)

- Use strong secrets in `.env` (do not commit `.env`)
- Consider Docker secrets / external secret managers
- Enable TLS where applicable (Kafka/MySQL) as needed
- Prefer managed MySQL/Kafka for durability at scale

## Troubleshooting

### Services won’t start

```bash
docker compose ps
docker compose logs -f
docker compose logs -f api mysql kafka
```

### Database connection errors

```bash
docker compose logs mysql
docker exec -it event-trigger-mysql mysql -u appuser -papppassword event_trigger
```

### Kafka connection errors

```bash
docker compose logs kafka
docker exec event-trigger-kafka kafka-broker-api-versions.sh --bootstrap-server localhost:9092
docker exec event-trigger-kafka kafka-topics.sh --list --bootstrap-server localhost:9092
```

### Scheduler publish failures

- Check Kafka availability and networking; restart Kafka if needed
- The scheduler retries automatically on next ticks

### Reset all data

```bash
docker compose down -v
docker compose up -d --build
```

## References

- Architecture, features, API examples: `../README.md`
- Interactive API docs (when running): <http://localhost:8080/swagger/index.html>
