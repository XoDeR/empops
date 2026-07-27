import { useForm } from 'react-hook-form'
import { z } from 'zod'
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery } from '@tanstack/react-query'
import { apiFetch } from '@/lib/api'
import { useAuthStore, type AuthUser } from '@/stores/auth'

const loginSchema = z.object({
  email: z.string().email(),
  password: z.string().min(1, 'Password is required'),
})

type LoginValues = z.infer<typeof loginSchema>

type TokenPayload = {
  access_token: string
  refresh_token: string
  expires_in: number
  token_type: string
  user: AuthUser
}

type HealthData = { status: string }
type MeData = AuthUser

export default function App() {
  const { accessToken, user, setSession, clearSession } = useAuthStore()

  const healthQuery = useQuery({
    queryKey: ['health'],
    queryFn: async () => {
      const res = await apiFetch<HealthData>('/health')
      return res.data
    },
    retry: false,
  })

  const meQuery = useQuery({
    queryKey: ['me', accessToken],
    enabled: Boolean(accessToken),
    queryFn: async () => {
      const res = await apiFetch<MeData>('/auth/me', { token: accessToken })
      return res.data
    },
  })

  const loginMutation = useMutation({
    mutationFn: async (values: LoginValues) => {
      const res = await apiFetch<TokenPayload>('/auth/login', {
        method: 'POST',
        body: JSON.stringify(values),
      })
      return res.data
    },
    onSuccess: (data) => {
      setSession({
        accessToken: data.access_token,
        refreshToken: data.refresh_token,
        user: data.user,
      })
    },
  })

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<LoginValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: { email: 'dev@empops.local', password: 'secret' },
  })

  return (
    <main className="mx-auto flex min-h-screen max-w-lg flex-col justify-center gap-8 px-6 py-12">
      <header className="space-y-2">
        <p className="text-sm uppercase tracking-[0.2em] text-[var(--empops-accent)]">EmpOps</p>
        <h1 className="text-4xl font-semibold tracking-tight">Platform shell</h1>
        <p className="text-base text-black/70">
          Step 0 React app — health check and stub JWT login against the Laravel API.
        </p>
      </header>

      <section className="rounded-2xl border border-black/10 bg-white/70 p-5 shadow-sm backdrop-blur">
        <h2 className="mb-2 text-sm font-semibold uppercase tracking-wide text-black/60">API health</h2>
        {healthQuery.isLoading && <p>Checking…</p>}
        {healthQuery.isError && (
          <p className="text-red-700">
            Unreachable. Start Laravel on :8000 or set <code>VITE_API_BASE_URL</code>.
          </p>
        )}
        {healthQuery.data && (
          <p className="text-[var(--empops-accent)]">Status: {healthQuery.data.status}</p>
        )}
      </section>

      {!accessToken ? (
        <form
          className="space-y-4 rounded-2xl border border-black/10 bg-white/80 p-5 shadow-sm"
          onSubmit={handleSubmit((values) => loginMutation.mutate(values))}
        >
          <h2 className="text-lg font-semibold">Sign in (stub)</h2>
          <label className="block space-y-1">
            <span className="text-sm">Email</span>
            <input
              className="w-full rounded-lg border border-black/15 bg-white px-3 py-2 outline-none focus:border-[var(--empops-accent)]"
              type="email"
              {...register('email')}
            />
            {errors.email && <span className="text-sm text-red-700">{errors.email.message}</span>}
          </label>
          <label className="block space-y-1">
            <span className="text-sm">Password</span>
            <input
              className="w-full rounded-lg border border-black/15 bg-white px-3 py-2 outline-none focus:border-[var(--empops-accent)]"
              type="password"
              {...register('password')}
            />
            {errors.password && (
              <span className="text-sm text-red-700">{errors.password.message}</span>
            )}
          </label>
          {loginMutation.isError && (
            <p className="text-sm text-red-700">{(loginMutation.error as Error).message}</p>
          )}
          <button
            type="submit"
            className="w-full rounded-lg bg-[var(--empops-accent)] px-4 py-2.5 font-medium text-white hover:opacity-90 disabled:opacity-60"
            disabled={loginMutation.isPending}
          >
            {loginMutation.isPending ? 'Signing in…' : 'Sign in'}
          </button>
        </form>
      ) : (
        <section className="space-y-3 rounded-2xl border border-black/10 bg-white/80 p-5 shadow-sm">
          <h2 className="text-lg font-semibold">Signed in</h2>
          <p className="text-sm text-black/70">
            {user?.name ?? meQuery.data?.name} · {user?.email ?? meQuery.data?.email}
          </p>
          {meQuery.data && (
            <pre className="overflow-x-auto rounded-lg bg-black/[0.04] p-3 text-xs">
              {JSON.stringify(meQuery.data, null, 2)}
            </pre>
          )}
          <button
            type="button"
            className="rounded-lg border border-black/15 px-4 py-2 text-sm hover:bg-black/[0.03]"
            onClick={() => clearSession()}
          >
            Sign out
          </button>
        </section>
      )}
    </main>
  )
}
