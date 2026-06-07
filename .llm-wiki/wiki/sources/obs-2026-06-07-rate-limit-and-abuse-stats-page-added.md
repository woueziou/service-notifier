---
type: source
title: "Observation: Rate limit and abuse stats page added"
slug: obs-2026-06-07-rate-limit-and-abuse-stats-page-added
status: observation
created: 2026-06-07
updated: 2026-06-07
relevance: high
observed_at: 2026-06-07T11:27:54.382Z
tags: ["frontend", "backend", "stats", "rate-limit", "abuse"]
source_context: "Phase 4 — rate limit and abuse stats page"
---
# ⭐ Observation: Rate limit and abuse stats page added
Added GET /admin/stats endpoint returning per-consumer rate-limit (current count in sliding window) and abuse stats (bounce rate, total jobs, suspended status). Added RateLimiter.GetCurrentCount() method that peeks at the window without consuming a slot. Frontend stats page at /stats with summary cards (total consumers, active, suspended, total jobs) and per-consumer table showing status badges, bounce rate color-coding, and rate limit progress bars that change color based on utilization. Auto-refreshes every 15s.
*Relevance: high*

*Context: Phase 4 — rate limit and abuse stats page*

*Tags: frontend backend stats rate-limit abuse*
---
*Observed: 2026-06-07T11:27:54.382Z*