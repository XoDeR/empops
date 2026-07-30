import { useEffect } from 'react'
import { NavLink, Outlet, useOutletContext, useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { NotificationBell } from '@/components/NotificationBell'
import { authFetch } from '@/lib/authFetch'
import { useCompanyStore } from '@/stores/company'
import type { CompanyMembership } from '@/types/api'

export type CompanyContextValue = {
  company: CompanyMembership
  isAdmin: boolean
  isHrOrAdmin: boolean
  isManager: boolean
  isAccountant: boolean
}

function tabClass({ isActive }: { isActive: boolean }) {
  return `border-b-2 pb-2 ${
    isActive
      ? 'border-[var(--empops-accent)] font-semibold'
      : 'border-transparent text-black/60 hover:text-black'
  }`
}

export default function CompanyLayout() {
  const { companyId } = useParams<{ companyId: string }>()
  const setSelectedCompanyId = useCompanyStore((s) => s.setSelectedCompanyId)

  const companyQuery = useQuery({
    queryKey: ['company', companyId],
    queryFn: async () => {
      const res = await authFetch<CompanyMembership>(`/companies/${companyId}`)
      return res.data
    },
    enabled: Boolean(companyId),
    retry: false,
  })

  useEffect(() => {
    if (companyId) setSelectedCompanyId(companyId)
  }, [companyId, setSelectedCompanyId])

  if (companyQuery.isLoading) {
    return <p className="text-black/60">Loading company…</p>
  }

  if (companyQuery.isError || !companyQuery.data) {
    return (
      <div className="rounded-2xl border border-red-200 bg-red-50 p-5 text-red-700">
        {(companyQuery.error as Error)?.message ?? 'Unable to load company.'}
      </div>
    )
  }

  const company = companyQuery.data
  const roles = company.roles
  const isAdmin = roles.includes('administrator')
  const isHrOrAdmin = isAdmin || roles.includes('hr')
  const isManager = roles.includes('manager') || isHrOrAdmin
  const isAccountant = roles.includes('accountant') || isHrOrAdmin

  const context: CompanyContextValue = {
    company,
    isAdmin,
    isHrOrAdmin,
    isManager,
    isAccountant,
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <p className="text-sm text-black/50">Company</p>
          <h1 className="text-2xl font-semibold">{company.name}</h1>
        </div>
        {companyId && <NotificationBell companyId={companyId} />}
      </div>

      <nav className="flex flex-wrap gap-4 border-b border-black/10 text-sm">
        <NavLink to={`/companies/${companyId}/dashboard/me`} className={tabClass}>
          Dashboard
        </NavLink>
        <NavLink to={`/companies/${companyId}/employees`} className={tabClass}>
          Employees
        </NavLink>
        <NavLink to={`/companies/${companyId}/teams`} className={tabClass}>
          Teams
        </NavLink>
        {isHrOrAdmin && (
          <NavLink to={`/companies/${companyId}/adminland`} className={tabClass}>
            Adminland
          </NavLink>
        )}
      </nav>

      <Outlet context={context} />
    </div>
  )
}

export function useCompanyContext() {
  return useOutletContext<CompanyContextValue>()
}
