---
type: source
title: "Observation: Migrated from chi to Fuego with modular architecture"
slug: obs-2026-06-07-migrated-from-chi-to-fuego-with-modular-architecture
status: observation
created: 2026-06-07
updated: 2026-06-07
relevance: high
observed_at: 2026-06-07T13:23:00.722Z
tags: ["fuego", "migration", "openapi", "modular", "architecture"]
source_context: "chi to Fuego migration"
---
# ⭐ Observation: Migrated from chi to Fuego with modular architecture
Complete migration from go-chi/chi to go-fuego/fuego v0.19.0. Router replaced with fuego's net/http handler. Each API concern is now a self-contained module with its own file and Register(s, middlewares...) method. Modules: module_health, module_dispatch, module_consumer, module_admin, module_stats. OpenAPI spec is auto-generated from Go route definitions — no more swaggo annotations or swag init step. Old handler files (admin.go, consumer.go, dispatch.go, health.go, stats.go, docs.go), validate.go, and helpers_test.go removed. Helpers.go kept for ConsumerContextKey/getConsumer used by middleware. Removed docs/ and swaggo dependency. The Makefile swag target removed. Fuego's WithAddr set to :8080 in routes.go. OpenAPI spec available at /swagger/openapi.json with docs UI at /swagger/index.html (using Stoplight Elements). All 12 API routes auto-documented. Backend build + all tests pass.
*Relevance: high*

*Context: chi to Fuego migration*

*Tags: fuego migration openapi modular architecture*
---
*Observed: 2026-06-07T13:23:00.722Z*