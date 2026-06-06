---
type: entity
created: 2026-06-06
updated: 2026-06-06
sources: [[[sources/SRC-2026-06-06-001]]]
---

# Notifier

Standalone service that dispatches email and push notifications on behalf of other services.

## Overview

Notifier is deployed on-premise via Docker. It receives notification requests from [[consumer]]s — each authenticated via [[concepts/api-key-based-authentication]] — and dispatches them to the appropriate [[mail-server]] or push notification channel. It is the [[concepts/single-source-of-truth-notification-dispatch]] for all outgoing notifications.

### Key features

- **Email & push notification dispatch** — receives requests, handles delivery
- **API key auth** — each [[consumer]] is authenticated per request
- **Abuse detection** — anti-spam, anti-DDoS, rate limiting via [[concepts/abuse-detection-anti-spam]]
- **Job status tracking** — consumers can check delivery status via [[concepts/job-status-tracking]]
- **Key rotation** — API keys are rotated periodically via [[concepts/api-key-rotation]]

## Related

- [[consumer]] — authenticated processes
- [[mail-server]] — downstream email infrastructure
- [[sources/SRC-2026-06-06-001]]
