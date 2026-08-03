import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { authFetch } from '@/lib/authFetch'
import { useCompanyContext } from '@/routes/CompanyLayout'
import type { AmaSession } from '@/types/api'

export default function AmaSessionPage() {
  const { companyId, sessionId } = useParams<{ companyId: string; sessionId: string }>()
  const { isHrOrAdmin } = useCompanyContext()
  const qc = useQueryClient()
  const [question, setQuestion] = useState('')
  const [anonymous, setAnonymous] = useState(false)
  const query = useQuery({
    queryKey: ['ama-session', companyId, sessionId],
    queryFn: async () => (await authFetch<AmaSession>(`/companies/${companyId}/ama-sessions/${sessionId}`)).data,
    enabled: Boolean(companyId && sessionId),
  })
  const ask = useMutation({
    mutationFn: () => authFetch(`/companies/${companyId}/ama-sessions/${sessionId}/questions`, { method: 'POST', body: JSON.stringify({ question, anonymous }) }),
    onSuccess: () => {
      setQuestion('')
      setAnonymous(false)
      void qc.invalidateQueries({ queryKey: ['ama-session', companyId, sessionId] })
    },
  })
  const mark = useMutation({
    mutationFn: ({ id, answered }: { id: string; answered: boolean }) => authFetch(`/companies/${companyId}/ama-sessions/${sessionId}/questions/${id}`, { method: 'PATCH', body: JSON.stringify({ answered }) }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['ama-session', companyId, sessionId] }),
  })
  if (query.isLoading) return <p className="text-black/55">Loading session…</p>
  if (!query.data) return <p className="text-red-700">Session not found.</p>
  return (
    <div className="space-y-6">
      <header><Link to={`/companies/${companyId}/ama`} className="text-sm text-black/50">← AMA</Link><h2 className="mt-2 text-xl font-semibold">{query.data.theme || 'Ask Me Anything'}</h2><p className="text-sm text-black/55">{query.data.happened_at} · {query.data.active ? 'Active' : 'Closed'}</p></header>
      {query.data.active && (
        <form className="space-y-3 rounded-2xl border border-black/10 bg-white/70 p-4" onSubmit={(e) => { e.preventDefault(); if (question.trim()) ask.mutate() }}>
          <textarea className="min-h-24 w-full rounded-lg border border-black/15 px-3 py-2 text-sm" placeholder="Ask a question…" value={question} onChange={(e) => setQuestion(e.target.value)} />
          <div className="flex items-center justify-between gap-3"><label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={anonymous} onChange={(e) => setAnonymous(e.target.checked)} />Ask anonymously</label><button className="rounded-lg bg-[var(--empops-accent)] px-4 py-2 text-sm text-white">Submit question</button></div>
        </form>
      )}
      <section className="space-y-3">
        {(query.data.questions ?? []).map((item) => (
          <article key={item.id} className="rounded-2xl border border-black/10 bg-white/70 p-4">
            <div className="flex items-start justify-between gap-3"><p className="whitespace-pre-wrap">{item.question}</p><span className={`rounded-full px-2 py-1 text-xs ${item.answered ? 'bg-green-100 text-green-800' : 'bg-black/[0.05]'}`}>{item.answered ? 'Answered' : 'Open'}</span></div>
            <div className="mt-3 flex items-center justify-between text-xs text-black/50"><span>{item.anonymous ? 'Anonymous' : item.employee ? `${item.employee.first_name} ${item.employee.last_name}` : 'Member'}</span>{isHrOrAdmin && <button type="button" className="text-[var(--empops-accent)]" onClick={() => mark.mutate({ id: item.id, answered: !item.answered })}>Mark {item.answered ? 'open' : 'answered'}</button>}</div>
          </article>
        ))}
        {!query.data.questions?.length && <p className="text-sm text-black/50">No questions yet.</p>}
      </section>
    </div>
  )
}
