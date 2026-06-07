# Notifier

A standalone email dispatch service — single source of truth for all outgoing email notifications across your infrastructure.

Instead of every internal service needing its own SMTP credentials, SPF/DKIM setup, and IP whitelisting, Notifier gives you one secure ingress point. Services authenticate via API key or HMAC-signed requests, and Notifier handles the rest: validation, rate limiting, queuing, retries, delivery, and audit logging.

## Features

- **Single ingress for email** — Every internal service calls Notifier instead of talking directly to SMTP
- **API Key + HMAC authentication** — SHA-256 hashed API keys with constant-time comparison, plus optional HMAC request signing
- **Redis Streams job queue** — Horizontally scalable, persistent, with consumer groups for exactly-once delivery
- **Worker pool** — Configurable N goroutines per container for parallel SMTP dispatch
- **Retry with exponential backoff** — Automatic retries before moving to Dead Letter Queue
- **Dead Letter Queue (DLQ)** — Inspect and replay failed jobs from the admin API
- **Rate limiting** — Per-consumer (60 req/min) + per-IP (120 req/min) with sliding window via Redis sorted sets
- **Abuse detection** — Auto-suspends consumers exceeding bounce rate threshold (default 20%)
- **Audit logging** — Immutable, append-only table tracking every API call
- **Request validation** — Struct-tag validation on all inputs, 10 MB body size limit
- **Prometheus metrics** — Request count, duration, in-flight gauge, queue depth, DLQ depth
- **Structured JSON logging** — Via Go's `log/slog`
- **Health check** — Checks PostgreSQL + Redis connectivity
- **Graceful shutdown** — Workers finish in-flight jobs before exit on SIGTERM
- **Stateless by design** — All state lives in PostgreSQL + Redis; any container handles any request
- **Admin dashboard** — React frontend to manage consumers, view jobs, inspect DLQ

## Architecture

```
                        ┌──────────────────────┐
    Internal Services ──│   Load Balancer       │── Round-robin
    (API consumers)     └──────┬───────┬───────┘
                               │       │
                        ┌──────┼───────┼───────┐
                        ▼      ▼       ▼       ▼
                   ┌──────┐ ┌──────┐ ┌──────┐
                   │ Pod A│ │ Pod B│ │ Pod C│   All identical, stateless
                   └──┬───┘ └──┬───┘ └──┬───┘
                      │        │        │
                      └────────┼────────┘
                               ▼
                        ┌─────────────┐
                        │    Redis     │   Streams + rate-limit counters
                        │  Streams     │   Consumer group: "notifier-workers"
                        └──────┬──────┘
                               │
                               ▼
                        ┌─────────────┐
                        │ PostgreSQL  │   Consumers, keys, jobs, audit log
                        └─────────────┘
                               │
                               ▼
                        ┌─────────────┐
                        │    SMTP     │   Mailpit (dev) / SES / SendGrid (prod)
                        │   Relay     │
                        └─────────────┘
```

**Data flow:**

1. `POST /v1/send` → validate → rate-check → create job in PostgreSQL → `XADD` to Redis Stream → return `202 Accepted`
2. Worker goroutine → `XREADGROUP` from stream → send via SMTP → `XACK` on success
3. On failure → retry with backoff (re-enqueue) → after max retries → `XADD` to `email:dlq`
4. Admin can inspect DLQ → replay individual messages back to `email:jobs`

## Tech Stack

| Concern | Technology |
|---------|-----------|
| Language | Go 1.22+ |
| HTTP Router | chi (go-chi/chi v5) |
| ORM | GORM with PostgreSQL driver |
| Database | PostgreSQL |
| Job Queue | Redis Streams (go-redis v9) |
| Rate Limiting | Redis sorted sets (sliding window) |
| Auth | SHA-256 API keys + optional HMAC-SHA256 request signing |
| Request Validation | go-playground/validator v10 |
| Migrations | golang-migrate |
| Config | caarlos0/env v11 (env vars → struct) |
| Metrics | prometheus/client_golang |
| Logging | log/slog (structured JSON) |
| SMTP | net/smtp (stdlib) via `EmailEngine` interface |
| Secrets | AES-256-GCM encryption for HMAC secrets at rest |

**Frontend:**

| Concern | Technology |
|---------|-----------|
| Framework | React 19 |
| Build | Vite 6 |
| Routing | TanStack Router (file-based) |
| Data Fetching | TanStack Query (React Query v5) |
| Styling | Tailwind CSS 3 |
| Type Checking | TypeScript 5.7 |

**Infrastructure:**

| Concern | Technology |
|---------|-----------|
| Containerization | Docker, Docker Compose |
| Dev SMTP | Mailpit (catches all outbound mail) |
| CI/CD | GitHub Actions |

