import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { apiFetch } from '@/lib/api'
import type { PublicJobCompany } from '@/types/api'

function PublicShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen">
      <header className="border-b border-black/10 bg-white/70 backdrop-blur">
        <div className="mx-auto max-w-5xl px-6 py-4">
          <Link
            to="/jobs"
            className="text-sm font-semibold uppercase tracking-[0.2em] text-[var(--empops-accent)]"
          >
            EmpOps Careers
          </Link>
        </div>
      </header>
      <main className="mx-auto max-w-5xl px-6 py-8">{children}</main>
    </div>
  )
}

export default function JobsPage() {
  const companiesQuery = useQuery({
    queryKey: ['public-jobs'],
    queryFn: async () => {
      const res = await apiFetch<PublicJobCompany[]>('/jobs')
      return res.data
    },
  })

  return (
    <PublicShell>
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-semibold">Open positions</h1>
          <p className="text-sm text-black/60">Browse companies hiring on EmpOps.</p>
        </div>

        {companiesQuery.isLoading && <p className="text-black/60">Loading…</p>}
        {companiesQuery.isError && (
          <div className="rounded-2xl border border-red-200 bg-red-50 p-4 text-red-700">
            {(companiesQuery.error as Error).message}
          </div>
        )}

        <ul className="space-y-3">
          {(companiesQuery.data ?? []).map((company) => (
            <li key={company.slug}>
              <Link
                to={`/jobs/${company.slug}`}
                className="block rounded-2xl border border-black/10 bg-white/70 p-4 transition hover:border-black/20"
              >
                <h2 className="font-medium">{company.name}</h2>
                <p className="mt-1 text-sm text-black/60">
                  {company.openings_count} open{' '}
                  {company.openings_count === 1 ? 'position' : 'positions'}
                </p>
              </Link>
            </li>
          ))}
        </ul>

        {companiesQuery.data?.length === 0 && (
          <p className="text-sm text-black/55">No companies with open positions right now.</p>
        )}
      </div>
    </PublicShell>
  )
}
