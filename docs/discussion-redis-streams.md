# Redis Streams for Job Queue & DLQ — Discussion Recap

---

## Why Redis Streams Fits This Use Case

**Decision: Replace the in-process goroutine pool with Redis Streams for cross-container job distribution.**

| Requirement | How Redis Streams solves it |
|-------------|---------------------------|
| Horizontal scaling | Consumer groups distribute messages across containers — each message goes to exactly one consumer |
| Reliability | Pending Entries List (PEL) — unacknowledged messages are re-delivered on failure |
| Ordering | Streams are append-only, ordered logs |
| Retries | XAUTOCLAIM re-processes abandoned messages |
| DLQ | Separate stream for failed messages after max retries |
| No extra infra | Redis is already in the stack for rate-limiting |

---

## Architecture Overview

```
                      ┌──────────────────────┐
                      │   HTTP API Server     │
                      │  POST /v1/send        │
                      │  → validate & rate    │
                      │  → XADD email:jobs    │
                      │  → return 202 Accepted│
                      └──────────┬───────────┘
                                 │
                                 ▼
                      ┌──────────────────────┐
                      │    Redis Streams      │
                      │   "email:jobs"        │
                      │   Consumer Group:     │
                      │   "notifier-workers"  │
                      └──────┬─────────┬─────┘
                             │         │
                ┌────────────┘         └────────────┐
                ▼                                     ▼
     ┌──────────────────────┐         ┌──────────────────────┐
     │  Container 1         │         │  Container 2         │
     │  Worker goroutine x N │         │  Worker goroutine x N │
     │  XREADGROUP → SMTP   │         │  XREADGROUP → SMTP   │
     │  XACK on success     │         │  XACK on success     │
     │  XADD email:dlq on   │         │  XADD email:dlq on   │
     │  max retries         │         │  max retries         │
     └──────────────────────┘         └──────────────────────┘
                                 │
                                 ▼
                      ┌──────────────────────┐
                      │    Redis Streams      │
                      │   "email:dlq"         │
                      │   Dead Letter Queue   │
                      └──────────────────────┘
                                 │
                                 ▼
                      ┌──────────────────────┐
                      │   Admin / Frontend    │
                      │  View DLQ             │
                      │  Replay single job    │
                      │  Purge                │
                      └──────────────────────┘
```

---

## Data Flow

### 1. Ingest (API Handler → Stream)

```
POST /v1/send → validate → rate-check → XADD email:jobs * \
    consumer_id c_abc123 \
    to user@example.com \
    subject "Your report is ready" \
    body "<html>..." \
    max_retries 3
```

Returns `202 Accepted` with `{ "job_id": "j_uuid" }`. The actual sending is async.

### 2. Process (Worker → SMTP)

Each container runs N worker goroutines. Each worker loops:

```
loop:
    messages = XREADGROUP GROUP notifier-workers container-1-worker-0 \
                         COUNT 1 BLOCK 5000 STREAMS email:jobs >

    for msg in messages:
        send_email(msg)
        if success:
            XACK email:jobs notifier-workers msg.id
        else:
            retry_count = get_retry_count(msg)
            if retry_count >= max_retries:
                XADD email:dlq * \
                    original_job_id msg.id \
                    consumer_id ... \
                    error "SMTP timeout after 3 retries"
                XACK email:jobs notifier-workers msg.id
            else:
                XACK email:jobs notifier-workers msg.id
                // Re-enqueue with backoff
                XADD email:jobs * \
                    ... msg fields ...
                    retry_count retry_count + 1
```

### 3. Dead Letter Queue

```
XADD email:dlq * \
    original_job_id "1712345678000-0" \
    consumer_id "c_abc123" \
    to "user@example.com" \
    subject "Your report" \
    error "SMTP connection refused after 3 retries" \
    failed_at "2025-04-05T12:00:00Z"
```

DLQ supports:
- **List**: `XRANGE email:dlq - + COUNT 50`
- **Replay**: `XREAD` from DLQ, re-`XADD` to `email:jobs`
- **Purge**: `XTRIM email:dlq MINID ...` or `DEL email:dlq`

---

## Key Redis Streams Commands

| Operation | Command | Purpose |
|-----------|---------|---------|
| Enqueue | `XADD email:jobs * field1 val1 ...` | Add job to stream |
| Create group | `XGROUP CREATE email:jobs notifier-workers $ MKSTREAM` | Create consumer group (once) |
| Consume | `XREADGROUP GROUP notifier-workers my-worker COUNT 1 BLOCK 5000 STREAMS email:jobs >` | Read next unread message |
| Acknowledge | `XACK email:jobs notifier-workers msg-id` | Mark as processed |
| Claim abandoned | `XAUTOCLAIM email:jobs notifier-workers my-worker 30000 0-0 COUNT 10` | Reclaim messages stuck >30s |
| DLQ enqueue | `XADD email:dlq * ...` | Add to dead letter stream |
| List DLQ | `XRANGE email:dlq - + COUNT 50` | Browse dead letters |
| Stream info | `XINFO STREAM email:jobs` | Stream stats (length, groups) |
| Group info | `XINFO GROUPS email:jobs` | Consumer group status |
| Trim | `XTRIM email:jobs MAXLEN ~ 100000` | Cap stream size (prevent memory growth) |

---

## Horizontal Scaling — How It Works

```
        Redis Stream "email:jobs"
        Consumer Group: "notifier-workers"
        ┌────┬────┬────┬────┬────┬────┬────┬────┐
        │ m1 │ m2 │ m3 │ m4 │ m5 │ m6 │ m7 │ m8 │
        └─┬──┴─┬──┴─┬──┴─┬──┴─┬──┴─┬──┴─┬──┴─┬──┘
          │    │    │    │    │    │    │    │
    ┌─────┘    │    └────┼────┘    │    └────┼────┐
    ▼          ▼         ▼         ▼         ▼     ▼
 Container-1 (3 workers)    Container-2 (2 workers)
   w0  w1  w2                w0  w1
```

