# Notifier — Architecture & Tech Stack Discussion

## 1️⃣ Project Feasibility

**Yes, this is a well-scoped and highly feasible project.** The core value proposition is clear:

> *"One ingress point for all internal services that need to send email, instead of each service needing SMTP credentials or IP whitelisting."*

### What makes it feasible
- **Narrow scope**: A focused dispatch service — receive a request, validate it, send email, track status. No CMS, no complex UI.
- **Well-defined boundaries**: The only inputs are API calls (from internal services), the only output is SMTP email.
- **Real-world need**: Enterprises deal with this at scale — every microservice shouldn't need its own SMTP config, SPF/DKIM setup, and IP authorization.

### Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Abuse / spam (compromised API key) | Rate-limiting, per-consumer quotas, daily caps, abuse detection |
| Deliverability (emails landing in spam) | Proper SPF/DKIM/DMARC setup per sender domain |
| SMTP relay dependency | Support multiple SMTP providers (SendGrid, AWS SES, Postmark) as backends |
| Single point of failure | Stateless design + horizontal scaling behind a load balancer |

---

## 2️⃣ Tech Stack

### Backend — **Go** (strongly recommended)
- **Why**: Compiles to a single binary, excellent for a standalone service. Great performance for concurrent email processing. Built-in HTTP server, no framework dependency. Low memory footprint.
- **Alternatives**: Rust (more complex for this use case), Node.js/TypeScript (good for rapid prototyping but heavier, not ideal for a standalone long-running email service), Python (too slow for high-volume dispatch).

### API Layer
- **RESTful JSON API** — standard, simple, every consumer can call it.
- Health endpoint: `GET /health`
- Send email: `POST /v1/send`
- Job status: `GET /v1/jobs/{id}`
- Consumer management: `POST/GET /v1/consumers`

### Database — **PostgreSQL**
- Queue state, consumer keys, rate-limit counters, delivery logs, abuse metrics.

### Email Engine
- Use **Go's `net/smtp`** for direct SMTP, or wrap an SDK like **AWS SES**, **SendGrid**, **Mailgun**, or **Postmark** behind a common interface.

### Observability
- Structured JSON logging (slog or zerolog)
- Prometheus metrics (request count, rate, failures, delivery latency)

---

## 3️⃣ Storage & Streaming Tech

### Storage

| Data | Storage | Why |
|------|---------|-----|
| Consumers + API keys | PostgreSQL | Relational, ACID, easy to query |
| Email jobs (queue) | PostgreSQL (or Redis + PG for audit) | Job queue with status tracking |
| Rate-limit counters | **Redis** (TTL-based counters) | Fast, atomic INCR + EXPIRE, perfect for sliding window |
| Abuse metrics | PostgreSQL (time-series style) | Queryable for dashboards and audit trails |
| Delivery logs | PostgreSQL | Long-term audit trail |
| Attachments | **S3-compatible object storage** | Don't store in DB; reference by URL |

### Streaming — when do you need it?

At MVP stage, you **don't** need a stream processor. A simple in-process or DB-backed job queue is fine:

1. **In-process goroutine pool** (simplest): An internal buffered channel + worker goroutines. Non-blocking, fault-tolerant at the single-binary level.
2. **PostgreSQL-based queue** (next step): A `jobs` table with `status` (pending/processing/done/failed). Workers poll or use `LISTEN/NOTIFY`.
3. **Redis Streams** (for scale-out): If you need multiple instances, Redis Streams with consumer groups provide exactly-once delivery semantics.

**When to introduce streaming:**
- When you have **multiple Notifier instances** competing for jobs
- When you need **retry with backoff** and **dead-letter queues**
- Then add **Redis Streams** or **NATS JetStream** (lightweight, Go-native)

---

## 4️⃣ Authentication & Protection

### Authentication — API Key Based

```
POST /v1/consumers
Authorization: Bearer <master-admin-key>
Body: { "name": "automater", "email_prefix": "automater-noreply" }

→ Response: { "consumer_id": "c_abc123", "api_key": "nk_xxxxxxxxx", "sender": "automater-noreply@domain.com" }
```

**Key design choices:**
- **Prefix + high-entropy suffix**: `nk_c123f8a7b3...` — the prefix identifies the hash type / key scope
- **Hash on store**: Store only `SHA-256(api_key)` in DB. The raw key is shown **once** at creation.
- **Scoped**: Each API key is associated with exactly one consumer (one sender email).
- **Rotation**: Support key rotation with a grace period (two active keys per consumer).

### Protection

#### 4.1 Rate Limiting (mandatory)

```
Per consumer:
  - Max N emails/minute (sliding window via Redis)
  - Max M emails/hour
  - Max L emails/day
```

Enforce at **two levels**:
- **HTTP middleware**: Quick rejection for obvious bursts (allow/deny based on Redis counter)
- **Job creation**: Re-check before enqueuing

#### 4.2 Abuse Detection

```
Track per consumer:
  - Bounce rate (rejected / delivered)
  - Complaint rate (spam reports)
  - Recipient variety (are they emailing the same few users or spraying?)

→ Auto-suspend consumer if thresholds exceeded
```

#### 4.3 Anti-DDoS / IP Protection
- **Rate-limit per IP** (not just per API key) — prevents brute-force key guessing
- **Global rate limit** on `POST /v1/send` endpoint
- **Body size limit** (e.g., 10MB max, with multipart for attachments)
- **Connection pooling / keep-alive limits**

#### 4.4 Audit Trail

```
Every API call logged:
  - consumer_id, ip, endpoint, timestamp, status_code, job_id

Immutable audit table (append-only, no UPDATE/DELETE allowed).
```

#### 4.5 Job Status API

```
GET /v1/jobs/{job_id}
Authorization: Bearer <api_key>

→ {
    "id": "j_xxx",
    "status": "delivered" | "failed" | "pending" | "bounced",
    "consumer_id": "c_abc123",
    "to": ["user@example.com"],
    "subject": "...",
    "delivered_at": "...",
    "error": null
  }
```

---

## Summary Decision Matrix

| Concern | Recommended Tech |
|---------|-----------------|
| Language | **Go** |
| Database | **PostgreSQL** |
| Cache / Rate-limiter | **Redis** |
| Object storage | **S3-compatible** (for attachments) |
| Job queue (MVP) | In-process goroutine pool → PostgreSQL-based |
| Job queue (scale) | Redis Streams or NATS JetStream |
| Observability | Prometheus + structured JSON logs |
| Deployment | Docker container → any cloud / Kubernetes |
| Email backends | Interface pattern: support SMTP + SES + SendGrid |
