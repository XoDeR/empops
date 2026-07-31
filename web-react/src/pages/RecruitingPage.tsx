import { useState } from 'react'
import { Link, useParams, useSearchParams } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { authFetch } from '@/lib/authFetch'
import { useCompanyContext } from '@/routes/CompanyLayout'
import type { JobOpening, Position, RecruitingStageTemplate, Team } from '@/types/api'

const inputClass =
  'w-full rounded-lg border border-black/15 bg-white px-3 py-2 text-sm outline-none focus:border-[var(--empops-accent)]'

const createSchema = z.object({
  title: z.string().min(1, 'Required'),
  description: z.string().min(1, 'Required'),
  position_id: z.string().min(1, 'Required'),
  template_id: z.string().min(1, 'Required'),
  team_id: z.string().optional(),
  reference_number: z.string().optional(),
})
type CreateValues = z.infer<typeof createSchema>

function tabBtn(active: boolean) {
  return `rounded-lg px-3 py-1.5 text-sm ${
    active
      ? 'bg-[var(--empops-accent)] text-white'
      : 'bg-black/[0.04] text-black/70 hover:bg-black/[0.08]'
  }`
}

export default function RecruitingPage() {
  const { companyId } = useParams<{ companyId: string }>()
  const { isHrOrAdmin } = useCompanyContext()
  const qc = useQueryClient()
  const [searchParams, setSearchParams] = useSearchParams()
  const fulfilled = searchParams.get('fulfilled') === 'true'
  const [error, setError] = useState<string | null>(null)

  const openingsQuery = useQuery({
    queryKey: ['job-openings', companyId, fulfilled],
    queryFn: async () => {
      const res = await authFetch<JobOpening[]>(
        `/companies/${companyId}/job-openings?fulfilled=${fulfilled}`,
      )
      return res.data
    },
    enabled: Boolean(companyId),
  })

  const positionsQuery = useQuery({
    queryKey: ['positions', companyId],
    queryFn: async () => {
      const res = await authFetch<Position[]>(`/companies/${companyId}/positions`)
      return res.data
    },
    enabled: Boolean(companyId) && isHrOrAdmin,
  })

  const templatesQuery = useQuery({
    queryKey: ['recruiting-templates', companyId],
    queryFn: async () => {
      const res = await authFetch<RecruitingStageTemplate[]>(
        `/companies/${companyId}/recruiting/stage-templates`,
      )
      return res.data
    },
    enabled: Boolean(companyId) && isHrOrAdmin,
  })

  const teamsQuery = useQuery({
    queryKey: ['teams', companyId],
    queryFn: async () => {
      const res = await authFetch<Team[]>(`/companies/${companyId}/teams`)
      return res.data
    },
    enabled: Boolean(companyId) && isHrOrAdmin,
  })

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<CreateValues>({
    resolver: zodResolver(createSchema),
    defaultValues: {
      title: '',
      description: '',
      position_id: '',
      template_id: '',
      team_id: '',
      reference_number: '',
    },
  })

  const createMutation = useMutation({
    mutationFn: async (values: CreateValues) => {
      const res = await authFetch<JobOpening>(`/companies/${companyId}/job-openings`, {
        method: 'POST',
        body: JSON.stringify({
          title: values.title,
          description: values.description,
          position_id: values.position_id,
          recruiting_stage_template_id: values.template_id,
          team_id: values.team_id || null,
          reference_number: values.reference_number || null,
        }),
      })
      return res.data
    },
    onSuccess: () => {
      setError(null)
      reset()
      void qc.invalidateQueries({ queryKey: ['job-openings', companyId] })
    },
    onError: (e: Error) => setError(e.message),
  })

  const toggleMutation = useMutation({
    mutationFn: async (jobOpeningId: string) => {
      const res = await authFetch<JobOpening>(
        `/companies/${companyId}/job-openings/${jobOpeningId}/toggle`,
        { method: 'POST' },
      )
      return res.data
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['job-openings', companyId] })
    },
    onError: (e: Error) => setError(e.message),
  })

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold">Recruiting</h2>
        <p className="text-sm text-black/55">Job openings and candidate pipeline.</p>
      </div>

      <nav className="flex flex-wrap gap-2">
        <button
          type="button"
          className={tabBtn(!fulfilled)}
          onClick={() => setSearchParams({ fulfilled: 'false' })}
        >
          Open
        </button>
        <button
          type="button"
          className={tabBtn(fulfilled)}
          onClick={() => setSearchParams({ fulfilled: 'true' })}
        >
          Fulfilled
        </button>
      </nav>

      {isHrOrAdmin && (
        <form
          onSubmit={handleSubmit((v) => createMutation.mutate(v))}
          className="grid gap-3 rounded-2xl border border-black/10 bg-white/70 p-4 sm:grid-cols-2"
        >
          <label className="block space-y-1 sm:col-span-2">
            <span className="text-xs text-black/60">Title</span>
            <input className={inputClass} {...register('title')} />
            {errors.title && <span className="text-xs text-red-700">{errors.title.message}</span>}
          </label>
          <label className="block space-y-1 sm:col-span-2">
            <span className="text-xs text-black/60">Description</span>
            <textarea
              className={`${inputClass} min-h-[80px]`}
              {...register('description')}
            />
            {errors.description && (
              <span className="text-xs text-red-700">{errors.description.message}</span>
            )}
          </label>
          <label className="block space-y-1">
            <span className="text-xs text-black/60">Position</span>
            <select className={inputClass} {...register('position_id')}>
              <option value="">Select…</option>
              {(positionsQuery.data ?? []).map((p) => (
                <option key={p.id} value={p.id}>
                  {p.title}
                </option>
              ))}
            </select>
            {errors.position_id && (
              <span className="text-xs text-red-700">{errors.position_id.message}</span>
            )}
          </label>
          <label className="block space-y-1">
            <span className="text-xs text-black/60">Stage template</span>
            <select className={inputClass} {...register('template_id')}>
              <option value="">Select…</option>
              {(templatesQuery.data ?? []).map((t) => (
                <option key={t.id} value={t.id}>
                  {t.name}
                </option>
              ))}
            </select>
            {errors.template_id && (
              <span className="text-xs text-red-700">{errors.template_id.message}</span>
            )}
          </label>
          <label className="block space-y-1">
            <span className="text-xs text-black/60">Team (optional)</span>
            <select className={inputClass} {...register('team_id')}>
              <option value="">None</option>
              {(teamsQuery.data ?? []).map((t) => (
                <option key={t.id} value={t.id}>
                  {t.name}
                </option>
              ))}
            </select>
          </label>
          <label className="block space-y-1">
            <span className="text-xs text-black/60">Reference number (optional)</span>
            <input className={inputClass} {...register('reference_number')} />
          </label>
          <div className="flex items-end sm:col-span-2">
            <button
              type="submit"
              disabled={createMutation.isPending}
              className="rounded-lg bg-[var(--empops-accent)] px-4 py-2 text-sm font-medium text-white disabled:opacity-60"
            >
              Create opening
            </button>
          </div>
        </form>
      )}

      {error && (
        <div className="rounded-2xl border border-red-200 bg-red-50 p-4 text-red-700">{error}</div>
      )}

      {openingsQuery.isLoading && <p className="text-black/60">Loading openings…</p>}
      {openingsQuery.isError && (
        <div className="rounded-2xl border border-red-200 bg-red-50 p-4 text-red-700">
          {(openingsQuery.error as Error).message}
        </div>
      )}

      <ul className="space-y-3">
        {(openingsQuery.data ?? []).map((opening) => (
          <li
            key={opening.id}
            className="rounded-2xl border border-black/10 bg-white/70 p-4"
          >
            <div className="flex flex-wrap items-start justify-between gap-3">
              <Link
                to={`/companies/${companyId}/recruiting/${opening.id}`}
                className="font-medium hover:text-[var(--empops-accent)]"
              >
                {opening.title}
                {opening.reference_number && (
                  <span className="ml-2 text-xs text-black/50">#{opening.reference_number}</span>
                )}
              </Link>
              <div className="flex flex-wrap items-center gap-2">
                <span
                  className={`rounded-full px-2 py-0.5 text-xs ${
                    opening.active
                      ? 'bg-green-100 text-green-800'
                      : 'bg-black/10 text-black/60'
                  }`}
                >
                  {opening.active ? 'Active' : 'Inactive'}
                </span>
                {isHrOrAdmin && !opening.fulfilled && (
                  <button
                    type="button"
                    disabled={toggleMutation.isPending}
                    onClick={() => toggleMutation.mutate(opening.id)}
                    className="rounded-lg border border-black/15 px-2 py-1 text-xs hover:bg-black/[0.03] disabled:opacity-60"
                  >
                    {opening.active ? 'Deactivate' : 'Activate'}
                  </button>
                )}
              </div>
            </div>
            {opening.position && (
              <p className="mt-1 text-sm text-black/60">{opening.position.title}</p>
            )}
            <p className="mt-2 text-xs text-black/50">
              {opening.page_views} views
              {opening.sponsors.length > 0 &&
                ` · Sponsors: ${opening.sponsors.map((s) => `${s.first_name} ${s.last_name}`).join(', ')}`}
            </p>
          </li>
        ))}
      </ul>

      {openingsQuery.data?.length === 0 && (
        <p className="text-sm text-black/55">No {fulfilled ? 'fulfilled' : 'open'} job openings.</p>
      )}
    </div>
  )
}
