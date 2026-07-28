import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { AuthUser } from '@/types/api'

export type { AuthUser }

type AuthState = {
  accessToken: string | null
  refreshToken: string | null
  user: AuthUser | null
  setSession: (payload: {
    accessToken: string
    refreshToken: string
    user: AuthUser
  }) => void
  clearSession: () => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      accessToken: null,
      refreshToken: null,
      user: null,
      setSession: ({ accessToken, refreshToken, user }) =>
        set({ accessToken, refreshToken, user }),
      clearSession: () =>
        set({ accessToken: null, refreshToken: null, user: null }),
    }),
    { name: 'empops-auth' },
  ),
)
