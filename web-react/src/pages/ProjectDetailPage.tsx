import { useRef, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { authFetch } from '@/lib/authFetch'
import { ChunkedUploader } from '@/lib/upload/chunked-upload'
import { resolveMediaUrl } from '@/lib/mediaUrl'
import type {
  Comment,
  Project,
  ProjectBoard,
  ProjectDecision,
  ProjectFile,
  ProjectLink,
  ProjectMessage,
  ProjectStatus,
  ProjectStatusUpdate,
  ProjectTaskList,
} from '@/types/api'

const TABS = ['overview', 'messages', 'decisions', 'tasks', 'board', 'files'] as const
type Tab = (typeof TABS)[number]

const PROJECT_STATUSES: ProjectStatus[] = [
  'created',
  'started',
  'paused',
  'cancelled',
  'closed',
]

const inputClass =
  'w-full rounded-lg border border-black/15 bg-white px-3 py-2 text-sm outline-none focus:border-[var(--empops-accent)]'

function tabBtn(active: boolean) {
  return `rounded-lg px-3 py-1.5 text-sm ${
    active
      ? 'bg-[var(--empops-accent)] text-white'
      : 'bg-black/[0.04] text-black/70 hover:bg-black/[0.08]'
  }`
}

function panelClass() {
  return 'rounded-2xl border border-black/10 bg-white/70 p-4'
}

export default function ProjectDetailPage() {
  const { companyId, projectId } = useParams<{ companyId: string; projectId: string }>()
  const qc = useQueryClient()
  const [tab, setTab] = useState<Tab>('overview')
  const [error, setError] = useState<string | null>(null)

  const base = `/companies/${companyId}/projects/${projectId}`

  const projectQuery = useQuery({
    queryKey: ['project', companyId, projectId],
    queryFn: async () => {
      const res = await authFetch<Project>(`${base}`)
      return res.data
    },
    enabled: Boolean(companyId && projectId),
  })

  const invalidateProject = () => {
    void qc.invalidateQueries({ queryKey: ['project', companyId, projectId] })
    void qc.invalidateQueries({ queryKey: ['projects', companyId] })
  }

  if (projectQuery.isLoading) return <p className="text-black/60">Loading project…</p>
  if (projectQuery.isError || !projectQuery.data) {
    return (
      <div className="rounded-2xl border border-red-200 bg-red-50 p-4 text-red-700">
        {(projectQuery.error as Error)?.message ?? 'Project not found'}
      </div>
    )
  }

  const project = projectQuery.data

  return (
    <div className="space-y-6">
      <div>
        <Link
          to={`/companies/${companyId}/projects`}
          className="text-sm text-black/50 hover:text-black"
        >
          ← Projects
        </Link>
        <h2 className="mt-2 text-xl font-semibold">
          {project.emoji ? `${project.emoji} ` : ''}
          {project.name}
        </h2>
        {project.summary && <p className="text-sm text-black/60">{project.summary}</p>}
        <p className="mt-1 text-xs text-black/50">
          Status: {project.status.replaceAll('_', ' ')} · Lead:{' '}
          {project.lead
            ? `${project.lead.first_name} ${project.lead.last_name}`
            : 'None'}
        </p>
      </div>

      {error && (
        <div className="rounded-2xl border border-red-200 bg-red-50 p-4 text-red-700">{error}</div>
      )}

      <nav className="flex flex-wrap gap-2">
        {TABS.map((t) => (
          <button key={t} type="button" className={tabBtn(tab === t)} onClick={() => setTab(t)}>
            {t.charAt(0).toUpperCase() + t.slice(1)}
          </button>
        ))}
      </nav>

      {tab === 'overview' && (
        <OverviewTab
          base={base}
          project={project}
          onError={setError}
          onInvalidate={invalidateProject}
        />
      )}
      {tab === 'messages' && (
        <MessagesTab base={base} onError={setError} />
      )}
      {tab === 'decisions' && (
        <DecisionsTab base={base} onError={setError} />
      )}
      {tab === 'tasks' && (
        <TasksTab base={base} onError={setError} />
      )}
      {tab === 'board' && (
        <BoardTab base={base} onError={setError} />
      )}
      {tab === 'files' && (
        <FilesTab base={base} onError={setError} />
      )}
    </div>
  )
}

function OverviewTab({
  base,
  project,
  onError,
  onInvalidate,
}: {
  base: string
  project: Project
  onError: (msg: string | null) => void
  onInvalidate: () => void
}) {
  const [memberId, setMemberId] = useState('')
  const [linkType, setLinkType] = useState('url')
  const [linkUrl, setLinkUrl] = useState('')
  const [linkLabel, setLinkLabel] = useState('')
  const [statusTitle, setStatusTitle] = useState('')
  const [statusValue, setStatusValue] = useState('started')
  const [statusDesc, setStatusDesc] = useState('')

  const linksQuery = useQuery({
    queryKey: ['project-links', base],
    queryFn: async () => {
      const res = await authFetch<ProjectLink[]>(`${base}/links`)
      return res.data
    },
  })

  const statusesQuery = useQuery({
    queryKey: ['project-statuses', base],
    queryFn: async () => {
      const res = await authFetch<ProjectStatusUpdate[]>(`${base}/statuses`)
      return res.data
    },
  })

  const updateStatus = useMutation({
    mutationFn: async (status: ProjectStatus) => {
      await authFetch(`${base}`, {
        method: 'PATCH',
        body: JSON.stringify({ status }),
      })
    },
    onSuccess: () => {
      onError(null)
      onInvalidate()
    },
    onError: (e: Error) => onError(e.message),
  })

  const addMember = useMutation({
    mutationFn: async (employeeId: string) => {
      await authFetch(`${base}/members/${employeeId}`, { method: 'POST' })
    },
    onSuccess: () => {
      onError(null)
      setMemberId('')
      onInvalidate()
    },
    onError: (e: Error) => onError(e.message),
  })

  const removeMember = useMutation({
    mutationFn: async (employeeId: string) => {
      await authFetch(`${base}/members/${employeeId}`, { method: 'DELETE' })
    },
    onSuccess: () => {
      onError(null)
      onInvalidate()
    },
    onError: (e: Error) => onError(e.message),
  })

  const addLink = useMutation({
    mutationFn: async () => {
      await authFetch(`${base}/links`, {
        method: 'POST',
        body: JSON.stringify({
          type: linkType,
          url: linkUrl,
          label: linkLabel || null,
        }),
      })
    },
    onSuccess: () => {
      onError(null)
      setLinkUrl('')
      setLinkLabel('')
      void linksQuery.refetch()
    },
    onError: (e: Error) => onError(e.message),
  })

  const deleteLink = useMutation({
    mutationFn: async (linkId: string) => {
      await authFetch(`${base}/links/${linkId}`, { method: 'DELETE' })
    },
    onSuccess: () => void linksQuery.refetch(),
    onError: (e: Error) => onError(e.message),
  })

  const createStatus = useMutation({
    mutationFn: async () => {
      await authFetch(`${base}/statuses`, {
        method: 'POST',
        body: JSON.stringify({
          title: statusTitle,
          status: statusValue,
          description: statusDesc,
        }),
      })
    },
    onSuccess: () => {
      onError(null)
      setStatusTitle('')
      setStatusDesc('')
      void statusesQuery.refetch()
      onInvalidate()
    },
    onError: (e: Error) => onError(e.message),
  })

  return (
    <div className="space-y-4">
      <section className={panelClass()}>
        <h3 className="font-medium">Project status</h3>
        <select
          className={`mt-2 ${inputClass} max-w-xs`}
          value={project.status}
          onChange={(e) => updateStatus.mutate(e.target.value as ProjectStatus)}
        >
          {PROJECT_STATUSES.map((s) => (
            <option key={s} value={s}>
              {s.replaceAll('_', ' ')}
            </option>
          ))}
        </select>
      </section>

      <section className={panelClass()}>
        <h3 className="font-medium">Members</h3>
        <ul className="mt-3 space-y-2">
          {project.members.map((m) => (
            <li key={m.id} className="flex items-center justify-between gap-3 text-sm">
              <span>
                {m.first_name} {m.last_name}
                <span className="ml-2 text-black/45">{m.email}</span>
              </span>
              <button
                type="button"
                className="text-xs text-red-700 hover:underline"
                onClick={() => removeMember.mutate(m.id)}
              >
                Remove
              </button>
            </li>
          ))}
          {project.members.length === 0 && (
            <li className="text-sm text-black/50">No members yet.</li>
          )}
        </ul>
        <div className="mt-3 flex flex-wrap gap-2 border-t border-black/5 pt-3">
          <input
            className={`${inputClass} max-w-xs`}
            placeholder="Employee ID"
            value={memberId}
            onChange={(e) => setMemberId(e.target.value)}
          />
          <button
            type="button"
            disabled={!memberId.trim() || addMember.isPending}
            className="rounded-lg bg-[var(--empops-accent)] px-3 py-1.5 text-sm text-white disabled:opacity-60"
            onClick={() => addMember.mutate(memberId.trim())}
          >
            Add member
          </button>
        </div>
      </section>

      <section className={panelClass()}>
        <h3 className="font-medium">Links</h3>
        <ul className="mt-3 space-y-2">
          {(linksQuery.data ?? []).map((link) => (
            <li key={link.id} className="flex items-center justify-between gap-3 text-sm">
              <a
                href={link.url}
                target="_blank"
                rel="noreferrer"
                className="text-[var(--empops-accent)] hover:underline"
              >
                {link.label || link.url}
              </a>
              <span className="text-xs text-black/45">{link.type}</span>
              <button
                type="button"
                className="text-xs text-red-700 hover:underline"
                onClick={() => deleteLink.mutate(link.id)}
              >
                Delete
              </button>
            </li>
          ))}
          {(linksQuery.data ?? []).length === 0 && (
            <li className="text-sm text-black/50">No links yet.</li>
          )}
        </ul>
        <div className="mt-3 grid gap-2 border-t border-black/5 pt-3 sm:grid-cols-3">
          <input
            className={inputClass}
            placeholder="Type"
            value={linkType}
            onChange={(e) => setLinkType(e.target.value)}
          />
          <input
            className={inputClass}
            placeholder="URL"
            value={linkUrl}
            onChange={(e) => setLinkUrl(e.target.value)}
          />
          <input
            className={inputClass}
            placeholder="Label (optional)"
            value={linkLabel}
            onChange={(e) => setLinkLabel(e.target.value)}
          />
        </div>
        <button
          type="button"
          disabled={!linkUrl.trim() || addLink.isPending}
          className="mt-2 rounded-lg bg-[var(--empops-accent)] px-3 py-1.5 text-sm text-white disabled:opacity-60"
          onClick={() => addLink.mutate()}
        >
          Add link
        </button>
      </section>

      <section className={panelClass()}>
        <h3 className="font-medium">Status updates</h3>
        <ul className="mt-3 space-y-2">
          {(statusesQuery.data ?? []).map((s) => (
            <li key={s.id} className="rounded-lg border border-black/5 p-3 text-sm">
              <p className="font-medium">{s.title}</p>
              <p className="text-xs text-black/50">
                {s.status.replaceAll('_', ' ')}
                {s.author_name ? ` · ${s.author_name}` : ''}
              </p>
              <p className="mt-1 whitespace-pre-wrap text-black/70">{s.description}</p>
            </li>
          ))}
          {(statusesQuery.data ?? []).length === 0 && (
            <li className="text-sm text-black/50">No status updates yet.</li>
          )}
        </ul>
        <div className="mt-3 space-y-2 border-t border-black/5 pt-3">
          <input
            className={inputClass}
            placeholder="Title"
            value={statusTitle}
            onChange={(e) => setStatusTitle(e.target.value)}
          />
          <select
            className={inputClass}
            value={statusValue}
            onChange={(e) => setStatusValue(e.target.value)}
          >
            {PROJECT_STATUSES.map((s) => (
              <option key={s} value={s}>
                {s.replaceAll('_', ' ')}
              </option>
            ))}
          </select>
          <textarea
            className={inputClass}
            rows={2}
            placeholder="Description"
            value={statusDesc}
            onChange={(e) => setStatusDesc(e.target.value)}
          />
          <button
            type="button"
            disabled={!statusTitle.trim() || !statusDesc.trim() || createStatus.isPending}
            className="rounded-lg bg-[var(--empops-accent)] px-3 py-1.5 text-sm text-white disabled:opacity-60"
            onClick={() => createStatus.mutate()}
          >
            Post update
          </button>
        </div>
      </section>
    </div>
  )
}

function MessagesTab({
  base,
  onError,
}: {
  base: string
  onError: (msg: string | null) => void
}) {
  const qc = useQueryClient()
  const [title, setTitle] = useState('')
  const [content, setContent] = useState('')
  const [commentDrafts, setCommentDrafts] = useState<Record<string, string>>({})

  const messagesQuery = useQuery({
    queryKey: ['project-messages', base],
    queryFn: async () => {
      const res = await authFetch<ProjectMessage[]>(`${base}/messages`)
      return res.data
    },
  })

  const createMessage = useMutation({
    mutationFn: async () => {
      await authFetch(`${base}/messages`, {
        method: 'POST',
        body: JSON.stringify({ title, content }),
      })
    },
    onSuccess: () => {
      onError(null)
      setTitle('')
      setContent('')
      void qc.invalidateQueries({ queryKey: ['project-messages', base] })
    },
    onError: (e: Error) => onError(e.message),
  })

  const addComment = useMutation({
    mutationFn: async ({ messageId, content: body }: { messageId: string; content: string }) => {
      await authFetch(`${base}/messages/${messageId}/comments`, {
        method: 'POST',
        body: JSON.stringify({ content: body }),
      })
    },
    onSuccess: (_data, vars) => {
      onError(null)
      setCommentDrafts((d) => ({ ...d, [vars.messageId]: '' }))
      void qc.invalidateQueries({ queryKey: ['project-messages', base] })
    },
    onError: (e: Error) => onError(e.message),
  })

  return (
    <div className="space-y-4">
      <section className={panelClass()}>
        <h3 className="font-medium">Messages</h3>
        <ul className="mt-3 space-y-3">
          {(messagesQuery.data ?? []).map((msg) => (
            <li key={msg.id} className="rounded-lg border border-black/5 p-3 text-sm">
              <p className="font-medium">{msg.title}</p>
              <p className="mt-1 whitespace-pre-wrap text-black/70">{msg.content}</p>
              <p className="mt-1 text-xs text-black/45">
                {msg.author_name ?? 'Unknown author'}
              </p>
              {(msg.comments ?? []).length > 0 && (
                <ul className="mt-2 space-y-1 border-l-2 border-black/10 pl-3">
                  {(msg.comments ?? []).map((c: Comment) => (
                    <li key={c.id} className="text-xs text-black/65">
                      <span className="font-medium">{c.author_name}:</span> {c.content}
                    </li>
                  ))}
                </ul>
              )}
              <div className="mt-2 flex gap-2">
                <input
                  className={`${inputClass} flex-1`}
                  placeholder="Add comment…"
                  value={commentDrafts[msg.id] ?? ''}
                  onChange={(e) =>
                    setCommentDrafts((d) => ({ ...d, [msg.id]: e.target.value }))
                  }
                />
                <button
                  type="button"
                  disabled={!commentDrafts[msg.id]?.trim() || addComment.isPending}
                  className="rounded-lg bg-[var(--empops-accent)] px-3 py-1.5 text-xs text-white disabled:opacity-60"
                  onClick={() =>
                    addComment.mutate({
                      messageId: msg.id,
                      content: commentDrafts[msg.id] ?? '',
                    })
                  }
                >
                  Comment
                </button>
              </div>
            </li>
          ))}
          {(messagesQuery.data ?? []).length === 0 && (
            <li className="text-sm text-black/50">No messages yet.</li>
          )}
        </ul>
        <div className="mt-3 space-y-2 border-t border-black/5 pt-3">
          <input
            className={inputClass}
            placeholder="Title"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
          />
          <textarea
            className={inputClass}
            rows={3}
            placeholder="Content"
            value={content}
            onChange={(e) => setContent(e.target.value)}
          />
          <button
            type="button"
            disabled={!title.trim() || !content.trim() || createMessage.isPending}
            className="rounded-lg bg-[var(--empops-accent)] px-3 py-1.5 text-sm text-white disabled:opacity-60"
            onClick={() => createMessage.mutate()}
          >
            Post message
          </button>
        </div>
      </section>
    </div>
  )
}

function DecisionsTab({
  base,
  onError,
}: {
  base: string
  onError: (msg: string | null) => void
}) {
  const qc = useQueryClient()
  const [title, setTitle] = useState('')
  const [decidedAt, setDecidedAt] = useState('')

  const decisionsQuery = useQuery({
    queryKey: ['project-decisions', base],
    queryFn: async () => {
      const res = await authFetch<ProjectDecision[]>(`${base}/decisions`)
      return res.data
    },
  })

  const createDecision = useMutation({
    mutationFn: async () => {
      await authFetch(`${base}/decisions`, {
        method: 'POST',
        body: JSON.stringify({
          title,
          decided_at: decidedAt || null,
        }),
      })
    },
    onSuccess: () => {
      onError(null)
      setTitle('')
      setDecidedAt('')
      void qc.invalidateQueries({ queryKey: ['project-decisions', base] })
    },
    onError: (e: Error) => onError(e.message),
  })

  const deleteDecision = useMutation({
    mutationFn: async (decisionId: string) => {
      await authFetch(`${base}/decisions/${decisionId}`, { method: 'DELETE' })
    },
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['project-decisions', base] }),
    onError: (e: Error) => onError(e.message),
  })

  return (
    <section className={panelClass()}>
      <h3 className="font-medium">Decisions</h3>
      <ul className="mt-3 space-y-2">
        {(decisionsQuery.data ?? []).map((d) => (
          <li key={d.id} className="flex items-start justify-between gap-3 rounded-lg border border-black/5 p-3 text-sm">
            <div>
              <p className="font-medium">{d.title}</p>
              {d.decided_at && (
                <p className="text-xs text-black/50">Decided: {d.decided_at}</p>
              )}
              {d.author_name && (
                <p className="text-xs text-black/45">by {d.author_name}</p>
              )}
            </div>
            <button
              type="button"
              className="text-xs text-red-700 hover:underline"
              onClick={() => deleteDecision.mutate(d.id)}
            >
              Delete
            </button>
          </li>
        ))}
        {(decisionsQuery.data ?? []).length === 0 && (
          <li className="text-sm text-black/50">No decisions yet.</li>
        )}
      </ul>
      <div className="mt-3 grid gap-2 border-t border-black/5 pt-3 sm:grid-cols-2">
        <input
          className={inputClass}
          placeholder="Title"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
        />
        <input
          type="date"
          className={inputClass}
          value={decidedAt}
          onChange={(e) => setDecidedAt(e.target.value)}
        />
      </div>
      <button
        type="button"
        disabled={!title.trim() || createDecision.isPending}
        className="mt-2 rounded-lg bg-[var(--empops-accent)] px-3 py-1.5 text-sm text-white disabled:opacity-60"
        onClick={() => createDecision.mutate()}
      >
        Record decision
      </button>
    </section>
  )
}

