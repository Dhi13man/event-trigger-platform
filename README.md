## Event Trigger Platform

Production-ready event trigger management service with API + scheduler orchestration. Ships with a Go API layer, MySQL persistence, and Kafka-facing worker interfaces (under construction).

### Architecture Overview
- **API (`cmd/api`)** issues CRUD operations over triggers, lists event logs, and exposes health/metrics + Swagger docs.
- **Scheduler (`cmd/scheduler`)** *(stub)* will poll `trigger_schedules` for due jobs and publish Kafka messages for workers.
- **Worker (`cmd/worker`)** *(stub)* consumes Kafka jobs, validates payloads (webhook schema / scheduled payload), fires downstream HTTP endpoints, and writes `event_logs`.
- **MySQL** stores canonical trigger definitions, scheduling queue, and event history. Migrations live in `db/migrations`.
- **Kafka** uses `platform/events` utility package for publishing/consuming jobs (queue wiring to be completed).

### Data Model
| Table | Purpose | Key Columns |
|-------|---------|-------------|
| `triggers` | Canonical definition for each trigger | `id`, `name`, `type (webhook|time_scheduled|cron_scheduled)`, `config (JSON)`, `status` |
| `trigger_schedules` | Per-occurrence queue for scheduled triggers | `id`, `trigger_id`, `fire_at`, `status (pending|processing|completed|cancelled)`, `attempt_count`, `last_attempt_at` |
| `event_logs` | Historical record of every fire | `id`, `trigger_id (nullable)`, `trigger_type`, `source (webhook|scheduler|manual-test)`, `payload`, `execution_status`, `retention_status`, `is_test_run` |
| `idempotency_keys` | Future use for dedupe/ retries | `job_id`, `event_id` |

Trigger `config` payloads are validated & normalized per type:
- `webhook` → endpoint, method, optional headers + JSON schema.
- `time_scheduled` → ISO timestamp + endpoint metadata; first schedule row inserted immediately.
- `cron_scheduled` → cron expression + endpoint metadata; next occurrence calculated via cron parser.

### API Surface (current status)
| Endpoint | Status | Notes |
|----------|--------|-------|
| `POST /api/v1/triggers` | ✅ | Validates payload per type, persists trigger, schedules first run when applicable. |
| `GET /api/v1/triggers` | ✅ | Pagination with optional `type`/`status` filters, includes `next_scheduled_run`. |
| `GET /api/v1/triggers/{id}` | ✅ | Fetch single trigger. |
| `PUT /api/v1/triggers/{id}` | ✅ | Update name/status/config. Recomputes schedule automatically. |
| `DELETE /api/v1/triggers/{id}` | ✅ | Removes trigger and pending schedules. |
| `POST /api/v1/triggers/{id}/test` | ⏳ | Stub – needs worker integration to enqueue manual/test execution. |
| `POST /api/v1/webhook/{trigger_id}` | ⏳ | Stub – should validate payload + enqueue to Kafka. |
| Event Logs (`GET /api/v1/events/...`) | ⏳ | Handlers scaffolded; storage wiring still pending. |

Swagger is available at `/swagger/index.html` once the API server is running.

### Running Locally
1. **Dependencies**: Go ≥ 1.24, Docker, Docker Compose, MySQL 8, Kafka (via docker compose template TBA).  
2. **Environment**: copy `.env.example` (coming soon) or set:
   - `DATABASE_URL` (e.g. `user:pass@tcp(localhost:3306)/event_triggers?parseTime=true`)
   - `KAFKA_BROKERS` (default `localhost:9092`)
3. **Migrations**: pending migration runner script; use `mysql` CLI or preferred tool to apply files under `db/migrations`.
4. **Run API**: `go run ./cmd/api` (needs valid DB connection).  
5. **Scheduler / Worker**: stubs exist; wiring to follow once Kafka contract is finalized.

`go test ./...` executes available unit tests (currently scaffolding, expect expansion).

### Known Limitations & Future Scope
- Scheduler drift, uniqueness tokens, and dedupe keys are **not implemented** yet; `trigger_schedules` keeps simple pending/processing semantics.
- No background retention pipeline yet; event lifecycle jobs remain TODO.
- Kafka publishing/consuming layers still have placeholders; no actual event dispatch.
- Webhook receiver, manual/test run path, and event log querying require storage + worker integration.
- Auth, rate-limiting, metrics, dashboards, and deployment scripts are out of scope for this iteration.

### Next Steps
1. Implement scheduler polling loop with safe row locking + status transitions.
2. Wire Kafka publisher/consumer and worker execution (HTTP client, retries, attempt tracking).
3. Persist event logs + retention workflow.
4. Fill webhook/test run gaps and build integration tests (API + scheduler + worker).
5. Flesh out Docker Compose stack & deployment pipeline; document in README once available.