## Project Structure

```
notifier/
├── server/                         # Go backend
│   ├── cmd/notifier/main.go        # Entry point, wiring, graceful shutdown
│   ├── internal/
│   │   ├── auth/                   # API key generation, hashing, HMAC signing, AES secrets
│   │   ├── config/config.go        # Env-driven config struct
│   │   ├── engine/
│   │   │   ├── engine.go           # EmailEngine interface
│   │   │   └── smtp.go             # net/smtp implementation
│   │   ├── handler/
│   │   │   ├── admin.go            # DLQ, jobs, suspend/reactivate
│   │   │   ├── consumer.go         # Consumer CRUD
│   │   │   ├── dispatch.go         # POST /v1/send, GET /v1/jobs/:id
│   │   │   ├── health.go           # Health check
│   │   │   ├── helpers.go          # writeJSON, writeError, context helpers
│   │   │   └── validate.go         # go-playground/validator wrapper
│   │   ├── model/
│   │   │   ├── api.go              # Request/response DTOs
│   │   │   ├── audit.go            # AuditLog entity
│   │   │   ├── consumer.go         # Consumer entity
│   │   │   └── job.go              # Job entity + status constants
│   │   ├── repository/
│   │   │   ├── audit.go            # Audit log writes
│   │   │   ├── consumer.go         # Consumer CRUD + auth + HMAC
│   │   │   └── job.go              # Job CRUD + bounce rate queries
│   │   ├── server/
│   │   │   ├── server.go           # HTTP server, DB/Redis connection, stream setup
│   │   │   ├── routes.go           # Route registration, middleware chain
│   │   │   ├── middleware.go        # Auth, audit, rate-limit, body-size middleware
│   │   │   ├── middleware_test.go
│   │   │   └── metrics.go          # Prometheus metrics collector
│   │   ├── service/
│   │   │   ├── abuse.go            # Abuse detection goroutine
│   │   │   ├── consumer.go         # Consumer business logic
│   │   │   ├── dispatch.go         # Enqueue + job lookup
│   │   │   └── ratelimit.go        # Redis sorted-set sliding window
│   │   └── worker/
│   │       └── worker.go           # Redis Stream consumer → SMTP sender
│   ├── migrations/                 # golang-migrate SQL files
│   │   ├── 000001_create_consumers.up.sql
│   │   ├── 000002_create_jobs.up.sql
│   │   ├── 000003_create_audit_logs.up.sql
│   │   └── 000004_add_hmac_secret_to_consumers.up.sql
│   ├── Dockerfile
│   ├── go.mod
│   └── go.sum
│
├── client/                         # React admin dashboard
│   ├── src/
│   │   ├── routes/
│   │   │   ├── __root.tsx          # Layout shell (nav, outlet)
│   │   │   ├── index.tsx           # Dashboard (placeholder cards)
│   │   │   ├── consumers/
│   │   │   │   ├── index.tsx       # Consumer list
│   │   │   │   ├── $consumerId.tsx  # Consumer detail
│   │   │   │   └── create.tsx      # Create consumer with one-time key
│   │   │   └── jobs/
│   │   │       ├── index.tsx       # Jobs list
│   │   │       └── $jobId.tsx      # Job detail
│   │   ├── lib/api.ts              # Fetch wrapper with Bearer token
│   │   ├── main.tsx
│   │   └── index.css
│   ├── index.html
│   ├── package.json
│   ├── vite.config.ts
│   ├── tsconfig.json
│   └── tailwind.config.ts
│
├── docs/                           # Architecture discussions
│   ├── project description.md
│   ├── roadmap.md
│   ├── discussion-tech-stack.md
│   ├── discussion-language-go.md
│   ├── discussion-routing-framework.md
│   ├── discussion-redis-streams.md
│   └── discussion-statelessness.md
│
├── docker-compose.yml              # Mailpit + Notifier (postgres/redis external)
├── Makefile                        # Build, run, test, migrate, docker targets
└── README.md
```

## Getting Started

### Prerequisites

- Go 1.22+
- Node.js 20+
- Docker (for PostgreSQL, Redis, and Mailpit)
- PostgreSQL container running (named `postgres`)
- Redis container running (named `redis`)

All three must be on the same Docker network.

### Quick Start (Docker Compose)

```bash
# 1. Ensure postgres and redis containers are running on the same Docker network
docker network ls
docker ps | grep -E 'postgres|redis'

# 2. Start Notifier + Mailpit
make docker-up

# Notifier:  http://localhost:8080
# Mailpit UI: http://localhost:8025  (catches all outbound email)
# Health:    http://localhost:8080/health
# Metrics:   http://localhost:8080/metrics
```