- Each consumer in the group gets a **unique name**: `{container-id}-{worker-index}`
- Each message is delivered to **exactly one consumer** in the group
- If Container-1 dies, its pending messages become idle → `XAUTOCLAIM` reassigns them to Container-2 after a timeout
- Adding more containers = more consumers = more throughput

**No sticky routing, no leader election, no coordination needed.**

---

## Within Each Container

Though the queue is external, within each container you still want controlled concurrency:

```
┌──────────────────────────────────────┐
│ Container                            │
│                                      │
│  ┌──────────┐  ┌──────────┐         │
│  │ Worker-0 │  │ Worker-1 │  ...    │
│  │ goroutine│  │ goroutine│         │
│  │ XREADGROUP│  │ XREADGROUP│        │
│  │ → SMTP   │  │ → SMTP   │         │
│  │ → XACK   │  │ → XACK   │         │
│  └──────────┘  └──────────┘         │
│                                      │
│  Config: WORKER_COUNT=5              │
│  Config: CONTAINER_ID=pod-abc123     │
└──────────────────────────────────────┘
```

The goroutine pool still exists — it's just that the work source is now Redis Streams instead of an in-process channel.

---

## Retry & Backoff Strategy

```
Attempt 1: send → fails → XACK + XADD with retry_count=1
Wait: 30 seconds (consumer will pick it up)

Attempt 2: send → fails → XACK + XADD with retry_count=2
Wait: 2 minutes

Attempt 3: send → fails → XACK + XADD with retry_count=3
Wait: 10 minutes

Attempt 4: send → fails → XADD to DLQ, XACK
```

Implementation: store `retry_count` and `next_attempt_after` as fields in the stream message. The worker checks these before processing.

Simpler alternative (no delayed retry): just XADD immediately with incremented retry_count. The worker checks at the top:

```go
if msg.retry_count > 0 {
    backoff := time.Duration(math.Pow(2, float64(msg.retry_count))) * time.Second
    age := time.Since(msg.timestamp)
    if age < backoff {
        XADD back to stream with same retry_count
        continue // let another worker pick it
    }
}
```

---

## DLQ Management (Admin API)

```
GET  /admin/dlq              → list dead letters (paginated)
POST /admin/dlq/{id}/replay  → re-enqueue to email:jobs
POST /admin/dlq/purge        → delete all dead letters
GET  /admin/dlq/stats        → DLQ count, oldest, by consumer
```

Backend implementation:

```go
func (h *AdminHandler) ListDLQ(w http.ResponseWriter, r *http.Request) {
    stream := r.URL.Query().Get("stream") // "email:dlq"
    start := r.URL.Query().Get("start")   // "-" for beginning
    end := r.URL.Query().Get("end")       // "+" for end
    count, _ := strconv.Atoi(r.URL.Query().Get("count"))
    
    entries, err := h.redis.XRange(ctx, stream, start, end, count).Result()
    // return as JSON
}

func (h *AdminHandler) ReplayDLQ(w http.ResponseWriter, r *http.Request) {
    msgID := chi.URLParam(r, "id")
    
    // Read the original message from DLQ
    entries, _ := h.redis.XRange(ctx, "email:dlq", msgID, msgID, 1).Result()
    
    // XADD to main stream (strip error fields)
    h.redis.XAdd(ctx, &redis.XAddArgs{
        Stream: "email:jobs",
        Values: cleanValues(entries[0].Values),
    })
    
    // Delete from DLQ
    h.redis.XDel(ctx, "email:dlq", msgID)
}
```

---

## Redis Memory Management

Streams grow unbounded without trimming. Configure:

```go
// Cap at 100K messages (more than enough for a stream that drains)
redis.XTrimMaxLen(ctx, "email:jobs", 100_000).Result()
// Or trim by min ID (keep last 1 hour)
redis.XTrimMinID(ctx, "email:jobs", oneHourAgoID).Result()
```

Run trim periodically (every 5 minutes) or on every N-th XADD.

---

## Container Identity

Each container needs a unique identity for consumer naming:

| Approach | How |
|----------|-----|
| Hostname | `os.Hostname()` — works out of the box in k8s/Docker |
| Env var | `CONTAINER_ID=pod-abc` — explicit |
| UUID | Generate on startup — simplest for dev |

```go
containerID, _ := os.Hostname() // "pod-abc123-def456"

for i := 0; i < config.WorkerCount; i++ {
    consumerName := fmt.Sprintf("%s-worker-%d", containerID, i)
    go worker.Run(ctx, redisClient, consumerName)
}
```

---

## Comparison: In-Process Channel vs Redis Streams

| Aspect | In-Process Channel | Redis Streams |
|--------|-------------------|---------------|
| Scaling | Single container | Any number of containers |
| Persistence | Lost on crash | Persistent (RDB/AOF) |
| Reliability | No retry mechanism | PEL + XAUTOCLAIM |
| DLQ | Manual | Separate stream |
| Complexity | Zero | Moderate (Redis Streams API) |
| Latency | Microseconds | ~1ms (network hop) |
| Observability | None built-in | XINFO, XLEN, debug tools |

**Verdict:** For a service that needs horizontal scaling from day one, Redis Streams is the right choice. The latency overhead (~1ms) is invisible compared to SMTP delivery time (100ms–5s).

---

## Stream Names Summary

| Stream | Consumer Group | Purpose |
|--------|---------------|---------|
| `email:jobs` | `notifier-workers` | Main job queue (all containers) |
| `email:dlq` | — | Dead letter queue (admin-inspected) |
