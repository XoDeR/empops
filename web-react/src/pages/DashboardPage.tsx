import { useState } from 'react'
import { NavLink, Navigate, useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { authFetch } from '@/lib/authFetch'
import { useCompanyContext } from '@/routes/CompanyLayout'
import type {
  DashboardShell,
  DashboardWidget,
  Expense,
  ExpenseCategory,
  ProjectSummary,
  ProjectTaskSummary,
  Timesheet,
  Worklog,
} from '@/types/api'

const VIEWS = ['me', 'team', 'manager', 'hr', 'accountant'] as const
type View = (typeof VIEWS)[number]

function isView(v: string | undefined): v is View {
  return VIEWS.includes(v as View)
}

function todayISO() {
  return new Date().toISOString().slice(0, 10)
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

function TimesheetWidget({
  companyId,
  data,
}: {
  companyId: string
  data: Timesheet | null
}) {
  const qc = useQueryClient()
  const [day, setDay] = useState(todayISO())
  const [hours, setHours] = useState('8')
  const [description, setDescription] = useState('')
  const [projectId, setProjectId] = useState('')
  const [projectTaskId, setProjectTaskId] = useState('')
  const [error, setError] = useState<string | null>(null)

  const editable = data?.status === 'open' || data?.status === 'rejected'

  const projectsQuery = useQuery({
    queryKey: ['timesheet-projects', companyId],
    queryFn: async () => {
      const res = await authFetch<ProjectSummary[]>(
        `/companies/${companyId}/timesheets/projects`,
      )
      return res.data
    },
    enabled: Boolean(companyId) && editable,
  })

  const tasksQuery = useQuery({
    queryKey: ['timesheet-project-tasks', companyId, projectId],
    queryFn: async () => {
      const res = await authFetch<ProjectTaskSummary[]>(
        `/companies/${companyId}/timesheets/projects/${projectId}/tasks`,
      )
      return res.data
    },
    enabled: Boolean(companyId && projectId) && editable,
  })

  const upsert = useMutation({
    mutationFn: async () => {
      if (!data) return
      const duration = Math.round(Number(hours) * 60)
      await authFetch(`/companies/${companyId}/timesheets/${data.id}/entries`, {
        method: 'POST',
        body: JSON.stringify({
          happened_at: day,
          duration,
          description: description.trim() || null,
          ...(projectId ? { project_id: projectId } : {}),
          ...(projectTaskId ? { project_task_id: projectTaskId } : {}),
        }),
      })
    },
    onSuccess: () => {
      setDescription('')
      setError(null)
      void qc.invalidateQueries({ queryKey: ['dashboard', companyId, 'me'] })
    },
    onError: (e: Error) => setError(e.message),
  })

  const submit = useMutation({
    mutationFn: async () => {
      if (!data) return
      await authFetch(`/companies/${companyId}/timesheets/${data.id}/submit`, { method: 'POST' })
    },
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['dashboard', companyId, 'me'] }),
    onError: (e: Error) => setError(e.message),
  })

  if (!data) {
    return (
      <section className="rounded-2xl border border-black/10 bg-white/70 p-4">
        <h3 className="font-medium">This week’s timesheet</h3>
        <p className="mt-2 text-sm text-black/55">Unable to load timesheet.</p>
      </section>
    )
  }

  return (
    <section className="rounded-2xl border border-black/10 bg-white/70 p-4 lg:col-span-2">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h3 className="font-medium">This week’s timesheet</h3>
        <p className="text-xs text-black/55">
          {data.started_at} → {data.ended_at} · {data.status.replaceAll('_', ' ')} ·{' '}
          {(data.total_duration / 60).toFixed(1)}h
        </p>
      </div>
      <ul className="mt-3 space-y-1 text-sm text-black/75">
        {data.entries.map((entry) => (
          <li key={entry.id}>
            {entry.happened_at}: {(entry.duration / 60).toFixed(1)}h
            {entry.project_name && (
              <span className="text-black/55"> · {entry.project_name}</span>
            )}
            {entry.project_task_title && (
              <span className="text-black/55"> / {entry.project_task_title}</span>
            )}
            {entry.description ? ` — ${entry.description}` : ''}
          </li>
        ))}
        {data.entries.length === 0 && (
          <li className="text-black/50">No entries yet.</li>
        )}
      </ul>
      {editable && (
        <div className="mt-3 grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
          <input
            type="date"
            className="rounded-lg border border-black/15 bg-white px-3 py-2 text-sm"
            value={day}
            min={data.started_at}
            max={data.ended_at}
            onChange={(e) => setDay(e.target.value)}
          />
          <input
            type="number"
            min={0.25}
            max={24}
            step={0.25}
            className="rounded-lg border border-black/15 bg-white px-3 py-2 text-sm"
            value={hours}
            onChange={(e) => setHours(e.target.value)}
            placeholder="Hours"
          />
          <select
            className="rounded-lg border border-black/15 bg-white px-3 py-2 text-sm"
            value={projectId}
            onChange={(e) => {
              setProjectId(e.target.value)
              setProjectTaskId('')
            }}
          >
            <option value="">No project (ad-hoc)</option>
            {(projectsQuery.data ?? []).map((p) => (
              <option key={p.id} value={p.id}>
                {p.emoji ? `${p.emoji} ` : ''}
                {p.name}
              </option>
            ))}
          </select>
          <select
            className="rounded-lg border border-black/15 bg-white px-3 py-2 text-sm sm:col-span-2"
            value={projectTaskId}
            disabled={!projectId}
            onChange={(e) => setProjectTaskId(e.target.value)}
          >
            <option value="">No task</option>
            {(tasksQuery.data ?? []).map((t) => (
              <option key={t.id} value={t.id}>
                {t.completed ? '✓ ' : ''}
                {t.title}
              </option>
            ))}
          </select>
          <input
            className="rounded-lg border border-black/15 bg-white px-3 py-2 text-sm lg:col-span-3"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Description (optional)"
          />
        </div>
      )}
      {error && <p className="mt-2 text-sm text-red-700">{error}</p>}
      {editable && (
        <div className="mt-3 flex flex-wrap gap-2">
          <button
            type="button"
            disabled={upsert.isPending || !hours}
            className="rounded-lg bg-[var(--empops-accent)] px-3 py-1.5 text-sm text-white disabled:opacity-60"
            onClick={() => upsert.mutate()}
          >
            Save entry
          </button>
          <button
            type="button"
            disabled={submit.isPending || data.entries.length === 0}
            className="rounded-lg border border-black/15 px-3 py-1.5 text-sm hover:bg-white disabled:opacity-60"
            onClick={() => submit.mutate()}
          >
            Submit week
          </button>
        </div>
      )}
    </section>
  )
}

