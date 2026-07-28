import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { authFetch } from '@/lib/authFetch'
import { useCompanyStore } from '@/stores/company'
import type { CompanyMembership } from '@/types/api'

const createCompanySchema = z.object({
  name: z.string().min(1, 'Name is required'),
  currency: z
    .string()
    .trim()
    .refine((v) => v === '' || v.length === 3, 'Use a 3-letter currency code')
    .optional(),
})
type CreateCompanyValues = z.infer<typeof createCompanySchema>

const joinCompanySchema = z.object({
  code: z.string().min(1, 'Join code is required'),
})
type JoinCompanyValues = z.infer<typeof joinCompanySchema>

type CreateOrJoinResponse = {
  company: CompanyMembership
}

export default function CompaniesPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const setSelectedCompanyId = useCompanyStore((s) => s.setSelectedCompanyId)
  const [createdJoinCode, setCreatedJoinCode] = useState<string | null>(null)

  const companiesQuery = useQuery({
    queryKey: ['companies'],
    queryFn: async () => {
      const res = await authFetch<CompanyMembership[]>('/companies')
      return res.data
    },
  })

  const createForm = useForm<CreateCompanyValues>({
    resolver: zodResolver(createCompanySchema),
    defaultValues: { name: '', currency: 'EUR' },
  })

  const createMutation = useMutation({
    mutationFn: async (values: CreateCompanyValues) => {
      const res = await authFetch<CreateOrJoinResponse>('/companies', {
        method: 'POST',
        body: JSON.stringify({
          name: values.name,
          ...(values.currency ? { currency: values.currency.toUpperCase() } : {}),
        }),
      })
      return res.data
    },
    onSuccess: (data) => {
      setCreatedJoinCode(data.company.code_to_join_company ?? null)
      createForm.reset({ name: '', currency: 'EUR' })
      void queryClient.invalidateQueries({ queryKey: ['companies'] })
    },
  })

  const joinForm = useForm<JoinCompanyValues>({
    resolver: zodResolver(joinCompanySchema),
    defaultValues: { code: '' },
  })

  const joinMutation = useMutation({
    mutationFn: async (values: JoinCompanyValues) => {
      const res = await authFetch<CreateOrJoinResponse>('/companies/join', {
        method: 'POST',
        body: JSON.stringify(values),
      })
      return res.data
    },
    onSuccess: (data) => {
      joinForm.reset({ code: '' })
      void queryClient.invalidateQueries({ queryKey: ['companies'] })
      setSelectedCompanyId(data.company.id)
      navigate(`/companies/${data.company.id}/employees`)
    },
  })

  const openCompany = (companyId: string) => {
    setSelectedCompanyId(companyId)
    navigate(`/companies/${companyId}/employees`)
  }

  return (
    <div className="space-y-8">
      <section>
        <h1 className="text-2xl font-semibold">Your companies</h1>
        <p className="text-sm text-black/60">
          Pick a company to manage employees, or create/join one below.
        </p>
      </section>

      <section className="space-y-3">
        {companiesQuery.isLoading && <p className="text-black/60">Loading…</p>}
        {companiesQuery.isError && (
          <p className="text-red-700">{(companiesQuery.error as Error).message}</p>
        )}
        {companiesQuery.data?.length === 0 && (
          <p className="text-black/60">You aren&apos;t part of any company yet.</p>
        )}
        <ul className="grid gap-3 sm:grid-cols-2">
          {companiesQuery.data?.map((c) => (
            <li key={c.id}>
              <button
                type="button"
                onClick={() => openCompany(c.id)}
                className="w-full rounded-2xl border border-black/10 bg-white/80 p-4 text-left shadow-sm hover:border-[var(--empops-accent)]"
              >
                <p className="font-semibold">{c.name}</p>
                <p className="text-sm text-black/60">{c.currency}</p>
                <p className="mt-2 flex flex-wrap gap-1">
                  {c.roles.map((r) => (
                    <span key={r} className="rounded-full bg-black/[0.06] px-2 py-0.5 text-xs">
                      {r}
                    </span>
                  ))}
                </p>
              </button>
            </li>
          ))}
        </ul>
      </section>

      <div className="grid gap-6 sm:grid-cols-2">
        <form
          className="space-y-3 rounded-2xl border border-black/10 bg-white/80 p-5 shadow-sm"
          onSubmit={createForm.handleSubmit((values) => createMutation.mutate(values))}
        >
          <h2 className="text-lg font-semibold">Create a company</h2>
          <label className="block space-y-1">
            <span className="text-sm">Name</span>
            <input
              className="w-full rounded-lg border border-black/15 bg-white px-3 py-2 outline-none focus:border-[var(--empops-accent)]"
              {...createForm.register('name')}
            />
            {createForm.formState.errors.name && (
              <span className="text-sm text-red-700">
                {createForm.formState.errors.name.message}
              </span>
            )}
          </label>
          <label className="block space-y-1">
            <span className="text-sm">Currency (optional)</span>
            <input
              className="w-full rounded-lg border border-black/15 bg-white px-3 py-2 uppercase outline-none focus:border-[var(--empops-accent)]"
              maxLength={3}
              placeholder="EUR"
              {...createForm.register('currency')}
            />
            {createForm.formState.errors.currency && (
              <span className="text-sm text-red-700">
                {createForm.formState.errors.currency.message}
              </span>
            )}
          </label>
          {createMutation.isError && (
            <p className="text-sm text-red-700">{(createMutation.error as Error).message}</p>
          )}
          <button
            type="submit"
            className="w-full rounded-lg bg-[var(--empops-accent)] px-4 py-2.5 font-medium text-white hover:opacity-90 disabled:opacity-60"
            disabled={createMutation.isPending}
          >
            {createMutation.isPending ? 'Creating…' : 'Create company'}
          </button>
          {createdJoinCode && (
            <p className="text-sm text-black/70">
              Share this join code with your team:{' '}
              <code className="rounded bg-black/[0.06] px-1.5 py-0.5">{createdJoinCode}</code>
            </p>
          )}
        </form>

        <form
          className="space-y-3 rounded-2xl border border-black/10 bg-white/80 p-5 shadow-sm"
          onSubmit={joinForm.handleSubmit((values) => joinMutation.mutate(values))}
        >
          <h2 className="text-lg font-semibold">Join a company</h2>
          <label className="block space-y-1">
            <span className="text-sm">Join code</span>
            <input
              className="w-full rounded-lg border border-black/15 bg-white px-3 py-2 uppercase outline-none focus:border-[var(--empops-accent)]"
              {...joinForm.register('code')}
            />
            {joinForm.formState.errors.code && (
              <span className="text-sm text-red-700">
                {joinForm.formState.errors.code.message}
              </span>
            )}
          </label>
          {joinMutation.isError && (
            <p className="text-sm text-red-700">{(joinMutation.error as Error).message}</p>
          )}
          <button
            type="submit"
            className="w-full rounded-lg border border-black/15 px-4 py-2.5 font-medium hover:bg-black/[0.03] disabled:opacity-60"
            disabled={joinMutation.isPending}
          >
            {joinMutation.isPending ? 'Joining…' : 'Join company'}
          </button>
        </form>
      </div>
    </div>
  )
}
