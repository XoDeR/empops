import { Link, Navigate, useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { authFetch } from '@/lib/authFetch'
import { useCompanyContext } from '@/routes/CompanyLayout'
import type { Employee, Hardware } from '@/types/api'

export default function HardwareDetailPage() {
  const { companyId, hardwareId } = useParams<{ companyId: string; hardwareId: string }>()
  const { isHrOrAdmin } = useCompanyContext()
  const qc = useQueryClient()
  const [name, setName] = useState('')
  const [serial, setSerial] = useState('')
  const [lendTo, setLendTo] = useState('')
  const [error, setError] = useState<string | null>(null)

  const query = useQuery({
    queryKey: ['hardware', companyId, hardwareId],
    queryFn: async () => {
      const res = await authFetch<Hardware>(`/companies/${companyId}/hardware/${hardwareId}`)
      return res.data
    },
    enabled: Boolean(companyId && hardwareId) && isHrOrAdmin,
  })

  useEffect(() => {
    if (query.data) {
      setName(query.data.name)
      setSerial(query.data.serial_number ?? '')
    }
  }, [query.data])

  const employeesQuery = useQuery({
    queryKey: ['employees', companyId],
    queryFn: async () => {
      const res = await authFetch<Employee[]>(`/companies/${companyId}/employees`)
      return res.data
    },
    enabled: Boolean(companyId) && isHrOrAdmin,
  })

  const invalidate = () => {
    void qc.invalidateQueries({ queryKey: ['hardware', companyId] })
  }

  const update = useMutation({
    mutationFn: async () => {
      await authFetch(`/companies/${companyId}/hardware/${hardwareId}`, {
        method: 'PATCH',
        body: JSON.stringify({ name, serial_number: serial || null }),
      })
    },
    onSuccess: () => {
      setError(null)
      invalidate()
    },
    onError: (e: Error) => setError(e.message),
  })

  const lend = useMutation({
    mutationFn: async () => {
      await authFetch(`/companies/${companyId}/hardware/${hardwareId}/lend`, {
        method: 'POST',
        body: JSON.stringify({ employee_id: lendTo }),
      })
    },
    onSuccess: () => {
      setLendTo('')
      setError(null)
      invalidate()
    },
    onError: (e: Error) => setError(e.message),
  })

  const regain = useMutation({
    mutationFn: async () => {
      await authFetch(`/companies/${companyId}/hardware/${hardwareId}/regain`, {
        method: 'POST',
      })
    },
    onSuccess: () => {
      setError(null)
      invalidate()
    },
    onError: (e: Error) => setError(e.message),
  })

  const destroy = useMutation({
    mutationFn: async () => {
      await authFetch(`/companies/${companyId}/hardware/${hardwareId}`, { method: 'DELETE' })
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['hardware', companyId] })
    },
  })

  if (!isHrOrAdmin) {
    return <Navigate to={`/companies/${companyId}/dashboard/me`} replace />
  }

  if (query.isLoading) return <p className="text-black/60">Loading…</p>
  if (query.isError || !query.data) {
    return (
      <div className="rounded-2xl border border-red-200 bg-red-50 p-4 text-red-700">
        {(query.error as Error)?.message ?? 'Not found'}
      </div>
    )
  }

  const item = query.data

  if (destroy.isSuccess) {
    return <Navigate to={`/companies/${companyId}/hardware`} replace />
  }

  return (
    <div className="space-y-6">
      <div>
        <Link
          to={`/companies/${companyId}/hardware`}
          className="text-sm text-black/55 hover:text-black"
        >
          ← Hardware
        </Link>
        <h2 className="mt-2 text-xl font-semibold">{item.name}</h2>
        <p className="text-sm text-black/55">
          {item.employee
            ? `Lent to ${item.employee.first_name} ${item.employee.last_name}`
            : 'Available'}
        </p>
      </div>

      {error && (
        <div className="rounded-2xl border border-red-200 bg-red-50 p-4 text-red-700">{error}</div>
      )}

      <section className="rounded-2xl border border-black/10 bg-white/70 p-4 space-y-3">
        <h3 className="font-medium">Details</h3>
        <form
          className="grid gap-2 sm:grid-cols-2"
          onSubmit={(e) => {
            e.preventDefault()
            update.mutate()
          }}
        >
          <input
            className="rounded-lg border border-black/15 px-3 py-2 text-sm"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
          <input
            className="rounded-lg border border-black/15 px-3 py-2 text-sm"
            placeholder="Serial number"
            value={serial}
            onChange={(e) => setSerial(e.target.value)}
          />
          <button
            type="submit"
            className="rounded-lg border border-black/15 px-3 py-2 text-sm sm:col-span-2"
          >
            Save
          </button>
        </form>
      </section>

      <section className="rounded-2xl border border-black/10 bg-white/70 p-4 space-y-3">
        <h3 className="font-medium">Assignment</h3>
        {item.employee_id ? (
          <button
            type="button"
            className="rounded-lg border border-black/15 px-3 py-2 text-sm"
            onClick={() => regain.mutate()}
          >
            Regain from employee
          </button>
        ) : (
          <div className="flex flex-wrap gap-2">
            <select
              className="rounded-lg border border-black/15 px-3 py-2 text-sm"
              value={lendTo}
              onChange={(e) => setLendTo(e.target.value)}
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
              disabled={!lendTo}
              className="rounded-lg bg-[var(--empops-accent)] px-3 py-2 text-sm text-white disabled:opacity-50"
              onClick={() => lend.mutate()}
            >
              Lend
            </button>
          </div>
        )}
      </section>

      <button
        type="button"
        className="text-sm text-red-600 hover:underline"
        onClick={() => {
          if (confirm('Delete this hardware asset?')) destroy.mutate()
        }}
      >
        Delete hardware
      </button>
    </div>
  )
}
