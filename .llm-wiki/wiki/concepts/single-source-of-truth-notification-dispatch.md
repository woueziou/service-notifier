---
type: concept
created: 2026-06-06
updated: 2026-06-06
sources: [[[sources/SRC-2026-06-06-001]]]
---

# Single source of truth (notification dispatch)

Unified notification service that all other services send requests to, instead of each talking directly to a mail server.

## Definition

Instead of each enterprise service connecting directly to a [[mail-server]] (which requires IP authorization, SMTP credentials, and per-service maintenance), all requests are routed through [[entities/notifier]]. Notifier handles dispatch, authentication via [[concepts/api-key-based-authentication]], abuse detection via [[concepts/abuse-detection-anti-spam]], and job status tracking via [[concepts/job-status-tracking]].

## Related concepts

- [[concepts/abuse-detection-anti-spam]] — mandatory protections
- [[concepts/api-key-based-authentication]] — consumer authentication
- [[concepts/api-key-rotation]] — credential lifecycle

## Related entities

- [[entities/notifier]] — the service
- [[consumer]] — authenticated process

## Sources

- [[sources/SRC-2026-06-06-001]]
