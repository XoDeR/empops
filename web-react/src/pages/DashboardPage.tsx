import { NavLink, Navigate, useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { authFetch } from '@/lib/authFetch'
import { useCompanyContext } from '@/routes/CompanyLayout'
import type { DashboardShell } from '@/types/api'

const VIEWS = ['me', 'team', 'manager', 'hr'] as const
type View = (typeof VIEWS)[number]

function isView(v: string | undefined): v is View {
  return VIEWS.includes(v as View)
}

export default function DashboardPage() {
  const { companyId, view } = useParams<{ companyId: string; view: string }>()
  const { isHrOrAdmin, isManager } = useCompanyContext()

  if (!isView(view)) {
    return <Navigate to={`/companies/${companyId}/dashboard/me`} replace />
  }

  if (view === 'manager' && !isManager) {
    return <Navigate to={`/companies/${companyId}/dashboard/me`} replace />
  }

  if (view === 'hr' && !isHrOrAdmin) {
    return <Navigate to={`/companies/${companyId}/dashboard/me`} replace />
  }

  const dashQuery = useQuery({
    queryKey: ['dashboard', companyId, view],
    queryFn: async () => {
      const res = await authFetch<DashboardShell>(`/companies/${companyId}/dashboard/${view}`)
      return res.data
    },
    enabled: Boolean(companyId),
  })

  const linkClass = ({ isActive }: { isActive: boolean }) =>
    `rounded-lg px-3 py-1.5 text-sm ${
      isActive
        ? 'bg-[var(--empops-accent)] text-white'
        : 'bg-black/[0.04] text-black/70 hover:bg-black/[0.08]'
    }`

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold capitalize">{view} dashboard</h2>
        <p className="text-sm text-black/55">Shell ready — widgets arrive in later steps.</p>
      </div>

      <nav className="flex flex-wrap gap-2">
        <NavLink to={`/companies/${companyId}/dashboard/me`} className={linkClass}>
          Me
        </NavLink>
        <NavLink to={`/companies/${companyId}/dashboard/team`} className={linkClass}>
          Team
        </NavLink>
        {isManager && (
          <NavLink to={`/companies/${companyId}/dashboard/manager`} className={linkClass}>
            Manager
          </NavLink>
        )}
        {isHrOrAdmin && (
          <NavLink to={`/companies/${companyId}/dashboard/hr`} className={linkClass}>
            HR
          </NavLink>
        )}
      </nav>

      {dashQuery.isLoading && <p className="text-black/60">Loading…</p>}
      {dashQuery.isError && (
        <div className="rounded-2xl border border-red-200 bg-red-50 p-4 text-red-700">
          {(dashQuery.error as Error).message}
        </div>
      )}
      {dashQuery.data && (
        <div className="rounded-2xl border border-black/10 bg-white/70 p-6">
          <p className="text-sm text-black/50">No widgets yet</p>
          <ul className="mt-3 text-sm text-black/70">
            <li>is_manager: {String(dashQuery.data.flags.is_manager)}</li>
            <li>can_manage_hr: {String(dashQuery.data.flags.can_manage_hr)}</li>
            <li>is_admin: {String(dashQuery.data.flags.is_admin)}</li>
          </ul>
        </div>
      )}
    </div>
  )
}
