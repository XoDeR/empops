import { Link, useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { apiFetch } from '@/lib/api'
import type { PublicJobOpening } from '@/types/api'

type PublicCompanyJobs = {
  slug: string
  name: string
  openings: PublicJobOpening[]
}

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

export default function CompanyJobsPage() {
  const { companySlug } = useParams<{ companySlug: string }>()

  const jobsQuery = useQuery({
    queryKey: ['public-company-jobs', companySlug],
    queryFn: async () => {
      const res = await apiFetch<PublicCompanyJobs>(`/jobs/${companySlug}`)
      return res.data
    },
    enabled: Boolean(companySlug),
  })

  return (
    <PublicShell>
      <div className="space-y-6">
        <div>
          <Link to="/jobs" className="text-sm text-black/50 hover:text-black">
            ← All companies
          </Link>
          <h1 className="mt-2 text-2xl font-semibold">
            {jobsQuery.data?.name ?? 'Loading…'}
          </h1>
          <p className="text-sm text-black/60">Open positions</p>
        </div>

        {jobsQuery.isLoading && <p className="text-black/60">Loading openings…</p>}
        {jobsQuery.isError && (
          <div className="rounded-2xl border border-red-200 bg-red-50 p-4 text-red-700">
            {(jobsQuery.error as Error).message}
          </div>
        )}

        <ul className="space-y-3">
          {(jobsQuery.data?.openings ?? []).map((job) => (
            <li key={job.slug}>
              <Link
                to={`/jobs/${companySlug}/jobs/${job.slug}`}
                className="block rounded-2xl border border-black/10 bg-white/70 p-4 transition hover:border-black/20"
              >
                <h2 className="font-medium">{job.title}</h2>
                {job.reference_number && (
                  <p className="text-xs text-black/50">Ref #{job.reference_number}</p>
                )}
              </Link>
            </li>
          ))}
        </ul>

        {jobsQuery.data?.openings.length === 0 && (
          <p className="text-sm text-black/55">No open positions at this company.</p>
        )}
      </div>
    </PublicShell>
  )
}
