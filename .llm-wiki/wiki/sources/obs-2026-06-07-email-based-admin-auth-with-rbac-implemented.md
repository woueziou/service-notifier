---
type: source
title: "Observation: Email-based admin auth with RBAC implemented"
slug: obs-2026-06-07-email-based-admin-auth-with-rbac-implemented
status: observation
created: 2026-06-07
updated: 2026-06-07
relevance: high
observed_at: 2026-06-07T18:48:14.642Z
tags: ["auth", "rbac", "email", "session", "backend", "frontend"]
source_context: "Reimplementing admin auth with email-based login and role-based access"
---
# ⭐ Observation: Email-based admin auth with RBAC implemented
Replaced password-based admin auth with email magic-link flow and three roles (super_admin, admin, viewer).

Backend:
- model/admin_user.go — email (unique), role enum, created_by
- migrations/000005_create_admin_users.up.sql — new table schema
- repository/admin_user.go — FindByEmail, List, Create, Delete
- handler/module_auth.go — POST /auth/request-login (sends 6-digit code via SMTP), POST /auth/verify-login (validates code → session), POST /auth/logout, GET /auth/me, GET/POST/DELETE /auth/admin-users (role-guarded). SessionAuthMiddleware + RequireRole (for specific routes) + RequireRoleForWrite (for all admin routes — viewers can GET, admins+ can POST/PUT/DELETE)
- config/config.go — ADMIN_SEED_EMAIL env var seeds first super_admin
- routes.go — wired auth module with SMTP engine, role-based middleware on admin routes
- main.go — seeds super_admin by email on first run
- Removed password.go (unused bcrypt helpers) and AdminAuthMiddleware (replaced by session auth)

Frontend:
- routes/login.tsx — two-step login: enter email → receive code → enter code
- lib/api.ts — requestLogin(), verifyLogin(), logout(), whoami(), listAdminUsers(), addAdminUser(), deleteAdminUser(). API calls use credentials:'include' for cookie auth.
- routes/__root.tsx — auth guard with role badge, conditional "Admins" nav link for admin+
- routes/admin-users.tsx — management page: list users, add (admin+), delete (super_admin only)

Role hierarchy: viewer (read-only) < admin (can add users) < super_admin (can add & delete users)
Session: stored in Redis with 8h TTL, HTTP-only cookie
Login code: 6-digit, 15min TTL, stored in Redis, sent via SMTP
*Relevance: high*

*Context: Reimplementing admin auth with email-based login and role-based access*

*Tags: auth rbac email session backend frontend*
---
*Observed: 2026-06-07T18:48:14.642Z*