import { Link, Navigate, useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { authFetch } from '@/lib/authFetch'
import { ChunkedUploader } from '@/lib/upload/chunked-upload'
import { useCompanyContext } from '@/routes/CompanyLayout'
import type { Employee, Software } from '@/types/api'

export default function SoftwareDetailPage() {
  const { companyId, softwareId } = useParams<{ companyId: string; softwareId: string }>()
  const { isHrOrAdmin } = useCompanyContext()
  const qc = useQueryClient()
  const [name, setName] = useState('')
  const [productKey, setProductKey] = useState('')
  const [seats, setSeats] = useState('1')
  const [assignTo, setAssignTo] = useState('')
  const [error, setError] = useState<string | null>(null)

  const query = useQuery({
    queryKey: ['software', companyId, softwareId],
    queryFn: async () => {
      const res = await authFetch<Software>(`/companies/${companyId}/softwares/${softwareId}`)
      return res.data
    },
    enabled: Boolean(companyId && softwareId) && isHrOrAdmin,
  })

  useEffect(() => {
    if (query.data) {
      setName(query.data.name)
      setProductKey(query.data.product_key ?? '')
      setSeats(String(query.data.seats))
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

  const withoutQuery = useQuery({
    queryKey: ['software-without', companyId, softwareId],
    queryFn: async () => {
      const res = await authFetch<{
        employees_without: number
        remaining_seats: number
        seats: number
      }>(`/companies/${companyId}/softwares/${softwareId}/employees-without`)
      return res.data
    },
    enabled: Boolean(companyId && softwareId) && isHrOrAdmin,
  })

  const invalidate = () => {
    void qc.invalidateQueries({ queryKey: ['software', companyId, softwareId] })
    void qc.invalidateQueries({ queryKey: ['softwares', companyId] })
    void qc.invalidateQueries({ queryKey: ['software-without', companyId, softwareId] })
  }

  const update = useMutation({
    mutationFn: async () => {
      await authFetch(`/companies/${companyId}/softwares/${softwareId}`, {
        method: 'PATCH',
        body: JSON.stringify({
          name,
          product_key: productKey,
          seats: Number(seats),
        }),
      })
    },
    onSuccess: () => {
      setError(null)
      invalidate()
    },
    onError: (e: Error) => setError(e.message),
  })

  const giveSeat = useMutation({
    mutationFn: async () => {
      await authFetch(`/companies/${companyId}/softwares/${softwareId}/seats`, {
        method: 'POST',
        body: JSON.stringify({ employee_id: assignTo }),
      })
    },
    onSuccess: () => {
      setAssignTo('')
      setError(null)
      invalidate()
    },
    onError: (e: Error) => setError(e.message),
  })

  const revokeSeat = useMutation({
    mutationFn: async (employeeId: string) => {
      await authFetch(
        `/companies/${companyId}/softwares/${softwareId}/seats/${employeeId}`,
        { method: 'DELETE' },
      )
    },
    onSuccess: () => {
      setError(null)
      invalidate()
    },
    onError: (e: Error) => setError(e.message),
  })

  const giveAll = useMutation({
    mutationFn: async () => {
      const res = await authFetch<{ assigned: number; software: Software }>(
        `/companies/${companyId}/softwares/${softwareId}/seats/all`,
        { method: 'POST' },
      )
      return res.data
    },
    onSuccess: (data) => {
      setError(null)
      invalidate()
      if (data && data.assigned === 0) {
        setError('No seats assigned (none remaining or everyone already has a seat).')
      }
    },
    onError: (e: Error) => setError(e.message),
  })

  const destroy = useMutation({
    mutationFn: async () => {
      await authFetch(`/companies/${companyId}/softwares/${softwareId}`, { method: 'DELETE' })
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['softwares', companyId] })
    },
  })

  const uploadFile = async (file: File) => {
    const uploader = new ChunkedUploader(file)
    const result = await uploader.upload()
    if (result.media_id == null || result.temporary_upload_id == null) {
      throw new Error('Upload completed but media IDs were not returned')
    }
    await authFetch(`/companies/${companyId}/softwares/${softwareId}/files`, {
      method: 'POST',
      body: JSON.stringify({
        temporary_upload_id: result.temporary_upload_id,
        media_id: result.media_id,
      }),
    })
    invalidate()
  }

  const detachFile = useMutation({
    mutationFn: async (mediaId: number) => {
      await authFetch(`/companies/${companyId}/softwares/${softwareId}/files/${mediaId}`, {
        method: 'DELETE',
      })
    },
    onSuccess: () => invalidate(),
    onError: (e: Error) => setError(e.message),
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

  if (destroy.isSuccess) {
    return <Navigate to={`/companies/${companyId}/softwares`} replace />
  }

  const item = query.data
  const assignedIds = new Set((item.employees ?? []).map((e) => e.id))
  const candidates = (employeesQuery.data ?? []).filter((e) => !assignedIds.has(e.id))
  const without = withoutQuery.data

  return (
    <div className="space-y-6">
      <div>
        <Link
          to={`/companies/${companyId}/softwares`}
          className="text-sm text-black/55 hover:text-black"
        >
          ← Software
        </Link>
        <h2 className="mt-2 text-xl font-semibold">{item.name}</h2>
        <p className="text-sm text-black/55">
          {item.seats_used ?? 0}/{item.seats} seats used
          {item.remaining_seats != null ? ` · ${item.remaining_seats} remaining` : ''}
        </p>
      </div>

      {error && (
        <div className="rounded-2xl border border-red-200 bg-red-50 p-4 text-red-700">{error}</div>
      )}

      <section className="rounded-2xl border border-black/10 bg-white/70 p-4 space-y-2 text-sm">
        <h3 className="font-medium">Purchase</h3>
        <p>
          Amount:{' '}
          {item.purchase_amount != null && item.currency
            ? `${item.purchase_amount} ${item.currency}`
            : '—'}
        </p>
        <p>
          Converted:{' '}
          {item.converted_purchase_amount != null && item.converted_to_currency
            ? `${item.converted_purchase_amount} ${item.converted_to_currency}`
            : '—'}
          {item.exchange_rate != null ? ` (rate ${item.exchange_rate})` : ''}
        </p>
      </section>

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
            value={productKey}
            onChange={(e) => setProductKey(e.target.value)}
          />
          <input
            type="number"
            min={1}
            className="rounded-lg border border-black/15 px-3 py-2 text-sm"
            value={seats}
            onChange={(e) => setSeats(e.target.value)}
          />
          <button
            type="submit"
            className="rounded-lg border border-black/15 px-3 py-2 text-sm"
          >
            Save
          </button>
        </form>
      </section>

      <section className="rounded-2xl border border-black/10 bg-white/70 p-4 space-y-3">
        <h3 className="font-medium">Seats</h3>
        <ul className="divide-y divide-black/5 text-sm">
          {(item.employees ?? []).map((e) => (
            <li key={e.id} className="flex items-center justify-between gap-2 py-2">
              <span>
                {e.first_name} {e.last_name}
              </span>
              <button
                type="button"
                className="text-xs text-red-600 hover:underline"
                onClick={() => revokeSeat.mutate(e.id)}
              >
                Revoke
              </button>
            </li>
          ))}
        </ul>
        <div className="flex flex-wrap gap-2">
          <select
            className="rounded-lg border border-black/15 px-3 py-2 text-sm"
            value={assignTo}
            onChange={(e) => setAssignTo(e.target.value)}
          >
            <option value="">Assign employee…</option>
            {candidates.map((e) => (
              <option key={e.id} value={e.id}>
                {e.first_name} {e.last_name}
              </option>
            ))}
          </select>
          <button
            type="button"
            disabled={!assignTo}
            className="rounded-lg bg-[var(--empops-accent)] px-3 py-2 text-sm text-white disabled:opacity-50"
            onClick={() => giveSeat.mutate()}
          >
            Give seat
          </button>
        </div>
        {without && (
          <p className="text-xs text-black/50">
            {without.employees_without} employees without a seat · {without.remaining_seats}{' '}
            remaining
            {without.employees_without > without.remaining_seats
              ? ' (not enough seats for everyone)'
              : ''}
          </p>
        )}
        <button
          type="button"
          className="rounded-lg border border-black/15 px-3 py-2 text-sm"
          onClick={() => {
            if (
              without &&
              without.employees_without > without.remaining_seats &&
              !confirm(
                `Only ${without.remaining_seats} seats remain for ${without.employees_without} employees. Continue?`,
              )
            ) {
              return
            }
            giveAll.mutate()
          }}
        >
          Give seats to all remaining
        </button>
      </section>

      <section className="rounded-2xl border border-black/10 bg-white/70 p-4 space-y-3">
        <h3 className="font-medium">Files</h3>
        <ul className="space-y-1 text-sm">
          {(item.files ?? []).map((f) => (
            <li key={f.id} className="flex items-center justify-between gap-2">
              <a href={f.url} className="hover:underline" target="_blank" rel="noreferrer">
                {f.file_name}
              </a>
              <button
                type="button"
                className="text-xs text-red-600 hover:underline"
                onClick={() => detachFile.mutate(f.id)}
              >
                Remove
              </button>
            </li>
          ))}
        </ul>
        <input
          type="file"
          onChange={(e) => {
            const file = e.target.files?.[0]
            if (!file) return
            void uploadFile(file).catch((err: Error) => setError(err.message))
            e.target.value = ''
          }}
        />
      </section>

      <button
        type="button"
        className="text-sm text-red-600 hover:underline"
        onClick={() => {
          if (confirm('Delete this software license?')) destroy.mutate()
        }}
      >
        Delete software
      </button>
    </div>
  )
}
