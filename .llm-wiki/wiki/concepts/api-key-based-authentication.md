---
type: concept
created: 2026-06-06
updated: 2026-06-06
sources: [[[sources/SRC-2026-06-06-001]]]
---

# API-key based authentication

Authentication method where each consumer is issued an API key and assigned a unique sender email address.

## Definition

Each [[consumer]] is issued a unique API key at creation time. This key is included in every request header to Notifier, identifying which consumer is sending the notification and which sender email to use. Keys are hashed at rest (never stored in plaintext) and rotated periodically via [[concepts/api-key-rotation]].

## Related concepts

- [[concepts/api-key-rotation]] — periodic replacement of API keys
- [[concepts/abuse-detection-anti-spam]] — mandatory protections
- [[concepts/rate-limiting]] — request throttling per key

## Related entities

- [[consumer]] — the entity assigned an API key
- [[entities/notifier]] — the service that validates keys

## Sources

- [[sources/SRC-2026-06-06-001]]
