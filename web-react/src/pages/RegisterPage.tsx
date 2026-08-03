import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useMutation, useQuery } from '@tanstack/react-query'
import { Link, useNavigate } from 'react-router-dom'
import { apiFetch } from '@/lib/api'
import { useAuthStore } from '@/stores/auth'
import type { InstanceFlags, TokenPayload } from '@/types/api'

const registerSchema = z
  .object({
    name: z.string().min(1, 'Name is required'),
    email: z.string().email(),
    password: z.string().min(8, 'At least 8 characters'),
    password_confirmation: z.string().min(1, 'Please confirm your password'),
  })
  .refine((data) => data.password === data.password_confirmation, {
    message: 'Passwords do not match',
    path: ['password_confirmation'],
  })

type RegisterValues = z.infer<typeof registerSchema>

export default function RegisterPage() {
  const navigate = useNavigate()
  const setSession = useAuthStore((s) => s.setSession)
  const instanceQuery = useQuery({
    queryKey: ['instance'],
    queryFn: async () => (await apiFetch<InstanceFlags>('/instance')).data,
    retry: false,
  })

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<RegisterValues>({ resolver: zodResolver(registerSchema) })

  const registerMutation = useMutation({
    mutationFn: async (values: RegisterValues) => {
      const res = await apiFetch<TokenPayload>('/auth/register', {
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
        <h1 className="text-3xl font-semibold tracking-tight">Create your account</h1>
      </header>

      {instanceQuery.data && !instanceQuery.data.enable_signups ? (
        <section className="space-y-4 rounded-2xl border border-black/10 bg-white/80 p-5 text-center shadow-sm">
          <p className="font-medium">Registration is disabled for this instance.</p>
          <Link className="text-sm text-[var(--empops-accent)] underline" to="/login">Sign in instead</Link>
        </section>
      ) : <form
        className="space-y-4 rounded-2xl border border-black/10 bg-white/80 p-5 shadow-sm"
        onSubmit={handleSubmit((values) => registerMutation.mutate(values))}
      >
        <label className="block space-y-1">
          <span className="text-sm">Full name</span>
          <input
            className="w-full rounded-lg border border-black/15 bg-white px-3 py-2 outline-none focus:border-[var(--empops-accent)]"
            autoComplete="name"
            {...register('name')}
          />
          {errors.name && <span className="text-sm text-red-700">{errors.name.message}</span>}
        </label>
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
            autoComplete="new-password"
            {...register('password')}
          />
          {errors.password && (
            <span className="text-sm text-red-700">{errors.password.message}</span>
          )}
        </label>
        <label className="block space-y-1">
          <span className="text-sm">Confirm password</span>
          <input
            className="w-full rounded-lg border border-black/15 bg-white px-3 py-2 outline-none focus:border-[var(--empops-accent)]"
            type="password"
            autoComplete="new-password"
            {...register('password_confirmation')}
          />
          {errors.password_confirmation && (
            <span className="text-sm text-red-700">{errors.password_confirmation.message}</span>
          )}
        </label>
        {registerMutation.isError && (
          <p className="text-sm text-red-700">{(registerMutation.error as Error).message}</p>
        )}
        <button
          type="submit"
          className="w-full rounded-lg bg-[var(--empops-accent)] px-4 py-2.5 font-medium text-white hover:opacity-90 disabled:opacity-60"
          disabled={registerMutation.isPending || instanceQuery.isLoading}
        >
          {registerMutation.isPending ? 'Creating account…' : 'Register'}
        </button>
        <p className="text-center text-sm text-black/60">
          Already have an account?{' '}
          <Link className="text-[var(--empops-accent)] underline" to="/login">
            Sign in
          </Link>
        </p>
      </form>}
    </main>
  )
}
