import { apiFetch, ApiError } from '@/lib/api'
import { useAuthStore } from '@/stores/auth'
import type { ApiEnvelope, TokenPayload } from '@/types/api'

let refreshPromise: Promise<string | null> | null = null

async function refreshAccessToken(): Promise<string | null> {
  const { refreshToken } = useAuthStore.getState()
  if (!refreshToken) return null

  if (!refreshPromise) {
    refreshPromise = apiFetch<TokenPayload>('/auth/refresh', {
      method: 'POST',
      body: JSON.stringify({ refresh_token: refreshToken }),
    })
      .then((res) => {
        useAuthStore.getState().setSession({
          accessToken: res.data.access_token,
          refreshToken: res.data.refresh_token,
          user: res.data.user,
        })
        return res.data.access_token
      })
      .catch(() => {
        useAuthStore.getState().clearSession()
        return null
      })
      .finally(() => {
        refreshPromise = null
      })
  }

  return refreshPromise
}

/**
 * Fetch wrapper for authenticated endpoints. Attaches the current access
 * token and, on a 401, attempts a single refresh before retrying once.
 */
export async function authFetch<T>(
  path: string,
  options: RequestInit = {},
): Promise<ApiEnvelope<T>> {
  const accessToken = useAuthStore.getState().accessToken

  try {
    return await apiFetch<T>(path, { ...options, token: accessToken })
  } catch (e) {
    if (e instanceof ApiError && e.status === 401) {
      const newToken = await refreshAccessToken()
      if (newToken) {
        return await apiFetch<T>(path, { ...options, token: newToken })
      }
    }
    throw e
  }
}
