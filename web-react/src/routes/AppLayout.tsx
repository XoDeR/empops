import { Link, NavLink, Outlet, useNavigate, useParams } from 'react-router-dom'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { apiFetch } from '@/lib/api'
import { authFetch } from '@/lib/authFetch'
import { useAuthStore } from '@/stores/auth'
import { useCompanyStore } from '@/stores/company'
import type { CompanyMembership, InstanceFlags } from '@/types/api'

export default function AppLayout() {
  const navigate = useNavigate()
  const { companyId: activeCompanyId } = useParams()
  const user = useAuthStore((s) => s.user)
  const refreshToken = useAuthStore((s) => s.refreshToken)
  const clearSession = useAuthStore((s) => s.clearSession)
  const setSelectedCompanyId = useCompanyStore((s) => s.setSelectedCompanyId)
  const queryClient = useQueryClient()

  const companiesQuery = useQuery({
    queryKey: ['companies'],
    queryFn: async () => {
      const res = await authFetch<CompanyMembership[]>('/companies')
      return res.data
    },
  })
  const instanceQuery = useQuery({
    queryKey: ['instance'],
    queryFn: async () => (await apiFetch<InstanceFlags>('/instance')).data,
    retry: false,
  })

  const handleLogout = async () => {
    try {
      await apiFetch('/auth/logout', {
        method: 'POST',
        body: JSON.stringify({ refresh_token: refreshToken }),
      })
    } catch {
      // best-effort; clear local session regardless
    }
    clearSession()
    setSelectedCompanyId(null)
    queryClient.clear()
    navigate('/login', { replace: true })
  }

  const handleSwitchCompany = (companyId: string) => {
    if (!companyId) return
    setSelectedCompanyId(companyId)
    navigate(`/companies/${companyId}/employees`)
  }

  return (
    <div className="min-h-screen">
      {instanceQuery.data?.demo_mode && (
        <div className="bg-amber-100 px-4 py-2 text-center text-sm text-amber-900">
          Demo mode — data may be reset periodically.
        </div>
      )}
      <header className="border-b border-black/10 bg-white/70 backdrop-blur">
        <div className="mx-auto flex max-w-5xl flex-wrap items-center gap-4 px-6 py-4">
          <Link
            to="/"
            className="text-sm font-semibold uppercase tracking-[0.2em] text-[var(--empops-accent)]"
          >
            EmpOps
          </Link>

          <nav className="flex flex-1 flex-wrap items-center gap-3 text-sm">
            <NavLink
              to="/"
              end
              className={({ isActive }) =>
                isActive ? 'font-semibold' : 'text-black/60 hover:text-black'
              }
            >
              Companies
            </NavLink>

            {companiesQuery.data && companiesQuery.data.length > 0 && (
              <select
                aria-label="Switch company"
                className="rounded-lg border border-black/15 bg-white px-2 py-1 text-sm"
                value={activeCompanyId ?? ''}
                onChange={(e) => handleSwitchCompany(e.target.value)}
              >
                <option value="" disabled>
                  Switch company…
                </option>
                {companiesQuery.data.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.name}
                  </option>
                ))}
              </select>
            )}
          </nav>

          <div className="flex items-center gap-3 text-sm">
            <span className="text-black/60">{user?.name ?? user?.email}</span>
            <button
              type="button"
              onClick={() => void handleLogout()}
              className="rounded-lg border border-black/15 px-3 py-1.5 hover:bg-black/[0.03]"
            >
              Log out
            </button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-5xl px-6 py-8">
        <Outlet />
      </main>
    </div>
  )
}
