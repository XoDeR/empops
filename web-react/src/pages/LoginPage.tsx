import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useMutation } from '@tanstack/react-query'
import { Link, useNavigate } from 'react-router-dom'
import { apiFetch } from '@/lib/api'
import { useAuthStore } from '@/stores/auth'
import type { TokenPayload } from '@/types/api'

const loginSchema = z.object({
  email: z.string().email(),
  password: z.string().min(1, 'Password is required'),
})

type LoginValues = z.infer<typeof loginSchema>

export default function LoginPage() {
  const navigate = useNavigate()
  const setSession = useAuthStore((s) => s.setSession)

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<LoginValues>({ resolver: zodResolver(loginSchema) })

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
      navigate('/', { replace: true })
    },
  })

  return (
    <main className="mx-auto flex min-h-screen max-w-md flex-col justify-center gap-8 px-6 py-12">
      <header className="space-y-2">
        <p className="text-sm uppercase tracking-[0.2em] text-[var(--empops-accent)]">EmpOps</p>
        <h1 className="text-3xl font-semibold tracking-tight">Sign in</h1>
      </header>

      <form
        className="space-y-4 rounded-2xl border border-black/10 bg-white/80 p-5 shadow-sm"
        onSubmit={handleSubmit((values) => loginMutation.mutate(values))}
      >
        <label className="block space-y-1">
          <span className="text-sm">Email</span>
          <input
            className="w-full rounded-lg border border-black/15 bg-white px-3 py-2 outline-none focus:border-[var(--empops-accent)]"
            type="email"
            autoComplete="email"
            {...register('email')}
          />
          {errors.email && <span className="text-sm text-red-700">{errors.email.message}</span>}
        </label>
        <label className="block space-y-1">
          <span className="text-sm">Password</span>
          <input
            className="w-full rounded-lg border border-black/15 bg-white px-3 py-2 outline-none focus:border-[var(--empops-accent)]"
            type="password"
            autoComplete="current-password"
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
        <p className="text-center text-sm text-black/60">
          No account?{' '}
          <Link className="text-[var(--empops-accent)] underline" to="/register">
            Register
          </Link>
        </p>
      </form>
    </main>
  )
}
