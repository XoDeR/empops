import { Link, Navigate, useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { authFetch } from '@/lib/authFetch'
import { ChunkedUploader } from '@/lib/upload/chunked-upload'
import { useCompanyContext } from '@/routes/CompanyLayout'
import type { DisciplineCase } from '@/types/api'

export default function DisciplineDetailPage() {
  const { companyId, caseId } = useParams<{ companyId: string; caseId: string }>()
  const { isHrOrAdmin, isManager } = useCompanyContext()
  const qc = useQueryClient()
  const [description, setDescription] = useState('')
  const [happenedAt, setHappenedAt] = useState(new Date().toISOString().slice(0, 10))
  const [error, setError] = useState<string | null>(null)

  const query = useQuery({
    queryKey: ['discipline-case', companyId, caseId],
    queryFn: async () => {
      const res = await authFetch<DisciplineCase>(
        `/companies/${companyId}/discipline-cases/${caseId}`,
      )
      return res.data
    },
    enabled: Boolean(companyId && caseId) && (isHrOrAdmin || isManager),
  })

  const invalidate = () =>
    void qc.invalidateQueries({ queryKey: ['discipline-case', companyId, caseId] })

  const toggle = useMutation({
    mutationFn: async () => {
      await authFetch(`/companies/${companyId}/discipline-cases/${caseId}/toggle`, {
        method: 'POST',
      })
    },
    onSuccess: invalidate,
    onError: (e: Error) => setError(e.message),
  })

  const addEvent = useMutation({
    mutationFn: async () => {
      await authFetch(`/companies/${companyId}/discipline-cases/${caseId}/events`, {
        method: 'POST',
        body: JSON.stringify({ happened_at: happenedAt, description }),
      })
    },
    onSuccess: () => {
      setDescription('')
      setError(null)
      invalidate()
    },
    onError: (e: Error) => setError(e.message),
  })

  const uploadFile = async (eventId: string, file: File) => {
    const uploader = new ChunkedUploader(file)
    const result = await uploader.upload()
    if (result.media_id == null || result.temporary_upload_id == null) {
      throw new Error('Upload completed but media IDs were not returned')
    }
    await authFetch(
      `/companies/${companyId}/discipline-cases/${caseId}/events/${eventId}/files`,
      {
        method: 'POST',
        body: JSON.stringify({
          temporary_upload_id: result.temporary_upload_id,
          media_id: result.media_id,
        }),
      },
    )
    invalidate()
  }

  if (!isHrOrAdmin && !isManager) {
    return <Navigate to={`/companies/${companyId}/dashboard/me`} replace />
  }

  const c = query.data
  if (query.isLoading) return <p className="text-black/60">Loading…</p>
  if (!c) return <p className="text-red-700">Case not found.</p>

  return (
    <div className="space-y-6">
      <div>
        <Link
          to={`/companies/${companyId}/discipline`}
          className="text-sm text-black/55 hover:text-black"
        >
          ← Cases
        </Link>
        <h2 className="mt-2 text-xl font-semibold">
          {c.employee.first_name} {c.employee.last_name}
        </h2>
        <p className="text-sm text-black/55">{c.active ? 'Open' : 'Closed'}</p>
      </div>

      {error && (
        <div className="rounded-2xl border border-red-200 bg-red-50 p-3 text-sm text-red-700">
          {error}
        </div>
      )}

      {isHrOrAdmin && (
        <button
          type="button"
          className="rounded-lg border border-black/15 px-3 py-1.5 text-sm"
          onClick={() => toggle.mutate()}
        >
          {c.active ? 'Close case' : 'Reopen case'}
        </button>
      )}

      <section className="rounded-2xl border border-black/10 bg-white/70 p-4">
        <h3 className="font-medium">Events</h3>
        <ul className="mt-3 space-y-4">
          {(c.events ?? []).map((ev) => (
            <li key={ev.id} className="border-t border-black/5 pt-3 text-sm">
              <p className="font-medium">
                {ev.happened_at} · {ev.author_name}
              </p>
              <p className="mt-1 whitespace-pre-wrap text-black/75">{ev.description}</p>
              <ul className="mt-2 space-y-1">
                {ev.files.map((f) => (
                  <li key={f.id}>
                    <a className="text-[var(--empops-accent)] hover:underline" href={f.url}>
                      {f.file_name}
                    </a>
                  </li>
                ))}
              </ul>
              <input
                type="file"
                className="mt-2 block text-xs"
                onChange={(e) => {
                  const file = e.target.files?.[0]
                  if (file) void uploadFile(ev.id, file).catch((err: Error) => setError(err.message))
                }}
              />
            </li>
          ))}
        </ul>
      </section>

      <section className="rounded-2xl border border-black/10 bg-white/70 p-4">
        <h3 className="font-medium">Add event</h3>
        <input
          type="date"
          className="mt-2 rounded-lg border border-black/15 px-3 py-2 text-sm"
          value={happenedAt}
          onChange={(e) => setHappenedAt(e.target.value)}
        />
        <textarea
          className="mt-2 w-full rounded-lg border border-black/15 px-3 py-2 text-sm"
          rows={3}
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />
        <button
          type="button"
          className="mt-2 rounded-lg bg-[var(--empops-accent)] px-3 py-1.5 text-sm text-white disabled:opacity-60"
          disabled={!description.trim() || addEvent.isPending}
          onClick={() => addEvent.mutate()}
        >
          Add event
        </button>
      </section>
    </div>
  )
}
