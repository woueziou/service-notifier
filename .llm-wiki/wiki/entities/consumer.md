---
type: entity
created: 2026-06-06
updated: 2026-06-06
sources: [[sources/SRC-2026-06-06-001]]
---

# Consumer

An authenticated process or service that sends notifications via [[entities/notifier]].

## Overview

Each consumer is issued an [[concepts/api-key-based-authentication]] API key at creation time and assigned a unique sender email address (e.g., `automater-noreply@domain.com`). The sender email identifies which process the notification originates from.

## Key attributes

- **API key** — used to authenticate every request
- **Sender email** — unique email address for outbound notifications
- **Job status** — consumers can track the status of their sent notifications via [[concepts/job-status-tracking]]
- **Rate limits** — per-consumer throttling enforced by [[concepts/abuse-detection-anti-spam]]
- **Rotation schedule** — API keys are rotated periodically via [[concepts/api-key-rotation]]

## Related

- [[entities/notifier]]
- [[sources/SRC-2026-06-06-001]]
