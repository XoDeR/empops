import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { authFetch } from '@/lib/authFetch'
import { useCompanyContext } from '@/routes/CompanyLayout'
import type { Employee, Group, Meeting } from '@/types/api'

export default function GroupDetailPage() {
  const { companyId, groupId } = useParams<{ companyId: string; groupId: string }>()
  const { isHrOrAdmin } = useCompanyContext()
  const qc = useQueryClient()
  const [employeeId, setEmployeeId] = useState('')
  const [date, setDate] = useState('')
  const group = useQuery({ queryKey: ['group', companyId, groupId], queryFn: async () => (await authFetch<Group>(`/companies/${companyId}/groups/${groupId}`)).data, enabled: Boolean(companyId && groupId) })
  const meetings = useQuery({ queryKey: ['meetings', companyId, groupId], queryFn: async () => (await authFetch<Meeting[]>(`/companies/${companyId}/groups/${groupId}/meetings`)).data, enabled: Boolean(companyId && groupId) })
  const employees = useQuery({ queryKey: ['employees', companyId], queryFn: async () => (await authFetch<Employee[]>(`/companies/${companyId}/employees`)).data, enabled: Boolean(companyId) && isHrOrAdmin })
  const refresh = () => { void qc.invalidateQueries({ queryKey: ['group', companyId, groupId] }); void qc.invalidateQueries({ queryKey: ['meetings', companyId, groupId] }) }
  const member = useMutation({ mutationFn: ({ id, method }: { id: string; method: 'POST' | 'DELETE' }) => authFetch(`/companies/${companyId}/groups/${groupId}/members/${id}`, { method }), onSuccess: () => { setEmployeeId(''); refresh() } })
  const meeting = useMutation({ mutationFn: () => authFetch(`/companies/${companyId}/groups/${groupId}/meetings`, { method: 'POST', body: JSON.stringify({ happened_at: date || null }) }), onSuccess: () => { setDate(''); refresh() } })
  if (group.isLoading) return <p className="text-black/55">Loading group…</p>
  if (!group.data) return <p className="text-red-700">Group not found.</p>
  const memberIds = new Set((group.data.members ?? []).map((m) => m.id))
  return (
    <div className="space-y-6">
      <header><Link to={`/companies/${companyId}/groups`} className="text-sm text-black/50">← Groups</Link><h2 className="mt-2 text-xl font-semibold">{group.data.name}</h2><p className="text-sm text-black/55">{group.data.mission || 'No mission set.'}</p></header>
      <section className="rounded-2xl border border-black/10 bg-white/70 p-4"><h3 className="font-medium">Members</h3><ul className="mt-2 space-y-2 text-sm">{(group.data.members ?? []).map((m) => <li key={m.id} className="flex justify-between"><Link to={`/companies/${companyId}/employees/${m.id}`} className="hover:underline">{m.first_name} {m.last_name}</Link>{isHrOrAdmin && <button type="button" className="text-xs text-red-700" onClick={() => member.mutate({ id: m.id, method: 'DELETE' })}>Remove</button>}</li>)}</ul>{isHrOrAdmin && <div className="mt-3 flex gap-2"><select className="flex-1 rounded-lg border border-black/15 px-3 py-2 text-sm" value={employeeId} onChange={(e) => setEmployeeId(e.target.value)}><option value="">Add member…</option>{(employees.data ?? []).filter((e) => !memberIds.has(e.id)).map((e) => <option key={e.id} value={e.id}>{e.first_name} {e.last_name}</option>)}</select><button type="button" disabled={!employeeId} className="rounded-lg border border-black/15 px-3 py-2 text-sm disabled:opacity-50" onClick={() => member.mutate({ id: employeeId, method: 'POST' })}>Add</button></div>}</section>
      <section className="rounded-2xl border border-black/10 bg-white/70 p-4"><h3 className="font-medium">Meetings</h3><form className="mt-3 flex gap-2" onSubmit={(e) => { e.preventDefault(); meeting.mutate() }}><input type="date" className="rounded-lg border border-black/15 px-3 py-2 text-sm" value={date} onChange={(e) => setDate(e.target.value)} /><button className="rounded-lg bg-[var(--empops-accent)] px-3 py-2 text-sm text-white">New meeting</button></form><ul className="mt-3 divide-y divide-black/5">{(meetings.data ?? []).map((m) => <li key={m.id} className="flex justify-between py-3 text-sm"><Link className="font-medium hover:underline" to={`/companies/${companyId}/groups/${groupId}/meetings/${m.id}`}>{m.happened_at || 'Unscheduled meeting'}</Link><span className="text-black/50">{m.happened ? 'Completed' : 'Planned'}</span></li>)}</ul></section>
    </div>
  )
}
