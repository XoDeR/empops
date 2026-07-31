import { Link, Navigate, useNavigate, useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { authFetch } from '@/lib/authFetch'
import { useCompanyContext } from '@/routes/CompanyLayout'
import type { DisciplineCase, Employee } from '@/types/api'

export default function DisciplineListPage() {
  const { companyId } = useParams<{ companyId: string }>()
  const { isHrOrAdmin, isManager } = useCompanyContext()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const [employeeId, setEmployeeId] = useState('')
  const [error, setError] = useState<string | null>(null)

  const casesQuery = useQuery({
    queryKey: ['discipline-cases', companyId],
    queryFn: async () => {
      const res = await authFetch<DisciplineCase[]>(
        `/companies/${companyId}/discipline-cases?active=true`,
      )
      return res.data
    },
    enabled: Boolean(companyId) && (isHrOrAdmin || isManager),
  })

  const employeesQuery = useQuery({
    queryKey: ['employees', companyId],
    queryFn: async () => {
      const res = await authFetch<Employee[]>(`/companies/${companyId}/employees`)
      return res.data
    },
    enabled: Boolean(companyId) && (isHrOrAdmin || isManager),
  })

  const create = useMutation({
    mutationFn: async () => {
      const res = await authFetch<DisciplineCase>(
        `/companies/${companyId}/discipline-cases`,
        {
          method: 'POST',
          body: JSON.stringify({ employee_id: employeeId }),
        },
      )
      return res.data
    },
    onSuccess: (c) => {
      setError(null)
      void qc.invalidateQueries({ queryKey: ['discipline-cases', companyId] })
      navigate(`/companies/${companyId}/discipline/${c.id}`)
    },
    onError: (e: Error) => setError(e.message),
  })

  if (!isHrOrAdmin && !isManager) {
    return <Navigate to={`/companies/${companyId}/dashboard/me`} replace />
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold">Discipline cases</h2>
        <p className="text-sm text-black/55">Active cases</p>
      </div>

      {error && (
        <div className="rounded-2xl border border-red-200 bg-red-50 p-3 text-sm text-red-700">
          {error}
        </div>
      )}

      <section className="rounded-2xl border border-black/10 bg-white/70 p-4">
        <h3 className="font-medium">Open a case</h3>
        <div className="mt-2 flex flex-wrap gap-2">
          <select
            className="rounded-lg border border-black/15 px-3 py-2 text-sm"
            value={employeeId}
            onChange={(e) => setEmployeeId(e.target.value)}
          >
            <option value="">Select employee…</option>
            {(employeesQuery.data ?? []).map((e) => (
              <option key={e.id} value={e.id}>
                {e.first_name} {e.last_name}
              </option>
            ))}
          </select>
          <button
            type="button"
            className="rounded-lg bg-[var(--empops-accent)] px-3 py-1.5 text-sm text-white disabled:opacity-60"
            disabled={!employeeId || create.isPending}
            onClick={() => create.mutate()}
          >
            Create
          </button>
        </div>
      </section>

      <section className="rounded-2xl border border-black/10 bg-white/70 p-4">
        {casesQuery.isLoading && <p className="text-sm text-black/60">Loading…</p>}
        <ul className="divide-y divide-black/5">
          {(casesQuery.data ?? []).map((c) => (
            <li key={c.id} className="py-3">
              <Link
                to={`/companies/${companyId}/discipline/${c.id}`}
                className="font-medium hover:underline"
              >
                {c.employee.first_name} {c.employee.last_name}
              </Link>
              <p className="text-xs text-black/50">
                Opened by {c.opened_by_employee_name ?? '—'} ·{' '}
                {c.created_at.slice(0, 10)}
              </p>
            </li>
          ))}
        </ul>
      </section>
    </div>
  )
}
