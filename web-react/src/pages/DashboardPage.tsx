import { useState } from 'react'
import { NavLink, Navigate, useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { authFetch } from '@/lib/authFetch'
import { useCompanyContext } from '@/routes/CompanyLayout'
import type {
  DashboardShell,
  DashboardWidget,
  Worklog,
} from '@/types/api'

const VIEWS = ['me', 'team', 'manager', 'hr'] as const
type View = (typeof VIEWS)[number]

function isView(v: string | undefined): v is View {
  return VIEWS.includes(v as View)
}

function widgetOf<T extends DashboardWidget['type']>(
  widgets: DashboardWidget[],
  type: T,
): Extract<DashboardWidget, { type: T }> | undefined {
  return widgets.find((w): w is Extract<DashboardWidget, { type: T }> => w.type === type)
}

function WorklogWidget({
  companyId,
  data,
}: {
  companyId: string
  data: Extract<DashboardWidget, { type: 'worklog_today' }>['data']
}) {
  const qc = useQueryClient()
  const [content, setContent] = useState('')
  const [error, setError] = useState<string | null>(null)

  const save = useMutation({
    mutationFn: async () => {
      await authFetch<Worklog>(`/companies/${companyId}/worklogs`, {
        method: 'POST',
        body: JSON.stringify({ content }),
      })
    },
    onSuccess: () => {
      setContent('')
      setError(null)
      void qc.invalidateQueries({ queryKey: ['dashboard', companyId, 'me'] })
    },
    onError: (e: Error) => setError(e.message),
  })

  return (
    <section className="rounded-2xl border border-black/10 bg-white/70 p-4">
      <h3 className="font-medium">Today’s work log</h3>
      {data.logged && data.worklog ? (
        <p className="mt-2 whitespace-pre-wrap text-sm text-black/75">{data.worklog.content}</p>
      ) : (
        <div className="mt-3 space-y-2">
          <textarea
            className="w-full rounded-lg border border-black/15 bg-white px-3 py-2 text-sm"
            rows={3}
            placeholder="What did you work on today?"
            value={content}
            onChange={(e) => setContent(e.target.value)}
          />
          {error && <p className="text-sm text-red-700">{error}</p>}
          <button
            type="button"
            disabled={!content.trim() || save.isPending}
            className="rounded-lg bg-[var(--empops-accent)] px-3 py-1.5 text-sm text-white disabled:opacity-60"
            onClick={() => save.mutate()}
          >
            Log work
          </button>
        </div>
      )}
      {data.consecutive_missed > 0 && (
        <p className="mt-2 text-xs text-amber-800">
          Missed work logs in a row: {data.consecutive_missed}
        </p>
      )}
    </section>
  )
}

function ActiveQuestionWidget({
  companyId,
  data,
}: {
  companyId: string
  data: Extract<DashboardWidget, { type: 'active_question' }>['data']
}) {
  const qc = useQueryClient()
  const [body, setBody] = useState('')
  const [error, setError] = useState<string | null>(null)

  const save = useMutation({
    mutationFn: async (questionId: string) => {
      await authFetch(`/companies/${companyId}/questions/${questionId}/answers`, {
        method: 'POST',
        body: JSON.stringify({ body }),
      })
    },
    onSuccess: () => {
      setBody('')
      setError(null)
      void qc.invalidateQueries({ queryKey: ['dashboard', companyId, 'me'] })
      void qc.invalidateQueries({ queryKey: ['questions', companyId] })
    },
    onError: (e: Error) => setError(e.message),
  })

  if (!data) {
    return (
      <section className="rounded-2xl border border-black/10 bg-white/70 p-4">
        <h3 className="font-medium">Get to know colleagues</h3>
        <p className="mt-2 text-sm text-black/55">No active question right now.</p>
      </section>
    )
  }

  return (
    <section className="rounded-2xl border border-black/10 bg-white/70 p-4">
      <h3 className="font-medium">Get to know colleagues</h3>
      <p className="mt-2 text-sm text-black/80">{data.title}</p>
      {data.answered ? (
        <p className="mt-2 text-xs text-black/50">You already answered this question.</p>
      ) : (
        <div className="mt-3 space-y-2">
          <textarea
            className="w-full rounded-lg border border-black/15 bg-white px-3 py-2 text-sm"
            rows={2}
            placeholder="Your answer…"
            value={body}
            onChange={(e) => setBody(e.target.value)}
          />
          {error && <p className="text-sm text-red-700">{error}</p>}
          <button
            type="button"
            disabled={!body.trim() || save.isPending}
            className="rounded-lg bg-[var(--empops-accent)] px-3 py-1.5 text-sm text-white disabled:opacity-60"
            onClick={() => save.mutate(data.id)}
          >
            Submit answer
          </button>
        </div>
      )}
    </section>
  )
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

  const widgets = dashQuery.data?.widgets ?? []
  const worklog = widgetOf(widgets, 'worklog_today')
  const question = widgetOf(widgets, 'active_question')
  const unread = widgetOf(widgets, 'unread_notifications')

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold capitalize">{view} dashboard</h2>
        <p className="text-sm text-black/55">
          {view === 'me'
            ? 'Work logs, Q&A, and notifications for your day.'
            : 'Shell ready — more widgets arrive in later steps.'}
        </p>
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

      {dashQuery.data && view === 'me' && companyId && (
        <div className="grid gap-4 lg:grid-cols-2">
          {worklog && <WorklogWidget companyId={companyId} data={worklog.data} />}
          {question && <ActiveQuestionWidget companyId={companyId} data={question.data} />}
          {unread && (
            <section className="rounded-2xl border border-black/10 bg-white/70 p-4 lg:col-span-2">
              <h3 className="font-medium">Notifications</h3>
              <p className="mt-2 text-sm text-black/70">
                Unread: {unread.data.count}
              </p>
            </section>
          )}
        </div>
      )}

      {dashQuery.data && view !== 'me' && (
        <div className="rounded-2xl border border-black/10 bg-white/70 p-6">
          <p className="text-sm text-black/50">No widgets yet for this view.</p>
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
