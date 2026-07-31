import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { authFetch } from '@/lib/authFetch'
import { resolveMediaUrl } from '@/lib/mediaUrl'
import type { Candidate, Employee, JobOpening } from '@/types/api'

const BUCKETS = ['to_sort', 'selected', 'rejected'] as const
type Bucket = (typeof BUCKETS)[number]

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

const hireSchema = z.object({
  email: z.string().email(),
  first_name: z.string().min(1, 'Required'),
  last_name: z.string().min(1, 'Required'),
  hired_at: z.string().min(1, 'Required'),
})
type HireValues = z.infer<typeof hireSchema>

export default function JobOpeningDetailPage() {
  const { companyId, jobOpeningId } = useParams<{ companyId: string; jobOpeningId: string }>()
  const qc = useQueryClient()
  const [bucket, setBucket] = useState<Bucket>('to_sort')
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const base = `/companies/${companyId}/job-openings/${jobOpeningId}`

  const openingQuery = useQuery({
    queryKey: ['job-opening', companyId, jobOpeningId],
    queryFn: async () => {
      const res = await authFetch<JobOpening>(base)
      return res.data
    },
    enabled: Boolean(companyId && jobOpeningId),
  })

  const candidatesQuery = useQuery({
    queryKey: ['candidates', companyId, jobOpeningId, bucket],
    queryFn: async () => {
      const res = await authFetch<Candidate[]>(`${base}/candidates?bucket=${bucket}`)
      return res.data
    },
    enabled: Boolean(companyId && jobOpeningId),
  })

  const candidateQuery = useQuery({
    queryKey: ['candidate', companyId, jobOpeningId, selectedId],
    queryFn: async () => {
      const res = await authFetch<Candidate>(`${base}/candidates/${selectedId}`)
      return res.data
    },
    enabled: Boolean(selectedId),
  })

  const invalidate = () => {
    void qc.invalidateQueries({ queryKey: ['candidates', companyId, jobOpeningId] })
    void qc.invalidateQueries({ queryKey: ['candidate', companyId, jobOpeningId, selectedId] })
    void qc.invalidateQueries({ queryKey: ['job-opening', companyId, jobOpeningId] })
  }

  if (openingQuery.isLoading) return <p className="text-black/60">Loading opening…</p>
  if (openingQuery.isError || !openingQuery.data) {
    return (
      <div className="rounded-2xl border border-red-200 bg-red-50 p-4 text-red-700">
        {(openingQuery.error as Error)?.message ?? 'Opening not found'}
      </div>
    )
  }

  const opening = openingQuery.data

  return (
    <div className="space-y-6">
      <div>
        <Link
          to={`/companies/${companyId}/recruiting`}
          className="text-sm text-black/50 hover:text-black"
        >
          ← Recruiting
        </Link>
        <h2 className="mt-2 text-xl font-semibold">{opening.title}</h2>
        <p className="text-sm text-black/60">{opening.description}</p>
        <p className="mt-1 text-xs text-black/50">
          {opening.active ? 'Active' : 'Inactive'}
          {opening.position && ` · ${opening.position.title}`}
          {opening.reference_number && ` · Ref #${opening.reference_number}`}
        </p>
      </div>

      {error && (
        <div className="rounded-2xl border border-red-200 bg-red-50 p-4 text-red-700">{error}</div>
      )}

      <nav className="flex flex-wrap gap-2">
        {BUCKETS.map((b) => (
          <button
            key={b}
            type="button"
            className={tabBtn(bucket === b)}
            onClick={() => {
              setBucket(b)
              setSelectedId(null)
            }}
          >
            {b.replace('_', ' ')}
          </button>
        ))}
      </nav>

      <div className="grid gap-4 lg:grid-cols-[1fr_1.2fr]">
        <section className={panelClass()}>
          <h3 className="font-medium">Candidates</h3>
          {candidatesQuery.isLoading && (
            <p className="mt-2 text-sm text-black/60">Loading…</p>
          )}
          <ul className="mt-3 space-y-2">
            {(candidatesQuery.data ?? []).map((c) => (
              <li key={c.id}>
                <button
                  type="button"
                  onClick={() => setSelectedId(c.id)}
                  className={`w-full rounded-lg px-3 py-2 text-left text-sm transition ${
                    selectedId === c.id
                      ? 'bg-[var(--empops-accent)]/10 ring-1 ring-[var(--empops-accent)]'
                      : 'bg-black/[0.03] hover:bg-black/[0.06]'
                  }`}
                >
                  <span className="font-medium">{c.name}</span>
                  <span className="ml-2 text-xs text-black/50">{c.email}</span>
                  {c.employee_id && (
                    <span className="ml-2 text-xs text-green-700">Hired</span>
                  )}
                </button>
              </li>
            ))}
          </ul>
          {candidatesQuery.data?.length === 0 && (
            <p className="mt-2 text-sm text-black/55">No candidates in this bucket.</p>
          )}
        </section>

        {selectedId && candidateQuery.data && (
          <CandidatePanel
            base={base}
            candidate={candidateQuery.data}
            opening={opening}
            companyId={companyId!}
            onError={setError}
            onInvalidate={invalidate}
          />
        )}
        {selectedId && candidateQuery.isLoading && (
          <p className="text-black/60">Loading candidate…</p>
        )}
      </div>
    </div>
  )
}

