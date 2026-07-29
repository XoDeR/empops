const API_BASE = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8000/api/v1'

/** Resolve avatar/logo URLs from API (absolute or relative). */
export function resolveMediaUrl(path: string | null | undefined): string | null {
  if (!path) return null
  if (path.startsWith('http://') || path.startsWith('https://')) return path
  const origin = new URL(API_BASE).origin
  return `${origin}${path.startsWith('/') ? path : `/${path}`}`
}
