import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { authFetch } from '@/lib/authFetch'
import { useCompanyContext } from '@/routes/CompanyLayout'
import type { Group } from '@/types/api'

export default function GroupsPage() {
  const { companyId } = useParams<{ companyId: string }>()
  const { isHrOrAdmin } = useCompanyContext()
  const qc = useQueryClient()
  const [name, setName] = useState('')
  const [mission, setMission] = useState('')
  const query = useQuery({ queryKey: ['groups', companyId], queryFn: async () => (await authFetch<Group[]>(`/companies/${companyId}/groups`)).data, enabled: Boolean(companyId) })
  const create = useMutation({
    mutationFn: () => authFetch(`/companies/${companyId}/groups`, { method: 'POST', body: JSON.stringify({ name, mission: mission || null }) }),
    onSuccess: () => {
      setName(''); setMission('')
      void qc.invalidateQueries({ queryKey: ['groups', companyId] })
    },
  })
  return (
    <div className="space-y-6">
      <header><h2 className="text-xl font-semibold">Groups</h2><p className="text-sm text-black/55">Communities, committees, and recurring meetings.</p></header>
      {isHrOrAdmin && <form className="grid gap-2 rounded-2xl border border-black/10 bg-white/70 p-4 sm:grid-cols-2" onSubmit={(e) => { e.preventDefault(); if (name.trim()) create.mutate() }}><input className="rounded-lg border border-black/15 px-3 py-2 text-sm" placeholder="Group name" value={name} onChange={(e) => setName(e.target.value)} /><input className="rounded-lg border border-black/15 px-3 py-2 text-sm" placeholder="Mission (optional)" value={mission} onChange={(e) => setMission(e.target.value)} /><button className="rounded-lg bg-[var(--empops-accent)] px-4 py-2 text-sm text-white sm:col-span-2">Create group</button></form>}
      <section className="grid gap-4 sm:grid-cols-2">
        {(query.data ?? []).map((group) => <Link key={group.id} to={`/companies/${companyId}/groups/${group.id}`} className="rounded-2xl border border-black/10 bg-white/70 p-4 hover:border-[var(--empops-accent)]"><h3 className="font-medium">{group.name}</h3><p className="mt-1 text-sm text-black/55">{group.mission || 'No mission set.'}</p><p className="mt-3 text-xs text-black/45">{group.members?.length ?? 0} members</p></Link>)}
      </section>
      {query.isLoading && <p className="text-sm text-black/55">Loading…</p>}
      {!query.isLoading && !query.data?.length && <p className="text-sm text-black/50">No groups yet.</p>}
    </div>
  )
}
