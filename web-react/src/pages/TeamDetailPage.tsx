import { useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { authFetch } from '@/lib/authFetch'
import { useCompanyContext } from '@/routes/CompanyLayout'
import type { Employee, Ship, Team, TeamNews } from '@/types/api'

export default function TeamDetailPage() {
  const { companyId, teamId } = useParams<{ companyId: string; teamId: string }>()
  const { isHrOrAdmin } = useCompanyContext()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const [memberId, setMemberId] = useState('')
  const [leadId, setLeadId] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [newsTitle, setNewsTitle] = useState('')
  const [newsContent, setNewsContent] = useState('')
  const [shipTitle, setShipTitle] = useState('')
  const [shipDescription, setShipDescription] = useState('')
  const [shipEmployeeIds, setShipEmployeeIds] = useState<string[]>([])

  const teamQuery = useQuery({
    queryKey: ['team', companyId, teamId],
    queryFn: async () => {
      const res = await authFetch<Team>(`/companies/${companyId}/teams/${teamId}`)
      return res.data
    },
    enabled: Boolean(companyId && teamId),
  })

  const employeesQuery = useQuery({
    queryKey: ['employees', companyId],
    queryFn: async () => {
      const res = await authFetch<Employee[]>(`/companies/${companyId}/employees`)
      return res.data
    },
    enabled: Boolean(companyId) && isHrOrAdmin,
  })

  const newsQuery = useQuery({
    queryKey: ['team-news', companyId, teamId],
    queryFn: async () => {
      const res = await authFetch<TeamNews[]>(`/companies/${companyId}/teams/${teamId}/news`)
      return res.data
    },
    enabled: Boolean(companyId && teamId),
  })

  const shipsQuery = useQuery({
    queryKey: ['ships', companyId, teamId],
    queryFn: async () => {
      const res = await authFetch<Ship[]>(`/companies/${companyId}/teams/${teamId}/ships`)
      return res.data
    },
    enabled: Boolean(companyId && teamId),
  })

  const invalidate = () => {
    void qc.invalidateQueries({ queryKey: ['team', companyId, teamId] })
    void qc.invalidateQueries({ queryKey: ['teams', companyId] })
  }

  const addMember = useMutation({
    mutationFn: async (employeeId: string) => {
      await authFetch(`/companies/${companyId}/teams/${teamId}/members/${employeeId}`, {
        method: 'POST',
      })
    },
    onSuccess: () => {
      setError(null)
      setMemberId('')
      invalidate()
    },
    onError: (e: Error) => setError(e.message),
  })

  const removeMember = useMutation({
    mutationFn: async (employeeId: string) => {
      await authFetch(`/companies/${companyId}/teams/${teamId}/members/${employeeId}`, {
        method: 'DELETE',
      })
    },
    onSuccess: () => {
      setError(null)
      invalidate()
    },
    onError: (e: Error) => setError(e.message),
  })

  const setLead = useMutation({
    mutationFn: async (employee_id: string | null) => {
      await authFetch(`/companies/${companyId}/teams/${teamId}/lead`, {
        method: 'PUT',
        body: JSON.stringify({ employee_id }),
      })
    },
    onSuccess: () => {
      setError(null)
      invalidate()
    },
    onError: (e: Error) => setError(e.message),
  })

  const deleteTeam = useMutation({
    mutationFn: async () => {
      await authFetch(`/companies/${companyId}/teams/${teamId}`, { method: 'DELETE' })
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['teams', companyId] })
      navigate(`/companies/${companyId}/teams`)
    },
    onError: (e: Error) => setError(e.message),
  })

  const createNews = useMutation({
    mutationFn: async () => {
      await authFetch(`/companies/${companyId}/teams/${teamId}/news`, {
        method: 'POST',
        body: JSON.stringify({ title: newsTitle, content: newsContent }),
      })
    },
    onSuccess: () => {
      setNewsTitle('')
      setNewsContent('')
      setError(null)
      void qc.invalidateQueries({ queryKey: ['team-news', companyId, teamId] })
    },
    onError: (e: Error) => setError(e.message),
  })

  const deleteNews = useMutation({
    mutationFn: async (newsId: string) => {
      await authFetch(`/companies/${companyId}/teams/${teamId}/news/${newsId}`, {
        method: 'DELETE',
      })
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['team-news', companyId, teamId] })
    },
    onError: (e: Error) => setError(e.message),
  })

  const createShip = useMutation({
    mutationFn: async () => {
      await authFetch(`/companies/${companyId}/teams/${teamId}/ships`, {
        method: 'POST',
        body: JSON.stringify({
          title: shipTitle,
          description: shipDescription || null,
          employee_ids: shipEmployeeIds,
        }),
      })
    },
    onSuccess: () => {
      setShipTitle('')
      setShipDescription('')
      setShipEmployeeIds([])
      setError(null)
      void qc.invalidateQueries({ queryKey: ['ships', companyId, teamId] })
      void qc.invalidateQueries({ queryKey: ['notifications', companyId] })
    },
    onError: (e: Error) => setError(e.message),
  })

  const deleteShip = useMutation({
    mutationFn: async (shipId: string) => {
      await authFetch(`/companies/${companyId}/teams/${teamId}/ships/${shipId}`, {
        method: 'DELETE',
      })
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['ships', companyId, teamId] })
    },
    onError: (e: Error) => setError(e.message),
  })

  if (teamQuery.isLoading) return <p className="text-black/60">Loading team…</p>
  if (teamQuery.isError || !teamQuery.data) {
    return (
      <div className="rounded-2xl border border-red-200 bg-red-50 p-4 text-red-700">
        {(teamQuery.error as Error)?.message ?? 'Team not found'}
      </div>
    )
  }

  const team = teamQuery.data
  const memberIds = new Set(team.members.map((m) => m.id))
  const candidates = (employeesQuery.data ?? []).filter((e) => !memberIds.has(e.id))

  return (
    <div className="space-y-6">
      <div>
        <Link to={`/companies/${companyId}/teams`} className="text-sm text-black/50 hover:text-black">
          ← Teams
        </Link>
        <h2 className="mt-2 text-xl font-semibold">{team.name}</h2>
        {team.description && <p className="text-sm text-black/60">{team.description}</p>}
        <p className="mt-1 text-xs text-black/50">
          Lead:{' '}
          {team.leader
            ? `${team.leader.first_name} ${team.leader.last_name}`
            : 'None'}
        </p>
      </div>

      {error && (
        <div className="rounded-2xl border border-red-200 bg-red-50 p-4 text-red-700">{error}</div>
      )}

      <section className="rounded-2xl border border-black/10 bg-white/70 p-4">
        <h3 className="font-medium">Members</h3>
        <ul className="mt-3 space-y-2">
          {team.members.map((m) => (
            <li key={m.id} className="flex items-center justify-between gap-3 text-sm">
              <Link
                to={`/companies/${companyId}/employees/${m.id}`}
                className="hover:underline"
              >
                {m.first_name} {m.last_name}
                <span className="ml-2 text-black/45">{m.email}</span>
              </Link>
              {isHrOrAdmin && (
                <button
                  type="button"
                  className="text-xs text-red-700 hover:underline"
                  onClick={() => removeMember.mutate(m.id)}
                >
                  Remove
                </button>
              )}
            </li>
          ))}
          {team.members.length === 0 && (
            <li className="text-sm text-black/50">No members yet.</li>
          )}
        </ul>
      </section>

      <section className="space-y-3 rounded-2xl border border-black/10 bg-white/70 p-4">
        <h3 className="font-medium">Team news</h3>
        <ul className="space-y-2">
          {(newsQuery.data ?? []).map((n) => (
            <li key={n.id} className="rounded-lg border border-black/5 p-3 text-sm">
              <div className="flex items-start justify-between gap-2">
                <div>
                  <p className="font-medium">{n.title}</p>
                  <p className="mt-1 whitespace-pre-wrap text-black/70">{n.content}</p>
                  <p className="mt-1 text-xs text-black/45">by {n.author_name}</p>
                </div>
                <button
                  type="button"
                  className="text-xs text-red-700 hover:underline"
                  onClick={() => deleteNews.mutate(n.id)}
                >
                  Delete
                </button>
              </div>
            </li>
          ))}
          {(newsQuery.data ?? []).length === 0 && (
            <li className="text-sm text-black/50">No team news yet.</li>
          )}
        </ul>
        <div className="space-y-2 border-t border-black/5 pt-3">
          <input
            className="w-full rounded-lg border border-black/15 bg-white px-3 py-2 text-sm"
            placeholder="Title"
            value={newsTitle}
            onChange={(e) => setNewsTitle(e.target.value)}
          />
          <textarea
            className="w-full rounded-lg border border-black/15 bg-white px-3 py-2 text-sm"
            rows={2}
            placeholder="What’s new for the team?"
            value={newsContent}
            onChange={(e) => setNewsContent(e.target.value)}
          />
          <button
            type="button"
            disabled={!newsTitle.trim() || !newsContent.trim() || createNews.isPending}
            className="rounded-lg bg-[var(--empops-accent)] px-3 py-1.5 text-sm text-white disabled:opacity-60"
            onClick={() => createNews.mutate()}
          >
            Post news
          </button>
        </div>
      </section>

      <section className="space-y-3 rounded-2xl border border-black/10 bg-white/70 p-4">
        <h3 className="font-medium">Recent ships</h3>
        <ul className="space-y-2">
          {(shipsQuery.data ?? []).map((s) => (
            <li key={s.id} className="rounded-lg border border-black/5 p-3 text-sm">
              <div className="flex items-start justify-between gap-2">
                <div>
                  <p className="font-medium">{s.title}</p>
                  {s.description && (
                    <p className="mt-1 whitespace-pre-wrap text-black/70">{s.description}</p>
                  )}
                  <p className="mt-1 text-xs text-black/45">
                    by {s.author_name}
                    {s.employees.length > 0 &&
                      ` · ${s.employees.map((e) => `${e.first_name} ${e.last_name}`).join(', ')}`}
                  </p>
                </div>
                <button
                  type="button"
                  className="text-xs text-red-700 hover:underline"
                  onClick={() => deleteShip.mutate(s.id)}
                >
                  Delete
                </button>
              </div>
            </li>
          ))}
          {(shipsQuery.data ?? []).length === 0 && (
            <li className="text-sm text-black/50">No ships yet.</li>
          )}
        </ul>
        <div className="space-y-2 border-t border-black/5 pt-3">
          <input
            className="w-full rounded-lg border border-black/15 bg-white px-3 py-2 text-sm"
            placeholder="What shipped?"
            value={shipTitle}
            onChange={(e) => setShipTitle(e.target.value)}
          />
          <textarea
            className="w-full rounded-lg border border-black/15 bg-white px-3 py-2 text-sm"
            rows={2}
            placeholder="Optional description"
            value={shipDescription}
            onChange={(e) => setShipDescription(e.target.value)}
          />
          <div className="flex flex-wrap gap-2">
            {team.members.map((m) => {
              const checked = shipEmployeeIds.includes(m.id)
              return (
                <label key={m.id} className="flex items-center gap-1.5 text-xs text-black/70">
                  <input
                    type="checkbox"
                    checked={checked}
                    onChange={() =>
                      setShipEmployeeIds((ids) =>
                        checked ? ids.filter((id) => id !== m.id) : [...ids, m.id],
                      )
                    }
                  />
                  {m.first_name} {m.last_name}
                </label>
              )
            })}
          </div>
          <button
            type="button"
            disabled={!shipTitle.trim() || createShip.isPending}
            className="rounded-lg bg-[var(--empops-accent)] px-3 py-1.5 text-sm text-white disabled:opacity-60"
            onClick={() => createShip.mutate()}
          >
            Log ship
          </button>
        </div>
      </section>

      {isHrOrAdmin && (
        <section className="grid gap-4 rounded-2xl border border-black/10 bg-white/70 p-4 sm:grid-cols-2">
          <div className="space-y-2">
            <h3 className="font-medium">Add member</h3>
            <select
              className="w-full rounded-lg border border-black/15 bg-white px-3 py-2 text-sm"
              value={memberId}
              onChange={(e) => setMemberId(e.target.value)}
            >
              <option value="">Select employee…</option>
              {candidates.map((e) => (
                <option key={e.id} value={e.id}>
                  {e.first_name} {e.last_name}
                </option>
              ))}
            </select>
            <button
              type="button"
              disabled={!memberId || addMember.isPending}
              className="rounded-lg bg-[var(--empops-accent)] px-3 py-1.5 text-sm text-white disabled:opacity-60"
              onClick={() => addMember.mutate(memberId)}
            >
              Add
            </button>
          </div>

          <div className="space-y-2">
            <h3 className="font-medium">Set lead</h3>
            <select
              className="w-full rounded-lg border border-black/15 bg-white px-3 py-2 text-sm"
              value={leadId}
              onChange={(e) => setLeadId(e.target.value)}
            >
              <option value="">Select member…</option>
              {team.members.map((m) => (
                <option key={m.id} value={m.id}>
                  {m.first_name} {m.last_name}
                </option>
              ))}
            </select>
            <div className="flex gap-2">
              <button
                type="button"
                disabled={!leadId || setLead.isPending}
                className="rounded-lg bg-[var(--empops-accent)] px-3 py-1.5 text-sm text-white disabled:opacity-60"
                onClick={() => setLead.mutate(leadId)}
              >
                Set lead
              </button>
              <button
                type="button"
                disabled={setLead.isPending || !team.leader}
                className="rounded-lg border border-black/15 px-3 py-1.5 text-sm disabled:opacity-60"
                onClick={() => setLead.mutate(null)}
              >
                Clear lead
              </button>
            </div>
          </div>

          <div className="sm:col-span-2">
            <button
              type="button"
              className="text-sm text-red-700 hover:underline"
              onClick={() => {
                if (confirm('Delete this team?')) deleteTeam.mutate()
              }}
            >
              Delete team
            </button>
          </div>
        </section>
      )}
    </div>
  )
}
