# Request: DNS Records for Notifier Deployment

**Project:** [woueziou/service-notifier](https://github.com/woueziou/service-notifier) / `flyasky.com`
**Priority:** High

---

## Description

We need internal DNS records configured for the **Notifier** service so it can be reached within the network. The app will be accessible internally first, with internet exposure planned for a later phase.

Notifier has two components that need to be reachable:

| Component | Internal port | What it serves |
|-----------|---------------|----------------|
| **notifier-client** | 80 | React admin dashboard (SPA) — manage consumers, view jobs, inspect DLQ |
| **notifier** (server) | 8055 | REST API — email dispatch, admin endpoints, health check |

Both run as Docker containers on Dokploy behind a shared reverse proxy (ingress).

---

## Requested Records

We use **separate subdomains** for the admin UI and the API:

| Type | Name | Target | Notes |
|------|------|--------|-------|
| A / CNAME | `notifier` | `[internal-ip-or-cname]` | Admin UI: `https://notifier.flyasky.com` |
| A / CNAME | `api.notifier` | `[internal-ip-or-cname]` | API: `https://api.notifier.flyasky.com` |

**Client config:** `VITE_API_URL=https://api.notifier.flyasky.com` — cross-origin, server has CORS configured.

---

## Future (out of scope for this ticket)

- Public internet exposure — update DNS target and add TLS certificate
- SMTP deliverability (SPF / DKIM / DMARC) — once sending from a custom domain

---

## Next steps

1. Confirm the internal IP or CNAME the records should point to
2. Create the two A / CNAME records above
3. Set TTL — recommend `300` (5 min) for initial rollout, `3600` (1 h) for steady state

---

*Ticket generated from project configuration. Adjust names and target IP/CNAME as needed.*