The `docker-compose.yml` expects external `postgres` and `redis` containers. If they are on a custom network, uncomment the `networks` section in `docker-compose.yml` and set the network name.

### Development (Native Server)

```bash
# 1. Start infrastructure (Mailpit, Postgres, Redis should already be running)
make dev

# This starts Mailpit via Docker and runs the Go server natively.
# The server connects to postgres/redis on localhost:5432 and localhost:6379.

# 2. Or run everything in Docker
make dev-docker
```

### Frontend Development

```bash
cd client
npm install
npm run dev                         # → http://localhost:5173

# The Vite dev server proxies /api → http://localhost:8080
# Set VITE_ADMIN_KEY env var to your admin key (default: admin-key-change-me)
```

## Configuration

All configuration is via environment variables. See `server/internal/config/config.go`:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `HOST` | `0.0.0.0` | HTTP bind address |
| `DATABASE_URL` | `postgres://postgres:...@localhost:5432/notifier?sslmode=disable` | PostgreSQL connection string |
| `REDIS_HOST` | `localhost` | Redis host |
| `REDIS_PORT` | `6379` | Redis port |
| `REDIS_PASSWORD` | `Rd1s_P@ssw0rd_2024` | Redis password |
| `REDIS_DB` | `0` | Redis database number |
| `SMTP_HOST` | `localhost` | SMTP server host |
| `SMTP_PORT` | `1025` | SMTP server port |
| `SMTP_USER` | (empty) | SMTP auth username |
| `SMTP_PASSWORD` | (empty) | SMTP auth password |
| `SMTP_FROM` | `notifier@localhost` | Default sender address |
| `ADMIN_API_KEY` | `admin-key-change-me` | Admin API key (change in production!) |
| `WORKER_COUNT` | `5` | Number of worker goroutines per container |
| `MAX_RETRIES` | `3` | Max retry attempts before DLQ |
| `CONTAINER_ID` | (hostname) | Unique container identity for worker naming |
| `RUN_MIGRATIONS` | `true` | Run golang-migrate on startup |
| `MIGRATIONS_PATH` | `migrations` | Path to migration files |
| `REDIS_STREAM_NAME` | `email:jobs` | Redis stream for job queue |
| `REDIS_CONSUMER_GROUP` | `notifier-workers` | Consumer group name |
| `REDIS_DLQ_STREAM` | `email:dlq` | Dead letter queue stream |
| `HMAC_MASTER_KEY` | (generated in dev) | 64-char hex AES-256 key for encrypting HMAC secrets at rest |

## API Reference

### System

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check (DB + Redis). Returns 503 if unhealthy. |
| `GET` | `/metrics` | Prometheus metrics endpoint. |

### Dispatch (Consumer Auth)

