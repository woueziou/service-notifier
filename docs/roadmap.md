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

## Phase 2: Hardening & Missing Features — 🔶 IN PROGRESS

### Rate Limiting ❌

The `RateLimiter` service exists (`service/ratelimit.go`) using Redis sorted sets (sliding window), but it's **not wired into any handler or middleware**.

**Tasks:**
- [ ] Add rate-limit middleware for `/v1/*` routes
- [ ] Apply per-consumer limits (configurable: N/min, M/hour, L/day)
- [ ] Apply per-IP limits for DDoS protection
- [ ] Apply global rate limit on `POST /v1/send`
- [ ] Return `429 Too Many Requests` with `Retry-After` header
- [ ] Expose rate-limit stats in admin API

### Abuse Detection ❌

**Tasks:**
- [ ] Track bounce rate per consumer (delivered vs failed)
- [ ] Track recipient variety (are they emailing the same few users or spraying?)
- [ ] Auto-suspend consumer when thresholds exceeded
- [ ] Admin endpoint to view abuse metrics
- [ ] Admin endpoint to manually suspend/reactivate consumer

### Request Validation ❌

The `go-playground/validator/v10` package is in the dependency list but not wired.

**Tasks:**
- [ ] Add validation middleware or call `validate.Struct(req)` in handlers
- [ ] Validate email format, subject length, body size
- [ ] Enforce maximum body size (e.g., 10MB with multipart for attachments)

### DDoS / IP Protection ❌

**Tasks:**
- [ ] Per-IP rate limiting in middleware
- [ ] Body size limit middleware
- [ ] Connection pooling / keep-alive limits on HTTP server

### Admin Jobs List Endpoint ❌

The frontend has a `/jobs` route but the backend has no admin jobs listing endpoint.

**Tasks:**
- [ ] Add `GET /admin/jobs` endpoint (paginated, filterable by consumer, status, date)
- [ ] Add `GET /admin/jobs?consumer_id=x` to filter by consumer
- [ ] Wire into frontend jobs list page

### DLQ Replay Bug 🔶

The DLQ replay handler replays messages back to the DLQ stream instead of the main `email:jobs` stream.

**Tasks:**
- [ ] Fix `ReplayDLQ` to accept the main stream name and XADD there
- [ ] Verify DLQ → main stream replay works end-to-end

### Graceful Worker Shutdown ❌

Workers use `context.Context` for cancellation, but the main.go sends SIGTERM to the HTTP server without coordinating worker shutdown.

**Tasks:**
- [ ] Propagate cancellation to all workers on SIGTERM
- [ ] Wait for in-flight messages to complete (finish SMTP → XACK) before exit
- [ ] Drain the XREADGROUP before stopping

---

## Phase 3: Production Readiness

### Observability

| Task | Priority | Notes |
|------|----------|-------|
| Prometheus metrics endpoint | High | `GET /metrics` — request count, latency, queue depth, worker count |
| Structured JSON logging | ✅ | Done via `log/slog` with JSON handler |
| Request ID tracing | ✅ | Done via `chimw.RequestID` |
| Redis stream monitoring | Medium | Expose `XLEN`, `XINFO GROUPS`, DLQ count via admin API |
| Sentry or error reporting | Low | Optional integration |

### Testing

| Task | Priority | Notes |
|------|----------|-------|
| Unit tests for auth/apikey.go | High | Generate, hash, verify, edge cases |
| Unit tests for RateLimiter | High | Sliding window logic |
| Unit tests for Worker | Medium | Message processing, retry, DLQ |
| Integration test: API → Redis → Worker → DB | Medium | Full flow test |
| Integration test: DLQ flow | Medium | Max retries → DLQ → replay |
| Integration test: Auth middleware | High | Valid key, invalid key, suspended consumer |
| Load test with multiple containers | Low | Horizontal scaling verification |

### CI/CD

| Task | Priority | Notes |
|------|----------|-------|
| GitHub Actions — Go build + vet | High | On every PR |
| GitHub Actions — frontend build | High | TypeScript + Vite build |
| GitHub Actions — run tests | High | `go test ./...` |
| Docker image build + push | Medium | Tagged releases |
| Database migration in CI | Low | Run `golang-migrate` in CI pipeline |

### Production Config

| Task | Priority | Notes |
|------|----------|-------|
| `golang-migrate` for production | High | Replace `AutoMigrate` with file-based migrations |
| Configurable SMTP TLS/STARTTLS | Medium | For production SMTP relays |
| S3-compatible attachment storage | Low | Reference attachments by URL |
| Kubernetes manifests | Medium | Deployment, Service, ConfigMap, HPA |
| Terraform / Pulumi infra | Low | If deploying to cloud |

### Security

| Task | Priority | Notes |
|------|----------|-------|
| API key rotation endpoint | Medium | Grace period with two active keys |
| Consumer suspension endpoint | High | Admin API to suspend abusive consumers |
| Rate-limit bypass protection | High | Ensure rate-limit state is **never** cached in-memory |
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
