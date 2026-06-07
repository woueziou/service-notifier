---
type: source
title: "Observation: Redis queue depth and DLQ depth gauges wired"
slug: obs-2026-06-07-redis-queue-depth-and-dlq-depth-gauges-wired
status: observation
created: 2026-06-07
updated: 2026-06-07
relevance: medium
observed_at: 2026-06-07T10:14:59.514Z
tags: ["metrics", "observability", "redis", "streams"]
source_context: "Phase 3 — Redis stream queue depth and DLQ depth gauges"
---
# 🔍 Observation: Redis queue depth and DLQ depth gauges wired
Wired the existing SetQueueDepth and SetDLQDepth metrics into production code. Removed the QueueDepthReporter stub in metrics.go that always returned 0. In main.go, the gauges now call rdb.XLen() on the job stream (email:jobs) and DLQ stream (email:dlq) on each Prometheus scrape. Made NewMetricsCollector's registration safe against duplicates by replacing MustRegister with registerSafe (gracefully handles AlreadyRegisteredError). Added 8 new tests in metrics_test.go covering gauge values, zero values, concurrent reads, and request in-flight counting.
*Relevance: medium*

*Context: Phase 3 — Redis stream queue depth and DLQ depth gauges*

*Tags: metrics observability redis streams*
---
*Observed: 2026-06-07T10:14:59.514Z*