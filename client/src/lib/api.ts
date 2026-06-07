const API_BASE = import.meta.env.VITE_API_URL || "/api"

/**
 * Base API fetch with cookie-based session auth.
 * On 401, redirects to /login.
 */
export async function api<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...options?.headers,
    },
  })
  if (res.status === 401) {
    window.location.href = "/login"
    throw new Error("Session expired")
  }
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    throw new Error(err.message || `HTTP ${res.status}`)
  }
  return res.json()
}

// --- Auth ---

export interface MeResponse {
  email: string
  role: "viewer" | "admin" | "super_admin"
  created_at: string
}

export interface AdminUser {
  id: string
  email: string
  role: "viewer" | "admin" | "super_admin"
  created_at: string
}

export async function requestLogin(email: string): Promise<void> {
  const res = await fetch(`${API_BASE}/auth/request-login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email }),
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    throw new Error(err.message || `Request failed (${res.status})`)
  }
  // Always returns { status: "check_your_email" } — no leak
}

export async function verifyLogin(email: string, code: string): Promise<void> {
  const res = await fetch(`${API_BASE}/auth/verify-login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, code }),
    credentials: "include",
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    throw new Error(err.message || `Verification failed (${res.status})`)
  }
  // Session cookie is set by the server
}

export async function logout(): Promise<void> {
  await fetch(`${API_BASE}/auth/logout`, {
    method: "POST",
    credentials: "include",
  })
}

export async function whoami(): Promise<MeResponse> {
  const res = await fetch(`${API_BASE}/auth/me`, {
    credentials: "include",
  })
  if (!res.ok) {
    throw new Error("Not authenticated")
  }
  return res.json()
}

export async function listAdminUsers(): Promise<AdminUser[]> {
  return api<AdminUser[]>("/auth/admin-users")
}

export async function addAdminUser(
  email: string,
  role: "viewer" | "admin"
): Promise<AdminUser> {
  return api<AdminUser>("/auth/admin-users", {
    method: "POST",
    body: JSON.stringify({ email, role }),
  })
}

export async function deleteAdminUser(id: string): Promise<void> {
  await api(`/auth/admin-users/${id}`, { method: "DELETE" })
}

// --- Types ---

export interface Consumer {
  id: string
  name: string
  email_prefix: string
  sender_email: string
  active: boolean
  suspended: boolean
  created_at: string
}

export interface CreateConsumerRequest {
  name: string
  email_prefix: string
}

export interface CreateConsumerResponse {
  id: string
  name: string
  email_prefix: string
  sender_email: string
  api_key: string
}

export interface Job {
  id: string
  consumer_id: string
  status: "pending" | "delivered" | "failed" | "bounced"
  to: string
  subject: string
  body?: string
  error?: string
  delivered_at?: string
  created_at: string
  updated_at: string
}

export interface JobListResponse {
  jobs: Job[]
  total: number
  limit: number
  offset: number
}

export interface DLQEntry {
  id: string
  fields: Record<string, string>
}

export interface DashboardStats {
  consumers: number
  jobs_today: number
  dlq_depth: number
}

// --- API functions ---

export async function listConsumers(): Promise<Consumer[]> {
  return api<Consumer[]>("/admin/consumers")
}

export async function getConsumer(id: string): Promise<Consumer> {
  return api<Consumer>(`/admin/consumers/${id}`)
}

export async function createConsumer(
  data: CreateConsumerRequest
): Promise<CreateConsumerResponse> {
  return api<CreateConsumerResponse>("/admin/consumers", {
    method: "POST",
    body: JSON.stringify(data),
  })
}

export async function listJobs(params?: {
  consumer_id?: string
  status?: string
  limit?: number
  offset?: number
}): Promise<JobListResponse> {
  const search = new URLSearchParams()
  if (params?.consumer_id) search.set("consumer_id", params.consumer_id)
  if (params?.status) search.set("status", params.status)
  if (params?.limit) search.set("limit", String(params.limit))
  if (params?.offset) search.set("offset", String(params.offset))
  const qs = search.toString()
  return api<JobListResponse>(`/admin/jobs${qs ? "?" + qs : ""}`)
}

export async function getJob(id: string): Promise<Job> {
  return api<Job>(`/admin/jobs/${id}`)
}

export async function suspendConsumer(id: string): Promise<void> {
  await api(`/admin/consumers/${id}/suspend`, { method: "POST" })
}

export async function reactivateConsumer(id: string): Promise<void> {
  await api(`/admin/consumers/${id}/reactivate`, { method: "POST" })
}

export async function listDLQ(params?: {
  count?: number
}): Promise<DLQEntry[]> {
  const search = new URLSearchParams()
  if (params?.count) search.set("count", String(params.count))
  const qs = search.toString()
  return api<DLQEntry[]>(`/admin/dlq${qs ? "?" + qs : ""}`)
}

export async function replayDLQ(id: string): Promise<void> {
  await api(`/admin/dlq/${id}/replay`, { method: "POST" })
}

// --- Stats ---

export interface ConsumerStats {
  id: string
  name: string
  active: boolean
  suspended: boolean
  sender_email: string
  total_jobs: number
  bounce_rate: number
  rate_limit_current: number
  rate_limit_max: number
}

export interface StatsSummary {
  total_consumers: number
  active_consumers: number
  suspended_consumers: number
  total_jobs: number
}

export interface StatsResponse {
  summary: StatsSummary
  consumers: ConsumerStats[]
}

export async function getStats(): Promise<StatsResponse> {
  return api<StatsResponse>("/admin/stats")
}
