# Language Choice: Go — Discussion Recap

## Decision: Go

**Unanimous choice.** Go is the right language for a standalone email dispatch service.

---

## 1️⃣ Performance

| Aspect | Why it matters for Notifier |
|--------|---------------------------|
| **Goroutines** | Each email send is a goroutine — lightweight (~2KB stack), millions possible. Perfect for async dispatch. |
| **HTTP throughput** | `net/http` serves thousands of concurrent API requests with negligible overhead. No framework tax. |
| **GC latency** | Low-pause GC (<1ms typically). Email dispatch tolerates micro-pauses; won't even notice them. |
| **Startup time** | ~10ms to ready. Important for containerized / auto-scaled deployments. |
| **Binary size** | ~10-20MB static binary. No JVM, no runtime, no npm. |

**Where Go wouldn't be ideal** (and why it doesn't matter here):
- CPU-bound numeric crunching → not applicable, we're I/O bound
- Real-time audio/video → not applicable
- Hot path needing single-digit microsecond latency → email delivery takes seconds, Go overhead is invisible

---

## 2️⃣ Implementation Architecture

```
┌─────────────────────────────────────────┐
│              HTTP Server                 │
│  ┌──────────┐  ┌──────────┐  ┌────────┐ │
│  │ Auth MW  │  │ Rate MW  │  │ Audit │ │
│  └────┬─────┘  └────┬─────┘  └────────┘ │
│       └──────┬──────┘                    │
│              ▼                           │
│  ┌──────────────────────┐                │
│  │    Handler Layer     │                │
│  │  POST /v1/send       │                │
│  │  GET  /v1/jobs/:id   │                │
│  │  POST /v1/consumers  │                │
│  └──────────┬───────────┘                │
│             ▼                            │
│  ┌──────────────────────┐                │
│  │   Service Layer      │                │
│  │  ConsumerService    │                │
│  │  DispatchService    │                │
│  │  AbuseDetection     │                │
│  └──────────┬───────────┘                │
│             ▼                            │
│  ┌──────────────────────┐  ┌──────────┐ │
│  │    Repository        │  │  Email   │ │
│  │  (PostgreSQL)        │  │  Engine  │ │
│  │                      │  │  (SMTP)  │ │
│  └──────────────────────┘  └──────────┘ │
│             ▼                            │
│  ┌──────────────────────┐                │
│  │   Worker Pool        │                │
│  │  (goroutines)        │                │
│  └──────────────────────┘                │
└─────────────────────────────────────────┘
```

### Key Patterns
- **Handler → Service → Repository** — three-layer, standard Go
- **Interface-driven email engine** — swap SMTP ↔ SES ↔ SendGrid easily
- **Middleware chain** — auth → rate-limit → audit log → handler
- **Worker pool** — channel-based job queue with configurable concurrency

---

## 3️⃣ Security

| Concern | Go Approach |
|---------|------------|
| **API key hashing** | `crypto/sha256` — hash on ingress, compare constant-time with `subtle.ConstantTimeCompare` |
| **TLS** | Built-in `crypto/tls` for HTTPS, SMTP with STARTTLS |
| **Input validation** | `go-playground/validator` for struct tags on request bodies |
| **SQL injection** | `database/sql` with parameterized queries (`$1`, `$2`) — never string concat |
| **CSRF/XSS** | REST API with Bearer tokens — not applicable (no cookies, no HTML rendering) |
| **Secrets management** | Load API keys, SMTP creds from env vars / Vault, never embed in binary |
| **Memory** | Auto-zeroed GC; no manual memory management (unlike C/C++) |

### Go-Specific Pitfalls to Avoid
- Don't `json.Unmarshal` into `interface{}` on untrusted input — limit struct sizes
- Set `ReadHeaderTimeout` / `ReadTimeout` / `WriteTimeout` on `http.Server` to prevent slow-loris
- Run `go vet`, `staticcheck`, and `gosec` in CI

---

## 4️⃣ Recommended Packages

