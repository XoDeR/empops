import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { authFetch } from '@/lib/authFetch'
import type { Project } from '@/types/api'

const createSchema = z.object({
  name: z.string().min(1, 'Required'),
  summary: z.string().optional(),
})
type CreateValues = z.infer<typeof createSchema>

export default function ProjectsPage() {
  const { companyId } = useParams<{ companyId: string }>()
  const qc = useQueryClient()
  const [error, setError] = useState<string | null>(null)

  const projectsQuery = useQuery({
    queryKey: ['projects', companyId],
    queryFn: async () => {
      const res = await authFetch<Project[]>(`/companies/${companyId}/projects`)
      return res.data
    },
    enabled: Boolean(companyId),
  })

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<CreateValues>({
    resolver: zodResolver(createSchema),
    defaultValues: { name: '', summary: '' },
  })

  const createMutation = useMutation({
    mutationFn: async (values: CreateValues) => {
      const res = await authFetch<Project>(`/companies/${companyId}/projects`, {
        method: 'POST',
        body: JSON.stringify({
          name: values.name,
          summary: values.summary || null,
        }),
      })
      return res.data
    },
    onSuccess: () => {
      setError(null)
      reset()
      void qc.invalidateQueries({ queryKey: ['projects', companyId] })
    },
    onError: (e: Error) => setError(e.message),
  })

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold">Projects</h2>
        <p className="text-sm text-black/55">Track work, tasks, boards, and decisions.</p>
      </div>

      <form
        onSubmit={handleSubmit((v) => createMutation.mutate(v))}
        className="grid gap-3 rounded-2xl border border-black/10 bg-white/70 p-4 sm:grid-cols-[1fr_1fr_auto]"
      >
        <label className="block space-y-1">
          <span className="text-xs text-black/60">Name</span>
          <input
            className="w-full rounded-lg border border-black/15 bg-white px-3 py-2 text-sm outline-none focus:border-[var(--empops-accent)]"
            {...register('name')}
          />
          {errors.name && <span className="text-xs text-red-700">{errors.name.message}</span>}
        </label>
        <label className="block space-y-1">
          <span className="text-xs text-black/60">Summary</span>
          <input
            className="w-full rounded-lg border border-black/15 bg-white px-3 py-2 text-sm outline-none focus:border-[var(--empops-accent)]"
            {...register('summary')}
          />
        </label>
        <div className="flex items-end">
          <button
            type="submit"
            disabled={createMutation.isPending}
            className="rounded-lg bg-[var(--empops-accent)] px-4 py-2 text-sm font-medium text-white disabled:opacity-60"
          >
            Create project
          </button>
        </div>
      </form>

      {error && (
        <div className="rounded-2xl border border-red-200 bg-red-50 p-4 text-red-700">{error}</div>
      )}

      {projectsQuery.isLoading && <p className="text-black/60">Loading projects…</p>}
      {projectsQuery.isError && (
        <div className="rounded-2xl border border-red-200 bg-red-50 p-4 text-red-700">
          {(projectsQuery.error as Error).message}
        </div>
      )}

      <ul className="space-y-3">
        {(projectsQuery.data ?? []).map((project) => (
          <li key={project.id}>
            <Link
              to={`/companies/${companyId}/projects/${project.id}`}
              className="block rounded-2xl border border-black/10 bg-white/70 p-4 transition hover:border-black/20"
            >
              <div className="flex items-baseline justify-between gap-3">
                <h3 className="font-medium">
                  {project.emoji ? `${project.emoji} ` : ''}
                  {project.name}
                </h3>
                <span className="text-xs text-black/50">{project.status.replaceAll('_', ' ')}</span>
              </div>
              {project.summary && (
                <p className="mt-1 text-sm text-black/60 line-clamp-2">{project.summary}</p>
              )}
              <p className="mt-2 text-xs text-black/50">
                {project.member_count} members
                {project.lead &&
                  ` · Lead: ${project.lead.first_name} ${project.lead.last_name}`}
              </p>
            </Link>
          </li>
        ))}
      </ul>

      {projectsQuery.data?.length === 0 && (
        <p className="text-sm text-black/55">No projects yet.</p>
      )}
    </div>
  )
}