function TasksTab({
  base,
  onError,
}: {
  base: string
  onError: (msg: string | null) => void
}) {
  const qc = useQueryClient()
  const [listTitle, setListTitle] = useState('')
  const [taskTitle, setTaskTitle] = useState('')
  const [taskListId, setTaskListId] = useState('')

  const listsQuery = useQuery({
    queryKey: ['project-task-lists', base],
    queryFn: async () => {
      const res = await authFetch<ProjectTaskList[]>(`${base}/task-lists`)
      return res.data
    },
  })

  const createList = useMutation({
    mutationFn: async () => {
      await authFetch(`${base}/task-lists`, {
        method: 'POST',
        body: JSON.stringify({ title: listTitle }),
      })
    },
    onSuccess: () => {
      onError(null)
      setListTitle('')
      void qc.invalidateQueries({ queryKey: ['project-task-lists', base] })
    },
    onError: (e: Error) => onError(e.message),
  })

  const createTask = useMutation({
    mutationFn: async () => {
      await authFetch(`${base}/tasks`, {
        method: 'POST',
        body: JSON.stringify({
          title: taskTitle,
          project_task_list_id: taskListId || null,
        }),
      })
    },
    onSuccess: () => {
      onError(null)
      setTaskTitle('')
      void qc.invalidateQueries({ queryKey: ['project-task-lists', base] })
    },
    onError: (e: Error) => onError(e.message),
  })

  const toggleTask = useMutation({
    mutationFn: async (taskId: string) => {
      await authFetch(`${base}/tasks/${taskId}/toggle`, { method: 'POST' })
    },
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['project-task-lists', base] }),
    onError: (e: Error) => onError(e.message),
  })

  const lists = listsQuery.data ?? []

  return (
    <div className="space-y-4">
      <section className={panelClass()}>
        <h3 className="font-medium">Task lists</h3>
        {lists.map((list) => (
          <div key={list.id} className="mt-4">
            <h4 className="text-sm font-medium">{list.title}</h4>
            <ul className="mt-2 space-y-1">
              {list.tasks.map((task) => (
                <li key={task.id} className="flex items-center gap-2 text-sm">
                  <input
                    type="checkbox"
                    checked={task.completed}
                    onChange={() => toggleTask.mutate(task.id)}
                  />
                  <span className={task.completed ? 'text-black/45 line-through' : ''}>
                    {task.title}
                  </span>
                  {task.assignee && (
                    <span className="text-xs text-black/45">
                      · {task.assignee.first_name} {task.assignee.last_name}
                    </span>
                  )}
                </li>
              ))}
              {list.tasks.length === 0 && (
                <li className="text-xs text-black/50">No tasks in this list.</li>
              )}
            </ul>
          </div>
        ))}
        {lists.length === 0 && <p className="mt-2 text-sm text-black/50">No task lists yet.</p>}
      </section>

      <section className={`grid gap-4 sm:grid-cols-2 ${panelClass()}`}>
        <div className="space-y-2">
          <h3 className="font-medium">Create list</h3>
          <input
            className={inputClass}
            placeholder="List title"
            value={listTitle}
            onChange={(e) => setListTitle(e.target.value)}
          />
          <button
            type="button"
            disabled={!listTitle.trim() || createList.isPending}
            className="rounded-lg bg-[var(--empops-accent)] px-3 py-1.5 text-sm text-white disabled:opacity-60"
            onClick={() => createList.mutate()}
          >
            Create list
          </button>
        </div>
        <div className="space-y-2">
          <h3 className="font-medium">Create task</h3>
          <input
            className={inputClass}
            placeholder="Task title"
            value={taskTitle}
            onChange={(e) => setTaskTitle(e.target.value)}
          />
          <select
            className={inputClass}
            value={taskListId}
            onChange={(e) => setTaskListId(e.target.value)}
          >
            <option value="">No list</option>
            {lists.map((l) => (
              <option key={l.id} value={l.id}>
                {l.title}
              </option>
            ))}
          </select>
          <button
            type="button"
            disabled={!taskTitle.trim() || createTask.isPending}
            className="rounded-lg bg-[var(--empops-accent)] px-3 py-1.5 text-sm text-white disabled:opacity-60"
            onClick={() => createTask.mutate()}
          >
            Create task
          </button>
        </div>
      </section>
    </div>
  )
}

