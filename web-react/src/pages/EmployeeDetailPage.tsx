import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { authFetch } from '@/lib/authFetch'
import { useCompanyContext } from '@/routes/CompanyLayout'
import { ImageUploadField } from '@/components/ImageUploadField'
import { PlacesSection } from '@/components/PlacesSection'
import { resolveMediaUrl } from '@/lib/mediaUrl'
import type { Employee, EmployeeSummary, Hardware, Software } from '@/types/api'

export default function EmployeeDetailPage() {
  const { companyId, employeeId } = useParams<{ companyId: string; employeeId: string }>()
  const { isHrOrAdmin, company } = useCompanyContext()
  const qc = useQueryClient()
  const [tab, setTab] = useState<'profile' | 'work'>('work')
  const [managerId, setManagerId] = useState('')
  const [error, setError] = useState<string | null>(null)

  const employeeQuery = useQuery({
    queryKey: ['employee', companyId, employeeId],
    queryFn: async () => {
      const res = await authFetch<Employee>(`/companies/${companyId}/employees/${employeeId}`)
      return res.data
    },
    enabled: Boolean(companyId && employeeId),
  })

  const reportsQuery = useQuery({
    queryKey: ['direct-reports', companyId, employeeId],
    queryFn: async () => {
      const res = await authFetch<EmployeeSummary[]>(
        `/companies/${companyId}/employees/${employeeId}/direct-reports`,
      )
      return res.data
    },
    enabled: Boolean(companyId && employeeId),
  })

  const employeesQuery = useQuery({
    queryKey: ['employees', companyId],
    queryFn: async () => {
      const res = await authFetch<Employee[]>(`/companies/${companyId}/employees`)
      return res.data
    },
    enabled: Boolean(companyId) && isHrOrAdmin,
  })

  const canSeeAssets =
    Boolean(companyId && employeeId) &&
    (isHrOrAdmin || employeeId === company.employee_id)

  const hardwareQuery = useQuery({
    queryKey: ['employee-hardware', companyId, employeeId],
    queryFn: async () => {
      const res = await authFetch<Hardware[]>(
        `/companies/${companyId}/employees/${employeeId}/hardware`,
      )
      return res.data
    },
    enabled: canSeeAssets,
  })

  const softwaresQuery = useQuery({
    queryKey: ['employee-softwares', companyId, employeeId],
    queryFn: async () => {
      const res = await authFetch<Software[]>(
        `/companies/${companyId}/employees/${employeeId}/softwares`,
      )
      return res.data
    },
    enabled: canSeeAssets,
  })

  const invalidate = () => {
    void qc.invalidateQueries({ queryKey: ['employee', companyId, employeeId] })
    void qc.invalidateQueries({ queryKey: ['direct-reports', companyId, employeeId] })
    void qc.invalidateQueries({ queryKey: ['employees', companyId] })
  }

  const assignManager = useMutation({
    mutationFn: async (manager_id: string) => {
      await authFetch(`/companies/${companyId}/employees/${employeeId}/managers`, {
        method: 'POST',
        body: JSON.stringify({ manager_id }),
      })
    },
    onSuccess: () => {
      setError(null)
      setManagerId('')
      invalidate()
    },
    onError: (e: Error) => setError(e.message),
  })

  const unassignManager = useMutation({
    mutationFn: async (mid: string) => {
      await authFetch(`/companies/${companyId}/employees/${employeeId}/managers/${mid}`, {
        method: 'DELETE',
      })
    },
    onSuccess: () => {
      setError(null)
      invalidate()
    },
    onError: (e: Error) => setError(e.message),
  })

  if (employeeQuery.isLoading) return <p className="text-black/60">Loading employee…</p>
  if (employeeQuery.isError || !employeeQuery.data) {
    return (
      <div className="rounded-2xl border border-red-200 bg-red-50 p-4 text-red-700">
        {(employeeQuery.error as Error)?.message ?? 'Employee not found'}
      </div>
    )
  }

  const employee = employeeQuery.data
  const isSelf = employee.id === company.employee_id
  const canEditAvatar = isSelf || isHrOrAdmin
  const canEditPlaces = isSelf || isHrOrAdmin
  const avatarUrl = resolveMediaUrl(employee.avatar_url)
  const managers = employee.managers ?? (employee.manager ? [employee.manager] : [])
  const managerIds = new Set(managers.map((m) => m.id))
  const managerCandidates = (employeesQuery.data ?? []).filter(
    (e) => e.id !== employee.id && !managerIds.has(e.id),
  )

  const tabClass = (active: boolean) =>
    `border-b-2 pb-2 text-sm ${
      active
        ? 'border-[var(--empops-accent)] font-semibold'
        : 'border-transparent text-black/60 hover:text-black'
    }`

  return (
    <div className="space-y-6">
      <div>
        <Link
          to={`/companies/${companyId}/employees`}
          className="text-sm text-black/50 hover:text-black"
        >
          ← Employees
        </Link>
        <h2 className="mt-2 text-xl font-semibold flex items-center gap-3">
          {avatarUrl && (
            <img src={avatarUrl} alt="" className="h-10 w-10 rounded-full object-cover" />
          )}
          <span>
            {employee.first_name} {employee.last_name}
            {isSelf && <span className="ml-2 text-sm font-normal text-black/45">(you)</span>}
          </span>
        </h2>
        <p className="text-sm text-black/55">{employee.email}</p>
      </div>

      <nav className="flex gap-4 border-b border-black/10">
        <button type="button" className={tabClass(tab === 'profile')} onClick={() => setTab('profile')}>
          Profile
        </button>
        <button type="button" className={tabClass(tab === 'work')} onClick={() => setTab('work')}>
          Work
        </button>
      </nav>

      {error && (
        <div className="rounded-2xl border border-red-200 bg-red-50 p-4 text-red-700">{error}</div>
      )}

      {tab === 'profile' && (
        <div className="space-y-4">
          <div className="rounded-2xl border border-black/10 bg-white/70 p-4 text-sm space-y-4">
            {canEditAvatar && companyId && employeeId && (
              <ImageUploadField
                label="Avatar"
                imageUrl={employee.avatar_url}
                disabled={!canEditAvatar}
                onUpload={async (ids) => {
                  await authFetch(`/companies/${companyId}/employees/${employeeId}/avatar`, {
                    method: 'PUT',
                    body: JSON.stringify(ids),
                  })
                  invalidate()
                }}
              />
            )}
            <p>
              <span className="text-black/50">Position:</span>{' '}
              {employee.position?.title ?? '—'}
            </p>
            <p>
              <span className="text-black/50">Status:</span> {employee.status?.name ?? '—'}
            </p>
            <p>
              <span className="text-black/50">Roles:</span> {employee.roles.join(', ')}
            </p>
            <p>
              <span className="text-black/50">Hired:</span> {employee.hired_at ?? '—'}
            </p>
          </div>

          {companyId && employeeId && (
            <PlacesSection
              companyId={companyId}
              employeeId={employeeId}
              canEdit={canEditPlaces}
            />
          )}
        </div>
      )}

      {tab === 'work' && (
        <div className="space-y-4">
          <section className="rounded-2xl border border-black/10 bg-white/70 p-4">
            <h3 className="font-medium">Manager</h3>
            <ul className="mt-2 space-y-1 text-sm">
              {managers.map((m) => (
                <li key={m.id} className="flex items-center justify-between gap-3">
                  <Link
                    to={`/companies/${companyId}/employees/${m.id}`}
                    className="hover:underline"
                  >
                    {m.first_name} {m.last_name}
                  </Link>
                  {isHrOrAdmin && (
                    <button
                      type="button"
                      className="text-xs text-red-700 hover:underline"
                      onClick={() => unassignManager.mutate(m.id)}
                    >
                      Unassign
                    </button>
                  )}
                </li>
              ))}
              {managers.length === 0 && (
                <li className="text-black/50">No manager assigned.</li>
              )}
            </ul>

            {isHrOrAdmin && (
              <div className="mt-3 flex flex-wrap gap-2">
                <select
                  className="rounded-lg border border-black/15 bg-white px-3 py-1.5 text-sm"
                  value={managerId}
                  onChange={(e) => setManagerId(e.target.value)}
                >
                  <option value="">Select manager…</option>
                  {managerCandidates.map((e) => (
                    <option key={e.id} value={e.id}>
                      {e.first_name} {e.last_name}
                    </option>
                  ))}
                </select>
                <button
                  type="button"
                  disabled={!managerId || assignManager.isPending}
                  className="rounded-lg bg-[var(--empops-accent)] px-3 py-1.5 text-sm text-white disabled:opacity-60"
                  onClick={() => assignManager.mutate(managerId)}
                >
                  Assign manager
                </button>
              </div>
            )}
          </section>

          <section className="rounded-2xl border border-black/10 bg-white/70 p-4">
            <h3 className="font-medium">Teams</h3>
            <ul className="mt-2 space-y-1 text-sm">
              {(employee.teams ?? []).map((t) => (
                <li key={t.id}>
                  <Link
                    to={`/companies/${companyId}/teams/${t.id}`}
                    className="hover:underline"
                  >
                    {t.name}
                  </Link>
                </li>
              ))}
              {(employee.teams ?? []).length === 0 && (
                <li className="text-black/50">Not on any team.</li>
              )}
            </ul>
          </section>

          <section className="rounded-2xl border border-black/10 bg-white/70 p-4">
            <h3 className="font-medium">Direct reports</h3>
            <ul className="mt-2 space-y-1 text-sm">
              {(reportsQuery.data ?? []).map((r) => (
                <li key={r.id}>
                  <Link
                    to={`/companies/${companyId}/employees/${r.id}`}
                    className="hover:underline"
                  >
                    {r.first_name} {r.last_name}
                  </Link>
                </li>
              ))}
              {(reportsQuery.data ?? []).length === 0 && (
                <li className="text-black/50">No direct reports.</li>
              )}
            </ul>
          </section>

          {canSeeAssets && (
            <>
              <section className="rounded-2xl border border-black/10 bg-white/70 p-4">
                <h3 className="font-medium">Assigned hardware</h3>
                <ul className="mt-2 space-y-1 text-sm">
                  {(hardwareQuery.data ?? []).map((h) => (
                    <li key={h.id}>
                      {isHrOrAdmin ? (
                        <Link
                          to={`/companies/${companyId}/hardware/${h.id}`}
                          className="hover:underline"
                        >
                          {h.name}
                          {h.serial_number ? ` (${h.serial_number})` : ''}
                        </Link>
                      ) : (
                        <span>
                          {h.name}
                          {h.serial_number ? ` (${h.serial_number})` : ''}
                        </span>
                      )}
                    </li>
                  ))}
                  {!hardwareQuery.isLoading && (hardwareQuery.data ?? []).length === 0 && (
                    <li className="text-black/50">No hardware assigned.</li>
                  )}
                </ul>
              </section>

              <section className="rounded-2xl border border-black/10 bg-white/70 p-4">
                <h3 className="font-medium">Assigned software</h3>
                <ul className="mt-2 space-y-1 text-sm">
                  {(softwaresQuery.data ?? []).map((s) => (
                    <li key={s.id}>
                      {isHrOrAdmin ? (
                        <Link
                          to={`/companies/${companyId}/softwares/${s.id}`}
                          className="hover:underline"
                        >
                          {s.name}
                        </Link>
                      ) : (
                        <span>{s.name}</span>
                      )}
                    </li>
                  ))}
                  {!softwaresQuery.isLoading && (softwaresQuery.data ?? []).length === 0 && (
                    <li className="text-black/50">No software seats.</li>
                  )}
                </ul>
              </section>
            </>
          )}
        </div>
      )}
    </div>
  )
}
