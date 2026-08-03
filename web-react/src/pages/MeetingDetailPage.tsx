import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { authFetch } from '@/lib/authFetch'
import type { Meeting } from '@/types/api'

export default function MeetingDetailPage() {
  const { companyId, groupId, meetingId } = useParams<{ companyId: string; groupId: string; meetingId: string }>()
  const qc = useQueryClient()
  const [summary, setSummary] = useState('')
  const [decisionText, setDecisionText] = useState<Record<string, string>>({})
  const query = useQuery({ queryKey: ['meeting', companyId, groupId, meetingId], queryFn: async () => (await authFetch<Meeting>(`/companies/${companyId}/groups/${groupId}/meetings/${meetingId}`)).data, enabled: Boolean(companyId && groupId && meetingId) })
  const refresh = () => void qc.invalidateQueries({ queryKey: ['meeting', companyId, groupId, meetingId] })
  const addItem = useMutation({ mutationFn: () => authFetch(`/companies/${companyId}/groups/${groupId}/meetings/${meetingId}/agenda`, { method: 'POST', body: JSON.stringify({ summary, description: null }) }), onSuccess: () => { setSummary(''); refresh() } })
  const toggleItem = useMutation({ mutationFn: ({ id, checked }: { id: string; checked: boolean }) => authFetch(`/companies/${companyId}/groups/${groupId}/meetings/${meetingId}/agenda/${id}`, { method: 'PATCH', body: JSON.stringify({ checked }) }), onSuccess: refresh })
  const addDecision = useMutation({ mutationFn: ({ itemId, description }: { itemId: string; description: string }) => authFetch(`/companies/${companyId}/groups/${groupId}/meetings/${meetingId}/agenda/${itemId}/decisions`, { method: 'POST', body: JSON.stringify({ description }) }), onSuccess: (_, v) => { setDecisionText((p) => ({ ...p, [v.itemId]: '' })); refresh() } })
  const complete = useMutation({ mutationFn: () => authFetch(`/companies/${companyId}/groups/${groupId}/meetings/${meetingId}`, { method: 'PATCH', body: JSON.stringify({ happened: true, happened_at: query.data?.happened_at ?? new Date().toISOString().slice(0, 10) }) }), onSuccess: refresh })
  if (query.isLoading) return <p className="text-black/55">Loading meeting…</p>
  if (!query.data) return <p className="text-red-700">Meeting not found.</p>
  const agenda = query.data.agenda_items ?? query.data.agenda ?? []
  return (
    <div className="space-y-6">
      <header><Link to={`/companies/${companyId}/groups/${groupId}`} className="text-sm text-black/50">← Group</Link><div className="mt-2 flex items-center justify-between gap-3"><div><h2 className="text-xl font-semibold">Meeting</h2><p className="text-sm text-black/55">{query.data.happened_at || 'Unscheduled'} · {query.data.happened ? 'Completed' : 'Planned'}</p></div>{!query.data.happened && <button type="button" className="rounded-lg border border-black/15 px-3 py-2 text-sm" onClick={() => complete.mutate()}>Mark completed</button>}</div></header>
      <form className="flex gap-2 rounded-2xl border border-black/10 bg-white/70 p-4" onSubmit={(e) => { e.preventDefault(); if (summary.trim()) addItem.mutate() }}><input className="flex-1 rounded-lg border border-black/15 px-3 py-2 text-sm" placeholder="Agenda item" value={summary} onChange={(e) => setSummary(e.target.value)} /><button className="rounded-lg bg-[var(--empops-accent)] px-3 py-2 text-sm text-white">Add</button></form>
      <section className="space-y-3">
        {agenda.map((item) => <article key={item.id} className="rounded-2xl border border-black/10 bg-white/70 p-4"><label className="flex items-start gap-3"><input type="checkbox" className="mt-1" checked={item.checked} onChange={(e) => toggleItem.mutate({ id: item.id, checked: e.target.checked })} /><span><strong className={item.checked ? 'line-through opacity-60' : ''}>{item.summary}</strong>{item.description && <p className="mt-1 text-sm text-black/55">{item.description}</p>}</span></label><div className="ml-6 mt-3"><h4 className="text-xs font-medium uppercase tracking-wide text-black/45">Decisions</h4><ul className="mt-1 space-y-1 text-sm">{(item.decisions ?? []).map((d) => <li key={d.id}>• {d.description}</li>)}</ul><form className="mt-2 flex gap-2" onSubmit={(e) => { e.preventDefault(); const text = (decisionText[item.id] ?? '').trim(); if (text) addDecision.mutate({ itemId: item.id, description: text }) }}><input className="flex-1 rounded-lg border border-black/15 px-2 py-1 text-sm" placeholder="Record a decision" value={decisionText[item.id] ?? ''} onChange={(e) => setDecisionText((p) => ({ ...p, [item.id]: e.target.value }))} /><button className="rounded-lg border border-black/15 px-2 py-1 text-xs">Add</button></form></div></article>)}
        {!agenda.length && <p className="text-sm text-black/50">No agenda items yet.</p>}
      </section>
    </div>
  )
}
