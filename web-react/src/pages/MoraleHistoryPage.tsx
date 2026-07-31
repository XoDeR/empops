import { Link, Navigate, useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import {
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { authFetch } from '@/lib/authFetch'
import { useCompanyContext } from '@/routes/CompanyLayout'
import type { MoraleHistoryPoint } from '@/types/api'

export default function MoraleHistoryPage() {
  const { companyId } = useParams<{ companyId: string }>()
  const { isHrOrAdmin, isManager } = useCompanyContext()

  const query = useQuery({
    queryKey: ['morale-history', companyId],
    queryFn: async () => {
      const res = await authFetch<MoraleHistoryPoint[]>(
        `/companies/${companyId}/morale/history/company`,
      )
      return res.data
    },
    enabled: Boolean(companyId) && (isHrOrAdmin || isManager),
  })

  if (!isHrOrAdmin && !isManager) {
    return <Navigate to={`/companies/${companyId}/dashboard/me`} replace />
  }

  const chartData = [...(query.data ?? [])]
    .reverse()
    .map((p) => ({
      date: p.created_at.slice(0, 10),
      average: Number(p.average.toFixed(2)),
      n: p.number_of_employees ?? 0,
    }))

  return (
    <div className="space-y-6">
      <div>
        <Link
          to={`/companies/${companyId}/dashboard/hr`}
          className="text-sm text-black/55 hover:text-black"
        >
          ← Dashboard
        </Link>
        <h2 className="mt-2 text-xl font-semibold">Company morale</h2>
        <p className="text-sm text-black/55">Daily average emotion (1–3)</p>
      </div>

      <section className="rounded-2xl border border-black/10 bg-white/70 p-4">
        {query.isLoading && <p className="text-sm text-black/60">Loading…</p>}
        {chartData.length === 0 && !query.isLoading && (
          <p className="text-sm text-black/55">No history yet. Run the nightly morale job.</p>
        )}
        {chartData.length > 0 && (
          <div className="h-72 w-full">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={chartData}>
                <CartesianGrid strokeDasharray="3 3" stroke="#00000022" />
                <XAxis dataKey="date" tick={{ fontSize: 12 }} />
                <YAxis domain={[1, 3]} tick={{ fontSize: 12 }} />
                <Tooltip />
                <Line
                  type="monotone"
                  dataKey="average"
                  stroke="var(--empops-accent)"
                  strokeWidth={2}
                  dot={false}
                />
              </LineChart>
            </ResponsiveContainer>
          </div>
        )}
      </section>
    </div>
  )
}
