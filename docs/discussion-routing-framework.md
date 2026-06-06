# Routing, API Docs, ORM & Monorepo — Discussion Recap

---

## 1️⃣ Go HTTP Router: `chi`

**Decision: `go-chi/chi`**

chi is the de facto Go router for production APIs. It's lightweight, stdlib-compatible, and supports everything we need without magic.

```go
r := chi.NewRouter()

// Global middleware chain
r.Use(middleware.Logger)
r.Use(middleware.Recoverer)
r.Use(myauth.Middleware)

// Route groups
r.Route("/v1", func(r chi.Router) {
    r.Post("/send", handler.SendEmail)
    r.Get("/jobs/{jobID}", handler.GetJob)

    r.Route("/consumers", func(r chi.Router) {
        r.Post("/", handler.CreateConsumer)
        r.Get("/", handler.ListConsumers)
        r.Get("/{consumerID}", handler.GetConsumer)
    })
})
```

**Why chi:**
- Middleware chaining — auth → rate-limit → audit, cleanly composable
- URL parameter extraction via `chi.URLParam(r, "jobID")`
- Route grouping — nest `/v1/consumers` under `/v1`
- 100% `net/http` compatible — no framework lock-in
- Most popular Go router

---

## 2️⃣ OpenAPI Docs: `swaggo/swag`

**Decision: Use `swaggo/swag` to generate OpenAPI 3.0 from Go comments.**

```go
// @Summary      Send an email
// @Description  Dispatch an email on behalf of a consumer
// @Tags         dispatch
// @Accept       json
// @Produce      json
// @Param        body  body  SendRequest  true  "Email payload"
// @Success      202   {object}  SendResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      429   {object}  ErrorResponse
// @Router       /v1/send [post]
func (h *DispatchHandler) SendEmail(w http.ResponseWriter, r *http.Request) { ... }
```

One command: `swag init` → generates `docs/swagger.json` + `docs/swagger.yaml`.

The generated docs can be:
- Served from the server itself (embedded Swagger UI)
- Used in the frontend to generate typed API clients (via `openapi-typescript`)

---

## 3️⃣ ORM: GORM

**Decision: Use `gorm.io/gorm` with `gorm.io/driver/postgres`.**

GORM simplifies CRUD, associations, and dev migrations. Production migrations still use file-based SQL.

```go
type Consumer struct {
    ID          string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
    Name        string `gorm:"uniqueIndex;not null"`
    EmailPrefix string `gorm:"not null"`
    SenderEmail string `gorm:"not null"`
    APIKeyHash  string `gorm:"not null"`
    Active      bool   `gorm:"default:true"`
    CreatedAt   time.Time
    UpdatedAt   time.Time
    Jobs        []Job `gorm:"foreignKey:ConsumerID"`
}

type Job struct {
    ID         string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
    ConsumerID string    `gorm:"index;not null"`
    Status     string    `gorm:"type:varchar(20);default:pending"`
    To         string    `gorm:"type:text;not null"`
    Subject    string    `gorm:"type:varchar(998)"`
    Body       string    `gorm:"type:text"`
    Error      string    `gorm:"type:text"`
    CreatedAt  time.Time
    UpdatedAt  time.Time
}
```

**GORM features to use:**
- `AutoMigrate` for dev
- `Preload` for eager loading relationships
- Hooks (`BeforeCreate` for UUID generation, key hashing)
- Scopes for reusable query filters

---

## 4️⃣ Auth: Bearer Token

**Decision: Simple `Authorization: Bearer <token>` header.**

```go
func AuthMiddleware(repo *repository.ConsumerRepo) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token := r.Header.Get("Authorization")
            if !strings.HasPrefix(token, "Bearer ") {
                http.Error(w, "missing token", http.StatusUnauthorized)
                return
            }
            key := strings.TrimPrefix(token, "Bearer ")
            consumer, err := repo.Authenticate(r.Context(), key)
            if err != nil {
                http.Error(w, "invalid token", http.StatusUnauthorized)
                return
            }
            ctx := context.WithValue(r.Context(), consumerKey, consumer)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

Key verification uses `crypto/subtle.ConstantTimeCompare` to prevent timing attacks.

---

## 5️⃣ Email Engine: SMTP Only (for now)

**Decision: Only `net/smtp` implementation initially.** The `EmailEngine` interface keeps the door open for SES/SendGrid later.

```go
// engine/engine.go
type EmailEngine interface {
    Send(ctx context.Context, msg *EmailMessage) error
}

