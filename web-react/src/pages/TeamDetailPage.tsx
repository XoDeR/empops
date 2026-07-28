import { useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { authFetch } from '@/lib/authFetch'
import { useCompanyContext } from '@/routes/CompanyLayout'
import type { Employee, Team } from '@/types/api'

export default function TeamDetailPage() {
  const { companyId, teamId } = useParams<{ companyId: string; teamId: string }>()
  const { isHrOrAdmin } = useCompanyContext()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const [memberId, setMemberId] = useState('')
  const [leadId, setLeadId] = useState('')
  const [error, setError] = useState<string | null>(null)

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
