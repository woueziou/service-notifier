# Statelessness Analysis — Architecture Review

After reviewing all four discussion documents (`discussion-tech-stack.md`, `discussion-language-go.md`, `discussion-routing-framework.md`, `discussion-redis-streams.md`), here is the assessment.

---

## Verdict: ✅ The architecture IS stateless — with one caveat.

---

## Component-by-Component Analysis

| Component | State Location | Stateless? | Why |
|-----------|---------------|:----------:|-----|
| **HTTP API** (chi) | Nowhere | ✅ | No sessions, no cookies, no sticky routing |
| **Auth (Bearer token)** | PostgreSQL (hashed keys) | ✅ | Every request independently verified — no local cache |
| **Rate limiting** | Redis (TTL counters) | ✅ | Shared external state via atomic INCR/EXPIRE — no local counters |
| **Job queue** | Redis Streams | ✅ | **Key enabler** — `XADD` from handler, `XREADGROUP` from workers |
| **Worker pool** | Redis Streams | ✅ | Workers read from shared stream, not from local channels |
| **SMTP engine** | Config (env vars) | ✅ | Same credentials across all pods, fresh connections per message |
| **DLQ** | Redis Streams (`email:dlq`) | ✅ | Separate stream, admin-accessible from any container |
| **Audit log** | PostgreSQL | ✅ | Direct writes to shared DB, no local buffer |
| **GORM** | PostgreSQL | ✅ | Connection pool is local helper state (recreatable on restart) — no application data in memory |
| **Config** | Env vars | ✅ | Loaded identically on every startup from environment |
| **Prometheus metrics** | Per-container | ✅ | Normal — Prometheus scrapes each container independently |
| **OpenAPI docs** | Generated files | ✅ | Static artifacts, same in every container |

---

## What Makes It Stateless

### 1. No in-process job queue

The earliest architecture sketch showed an in-process channel-based goroutine pool. That would be **stateful** — messages buffered in-process would be lost on crash and couldn't be shared across containers.

**The switch to Redis Streams eliminates this entirely.** The queue lives in Redis, not in process memory.

### 2. No in-memory caches

API key lookups hit PostgreSQL on every request. Rate-limit counters live in Redis with TTL.

There is no:
- `sync.Map` for caching API key lookups
- Local `map[string]int64` for rate-limit state
- In-process job buffer
- Session store

### 3. No leader election

Every container is equal. Redis Streams consumer groups (`notifier-workers`) handle message distribution without any coordination protocol.

- No Raft
- No consensus
- No "primary" / "replica" roles
- No coordination service (no etcd, no Zookeeper)

### 4. Container identity is cosmetic

The `{container-id}-worker-{N}` naming convention is purely for:
- Observability (XINFO shows which consumers are active)
- Debugging (logs show which container processed which job)

It does **not** affect routing, correctness, or data ownership.

---

## The One Caveat: In-Memory Caching

**If we ever add in-memory caching of auth or rate-limit state, we lose statelessness.**

### ❌ Patterns that would break statelessness

```go
// BREAKS STATELESSNESS — Container A has key, Container B doesn't
var apiKeyCache = sync.Map{}

func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        key := extractKey(r)
        cached, ok := apiKeyCache.Load(key)
        if ok {
            // Container B never sees newly created keys
        }
        // ...
    })
}
```

```go
// BREAKS STATELESSNESS — Counts diverge across containers
type RateLimiter struct {
    mu      sync.Mutex
    counts  map[string]*slidingWindow  // local, not shared
}
```

```go
// BREAKS STATELESSNESS — Lost on restart, different per container
var inFlightJobs = make(map[string]bool)
```

### ✅ The rule

> **Every piece of state that affects request handling must live in PostgreSQL or Redis — never in process memory.**

If caching is absolutely needed (e.g., to reduce PostgreSQL load on auth), the cache **must** be in Redis with a short TTL:

```go
// ACCEPTABLE — shared Redis cache
func (r *ConsumerRepo) Authenticate(ctx context.Context, key string) (*model.Consumer, error) {
    hash := sha256.Sum256([]byte(key))
    cacheKey := fmt.Sprintf("auth:%x", hash)

    // Try Redis cache first
    var consumer model.Consumer
    err := r.redis.Get(ctx, cacheKey).Scan(&consumer)
    if err == nil {
        return &consumer, nil
    }

    // Fall through to PostgreSQL
    result := r.db.Where("api_key_hash = ?", hash[:]).First(&consumer)
    if result.Error != nil {
        return nil, result.Error
    }

    // Cache in Redis with short TTL (30s)
    r.redis.Set(ctx, cacheKey, consumer, 30*time.Second)
    return &consumer, nil
}
```

Even this is optional — for MVP, direct PostgreSQL lookups are perfectly fine.

---

## Behavior in Real Deployment

```
        ┌──────────┐
        │  Load    │  Round-robin or least-connections
        │  Balancer│  No sticky sessions needed
        └────┬─────┘
             │
     ┌───────┼───────┐
     ▼       ▼       ▼
 ┌──────┐ ┌──────┐ ┌──────┐
 │ Pod A│ │ Pod B│ │ Pod C│  All identical
 └──┬───┘ └──┬───┘ └──┬───┘
    │        │        │
    └────────┼────────┘
             ▼
       ┌──────────┐
       │  Redis   │  Streams + rate-limit counters
       │  Streams │  Consumer group: "notifier-workers"
       └──────────┘
             │
             ▼
       ┌──────────┐
       │PostgreSQL│  Consumers, keys, audit, job status
       └──────────┘
```

### Pod A goes down mid-request
- Load balancer redirects to Pod B/C — no session affinity needed
- Unacknowledged stream messages from Pod A become idle in PEL
- **XAUTOCLAIM** (or automatic timeout) reassigns them to Pod B/C
- No data loss, no manual intervention

### Scale from 2 to 10 pods
- New pods start, connect to Redis, join consumer group
- Messages distribute automatically across all consumers
- No reconfiguration, no restart of existing pods

### Rolling update
- Old pods receive SIGTERM → finish current message → XACK → exit
- New pods start fresh → join consumer group → pick up remaining messages
- Zero downtime

---

## Statelessness Checklist

| Criteria | Status | How it's achieved |
|----------|--------|-------------------|
| Any container can handle any request | ✅ | Bearer token → PostgreSQL lookup, no local session |
| No local state that differs between instances | ✅ | All shared state in Redis + PostgreSQL |
| Scaling = just add/remove containers | ✅ | Redis Streams consumer group handles distribution |
| Graceful shutdown without data loss | ✅ | XACK before exit, XAUTOCLAIM for abandoned messages |
| No sticky sessions required | ✅ | No cookies, no stateful sessions |
| Zero-downtime deployment | ✅ | Drain → shutdown → startup flow |
| No leader election | ✅ | All containers equal peers |

---

## Summary

The architecture **is stateless by design**, with Redis Streams as the critical enabler. Every container runs identical code, reads from shared external stores, and writes to shared external stores.

The only discipline needed going forward:
> **Never cache auth or rate-limit state in process memory.**