function BoardTab({
  base,
  onError,
}: {
  base: string
  onError: (msg: string | null) => void
}) {
  const qc = useQueryClient()
  const [boardName, setBoardName] = useState('')
  const [selectedBoardId, setSelectedBoardId] = useState<string | null>(null)
  const [issueTitle, setIssueTitle] = useState('')
  const [issuePoints, setIssuePoints] = useState('')
  const [issueSprintId, setIssueSprintId] = useState('')

  const boardsQuery = useQuery({
    queryKey: ['project-boards', base],
    queryFn: async () => {
      const res = await authFetch<ProjectBoard[]>(`${base}/boards`)
      return res.data
    },
  })

  const boardDetailQuery = useQuery({
    queryKey: ['project-board', base, selectedBoardId],
    queryFn: async () => {
      const res = await authFetch<ProjectBoard>(`${base}/boards/${selectedBoardId}`)
      return res.data
    },
    enabled: Boolean(selectedBoardId),
  })

  const createBoard = useMutation({
    mutationFn: async () => {
      const res = await authFetch<ProjectBoard>(`${base}/boards`, {
        method: 'POST',
        body: JSON.stringify({ name: boardName }),
      })
      return res.data
    },
    onSuccess: (board) => {
      onError(null)
      setBoardName('')
      setSelectedBoardId(board.id)
      void qc.invalidateQueries({ queryKey: ['project-boards', base] })
    },
    onError: (e: Error) => onError(e.message),
  })

  const createIssue = useMutation({
    mutationFn: async () => {
      if (!selectedBoardId || !issueSprintId) return
      await authFetch(
        `${base}/boards/${selectedBoardId}/sprints/${issueSprintId}/issues`,
        {
          method: 'POST',
          body: JSON.stringify({
            title: issueTitle,
            story_points: issuePoints ? Number(issuePoints) : null,
          }),
        },
      )
    },
    onSuccess: () => {
      onError(null)
      setIssueTitle('')
      setIssuePoints('')
      void qc.invalidateQueries({ queryKey: ['project-board', base, selectedBoardId] })
    },
    onError: (e: Error) => onError(e.message),
  })

  const startSprint = useMutation({
    mutationFn: async (sprintId: string) => {
      if (!selectedBoardId) return
      await authFetch(`${base}/boards/${selectedBoardId}/sprints/${sprintId}/start`, {
        method: 'POST',
      })
    },
    onSuccess: () =>
      void qc.invalidateQueries({ queryKey: ['project-board', base, selectedBoardId] }),
    onError: (e: Error) => onError(e.message),
  })

  const toggleSprint = useMutation({
    mutationFn: async (sprintId: string) => {
      if (!selectedBoardId) return
      await authFetch(`${base}/boards/${selectedBoardId}/sprints/${sprintId}/toggle`, {
        method: 'POST',
      })
    },
    onSuccess: () =>
      void qc.invalidateQueries({ queryKey: ['project-board', base, selectedBoardId] }),
    onError: (e: Error) => onError(e.message),
  })

  const boards = boardsQuery.data ?? []
  const board = boardDetailQuery.data
  const sprints = board?.sprints ?? []

  return (
    <div className="space-y-4">
      <section className={panelClass()}>
        <h3 className="font-medium">Boards</h3>
        <ul className="mt-2 flex flex-wrap gap-2">
          {boards.map((b) => (
            <button
              key={b.id}
              type="button"
              className={tabBtn(selectedBoardId === b.id)}
              onClick={() => setSelectedBoardId(b.id)}
            >
              {b.name}
            </button>
          ))}
          {boards.length === 0 && <p className="text-sm text-black/50">No boards yet.</p>}
        </ul>
        <div className="mt-3 flex flex-wrap gap-2 border-t border-black/5 pt-3">
          <input
            className={`${inputClass} max-w-xs`}
            placeholder="Board name"
            value={boardName}
            onChange={(e) => setBoardName(e.target.value)}
          />
          <button
            type="button"
            disabled={!boardName.trim() || createBoard.isPending}
            className="rounded-lg bg-[var(--empops-accent)] px-3 py-1.5 text-sm text-white disabled:opacity-60"
            onClick={() => createBoard.mutate()}
          >
            Create board
          </button>
        </div>
      </section>

      {selectedBoardId && boardDetailQuery.isLoading && (
        <p className="text-sm text-black/55">Loading board…</p>
      )}

      {board && (
        <section className={panelClass()}>
          <h3 className="font-medium">{board.name}</h3>
          <div className="mt-4 space-y-4">
            {sprints.map((sprint) => (
              <div key={sprint.id} className="rounded-lg border border-black/5 p-3">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <h4 className="text-sm font-medium">
                    {sprint.name}
                    {sprint.active && (
                      <span className="ml-2 text-xs text-green-700">active</span>
                    )}
                    {sprint.completed_at && (
                      <span className="ml-2 text-xs text-black/45">completed</span>
                    )}
                  </h4>
                  <div className="flex gap-2">
                    {!sprint.active && !sprint.completed_at && (
                      <button
                        type="button"
                        className="rounded-lg border border-black/15 px-2 py-1 text-xs"
                        onClick={() => startSprint.mutate(sprint.id)}
                      >
                        Start
                      </button>
                    )}
                    <button
                      type="button"
                      className="rounded-lg border border-black/15 px-2 py-1 text-xs"
                      onClick={() => toggleSprint.mutate(sprint.id)}
                    >
                      Toggle
                    </button>
                  </div>
                </div>
                <ul className="mt-2 space-y-1">
                  {(sprint.issues ?? []).map((issue) => (
                    <li key={issue.id} className="text-sm text-black/75">
                      <span className="font-mono text-xs text-black/45">{issue.key}</span>{' '}
                      {issue.title}
                      {issue.story_points != null && (
                        <span className="ml-2 text-xs text-black/45">
                          {issue.story_points} pts
                        </span>
                      )}
                    </li>
                  ))}
                  {(sprint.issues ?? []).length === 0 && (
                    <li className="text-xs text-black/50">No issues.</li>
                  )}
                </ul>
              </div>
            ))}
            {sprints.length === 0 && (
              <p className="text-sm text-black/50">No sprints on this board.</p>
            )}
          </div>

          <div className="mt-4 space-y-2 border-t border-black/5 pt-3">
            <h4 className="text-sm font-medium">Create issue</h4>
            <select
              className={inputClass}
              value={issueSprintId}
              onChange={(e) => setIssueSprintId(e.target.value)}
            >
              <option value="">Select sprint…</option>
              {sprints.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name}
                </option>
              ))}
            </select>
            <input
              className={inputClass}
              placeholder="Issue title"
              value={issueTitle}
              onChange={(e) => setIssueTitle(e.target.value)}
            />
            <input
              type="number"
              min={0}
              className={inputClass}
              placeholder="Story points (optional)"
              value={issuePoints}
              onChange={(e) => setIssuePoints(e.target.value)}
            />
            <button
              type="button"
              disabled={
                !issueTitle.trim() || !issueSprintId || createIssue.isPending
              }
              className="rounded-lg bg-[var(--empops-accent)] px-3 py-1.5 text-sm text-white disabled:opacity-60"
              onClick={() => createIssue.mutate()}
            >
              Create issue
            </button>
          </div>
        </section>
      )}
    </div>
  )
}

