const API_BASE = import.meta.env.VITE_API_URL || "/api"
const ADMIN_KEY = import.meta.env.VITE_ADMIN_KEY || "admin-key-change-me"

export async function api<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${ADMIN_KEY}`,
      ...options?.headers,
    },
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    throw new Error(err.message || `HTTP ${res.status}`)
  }
  return res.json()
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
  error?: string
  delivered_at?: string
  created_at: string
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

export async function listJobs(consumerId?: string): Promise<Job[]> {
  const params = consumerId ? `?consumer_id=${consumerId}` : ""
  return api<Job[]>(`/admin/jobs${params}`)
}