All `/v1/*` routes require consumer authentication and are rate-limited (per-consumer + per-IP).

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/v1/send` | API Key or HMAC | Enqueue an email for delivery |
| `GET` | `/v1/jobs/{id}` | API Key or HMAC | Get job status |

**POST /v1/send** — Request body:

```json
{
  "to": ["user@example.com"],
  "subject": "Your report is ready",
  "body": "<html><body><h1>Report</h1></body></html>"
}
```

Response `202 Accepted`:

```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "queued"
}
```

**GET /v1/jobs/{id}** — Response:

```json
{
  "id": "550e8400-...",
  "consumer_id": "c_abc123",
  "status": "delivered",
  "to": "[\"user@example.com\"]",
  "subject": "Your report is ready",
  "delivered_at": "2025-01-01T12:00:00Z",
  "created_at": "2025-01-01T12:00:00Z"
}
```

### Admin

All `/admin/*` routes require the `ADMIN_API_KEY` as a Bearer token.

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/admin/consumers` | List all consumers |
| `POST` | `/admin/consumers` | Create a consumer (returns API key once) |
| `GET` | `/admin/consumers/{id}` | Get consumer details |
| `POST` | `/admin/consumers/{id}/suspend` | Suspend a consumer |
| `POST` | `/admin/consumers/{id}/reactivate` | Reactivate a consumer |
| `GET` | `/admin/jobs` | List jobs (filterable by `?consumer_id=&status=&limit=&offset=`) |
| `GET` | `/admin/dlq` | List dead letter queue entries (`?start=&end=&count=`) |
| `POST` | `/admin/dlq/{id}/replay` | Replay a DLQ message to main queue |

**POST /admin/consumers** — Request:

```json
{
  "name": "automater",
  "email_prefix": "automater-noreply"
}
```

Response `201 Created`:

```json
{
  "id": "c_uuid",
  "name": "automater",
  "email_prefix": "automater-noreply",
  "sender_email": "automater-noreply@yourdomain.com",
  "api_key": "nk_abc123...",
  "hmac_secret": "nsk_def456..."
}
```

> The `api_key` and `hmac_secret` are returned **only once**. Store them securely.

## Authentication

Notifier supports two authentication methods. HMAC is attempted first if the headers are present; otherwise it falls back to Bearer token.

### API Key Auth (Bearer Token)

```
Authorization: Bearer nk_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

- Keys are SHA-256 hashed on storage. The raw key is shown only at creation.
- Comparison uses `crypto/subtle.ConstantTimeCompare` to prevent timing attacks.

### HMAC Request Signing

For services that need stronger authentication:

```
X-Consumer-ID: c_uuid
X-Timestamp: 1700000000
X-Signature: <hex-encoded HMAC-SHA256>
```

The signature is computed as:

```
HMAC-SHA256(secret, "consumer_id:timestamp:canonicalJSON(body)")
```

Where `canonicalJSON` produces a deterministic, sorted-key JSON representation. The timestamp must be within 5 minutes of server time. The HMAC secret is encrypted at rest with AES-256-GCM using `HMAC_MASTER_KEY`.

## Key Concepts

### Consumers

A **consumer** represents an internal service authorized to send emails through Notifier. Each consumer has a unique name, email prefix, sender email (e.g., `automater-noreply@yourdomain.com`), API key and/or HMAC secret for authentication, an active/suspended status, and associated jobs (emails sent on its behalf).

### Job Queue & Workers

Jobs are persisted in PostgreSQL and enqueued to **Redis Streams** (`email:jobs`). A consumer group named `notifier-workers` distributes messages across all worker goroutines in all containers.

Each container runs `WORKER_COUNT` goroutines (default: 5). Each goroutine:
1. `XREADGROUP BLOCK 5000` — waits up to 5s for a message
2. Sends the email via SMTP
3. `XACK` on success, retries on failure

### Dead Letter Queue (DLQ)

After `MAX_RETRIES` failed delivery attempts, a job is moved to `email:dlq` — a separate Redis stream. Admin endpoints allow listing DLQ entries and replaying individual messages back to the main queue.

### Retry & Backoff

| Attempt | Backoff |
|---------|---------|
| 1 → 2 | 2 seconds |
| 2 → 3 | 4 seconds |
| 3 → 4 (DLQ) | 8 seconds |

### Rate Limiting

Two-layer rate limiting using Redis sorted sets (sliding window):

| Layer | Limit | Response |
|-------|-------|----------|
| Per consumer | 60 req/min | `429 Too Many Requests` + `Retry-After: 60` |
| Per IP | 120 req/min | `429 Too Many Requests` + `Retry-After: 60` |

### Abuse Detection

A background goroutine checks each active consumer every minute:
- Computes bounce rate: `(failed + bounced) / total jobs`
- If total jobs ≥ 10 and bounce rate > 20%: auto-suspends the consumer
- Admin can manually suspend/reactivate via API

## Database Migrations

Migrations use `golang-migrate` with file-based SQL:

```bash
make migrate-up       # Apply all pending migrations
make migrate-down     # Rollback last migration
```

Migration files are in `server/migrations/`:

| Migration | Tables |
|-----------|--------|
| `000001` | `consumers` |
| `000002` | `jobs` |
| `000003` | `audit_logs` |
| `000004` | `hmac_secret_encrypted` column on consumers |

## Observability

| Endpoint | Description |
|----------|-------------|
| `GET /health` | Liveness: checks PostgreSQL + Redis connectivity |
| `GET /metrics` | Prometheus metrics: request count, latency histogram, in-flight requests, queue depth, DLQ depth |

All server logs are structured JSON via Go's `log/slog`. Every request gets a unique request ID via chi's `RequestID` middleware.

## Deployment

### Docker

```bash
make docker-build   # Build the server image
make docker-up      # Start notifier + mailpit
make docker-down    # Stop all containers
```

### Production Considerations

- Change `ADMIN_API_KEY` to a strong random value
- Set `HMAC_MASTER_KEY` to a 64-char hex string (used to encrypt consumer HMAC secrets)
- Configure SMTP TLS/STARTTLS for your production relay
- Place behind a TLS-terminating load balancer (HTTPS)
- Set `RUN_MIGRATIONS=true` or run migrations separately via `make migrate-up`

## CI/CD

GitHub Actions runs on every PR:

- `go vet` + `staticcheck`
- `go test -short` with Redis service container
- Frontend TypeScript check + `npm run build`
- Docker build validation

## License

MIT
