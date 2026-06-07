---
type: source
title: "Observation: HMAC request signing implemented"
slug: obs-2026-06-06-hmac-request-signing-implemented
status: observation
created: 2026-06-06
updated: 2026-06-06
relevance: high
observed_at: 2026-06-06T20:02:15.133Z
tags: ["auth", "hmac", "security", "implementation"]
source_context: "Implementing HMAC request signing for Notifier"
---
# ⭐ Observation: HMAC request signing implemented
Implemented HMAC request signing as the primary authentication mechanism for Notifier. New files: server/internal/auth/hmac.go (signing/verification), server/internal/auth/secrets.go (AES-256-GCM encryption for secrets at rest). Updated: model/consumer.go (HMACSecretEncrypted field), repository/consumer.go (AuthenticateHMAC method + HMACSecretProvider interface), service/consumer.go (HMAC secret generation at consumer creation), server/middleware.go (dual auth — HMAC headers first, Bearer fallback), server/routes.go (wired SecretProvider), cmd/notifier/main.go (initSecretProvider with auto-generated dev key fallback), config/config.go (HMAC_MASTER_KEY env var), migrations/000004_add_hmac_secret_to_consumers. The secret never travels over the wire — only the signature does. Timestamp prevents replay attacks (5 min window). No API key rotation needed. Legacy Bearer token auth still works alongside for backward compatibility.
*Relevance: high*

*Context: Implementing HMAC request signing for Notifier*

*Tags: auth hmac security implementation*
---
*Observed: 2026-06-06T20:02:15.133Z*