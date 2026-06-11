---
type: source
title: "Observation: CORS middleware added to Go server"
slug: obs-2026-06-08-cors-middleware-added-to-go-server
status: observation
created: 2026-06-08
updated: 2026-06-08
relevance: medium
observed_at: 2026-06-08T08:43:48.045Z
tags: ["server", "cors", "middleware", "config"]
source_context: "Adding CORS support for cross-origin client-server deployment"
---
# 🔍 Observation: CORS middleware added to Go server
Added CORSMiddleware to the Go notifier server (server/internal/server/middleware.go). Configurable via CORS_ORIGIN env var — supports comma-separated explicit origins or "*" (echoes request origin for credentialed support). Wired early in the fuego middleware chain (before Logger/Recovery) to handle OPTIONS preflight. Config struct updated with CORSOrigin field. Env files updated: .env, .env.prod, .env.prod.example. Client .env files updated with clear VITE_API_URL docs explaining same-origin vs cross-origin modes.
*Relevance: medium*

*Context: Adding CORS support for cross-origin client-server deployment*

*Tags: server cors middleware config*
---
*Observed: 2026-06-08T08:43:48.045Z*