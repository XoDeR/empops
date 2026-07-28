import type { ApiEnvelope } from '@/types/api'

const API_BASE = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8000/api/v1'

export class ApiError extends Error {
  status: number

  constructor(message: string, status: number) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

export async function apiFetch<T>(
  path: string,
  options: RequestInit & { token?: string | null } = {},
): Promise<ApiEnvelope<T>> {
  const { token, headers, ...rest } = options
  const res = await fetch(`${API_BASE}${path}`, {
    ...rest,
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...headers,
    },
  })

  let body: ApiEnvelope<T>
  try {
    body = (await res.json()) as ApiEnvelope<T>
  } catch {
    throw new ApiError(`Request failed (${res.status})`, res.status)
  }

  if (!res.ok || !body.success) {
    throw new ApiError(body.message || `Request failed (${res.status})`, res.status)
  }
  return body
}

export { API_BASE }
export type { ApiEnvelope }