function FilesTab({
  base,
  onError,
}: {
  base: string
  onError: (msg: string | null) => void
}) {
  const qc = useQueryClient()
  const inputRef = useRef<HTMLInputElement>(null)
  const [progress, setProgress] = useState<number | null>(null)
  const [busy, setBusy] = useState(false)

  const filesQuery = useQuery({
    queryKey: ['project-files', base],
    queryFn: async () => {
      const res = await authFetch<ProjectFile[]>(`${base}/files`)
      return res.data
    },
  })

  const attachFile = useMutation({
    mutationFn: async (ids: { temporary_upload_id: number; media_id: number }) => {
      await authFetch(`${base}/files`, {
        method: 'POST',
        body: JSON.stringify(ids),
      })
    },
    onSuccess: () => {
      onError(null)
      void qc.invalidateQueries({ queryKey: ['project-files', base] })
    },
    onError: (e: Error) => onError(e.message),
  })

  const deleteFile = useMutation({
    mutationFn: async (mediaId: number) => {
      await authFetch(`${base}/files/${mediaId}`, { method: 'DELETE' })
    },
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['project-files', base] }),
    onError: (e: Error) => onError(e.message),
  })

  const handleFile = async (file: File) => {
    onError(null)
    setBusy(true)
    setProgress(0)
    try {
      const uploader = new ChunkedUploader(file, {
        onProgress: (p) => setProgress(p.percentage),
      })
      const result = await uploader.upload()
      if (result.media_id == null || result.temporary_upload_id == null) {
        throw new Error('Upload completed but media IDs were not returned')
      }
      await attachFile.mutateAsync({
        temporary_upload_id: result.temporary_upload_id,
        media_id: result.media_id,
      })
      setProgress(null)
    } catch (e) {
      onError(e instanceof Error ? e.message : 'Upload failed')
      setProgress(null)
    } finally {
      setBusy(false)
      if (inputRef.current) inputRef.current.value = ''
    }
  }

  return (
    <section className={panelClass()}>
      <h3 className="font-medium">Files</h3>
      <ul className="mt-3 space-y-2">
        {(filesQuery.data ?? []).map((f) => (
          <li key={f.id} className="flex items-center justify-between gap-3 text-sm">
            <a
              href={resolveMediaUrl(f.url) ?? f.url}
              target="_blank"
              rel="noreferrer"
              className="text-[var(--empops-accent)] hover:underline"
            >
              {f.file_name}
            </a>
            <span className="text-xs text-black/45">
              {(f.size / 1024).toFixed(1)} KB
            </span>
            <button
              type="button"
              className="text-xs text-red-700 hover:underline"
              onClick={() => deleteFile.mutate(f.id)}
            >
              Delete
            </button>
          </li>
        ))}
        {(filesQuery.data ?? []).length === 0 && (
          <li className="text-sm text-black/50">No files yet.</li>
        )}
      </ul>
      <div className="mt-3 border-t border-black/5 pt-3">
        <input
          ref={inputRef}
          type="file"
          className="text-sm"
          disabled={busy}
          onChange={(e) => {
            const file = e.target.files?.[0]
            if (file) void handleFile(file)
          }}
        />
        {progress != null && (
          <p className="mt-1 text-xs text-black/55">Uploading… {Math.round(progress)}%</p>
        )}
      </div>
    </section>
  )
}