function WfhWidget({
  companyId,
  employeeId,
  data,
}: {
  companyId: string
  employeeId: string
  data: { work_from_home: boolean }
}) {
  const qc = useQueryClient()
  const toggle = useMutation({
    mutationFn: async (work_from_home: boolean) => {
      await authFetch(`/companies/${companyId}/employees/${employeeId}/work-from-home`, {
        method: 'PUT',
        body: JSON.stringify({ date: todayISO(), work_from_home }),
      })
    },
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['dashboard', companyId, 'me'] }),
  })

  return (
    <section className="rounded-2xl border border-black/10 bg-white/70 p-4">
      <h3 className="font-medium">Work from home</h3>
      <p className="mt-2 text-sm text-black/70">
        Today: {data.work_from_home ? 'Working from home' : 'In office / not marked'}
      </p>
      <button
        type="button"
        className="mt-3 rounded-lg border border-black/15 px-3 py-1.5 text-sm hover:bg-white disabled:opacity-60"
        disabled={toggle.isPending}
        onClick={() => toggle.mutate(!data.work_from_home)}
      >
        {data.work_from_home ? 'Clear WFH' : 'Mark WFH today'}
      </button>
    </section>
  )
}

function ExpenseSubmitWidget({ companyId }: { companyId: string }) {
  const qc = useQueryClient()
  const { company } = useCompanyContext()
  const [title, setTitle] = useState('')
  const [amount, setAmount] = useState('')
  const [currency, setCurrency] = useState(company.currency)
  const [categoryId, setCategoryId] = useState('')
  const [error, setError] = useState<string | null>(null)

  const categories = useQuery({
    queryKey: ['expense-categories', companyId],
    queryFn: async () => {
      const res = await authFetch<ExpenseCategory[]>(`/companies/${companyId}/expense-categories`)
      return res.data
    },
  })

  const create = useMutation({
    mutationFn: async () => {
      const cents = Math.round(Number(amount) * 100)
      await authFetch(`/companies/${companyId}/expenses`, {
        method: 'POST',
        body: JSON.stringify({
          title,
          amount: cents,
          currency: currency.toUpperCase(),
          expensed_at: todayISO(),
          expense_category_id: categoryId || null,
        }),
      })
    },
    onSuccess: () => {
      setTitle('')
      setAmount('')
      setError(null)
      void qc.invalidateQueries({ queryKey: ['expenses', companyId] })
    },
    onError: (e: Error) => setError(e.message),
  })

  return (
    <section className="rounded-2xl border border-black/10 bg-white/70 p-4 lg:col-span-2">
      <h3 className="font-medium">Submit expense</h3>
      <div className="mt-3 grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
        <input
          className="rounded-lg border border-black/15 bg-white px-3 py-2 text-sm lg:col-span-2"
          placeholder="Title"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
        />
        <input
          type="number"
          min={0.01}
          step={0.01}
          className="rounded-lg border border-black/15 bg-white px-3 py-2 text-sm"
          placeholder="Amount"
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
        />
        <input
          className="rounded-lg border border-black/15 bg-white px-3 py-2 text-sm uppercase"
          maxLength={3}
          value={currency}
          onChange={(e) => setCurrency(e.target.value)}
        />
        <select
          className="rounded-lg border border-black/15 bg-white px-3 py-2 text-sm sm:col-span-2"
          value={categoryId}
          onChange={(e) => setCategoryId(e.target.value)}
        >
          <option value="">No category</option>
          {(categories.data ?? []).map((c) => (
            <option key={c.id} value={c.id}>
              {c.name}
            </option>
          ))}
        </select>
      </div>
      {error && <p className="mt-2 text-sm text-red-700">{error}</p>}
      <button
        type="button"
        className="mt-3 rounded-lg bg-[var(--empops-accent)] px-3 py-1.5 text-sm text-white disabled:opacity-60"
        disabled={!title.trim() || !amount || create.isPending}
        onClick={() => create.mutate()}
      >
        Submit expense
      </button>
    </section>
  )
}

