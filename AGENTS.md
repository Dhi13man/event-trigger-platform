# Event Trigger Platform – Agent Guide

## Mission Brief

- Deliver the production-ready, containerized event-trigger platform defined in `docs/backend-assignment-oct-2025.pdf`.
- Implement the system in Go and keep all runtime components locally orchestrated with Docker/Docker Compose (MySQL + Kafka). Avoid external cloud dependencies during development.
- Treat the assignment PDF as the authoritative specification for scope, quality bars, deliverables, and rubric priorities.
- Keep this guide aligned with current architecture and progress so future agents can ramp quickly.

## Platform Vision

The platform is an **event publisher**. We accept trigger definitions, ensure they fire on time, emit execution jobs to Kafka, and (by default) consume those jobs with our worker service to deliver payloads to user-specified HTTP endpoints. Advanced users can add extra consumers, but reliable delivery through the built-in worker remains our responsibility.

### Runtime Components

- **API Service (`cmd/api`)**: Exposes REST endpoints for trigger CRUD, manual/test runs, webhook ingestion, event queries, health, metrics, and Swagger docs. Executes lightweight validation and delegates durable work to services and queues.
- **Scheduler Service (`cmd/scheduler`)**: Polls `trigger_schedules` for due jobs, transitions them through lifecycle states, publishes Kafka messages, and enqueues the next occurrence for cron triggers.
- **Worker Service (`cmd/worker`)**: Consumes Kafka jobs, performs HTTP dispatch according to trigger config, applies retry/backoff, writes `event_logs`, and updates schedule state or reschedules as needed.
- **Infrastructure**: MySQL for persistence, Kafka for durable job delivery, Zap-based logging, and metrics endpoints for observability.

## Data Model Summary

- `triggers`: canonical definitions (fields: `id`, `name`, `type` in {`webhook`, `time_scheduled`, `cron_scheduled`}, `status`, `config` JSON, timestamps).
- `trigger_schedules`: per-occurrence queue rows (`id`, `trigger_id`, `fire_at`, `status` in {`pending`, `processing`, `completed`, `cancelled`}, `attempt_count`, `last_attempt_at`, timestamps).
- `event_logs`: execution history (`id`, `trigger_id` nullable, `trigger_type`, `fired_at`, `payload`, `source` in {`webhook`, `scheduler`, `manual-test`}, `execution_status`, `error_message`, `retention_status`, `is_test_run`, timestamps).
- `idempotency_keys`: job-to-event mapping for deduplication (`job_id`, `event_id`, `created_at`).

Trigger configurations are type-specific:

- **Webhook**: JSON schema, downstream endpoint, HTTP method, optional headers.
- **Time-scheduled**: `run_at` (ISO 8601 + timezone), endpoint, method, headers, payload.
- **Cron-scheduled**: CRON expression + timezone, endpoint, method, headers, payload, optional retry policy overrides.

## Functional Expectations

1. **Trigger Lifecycle**
   - Create/update/delete triggers via API with validation, pagination, and webhook URL generation.
   - Status changes affect only future executions; deleting triggers cascades pending schedules but preserves historical logs.
2. **Scheduling**
   - On creation/update of `time_scheduled` or `cron_scheduled` triggers, compute and store the next `trigger_schedules` row inside the same transaction.
   - Scheduler polls using the `(fire_at, status)` index, claims due rows by flipping `status` to `processing`, publishes jobs, and marks completion or reschedules.
   - Guarantee ≤10s drift between scheduled time and job publication under normal load.
3. **Execution & Idempotency**
   - Worker consumes jobs, validates payload (webhook/manual/test), executes HTTP dispatch, records outcomes in `event_logs`, and handles retry/backoff (with persisted `attempt_count`/`last_attempt_at`).
   - Use `idempotency_keys` to prevent duplicate logging when jobs are retried or redelivered.
4. **Manual/Test Runs**
   - `POST /api/v1/triggers/{id}/test` enqueues exactly one job flagged `is_test_run=true` without adding schedule rows.
5. **Retention Pipeline**
   - Background process (scheduler subroutine or dedicated worker) transitions event logs: Active (0–2h) → Archived (2–48h) → Deleted (>48h). Document accelerated timings for tests.
6. **Webhook Ingestion**
   - `POST /api/v1/webhook/{trigger_id}` validates payload against stored JSON schema, persists event log, and publishes job when valid; rejects with 400 on validation errors.

## Observability

- Structured logging (Zap) with request/job IDs everywhere.
- `/health` endpoints per service verifying DB, Kafka connectivity, and internal readiness.
- `/metrics` surfaces counts for pending/processing/completed schedules, retry attempts, retention buckets, trigger counts by type, and latency percentiles (p50/p95/p99).
- Add tracing hooks or context IDs as the system grows.

## Deployment & Operations

- Provide Dockerfiles for API, scheduler, and worker plus a `docker-compose.yml` that launches MySQL, Kafka (KRaft or with ZooKeeper), and all services with one command.
- Automate migrations (either run on container start or document a one-liner using `golang-migrate`).
- Supply `.env.example` capturing all relevant env vars (DB URL, Kafka brokers, retention intervals, retry settings).
- Publish a free-tier deployment and include link + setup notes in README once stable.

## Testing Strategy

- **Unit Tests**: config validation, cron parsing, schedule generation, trigger updates, worker retry policy, retention math.
- **Integration Tests**: API + MySQL persistence, scheduler-worker end-to-end with Kafka (tagged `//go:build integration`), webhook payload validation paths, retention transitions (with accelerated windows).
- **Concurrency Tests**: simultaneous trigger firings to verify idempotency and SLA adherence.
- Always run `go test ./...` (and `-race` for concurrency-sensitive changes) before commits.

## Current TODO Highlights

- Implement scheduler loop, Kafka publisher/consumer utilities, worker execution flow, and retention job.
- Finish webhook receiver + manual/test run execution path.
- Wire event log repository, API endpoints, and add pagination/filters.
- Extend README with architecture diagram, API examples, known limitations, deployment steps, and consumer integration guide.
- Add CI-friendly migrations and Docker Compose improvements.

## Workflow Guardrails

- Entry points reside in `cmd/`; reusable logic in `internal/`; shared helpers in `pkg/` / `platform/`.
- Maintain explicit dependency injection (logger, DB, Kafka) and avoid globals.
- Prefer ASCII text; run `gofmt`/`goimports` on all Go files.
- Never use destructive git commands (e.g., `git reset --hard`) without explicit user direction.
- Update this guide whenever architecture or priorities change.
