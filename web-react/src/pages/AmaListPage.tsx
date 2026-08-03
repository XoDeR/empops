import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { authFetch } from '@/lib/authFetch'
import { useCompanyContext } from '@/routes/CompanyLayout'
import type { AmaSession } from '@/types/api'

export default function AmaListPage() {
  const { companyId } = useParams<{ companyId: string }>()
  const { isHrOrAdmin } = useCompanyContext()
  const qc = useQueryClient()
  const [happenedAt, setHappenedAt] = useState(new Date().toISOString().slice(0, 10))
  const [theme, setTheme] = useState('')
  const query = useQuery({
    queryKey: ['ama-sessions', companyId],
    queryFn: async () => (await authFetch<AmaSession[]>(`/companies/${companyId}/ama-sessions`)).data,
    enabled: Boolean(companyId),
  })
  const create = useMutation({
    mutationFn: () => authFetch(`/companies/${companyId}/ama-sessions`, { method: 'POST', body: JSON.stringify({ happened_at: happenedAt, active: true, theme: theme || null }) }),
    onSuccess: () => {
      setTheme('')
      void qc.invalidateQueries({ queryKey: ['ama-sessions', companyId] })
    },
  })
  return (
    <div className="space-y-6">
      <header><h2 className="text-xl font-semibold">Ask Me Anything</h2><p className="text-sm text-black/55">Open questions and company conversations.</p></header>
      {isHrOrAdmin && (
        <form className="grid gap-2 rounded-2xl border border-black/10 bg-white/70 p-4 sm:grid-cols-[1fr_auto_auto]" onSubmit={(e) => { e.preventDefault(); create.mutate() }}>
          <input className="rounded-lg border border-black/15 px-3 py-2 text-sm" placeholder="Session theme (optional)" value={theme} onChange={(e) => setTheme(e.target.value)} />
          <input type="date" className="rounded-lg border border-black/15 px-3 py-2 text-sm" value={happenedAt} onChange={(e) => setHappenedAt(e.target.value)} />
          <button className="rounded-lg bg-[var(--empops-accent)] px-4 py-2 text-sm text-white">Create session</button>
        </form>
      )}
      <section className="rounded-2xl border border-black/10 bg-white/70 p-4">
        {query.isLoading && <p className="text-sm text-black/55">Loading…</p>}
        <ul className="divide-y divide-black/5">
          {(query.data ?? []).map((session) => (
            <li key={session.id} className="flex items-center justify-between gap-3 py-3">
              <div><Link className="font-medium hover:underline" to={`/companies/${companyId}/ama/${session.id}`}>{session.theme || 'Open AMA'}</Link><p className="text-xs text-black/50">{session.happened_at} · {session.active ? 'Active' : 'Closed'} · {session.questions?.length ?? 0} questions</p></div>
              <Link className="text-sm text-[var(--empops-accent)]" to={`/companies/${companyId}/ama/${session.id}`}>Open</Link>
            </li>
          ))}
        </ul>
        {!query.isLoading && !query.data?.length && <p className="text-sm text-black/50">No AMA sessions yet.</p>}
      </section>
    </div>
  )
}