// engine/smtp.go
type SMTPEngine struct {
    Host string
    Port int
    User string
    Pass string
}

func (e *SMTPEngine) Send(ctx context.Context, msg *EmailMessage) error {
    // smtp.SendMail(...)
}
```

---

## 6️⃣ Monorepo Structure

**Decision: Monorepo with `server/` (Go) + `client/` (Vite + React + TanStack Router).**

```
notifier/
├── server/                         # Go backend
│   ├── cmd/
│   │   └── notifier/
│   │       └── main.go
│   ├── internal/
│   │   ├── server/
│   │   │   ├── server.go
│   │   │   ├── middleware.go
│   │   │   └── routes.go
│   │   ├── handler/
│   │   │   ├── consumer.go
│   │   │   ├── dispatch.go
│   │   │   └── job.go
│   │   ├── service/
│   │   │   ├── consumer.go
│   │   │   ├── dispatch.go
│   │   │   └── ratelimit.go
│   │   ├── repository/
│   │   │   ├── consumer.go
│   │   │   └── job.go
│   │   ├── engine/
│   │   │   ├── engine.go       # EmailEngine interface
│   │   │   └── smtp.go         # SMTP implementation
│   │   ├── worker/
│   │   │   └── worker.go
│   │   ├── model/
│   │   │   ├── consumer.go
│   │   │   ├── job.go
│   │   │   └── api.go
│   │   ├── config/
│   │   │   └── config.go
│   │   └── auth/
│   │       └── apikey.go
│   ├── migrations/
│   │   ├── 000001_create_consumers.up.sql
│   │   ├── 000001_create_consumers.down.sql
│   │   ├── 000002_create_jobs.up.sql
│   │   └── 000002_create_jobs.down.sql
│   ├── docs/
│   │   ├── swagger.json        # Generated by swaggo
│   │   └── swagger.yaml
│   ├── go.mod
│   └── go.sum
│
├── client/                        # Vite + React + TanStack Router
│   ├── src/
│   │   ├── routes/
│   │   │   ├── __root.tsx
│   │   │   ├── index.tsx          # Dashboard
│   │   │   ├── consumers/
│   │   │   │   ├── index.tsx      # Consumer list
│   │   │   │   ├── $consumerId.tsx # Consumer detail
│   │   │   │   └── create.tsx     # Create consumer
│   │   │   └── jobs/
│   │   │       ├── index.tsx      # Jobs list
│   │   │       └── $jobId.tsx     # Job detail
│   │   ├── components/
│   │   │   ├── ui/               # shadcn/ui components
│   │   │   ├── layout.tsx        # App shell
│   │   │   └── ...
│   │   ├── lib/
│   │   │   ├── api.ts            # Fetch wrapper with Bearer token
│   │   │   └── auth.ts
│   │   ├── main.tsx
│   │   └── routeTree.gen.ts      # Generated by TanStack Router
│   ├── public/
│   ├── index.html
│   ├── package.json
│   ├── vite.config.ts
│   ├── tsconfig.json
│   └── tailwind.config.ts
│
├── docker-compose.yml             # PostgreSQL + Redis + server
├── Makefile
└── README.md
```

### API Client Setup (Frontend)

```ts
// client/src/lib/api.ts
const API_BASE = import.meta.env.VITE_API_URL || "http://localhost:8080"
const ADMIN_KEY = import.meta.env.VITE_ADMIN_KEY

export async function api<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${ADMIN_KEY}`,
      ...options?.headers,
    },
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    throw new Error(err.message || `HTTP ${res.status}`)
  }
  return res.json()
}
```

**TanStack Router file-based routing** mirrors the Go API structure:
- `/consumers` → consumer list
- `/consumers/$consumerId` → consumer detail
- `/jobs` → job list
- `/jobs/$jobId` → job detail
- `/` → dashboard

---

## Summary of All Decisions

| Concern | Decision |
|---------|----------|
| Language | **Go** |
| HTTP Router | **`chi`** |
| ORM | **GORM** (with `pgx` driver) |
| API Docs | **`swaggo/swag`** (generated from comments) |
| Auth | **Bearer token** (`Authorization: Bearer <key>`) |
| Email Engine | **SMTP only** (`net/smtp`) via interface |
| Database | **PostgreSQL** |
| Cache / Rate-limit | **Redis** |
| Frontend | **Vite + React + TanStack Router + shadcn/ui** |
| Monorepo | `server/` + `client/` under root |