function CandidatePanel({
  base,
  candidate,
  opening,
  companyId,
  onError,
  onInvalidate,
}: {
  base: string
  candidate: Candidate
  opening: JobOpening
  companyId: string
  onError: (msg: string | null) => void
  onInvalidate: () => void
}) {
  const candBase = `${base}/candidates/${candidate.id}`
  const [noteText, setNoteText] = useState('')
  const [noteStageId, setNoteStageId] = useState('')
  const [participantStageId, setParticipantStageId] = useState('')
  const [participantEmployeeId, setParticipantEmployeeId] = useState('')

  const employeesQuery = useQuery({
    queryKey: ['employees', companyId],
    queryFn: async () => {
      const res = await authFetch<Employee[]>(`/companies/${companyId}/employees`)
      return res.data
    },
  })

  const filesQuery = useQuery({
    queryKey: ['candidate-files', candidate.id],
    queryFn: async () => {
      const res = await authFetch<typeof candidate.files>(`${candBase}/files`)
      return res.data ?? []
    },
  })

  const processStage = useMutation({
    mutationFn: async ({ stageId, accepted }: { stageId: string; accepted: boolean }) => {
      await authFetch(`${candBase}/stages/${stageId}`, {
        method: 'POST',
        body: JSON.stringify({ accepted }),
      })
    },
    onSuccess: () => {
      onError(null)
      onInvalidate()
    },
    onError: (e: Error) => onError(e.message),
  })

  const addNote = useMutation({
    mutationFn: async ({ stageId, note }: { stageId: string; note: string }) => {
      await authFetch(`${candBase}/stages/${stageId}/notes`, {
        method: 'POST',
        body: JSON.stringify({ note }),
      })
    },
    onSuccess: () => {
      onError(null)
      setNoteText('')
      onInvalidate()
    },
    onError: (e: Error) => onError(e.message),
  })

  const addParticipant = useMutation({
    mutationFn: async ({ stageId, employee_id }: { stageId: string; employee_id: string }) => {
      await authFetch(`${candBase}/stages/${stageId}/participants`, {
        method: 'POST',
        body: JSON.stringify({ employee_id }),
      })
    },
    onSuccess: () => {
      onError(null)
      setParticipantEmployeeId('')
      onInvalidate()
    },
    onError: (e: Error) => onError(e.message),
  })

  const hireForm = useForm<HireValues>({
    resolver: zodResolver(hireSchema),
    defaultValues: {
      email: candidate.email,
      first_name: candidate.name.split(' ')[0] ?? '',
      last_name: candidate.name.split(' ').slice(1).join(' ') || candidate.name,
      hired_at: new Date().toISOString().slice(0, 10),
    },
  })

  const hireMutation = useMutation({
    mutationFn: async (values: HireValues) => {
      await authFetch(`${candBase}/hire`, {
        method: 'POST',
        body: JSON.stringify(values),
      })
    },
    onSuccess: () => {
      onError(null)
      onInvalidate()
    },
    onError: (e: Error) => onError(e.message),
  })

  const stages = candidate.stages ?? []
  const files = filesQuery.data ?? candidate.files ?? []

  return (
    <section className={`${panelClass()} space-y-4`}>
      <div>
        <h3 className="font-medium">{candidate.name}</h3>
        <p className="text-sm text-black/60">{candidate.email}</p>
        {candidate.url && (
          <a
            href={candidate.url}
            target="_blank"
            rel="noreferrer"
            className="text-sm text-[var(--empops-accent)] hover:underline"
          >
            Profile link
          </a>
        )}
        {candidate.desired_salary && (
          <p className="text-xs text-black/50">Desired salary: {candidate.desired_salary}</p>
        )}
        {candidate.notes && (
          <p className="mt-1 text-sm text-black/60">{candidate.notes}</p>
        )}
      </div>

      <div>
        <h4 className="text-sm font-medium">Stages</h4>
        <ul className="mt-2 space-y-3">
          {stages.map((stage) => (
            <li key={stage.id} className="rounded-lg bg-black/[0.03] p-3">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <span className="text-sm font-medium">
                  {stage.stage_name}
                  <span className="ml-2 text-xs text-black/50">({stage.status})</span>
                </span>
                {stage.status === 'pending' && (
                  <div className="flex gap-1">
                    <button
                      type="button"
                      disabled={processStage.isPending}
                      onClick={() => processStage.mutate({ stageId: stage.id, accepted: true })}
                      className="rounded-lg bg-green-600 px-2 py-1 text-xs text-white disabled:opacity-60"
                    >
                      Pass
                    </button>
                    <button
                      type="button"
                      disabled={processStage.isPending}
                      onClick={() => processStage.mutate({ stageId: stage.id, accepted: false })}
                      className="rounded-lg bg-red-600 px-2 py-1 text-xs text-white disabled:opacity-60"
                    >
                      Fail
                    </button>
                  </div>
                )}
              </div>
              {stage.decider_name && (
                <p className="mt-1 text-xs text-black/50">
                  Decided by {stage.decider_name}
                  {stage.decided_at && ` · ${new Date(stage.decided_at).toLocaleDateString()}`}
                </p>
              )}
              {(stage.notes ?? []).length > 0 && (
                <ul className="mt-2 space-y-1">
                  {stage.notes!.map((n) => (
                    <li key={n.id} className="text-xs text-black/70">
                      <span className="font-medium">{n.author_name}:</span> {n.note}
                    </li>
                  ))}
                </ul>
              )}
              {(stage.participants ?? []).length > 0 && (
                <p className="mt-1 text-xs text-black/50">
                  Participants:{' '}
                  {stage.participants!.map((p) => p.participant_name).join(', ')}
                </p>
              )}
            </li>
          ))}
        </ul>
      </div>

      <div className="space-y-2">
        <h4 className="text-sm font-medium">Add note</h4>
        <div className="flex flex-wrap gap-2">
          <select
            className={`${inputClass} max-w-[180px]`}
            value={noteStageId}
            onChange={(e) => setNoteStageId(e.target.value)}
          >
            <option value="">Stage…</option>
            {stages.map((s) => (
              <option key={s.id} value={s.id}>
                {s.stage_name}
              </option>
            ))}
          </select>
          <input
            className={`${inputClass} flex-1`}
            placeholder="Note text"
            value={noteText}
            onChange={(e) => setNoteText(e.target.value)}
          />
          <button
            type="button"
            disabled={!noteStageId || !noteText.trim() || addNote.isPending}
            onClick={() => addNote.mutate({ stageId: noteStageId, note: noteText.trim() })}
            className="rounded-lg bg-[var(--empops-accent)] px-3 py-2 text-sm text-white disabled:opacity-60"
          >
            Add
          </button>
        </div>
      </div>

      <div className="space-y-2">
        <h4 className="text-sm font-medium">Add participant</h4>
        <div className="flex flex-wrap gap-2">
          <select
            className={`${inputClass} max-w-[180px]`}
            value={participantStageId}
            onChange={(e) => setParticipantStageId(e.target.value)}
          >
            <option value="">Stage…</option>
            {stages.map((s) => (
              <option key={s.id} value={s.id}>
                {s.stage_name}
              </option>
            ))}
          </select>
          <select
            className={`${inputClass} flex-1`}
            value={participantEmployeeId}
            onChange={(e) => setParticipantEmployeeId(e.target.value)}
          >
            <option value="">Employee…</option>
            {(employeesQuery.data ?? []).map((e) => (
              <option key={e.id} value={e.id}>
                {e.first_name} {e.last_name}
              </option>
            ))}
          </select>
          <button
            type="button"
            disabled={
              !participantStageId ||
              !participantEmployeeId ||
              addParticipant.isPending
            }
            onClick={() =>
              addParticipant.mutate({
                stageId: participantStageId,
                employee_id: participantEmployeeId,
              })
            }
            className="rounded-lg bg-[var(--empops-accent)] px-3 py-2 text-sm text-white disabled:opacity-60"
          >
            Add
          </button>
        </div>
      </div>

      <div>
        <h4 className="text-sm font-medium">CV files</h4>
        <ul className="mt-2 space-y-1">
          {files.map((f) => (
            <li key={f.id}>
              <a
                href={resolveMediaUrl(f.url) ?? f.url}
                target="_blank"
                rel="noreferrer"
                className="text-sm text-[var(--empops-accent)] hover:underline"
              >
                {f.file_name}
              </a>
            </li>
          ))}
        </ul>
        {files.length === 0 && (
          <p className="text-xs text-black/50">No files attached.</p>
        )}
      </div>

      {!candidate.employee_id && opening.active && (
        <form
          className="space-y-2 border-t border-black/10 pt-4"
          onSubmit={hireForm.handleSubmit((v) => hireMutation.mutate(v))}
        >
          <h4 className="text-sm font-medium">Hire candidate</h4>
          <div className="grid gap-2 sm:grid-cols-2">
            <label className="block space-y-1">
              <span className="text-xs text-black/60">Email</span>
              <input className={inputClass} {...hireForm.register('email')} />
            </label>
            <label className="block space-y-1">
              <span className="text-xs text-black/60">Hired at</span>
              <input type="date" className={inputClass} {...hireForm.register('hired_at')} />
            </label>
            <label className="block space-y-1">
              <span className="text-xs text-black/60">First name</span>
              <input className={inputClass} {...hireForm.register('first_name')} />
            </label>
            <label className="block space-y-1">
              <span className="text-xs text-black/60">Last name</span>
              <input className={inputClass} {...hireForm.register('last_name')} />
            </label>
          </div>
          <button
            type="submit"
            disabled={hireMutation.isPending}
            className="rounded-lg bg-[var(--empops-accent)] px-4 py-2 text-sm font-medium text-white disabled:opacity-60"
          >
            Hire
          </button>
        </form>
      )}
    </section>
  )
}
