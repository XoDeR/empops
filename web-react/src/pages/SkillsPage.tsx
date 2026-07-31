import { Link, Navigate, useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { authFetch } from '@/lib/authFetch'
import { useCompanyContext } from '@/routes/CompanyLayout'
import type { Skill } from '@/types/api'

export default function SkillsPage() {
  const { companyId } = useParams<{ companyId: string }>()
  const { isHrOrAdmin } = useCompanyContext()
  const qc = useQueryClient()
  const [renameId, setRenameId] = useState<string | null>(null)
  const [name, setName] = useState('')

  const query = useQuery({
    queryKey: ['skills', companyId],
    queryFn: async () => {
      const res = await authFetch<Skill[]>(`/companies/${companyId}/skills`)
      return res.data
    },
    enabled: Boolean(companyId) && isHrOrAdmin,
  })

  const rename = useMutation({
    mutationFn: async () => {
      if (!renameId) return
      await authFetch(`/companies/${companyId}/skills/${renameId}`, {
        method: 'PATCH',
        body: JSON.stringify({ name }),
      })
    },
    onSuccess: () => {
      setRenameId(null)
      setName('')
      void qc.invalidateQueries({ queryKey: ['skills', companyId] })
    },
  })

  const destroy = useMutation({
    mutationFn: async (id: string) => {
      await authFetch(`/companies/${companyId}/skills/${id}`, { method: 'DELETE' })
    },
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['skills', companyId] }),
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
        <h2 className="mt-2 text-xl font-semibold">Skills directory</h2>
      </div>

      <section className="rounded-2xl border border-black/10 bg-white/70 p-4">
        {query.isLoading && <p className="text-sm text-black/60">Loading…</p>}
        <ul className="divide-y divide-black/5">
          {(query.data ?? []).map((s) => (
            <li key={s.id} className="flex flex-wrap items-center justify-between gap-2 py-3">
              <div>
                <p className="font-medium">{s.name}</p>
                <p className="text-xs text-black/50">{s.employees_count ?? 0} employees</p>
              </div>
              <div className="flex gap-2">
                <button
                  type="button"
                  className="rounded-lg border border-black/15 px-2 py-1 text-xs"
                  onClick={() => {
                    setRenameId(s.id)
                    setName(s.name)
                  }}
                >
                  Rename
                </button>
                <button
                  type="button"
                  className="rounded-lg border border-red-200 px-2 py-1 text-xs text-red-700"
                  onClick={() => destroy.mutate(s.id)}
                >
                  Delete
                </button>
              </div>
            </li>
          ))}
        </ul>
        {renameId && (
          <div className="mt-4 flex gap-2">
            <input
              className="flex-1 rounded-lg border border-black/15 px-3 py-2 text-sm"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
            <button
              type="button"
              className="rounded-lg bg-[var(--empops-accent)] px-3 py-1.5 text-sm text-white"
              onClick={() => rename.mutate()}
            >
              Save
            </button>
          </div>
        )}
      </section>
    </div>
  )
}