| Category | Package | Why |
|----------|---------|-----|
| **HTTP router** | `chi` (go-chi/chi) | Lightweight, stdlib-compatible, middleware chains, no magic |
| **Logger** | `log/slog` (stdlib, Go 1.21+) | Structured, leveled, zero dependencies |
| **Validation** | `go-playground/validator/v10` | Struct tag validation for request bodies |
| **Database** | `pgx` (jackc/pgx) | Best PostgreSQL driver for Go, connection pooling, prepared stmts |
| **Migrations** | `golang-migrate/migrate` | File-based SQL migrations, CLI + library |
| **Config** | `caarlos0/env/v11` | Load config from env vars into struct |
| **Email** | `net/smtp` + custom SMTP wrapper | Standard library covers SMTP; wrap for SES/SendGrid later |
| **Rate limit** | `redis` (redis/go-redis/v9) | Atomic INCR with EXPIRE for sliding windows |
| **Metrics** | `prometheus/client_golang` | Standard exposition format |
| **Testing** | `testing` + `testify` | Stdlib + assert/require helpers |
| **API docs** | No package (write OpenAPI 3.0 YAML) | Document the contract, generate clients |

---

## 5️⃣ Project Structure

```
notifier/
├── cmd/
│   └── notifier/
│       └── main.go              # Entry point, wire up dependencies
├── internal/
│   ├── server/
│   │   ├── server.go            # HTTP server setup, middleware chain
│   │   ├── middleware.go         # Auth, rate-limit, audit, recovery
│   │   └── routes.go            # Route registration
│   ├── handler/
│   │   ├── consumer.go          # CRUD consumers
│   │   ├── dispatch.go          # POST /v1/send
│   │   └── job.go               # GET /v1/jobs/:id
│   ├── service/
│   │   ├── consumer.go          # Business logic for consumers
│   │   ├── dispatch.go          # Email sending orchestration
│   │   ├── abuse.go             # Abuse detection engine
│   │   └── ratelimit.go         # Rate limiting logic
│   ├── repository/
│   │   ├── consumer.go          # SQL queries for consumers
│   │   ├── job.go               # SQL queries for jobs
│   │   └── audit.go             # SQL queries for audit log
│   ├── engine/
│   │   ├── engine.go            # EmailEngine interface
│   │   ├── smtp.go              # SMTP implementation
│   │   └── ses.go               # AWS SES implementation (future)
│   ├── worker/
│   │   └── worker.go            # Goroutine pool for async dispatch
│   ├── model/
│   │   ├── consumer.go          # Consumer struct
│   │   ├── job.go               # Job struct
│   │   └── api.go               # Request/response types
│   ├── config/
│   │   └── config.go            # Env-driven config struct
│   └── auth/
│       └── apikey.go            # API key generation, hashing, verification
├── migrations/
│   ├── 000001_create_consumers.up.sql
│   ├── 000001_create_consumers.down.sql
│   ├── 000002_create_jobs.up.sql
│   └── ...
├── frontend/                    # Admin web app (separate section below)
├── Dockerfile
├── docker-compose.yml           # Notifier + PostgreSQL + Redis (dev)
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

**Why `internal/`?** Go enforces that packages under `internal/` are not importable by external code — perfect for an application you want to keep monolithic internally but not expose as a library.

---

## 6️⃣ Frontend — Admin Web Interface

Since a web interface is needed to manage consumers, view jobs, and monitor the system.

### Admin Panel Screens (MVP)

1. **Dashboard** — total sent vs failed, rate graphs, recent activity
2. **Consumers list** — view all consumers, their sender emails, status (active / suspended)
3. **Create consumer** — form to name + generate API key (shown once)
4. **Consumer detail** — job history, rate usage, bounce rate
5. **Jobs list** — search / filter by consumer, status, date range
6. **Job detail** — full delivery info, headers, error logs

### Frontend Tech Options

| Option | Why | Why not |
|--------|-----|---------|
| **SvelteKit** | Minimal boilerplate, reactive, small bundle, great DX | Smaller ecosystem |
| **React + Vite** | Huge ecosystem, shadcn/ui for components, more hiring pool | Heavier, more boilerplate |
| **HTMX + Go templates** | Zero JS complexity, server-rendered, one language | Limited interactivity, poor for dashboards |

### Auth for the Admin UI
- Frontend calls the same API as consumers, but with a **separate admin API key**
- Or: Notifier has `/admin/*` routes protected by a master admin key (simpler)
- No session / cookie auth — just Bearer token for simplicity