function PendingTimesheetsPanel({ companyId }: { companyId: string }) {
  const qc = useQueryClient()
  const list = useQuery({
    queryKey: ['timesheets-pending', companyId],
    queryFn: async () => {
      const res = await authFetch<Timesheet[]>(`/companies/${companyId}/timesheets/pending`)
      return res.data
    },
  })

  const decide = useMutation({
    mutationFn: async ({ id, approve }: { id: string; approve: boolean }) => {
      await authFetch(
        `/companies/${companyId}/timesheets/${id}/${approve ? 'approve' : 'reject'}`,
        { method: 'POST' },
      )
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['timesheets-pending', companyId] })
      void qc.invalidateQueries({ queryKey: ['dashboard', companyId] })
    },
  })

  return (
    <section className="rounded-2xl border border-black/10 bg-white/70 p-4">
      <h3 className="font-medium">Pending timesheets</h3>
      {list.isLoading && <p className="mt-2 text-sm text-black/55">Loading…</p>}
      <ul className="mt-3 space-y-3">
        {(list.data ?? []).map((ts) => (
          <li key={ts.id} className="rounded-lg border border-black/10 p-3 text-sm">
            <p className="font-medium">
              {ts.employee
                ? `${ts.employee.first_name} ${ts.employee.last_name}`
                : ts.employee_id}
            </p>
            <p className="text-black/55">
              {ts.started_at} → {ts.ended_at} · {(ts.total_duration / 60).toFixed(1)}h
            </p>
            <div className="mt-2 flex gap-2">
              <button
                type="button"
                className="rounded-lg bg-[var(--empops-accent)] px-2 py-1 text-xs text-white"
                onClick={() => decide.mutate({ id: ts.id, approve: true })}
              >
                Approve
              </button>
              <button
                type="button"
                className="rounded-lg border border-black/15 px-2 py-1 text-xs"
                onClick={() => decide.mutate({ id: ts.id, approve: false })}
              >
                Reject
              </button>
            </div>
          </li>
        ))}
        {list.data?.length === 0 && (
          <li className="text-sm text-black/50">Nothing waiting.</li>
        )}
      </ul>
    </section>
  )
}

