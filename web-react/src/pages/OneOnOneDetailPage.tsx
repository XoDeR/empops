import { Link, useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { authFetch } from '@/lib/authFetch'
import type { OneOnOneEntry } from '@/types/api'

export default function OneOnOneDetailPage() {
  const { companyId, entryId } = useParams<{ companyId: string; entryId: string }>()
  const qc = useQueryClient()
  const [desc, setDesc] = useState('')
  const [note, setNote] = useState('')
  const [error, setError] = useState<string | null>(null)

  const query = useQuery({
    queryKey: ['one-on-one', companyId, entryId],
    queryFn: async () => {
      const res = await authFetch<OneOnOneEntry>(
        `/companies/${companyId}/one-on-ones/${entryId}`,
      )
      return res.data
    },
    enabled: Boolean(companyId && entryId),
  })

  const invalidate = () =>
    void qc.invalidateQueries({ queryKey: ['one-on-one', companyId, entryId] })

  const markHappened = useMutation({
    mutationFn: async () => {
      await authFetch(`/companies/${companyId}/one-on-ones/${entryId}/happened`, {
        method: 'POST',
      })
    },
    onSuccess: () => {
      setError(null)
      invalidate()
      void qc.invalidateQueries({ queryKey: ['dashboard', companyId] })
    },
    onError: (e: Error) => setError(e.message),
  })

  const addTalking = useMutation({
    mutationFn: async () => {
      await authFetch(`/companies/${companyId}/one-on-ones/${entryId}/talking-points`, {
        method: 'POST',
        body: JSON.stringify({ description: desc }),
      })
    },
    onSuccess: () => {
      setDesc('')
      invalidate()
    },
    onError: (e: Error) => setError(e.message),
  })

  const addAction = useMutation({
    mutationFn: async () => {
      await authFetch(`/companies/${companyId}/one-on-ones/${entryId}/action-items`, {
        method: 'POST',
        body: JSON.stringify({ description: desc }),
      })
    },
    onSuccess: () => {
      setDesc('')
      invalidate()
    },
    onError: (e: Error) => setError(e.message),
  })

  const addNote = useMutation({
    mutationFn: async () => {
      await authFetch(`/companies/${companyId}/one-on-ones/${entryId}/notes`, {
        method: 'POST',
        body: JSON.stringify({ note }),
      })
    },
    onSuccess: () => {
      setNote('')
      invalidate()
    },
    onError: (e: Error) => setError(e.message),
  })

  const toggle = async (kind: 'talking-points' | 'action-items', id: string) => {
    await authFetch(
      `/companies/${companyId}/one-on-ones/${entryId}/${kind}/${id}/toggle`,
      { method: 'POST' },
    )
    invalidate()
  }

  const entry = query.data
  if (query.isLoading) return <p className="text-black/60">Loading…</p>
  if (!entry) return <p className="text-red-700">One-on-one not found.</p>

  return (
    <div className="space-y-6">
      <div>
        <Link
          to={`/companies/${companyId}/dashboard/me`}
          className="text-sm text-black/55 hover:text-black"
        >
          ← Dashboard
        </Link>
        <h2 className="mt-2 text-xl font-semibold">
          1:1 — {entry.manager.first_name} & {entry.employee.first_name}
        </h2>
        <p className="text-sm text-black/55">
          {entry.happened ? 'Completed' : 'Open'}
        </p>
      </div>

      {error && (
        <div className="rounded-2xl border border-red-200 bg-red-50 p-3 text-sm text-red-700">
          {error}
        </div>
      )}

      {!entry.happened && (
        <button
          type="button"
          className="rounded-lg bg-[var(--empops-accent)] px-3 py-1.5 text-sm text-white"
          disabled={markHappened.isPending}
          onClick={() => markHappened.mutate()}
        >
          Mark as happened
        </button>
      )}

      <section className="rounded-2xl border border-black/10 bg-white/70 p-4">
        <h3 className="font-medium">Talking points</h3>
        <ul className="mt-2 space-y-1 text-sm">
          {entry.talking_points.map((p) => (
            <li key={p.id}>
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={p.checked}
                  onChange={() => void toggle('talking-points', p.id)}
                />
                <span className={p.checked ? 'line-through text-black/45' : ''}>
                  {p.description}
                </span>
              </label>
            </li>
          ))}
        </ul>
      </section>

      <section className="rounded-2xl border border-black/10 bg-white/70 p-4">
        <h3 className="font-medium">Action items</h3>
        <ul className="mt-2 space-y-1 text-sm">
          {entry.action_items.map((p) => (
            <li key={p.id}>
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={p.checked}
                  onChange={() => void toggle('action-items', p.id)}
                />
                <span className={p.checked ? 'line-through text-black/45' : ''}>
                  {p.description}
                </span>
              </label>
            </li>
          ))}
        </ul>
      </section>

      <section className="rounded-2xl border border-black/10 bg-white/70 p-4">
        <h3 className="font-medium">Notes</h3>
        <ul className="mt-2 space-y-2 text-sm text-black/75">
          {entry.notes.map((n) => (
            <li key={n.id} className="whitespace-pre-wrap">
              {n.note}
            </li>
          ))}
        </ul>
        <textarea
          className="mt-3 w-full rounded-lg border border-black/15 px-3 py-2 text-sm"
          rows={2}
          value={note}
          onChange={(e) => setNote(e.target.value)}
          placeholder="Add a note"
        />
        <button
          type="button"
          className="mt-2 rounded-lg border border-black/15 px-3 py-1.5 text-sm"
          disabled={!note.trim() || addNote.isPending}
          onClick={() => addNote.mutate()}
        >
          Add note
        </button>
      </section>

      <section className="rounded-2xl border border-black/10 bg-white/70 p-4">
        <h3 className="font-medium">Add item</h3>
        <input
          className="mt-2 w-full rounded-lg border border-black/15 px-3 py-2 text-sm"
          value={desc}
          onChange={(e) => setDesc(e.target.value)}
          placeholder="Description"
        />
        <div className="mt-2 flex gap-2">
          <button
            type="button"
            className="rounded-lg border border-black/15 px-3 py-1.5 text-sm"
            disabled={!desc.trim() || addTalking.isPending}
            onClick={() => addTalking.mutate()}
          >
            Talking point
          </button>
          <button
            type="button"
            className="rounded-lg border border-black/15 px-3 py-1.5 text-sm"
            disabled={!desc.trim() || addAction.isPending}
            onClick={() => addAction.mutate()}
          >
            Action item
          </button>
        </div>
      </section>
    </div>
  )
}
