---
type: concept
created: 2026-06-06
updated: 2026-06-06
sources: [[sources/SRC-2026-06-06-002]]
---

# HMAC request signing

Stateless authentication mechanism where a shared secret is established once per [[consumer]] at creation time. The consumer signs each request payload using HMAC-SHA256; the server recomputes and verifies the signature.

## Motivation

Replaces [[concepts/api-key-rotation]] entirely. The secret never changes, and auth is verified per-request without any session or token lifecycle. Designed for consumers that "just need to send an HTTP call" — no webhooks, no callbacks, no complex flows.

## How it works

### Consumer side (signing)

```python
import hmac, hashlib, json, time

SECRET = "nsk_..."  # shared once, never changes

def sign(consumer_id, body):
    ts = str(int(time.time()))
    msg = f"{consumer_id}:{ts}:{json.dumps(body, sort_keys=True)}"
    sig = hmac.new(SECRET.encode(), msg.encode(), hashlib.sha256).hexdigest()
    return {"X-Consumer-ID": consumer_id, "X-Timestamp": ts, "X-Signature": sig}
```

### Server side (verification)

Server looks up the consumer's stored HMAC secret (encrypted at rest), recomputes the signature, and compares in constant time. Also checks the timestamp is within a configurable clock skew (default 5 minutes) to prevent replay attacks.

## Headers

| Header | Value | Example |
|--------|-------|---------|
| `X-Consumer-ID` | Consumer UUID | `550e8400-...` |
| `X-Timestamp` | Unix timestamp | `1717000000` |
| `X-Signature` | Hex HMAC-SHA256 | `a1b2c3d4...` |

## Security properties

- **Secret never on wire** — only the signature is transmitted
- **Request integrity** — body is part of the signed message
- **Replay protection** — timestamp validation bounds the attack window
- **Stateless** — no session, no token, no rotation
- **No dependencies** — HMAC is in every standard library

## Implementation

Implemented in [[entities/notifier]] under `server/internal/auth/hmac.go`:

- `SignBody()` — produces HMAC-SHA256 signature over `consumerID:timestamp:canonicalBody`
- `VerifySignature()` — constant-time comparison of expected vs actual signature
- `CheckTimestamp()` — validates request age against clock skew
- `canonicalJSON()` — deterministic JSON serialization (sorted keys) so consumer and server produce identical signed messages

HMAC secrets are stored encrypted at rest using AES-256-GCM (`auth/secrets.go`) with a configurable master key (`HMAC_MASTER_KEY` env var).

## Related

- [[concepts/api-key-based-authentication]] — legacy Bearer token auth (still supported)
- [[concepts/api-key-rotation]] — the problem this approach eliminates
- [[sources/SRC-2026-06-06-002]]
