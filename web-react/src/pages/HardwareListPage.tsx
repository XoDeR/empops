import { Link, Navigate, useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { authFetch } from '@/lib/authFetch'
import { useCompanyContext } from '@/routes/CompanyLayout'
import type { Employee, Hardware } from '@/types/api'

type StatusFilter = 'all' | 'available' | 'lent'

export default function HardwareListPage() {
  const { companyId } = useParams<{ companyId: string }>()
  const { isHrOrAdmin } = useCompanyContext()
  const qc = useQueryClient()
  const [status, setStatus] = useState<StatusFilter>('all')
  const [q, setQ] = useState('')
  const [name, setName] = useState('')
  const [serial, setSerial] = useState('')
  const [employeeId, setEmployeeId] = useState('')
  const [error, setError] = useState<string | null>(null)

  const listQuery = useQuery({
    queryKey: ['hardware', companyId, status, q],
    queryFn: async () => {
      const params = new URLSearchParams({ status })
      if (q.trim()) params.set('q', q.trim())
      const res = await authFetch<Hardware[]>(
        `/companies/${companyId}/hardware?${params.toString()}`,
      )
      return res.data
    },
    enabled: Boolean(companyId) && isHrOrAdmin,
  })

  const employeesQuery = useQuery({
    queryKey: ['employees', companyId],
    queryFn: async () => {
      const res = await authFetch<Employee[]>(`/companies/${companyId}/employees`)
      return res.data
    },
    enabled: Boolean(companyId) && isHrOrAdmin,
  })

  const create = useMutation({
    mutationFn: async () => {
      await authFetch(`/companies/${companyId}/hardware`, {
        method: 'POST',
        body: JSON.stringify({
          name,
          serial_number: serial || null,
          employee_id: employeeId || null,
        }),
      })
    },
    onSuccess: () => {
      setName('')
      setSerial('')
      setEmployeeId('')
      setError(null)
      void qc.invalidateQueries({ queryKey: ['hardware', companyId] })
    },
    onError: (e: Error) => setError(e.message),
  })

  if (!isHrOrAdmin) {
    return <Navigate to={`/companies/${companyId}/dashboard/me`} replace />
  }

  return (
    <div className="space-y-6">
      <div>
        <Link
          to={`/companies/${companyId}/adminland`}
          className="text-sm text-black/55 hover:text-black"
        >
          ← Adminland
        </Link>
        <h2 className="mt-2 text-xl font-semibold">Hardware</h2>
        <p className="text-sm text-black/55">Company assets — lend and regain from employees.</p>
      </div>

      <div className="flex flex-wrap gap-2">
        {(['all', 'available', 'lent'] as StatusFilter[]).map((s) => (
          <button
            key={s}
            type="button"
            className={`rounded-lg border px-3 py-1 text-sm capitalize ${
              status === s
                ? 'border-[var(--empops-accent)] bg-[var(--empops-accent)]/10'
                : 'border-black/15'
            }`}
            onClick={() => setStatus(s)}
          >
            {s}
          </button>
        ))}
        <input
          className="rounded-lg border border-black/15 px-3 py-1 text-sm"
          placeholder="Search name or serial…"
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
      </div>

      <section className="rounded-2xl border border-black/10 bg-white/70 p-4 space-y-3">
        <h3 className="font-medium">Add hardware</h3>
        {error && <p className="text-sm text-red-600">{error}</p>}
        <form
          className="grid gap-2 sm:grid-cols-2"
          onSubmit={(e) => {
            e.preventDefault()
            create.mutate()
          }}
        >
          <input
            required
            className="rounded-lg border border-black/15 px-3 py-2 text-sm"
            placeholder="Name"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
          <input
            className="rounded-lg border border-black/15 px-3 py-2 text-sm"
            placeholder="Serial number"
            value={serial}
            onChange={(e) => setSerial(e.target.value)}
          />
          <select
            className="rounded-lg border border-black/15 px-3 py-2 text-sm sm:col-span-2"
            value={employeeId}
            onChange={(e) => setEmployeeId(e.target.value)}
          >
            <option value="">Available (not lent)</option>
            {(employeesQuery.data ?? []).map((e) => (
              <option key={e.id} value={e.id}>
                {e.first_name} {e.last_name}
              </option>
            ))}
          </select>
          <button
            type="submit"
            disabled={create.isPending}
            className="rounded-lg bg-[var(--empops-accent)] px-3 py-2 text-sm text-white sm:col-span-2"
          >
            Create
          </button>
        </form>
      </section>

      <section className="rounded-2xl border border-black/10 bg-white/70 p-4">
        {listQuery.isLoading && <p className="text-sm text-black/60">Loading…</p>}
        <ul className="divide-y divide-black/5">
          {(listQuery.data ?? []).map((h) => (
            <li key={h.id} className="flex flex-wrap items-center justify-between gap-2 py-3">
              <div>
                <Link
                  to={`/companies/${companyId}/hardware/${h.id}`}
                  className="font-medium hover:underline"
                >
                  {h.name}
                </Link>
                <p className="text-xs text-black/50">
                  {h.serial_number ? `S/N ${h.serial_number}` : 'No serial'}
                  {h.employee
                    ? ` · Lent to ${h.employee.first_name} ${h.employee.last_name}`
                    : ' · Available'}
                </p>
              </div>
              <Link
                to={`/companies/${companyId}/hardware/${h.id}`}
                className="text-sm text-[var(--empops-accent)] hover:underline"
              >
                Open
              </Link>
            </li>
          ))}
        </ul>
        {!listQuery.isLoading && (listQuery.data ?? []).length === 0 && (
          <p className="text-sm text-black/50">No hardware yet.</p>
        )}
      </section>
    </div>
  )
}
