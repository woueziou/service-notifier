---
type: concept
created: 2026-06-06
updated: 2026-06-06
sources: [[sources/SRC-2026-06-06-001]]
---

# API key rotation

Security practice of periodically replacing API keys with new ones and deactivating the old ones. A critical mechanism for limiting the damage window of leaked credentials.

## Definition

API key rotation is the process of issuing a new API key to a consumer and retiring the old one after a transition period. It ensures that even if a key is compromised (via logs, config files, or container breakout), the attacker's window of unauthorized access is bounded.

## Why rotation matters for Notifier

Notifier is an [[entities/notifier]] that sits between enterprise services and the [[mail-server]]. Each [[consumer]] is issued an [[concepts/api-key-based-authentication]] key. Since Notifier is deployed **on-premise via Docker**, API keys may be exposed through:

- Container environment variables or config files
- Application logs (accidental key logging)
- Compromised containers or hosts
- CI/CD pipeline leaks

Rotation reduces the blast radius of any single leak.

## Rotation patterns

### 1. Dual-key window (recommended)
Both the old and new keys are valid simultaneously for a configurable period. The consumer updates their configuration with the new key, then confirms the switch.

**Flow:**
1. Consumer calls `POST /api-keys/rotate` → server issues new key, marks old key as `rotating`
2. Consumer updates their config with the new key and tests it
3. Consumer calls `POST /api-keys/confirm-rotation` → old key is revoked
4. Server sends a notification to the consumer's registered contact

### 2. Grace period
The old key remains valid for N hours after rotation is requested, then auto-expires. No explicit confirmation needed — useful for distributed deployments where the new key takes time to propagate.

### 3. Immediate rotation (compromise response)
The old key is revoked the moment the new key is issued. Used when a security incident is suspected. May cause downtime for the consumer if they haven't updated their config yet.

## Design considerations for Notifier

| Requirement | Details |
|-------------|---------|
| **Rotation endpoint** | `POST /api-keys/rotate` to initiate, `POST /api-keys/confirm-rotation` to finalize |
| **Dual-key window** | Configurable per consumer (default: 1 hour) |
| **Notifications** | Email the consumer's registered address when rotation completes |
| **Audit trail** | Log every rotation: consumer ID, old key fingerprint, new key fingerprint, timestamp |
| **Mandatory schedule** | Force rotation every N days (configurable), warn the consumer N-7 days before expiry |
| **Force rotation** | Admin endpoint to instantly revoke all keys for a compromised consumer |

## Security properties

- **Forward secrecy for credentials**: a leaked old key cannot access the system after rotation
- **Compromise containment**: bounds the damage window to the rotation interval
- **Compliance**: satisfies standards requiring periodic credential rotation (SOC 2, PCI-DSS, ISO 27001)

## Note

API key rotation was the original design, but was **replaced by [[concepts/hmac-request-signing]]** as the primary auth mechanism. HMAC signing eliminates the need for rotation entirely — the shared secret never changes, and authentication is per-request.

## Related concepts

- [[concepts/hmac-request-signing]] — the approach that replaces rotation
- [[concepts/api-key-based-authentication]]
- [[concepts/abuse-detection-anti-spam]]
- [[concepts/job-status-tracking]]
- [[concepts/single-source-of-truth-notification-dispatch]]
- [[concepts/rate-limiting]]

## Related entities

- [[entities/notifier]]
- [[consumer]]
