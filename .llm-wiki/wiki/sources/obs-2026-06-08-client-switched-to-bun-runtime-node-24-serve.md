---
type: source
title: "Observation: Client switched to Bun runtime + Node 24 serve"
slug: obs-2026-06-08-client-switched-to-bun-runtime-node-24-serve
status: observation
created: 2026-06-08
updated: 2026-06-08
relevance: medium
observed_at: 2026-06-08T08:39:00.881Z
tags: ["docker", "bun", "node24", "client", "env"]
source_context: "Switching client to Bun runtime with Node 24 for serving"
---
# 🔍 Observation: Client switched to Bun runtime + Node 24 serve
Client Dockerfile now uses `oven/bun:alpine` for the build stage (bun install + bun run build) and `node:24-alpine` for the serve stage. Client package.json scripts use `bunx --bun vite`. Client .env and .env.example created with VITE_API_URL=/api. Docker compose files updated to pass VITE_API_URL as build arg from root .env and mount client/.env. bun.lockb generated.
*Relevance: medium*

*Context: Switching client to Bun runtime with Node 24 for serving*

*Tags: docker bun node24 client env*
---
*Observed: 2026-06-08T08:39:00.881Z*