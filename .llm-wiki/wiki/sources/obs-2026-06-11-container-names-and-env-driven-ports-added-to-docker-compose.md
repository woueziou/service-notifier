---
type: source
title: "Observation: Container names and env-driven ports added to docker-compose"
slug: obs-2026-06-11-container-names-and-env-driven-ports-added-to-docker-compose
status: observation
created: 2026-06-11
updated: 2026-06-11
relevance: medium
observed_at: 2026-06-11T15:22:18.652Z
tags: ["docker-compose", "devops", "networking"]
source_context: "Adding container names and env-driven ports to docker-compose"
---
# 🔍 Observation: Container names and env-driven ports added to docker-compose
Added container_name to all services in docker-compose.yml (notifier-mailpit, notifier-server, notifier-client) and docker-compose.prod.yml (notifier-server, notifier-client). Moved hardcoded host ports into .env variables: MAILPIT_SMTP_PORT, MAILPIT_UI_PORT for mailpit, CLIENT_PORT for notifier-client. Added fallback defaults via :- syntax so compose works without .env. Updated .env, .env.prod, and .env.prod.example with the new variables under a proper 'Client' section.
*Relevance: medium*

*Context: Adding container names and env-driven ports to docker-compose*

*Tags: docker-compose devops networking*
---
*Observed: 2026-06-11T15:22:18.652Z*