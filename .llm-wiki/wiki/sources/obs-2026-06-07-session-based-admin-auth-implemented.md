---
type: source
title: "Observation: Session-based admin auth implemented"
slug: obs-2026-06-07-session-based-admin-auth-implemented
status: observation
created: 2026-06-07
updated: 2026-06-07
relevance: high
observed_at: 2026-06-07T18:11:03.785Z
tags: ["auth", "backend", "frontend", "session"]
source_context: "Implementing username/password auth for admin dashboard"
---
# ⭐ Observation: Session-based admin auth implemented
Replaced the shared static ADMIN_API_KEY with session-based username/password authentication for the admin dashboard.

Backend changes:
- server/internal/model/admin_user.go — AdminUser GORM model (username, bcrypt-hashed password)
- server/migrations/000005_create_admin_users.up/down.sql — new migration
- server/internal/auth/password.go — bcrypt hash + verify helpers
- server/internal/repository/admin_user.go — FindByUsername, Create, Count
- server/internal/handler/module_auth.go — POST /auth/login, POST /auth/logout, GET /auth/me, SessionAuthMiddleware. Sessions stored in Redis with 8h TTL. HTTP-only, SameSite=Strict cookie.
- server/internal/config/config.go — removed ADMIN_API_KEY, added ADMIN_DEFAULT_USERNAME/ADMIN_DEFAULT_PASSWORD
- server/internal/server/routes.go — wired AuthModule, replaced AdminAuthMiddleware with SessionAuthMiddleware
- server/cmd/notifier/main.go — seeds default admin on first run if ADMIN_DEFAULT_PASSWORD is set

Frontend changes:
- client/src/routes/login.tsx — new login page with username/password form
- client/src/lib/api.ts — removed VITE_ADMIN_KEY, all calls use credentials:'include' for cookie auth. Added login(), logout(), whoami() helpers. On 401, redirects to /login.
- client/src/routes/__root.tsx — auth guard on mount (calls whoami(), redirects to /login if unauthenticated). Added Logout button and username display in nav.

Default credentials: admin / admin123 (set via .env)
*Relevance: high*

*Context: Implementing username/password auth for admin dashboard*

*Tags: auth backend frontend session*
---
*Observed: 2026-06-07T18:11:03.785Z*