function PendingExpensesPanel({
  companyId,
  mode,
}: {
  companyId: string
  mode: 'manager' | 'accounting'
}) {
  const qc = useQueryClient()
  const [reasons, setReasons] = useState<Record<string, string>>({})

  const list = useQuery({
    queryKey: ['expenses-pending', companyId, mode],
    queryFn: async () => {
      const res = await authFetch<Expense[]>(
        `/companies/${companyId}/expenses/pending/${mode}`,
      )
      return res.data
    },
  })

  const decide = useMutation({
    mutationFn: async ({
      id,
      approve,
    }: {
      id: string
      approve: boolean
    }) => {
      const action =
        mode === 'manager'
          ? approve
            ? 'manager-approve'
            : 'manager-reject'
          : approve
            ? 'accounting-approve'
            : 'accounting-reject'
      await authFetch(`/companies/${companyId}/expenses/${id}/${action}`, {
        method: 'POST',
        body: approve ? undefined : JSON.stringify({ reason: reasons[id] || 'Rejected' }),
      })
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['expenses-pending', companyId, mode] })
      void qc.invalidateQueries({ queryKey: ['dashboard', companyId] })
    },
  })

  return (
    <section className="rounded-2xl border border-black/10 bg-white/70 p-4">
      <h3 className="font-medium">
        Pending expenses ({mode === 'manager' ? 'manager' : 'accounting'})
      </h3>
      <ul className="mt-3 space-y-3">
        {(list.data ?? []).map((expense) => (
          <li key={expense.id} className="rounded-lg border border-black/10 p-3 text-sm">
            <p className="font-medium">{expense.title}</p>
            <p className="text-black/55">
              {expense.employee_name ?? expense.employee_id} ·{' '}
              {(expense.amount / 100).toFixed(2)} {expense.currency}
              {expense.converted_amount != null &&
                ` → ${(expense.converted_amount / 100).toFixed(2)} ${expense.converted_to_currency}`}
            </p>
            <input
              className="mt-2 w-full rounded-lg border border-black/15 bg-white px-2 py-1 text-xs"
              placeholder="Rejection reason"
              value={reasons[expense.id] ?? ''}
              onChange={(e) =>
                setReasons((prev) => ({ ...prev, [expense.id]: e.target.value }))
              }
            />
            <div className="mt-2 flex gap-2">
              <button
                type="button"
                className="rounded-lg bg-[var(--empops-accent)] px-2 py-1 text-xs text-white"
                onClick={() => decide.mutate({ id: expense.id, approve: true })}
              >
                Approve
              </button>
              <button
                type="button"
                className="rounded-lg border border-black/15 px-2 py-1 text-xs"
                onClick={() => decide.mutate({ id: expense.id, approve: false })}
              >
                Reject
              </button>
            </div>
          </li>
        ))}
        {list.data?.length === 0 && (
          <li className="text-sm text-black/50">Nothing waiting.</li>
        )}
      </ul>
    </section>
  )
}

