# Notifier — Project Roadmap

> **Current status:** MVP scaffold complete. Core flow works end-to-end (API → Redis Stream → SMTP worker → delivered).  
> See `/docs/` for architecture discussions.

---

## Legend

| Icon | Meaning |
|------|---------|
| ✅ | Done |
| 🔶 | Partially done |
| ❌ | Not started |

---

## Phase 1: Core MVP — DONE ✅

| Feature | Status | Notes |
|---------|--------|-------|
| Go project structure | ✅ | chi + GORM + go-redis, clean `internal/` layout |
| PostgreSQL models & migrations | ✅ | `consumers`, `jobs`, `audit_logs` tables |
| Redis Streams job queue | ✅ | Consumer group `notifier-workers`, XADD/XREADGROUP/XACK |
| SMTP email engine | ✅ | `net/smtp` via `EmailEngine` interface |
| Worker pool (goroutines) | ✅ | N workers per container, configurable |
| Dead Letter Queue (DLQ) | ✅ | Separate `email:dlq` stream |
| Health endpoint | ✅ | `GET /health` — checks DB + Redis |
| Create consumer + API key | ✅ | SHA-256 hashed, constant-time compare |
| Send email (consumer auth) | ✅ | `POST /v1/send` → 202 Accepted |
| Job status lookup | ✅ | `GET /v1/jobs/:id` |
| Bearer token auth | ✅ | Consumer auth + admin auth middleware |
| Audit logging | ✅ | Per-request, append-only table |
| Retry with backoff | ✅ | Exponential backoff → DLQ after max retries |
| Frontend scaffold | ✅ | Vite + React + TanStack Router + Tailwind |
| Consumer CRUD (frontend) | ✅ | List, create (with one-time key), detail |
| Docker compose | ✅ | Mailpit + Notifier (postgres/redis external) |

---

## Phase 2: Hardening & Missing Features — ✅ DONE

### Rate Limiting ✅
- [x] Add rate-limit middleware for `/v1/*` routes
- [x] Apply per-consumer limits (60 req/min)
- [x] Apply per-IP limits (120 req/min) for DDoS protection
- [x] Return `429 Too Many Requests` with `Retry-After` header

### Abuse Detection ✅
- [x] Track bounce rate per consumer (delivered vs failed)
- [x] Auto-suspend consumer when bounce rate exceeds 20% threshold
- [x] Admin endpoint to manually suspend/reactivate consumer

### Request Validation ✅
- [x] Add validation middleware using go-playground/validator
- [x] Validate email format, required fields, length constraints
- [x] Enforce maximum body size (10 MB)

### Admin Jobs List Endpoint ✅
- [x] Add `GET /admin/jobs` endpoint (paginated, filterable by consumer, status)

### DLQ Replay Fix ✅
- [x] Fix ReplayDLQ to enqueue to job stream and delete from DLQ

### Graceful Worker Shutdown ✅
- [x] Propagate cancellation to all workers on SIGTERM
- [x] Wait for in-flight messages to complete before exit

---

## Phase 3: Production Readiness — 🔶 IN PROGRESS

### Observability

| Task | Priority | Notes |
|------|----------|-------|
| Prometheus metrics endpoint | ✅ | `GET /metrics` — request count, latency, in-flight gauge |
| Structured JSON logging | ✅ | Done via `log/slog` with JSON handler |
| Request ID tracing | ✅ | Done via `chimw.RequestID` |
| Redis stream monitoring (XLEN gauges) | Medium | Queue depth + DLQ depth gauges (wiring in progress) |
| Sentry or error reporting | Low | Optional integration |

### CI/CD

| Task | Priority | Notes |
|------|----------|-------|
| GitHub Actions — Go build + vet | ✅ | `go vet` + `staticcheck` on every PR |
| GitHub Actions — frontend build | ✅ | TypeScript check + `npm run build` |
| GitHub Actions — run tests | ✅ | `go test -short` with Redis service container |
| GitHub Actions — Docker build | ✅ | `docker build` validation |
| Docker image push to registry | Medium | Add login + push steps |

### Production Config

| Task | Priority | Notes |
|------|----------|-------|
| `golang-migrate` for production | ✅ | File-based migrations, configurable via `RUN_MIGRATIONS`/`MIGRATIONS_PATH` |
| Configurable SMTP TLS/STARTTLS | Medium | For production SMTP relays |
| Kubernetes manifests | Medium | Deployment, Service, ConfigMap, HPA |

### Security

| Task | Priority | Notes |
|------|----------|-------|
| API key rotation endpoint | Medium | Grace period with two active keys |
| HTTPS / TLS on API | High | Terminate at load balancer or configure cert in server |

---

## Phase 4: Frontend Completion

### Admin Dashboard

| Screen | Status | Notes |
|--------|--------|-------|
| Dashboard with stats | 🔶 | Placeholder cards exist, need real data fetching |
| Consumers list | ✅ | Works |
| Consumer detail | ✅ | Works |
| Create consumer | ✅ | Works with one-time key display |
| Jobs list | ❌ | Backend endpoint + frontend table needed |
| Job detail | ❌ | Backend + frontend needed |
| DLQ viewer | ❌ | Backend exists, frontend missing |
| Rate-limit / abuse stats | ❌ | After backend is complete |

### Technical

| Task | Notes |
|------|-------|
| Generate TanStack Router route tree | `npm run dev` generates `routeTree.gen.ts` |
| Wire React Query for data fetching | Hooks already used in consumers page |
| Add shadcn/ui components | Button, Card, Table, Dialog, Toast |
| Dark mode | Optional, but nice to have |
| Proper error handling / toasts | User-friendly error display |

---

## Summary: Priority Matrix

```
                    URGENT
                      │
        Rate limiting  │  DLQ replay fix
        Abuse detection│  Graceful shutdown
        Input validation│  Admin jobs endpoint
        ───────────────┼──────────────────
        Frontend polish│  S3 attachments
        Prometheus     │  K8s manifests
        Production CI  │  Terraform
        TLS            │
                      │
                    LATER
```

### Immediate next steps (what to do now):

1. **Wire rate limiting** into the `/v1/*` middleware — the service exists, just needs to be plumbed
2. **Fix DLQ replay** — wrong stream target in `admin.go`
3. **Add admin jobs listing endpoint** — needed for frontend
4. **Graceful worker shutdown** — proper SIGTERM handling
5. **Write tests** — start with auth and rate-limiter unit tests
