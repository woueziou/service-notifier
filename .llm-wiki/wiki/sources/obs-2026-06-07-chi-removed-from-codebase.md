---
type: source
title: "Observation: Chi removed from codebase"
slug: obs-2026-06-07-chi-removed-from-codebase
status: observation
created: 2026-06-07
updated: 2026-06-07
relevance: high
observed_at: 2026-06-07T14:54:37.019Z
tags: ["chi", "fuego", "middleware", "cleanup"]
source_context: "Removing chi from codebase after Fuego migration"
---
# ⭐ Observation: Chi removed from codebase
Removed all go-chi/chi v5 references from the codebase. Replaced chimw.RequestID, chimw.RealIP, chimw.Recoverer with custom net/http middleware in middleware.go (RequestIDMiddleware, RealIPMiddleware, RecoveryMiddleware). Replaced chi.RouteContext in metrics.go with Go 1.22+ r.Pattern. Removed go-chi/chi from go.mod and go.sum. Also removed swaggo/swag dependency (no longer needed with Fuego's auto-generated OpenAPI).
*Relevance: high*

*Context: Removing chi from codebase after Fuego migration*

*Tags: chi fuego middleware cleanup*
---
*Observed: 2026-06-07T14:54:37.019Z*