export default function DashboardPage() {
  const { companyId, view } = useParams<{ companyId: string; view: string }>()
  const { company, isHrOrAdmin, isManager, isAccountant } = useCompanyContext()

  const dashQuery = useQuery({
    queryKey: ['dashboard', companyId, view],
    queryFn: async () => {
      const res = await authFetch<DashboardShell>(`/companies/${companyId}/dashboard/${view}`)
      return res.data
    },
    enabled: Boolean(companyId) && isView(view),
  })

  if (!isView(view)) {
    return <Navigate to={`/companies/${companyId}/dashboard/me`} replace />
  }

  if (view === 'manager' && !isManager) {
    return <Navigate to={`/companies/${companyId}/dashboard/me`} replace />
  }

  if (view === 'hr' && !isHrOrAdmin) {
    return <Navigate to={`/companies/${companyId}/dashboard/me`} replace />
  }

  if (view === 'accountant' && !isAccountant && !isHrOrAdmin) {
    return <Navigate to={`/companies/${companyId}/dashboard/me`} replace />
  }

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
  const timesheet = widgetOf(widgets, 'timesheet_current_week')
  const wfh = widgetOf(widgets, 'wfh_today')
  const pendingTs = widgetOf(widgets, 'pending_timesheets')
  const pendingEx = widgetOf(widgets, 'pending_expenses')
  const pendingAcc = widgetOf(widgets, 'pending_accounting_expenses')

  const subtitle =
    view === 'me'
      ? 'Work logs, timesheets, expenses, and Q&A.'
      : view === 'manager'
        ? 'Approve timesheets and expenses for your reports.'
        : view === 'hr'
          ? 'Approve orphan / past-week timesheets.'
          : view === 'accountant'
            ? 'Finalize expenses in the accounting queue.'
            : 'Team overview shell.'

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold capitalize">{view} dashboard</h2>
        <p className="text-sm text-black/55">{subtitle}</p>
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
        {(isAccountant || isHrOrAdmin) && (
          <NavLink to={`/companies/${companyId}/dashboard/accountant`} className={linkClass}>
            Accountant
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
          {timesheet && <TimesheetWidget companyId={companyId} data={timesheet.data} />}
          {wfh && (
            <WfhWidget
              companyId={companyId}
              employeeId={company.employee_id}
              data={wfh.data}
            />
          )}
          <ExpenseSubmitWidget companyId={companyId} />
          {unread && (
            <section className="rounded-2xl border border-black/10 bg-white/70 p-4 lg:col-span-2">
              <h3 className="font-medium">Notifications</h3>
              <p className="mt-2 text-sm text-black/70">Unread: {unread.data.count}</p>
            </section>
          )}
        </div>
      )}

      {dashQuery.data && view === 'manager' && companyId && (
        <div className="grid gap-4 lg:grid-cols-2">
          <div className="rounded-2xl border border-black/10 bg-white/70 p-4 text-sm text-black/70 lg:col-span-2">
            Pending timesheets: {pendingTs?.data.count ?? 0} · Pending expenses:{' '}
            {pendingEx?.data.count ?? 0}
          </div>
          <PendingTimesheetsPanel companyId={companyId} />
          <PendingExpensesPanel companyId={companyId} mode="manager" />
        </div>
      )}

      {dashQuery.data && view === 'hr' && companyId && (
        <div className="space-y-4">
          <p className="text-sm text-black/60">
            Pending orphan timesheets: {pendingTs?.data.count ?? 0}
          </p>
          <PendingTimesheetsPanel companyId={companyId} />
        </div>
      )}

      {dashQuery.data && view === 'accountant' && companyId && (
        <div className="space-y-4">
          <p className="text-sm text-black/60">
            Accounting queue: {pendingAcc?.data.count ?? 0}
          </p>
          <PendingExpensesPanel companyId={companyId} mode="accounting" />
        </div>
      )}

      {dashQuery.data && view === 'team' && (
        <div className="rounded-2xl border border-black/10 bg-white/70 p-6">
          <p className="text-sm text-black/50">Team dashboard shell — no Step 5 widgets.</p>
        </div>
      )}
    </div>
  )
}
