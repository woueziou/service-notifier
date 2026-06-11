---
type: source
title: "Observation: Server port now configurable via PORT env var end-to-end"
slug: obs-2026-06-11-server-port-now-configurable-via-port-env-var-end-to-end
status: observation
created: 2026-06-11
updated: 2026-06-11
relevance: medium
observed_at: 2026-06-11T15:36:32.218Z
tags: ["server", "docker", "fuego", "port"]
source_context: "Aligning Dockerfiles with docker-compose changes"
---
# 🔍 Observation: Server port now configurable via PORT env var end-to-end
The server's fuego HTTP server was hardcoded to port 8080 in routes.go. Fixed by adding Port field to ConfigAdapter, wiring cfg.Port from main.go through to fuego.WithAddr(fmt.Sprintf(":%d", cfg.Port)). Also updated server Dockerfile EXPOSE to use ARG PORT (default 8080) so it reflects the build-time default. The server now respects the PORT env var end-to-end, aligning with docker-compose's ${PORT:-8080} mapping. Build compiles cleanly.
*Relevance: medium*

*Context: Aligning Dockerfiles with docker-compose changes*

*Tags: server docker fuego port*
---
*Observed: 2026-06-11T15:36:32.218Z*