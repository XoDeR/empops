import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Navigate, useParams } from 'react-router-dom'
import { authFetch } from '@/lib/authFetch'
import { useCompanyContext } from '@/routes/CompanyLayout'
import { ImageUploadField } from '@/components/ImageUploadField'
import { CompanyNewsSection, QuestionsSection } from '@/components/CommunicateAdminSections'
import {
  AccountantsSection,
  ExpenseCategoriesSection,
  WorkFromHomeAdminSection,
} from '@/components/OperateAdminSections'
import type { EmployeeStatus, EmployeeStatusType, Position } from '@/types/api'

const settingsSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  currency: z
    .string()
    .trim()
    .length(3, 'Use a 3-letter currency code'),
})
type SettingsValues = z.infer<typeof settingsSchema>

const positionSchema = z.object({ title: z.string().min(1, 'Required') })
type PositionValues = z.infer<typeof positionSchema>

const statusSchema = z.object({
  name: z.string().min(1, 'Required'),
  type: z.enum(['internal', 'external']),
})
type StatusValues = z.infer<typeof statusSchema>

function CompanySettingsSection({ companyId }: { companyId: string }) {
  const { company } = useCompanyContext()
  const queryClient = useQueryClient()

  const { register, handleSubmit, formState: { errors } } = useForm<SettingsValues>({
    resolver: zodResolver(settingsSchema),
    defaultValues: { name: company.name, currency: company.currency },
  })

  const updateMutation = useMutation({
    mutationFn: async (values: SettingsValues) => {
      const res = await authFetch(`/companies/${companyId}`, {
        method: 'PATCH',
        body: JSON.stringify({ name: values.name, currency: values.currency.toUpperCase() }),
      })
      return res.data
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['company', companyId] })
      void queryClient.invalidateQueries({ queryKey: ['companies'] })
    },
  })

  return (
    <section className="space-y-3 rounded-2xl border border-black/10 bg-white/80 p-5 shadow-sm">
      <h2 className="text-lg font-semibold">Company settings</h2>
      <form
        className="grid gap-3 sm:grid-cols-2"
        onSubmit={handleSubmit((values) => updateMutation.mutate(values))}
      >
        <label className="block space-y-1">
          <span className="text-sm">Name</span>
          <input
            className="w-full rounded-lg border border-black/15 bg-white px-3 py-2 outline-none focus:border-[var(--empops-accent)]"
            {...register('name')}
          />
          {errors.name && <span className="text-sm text-red-700">{errors.name.message}</span>}
        </label>
        <label className="block space-y-1">
          <span className="text-sm">Currency</span>
          <input
            className="w-full rounded-lg border border-black/15 bg-white px-3 py-2 uppercase outline-none focus:border-[var(--empops-accent)]"
            maxLength={3}
            {...register('currency')}
          />
          {errors.currency && (
            <span className="text-sm text-red-700">{errors.currency.message}</span>
          )}
        </label>
        <div className="col-span-full flex items-center gap-3">
          <button
            type="submit"
            className="rounded-lg bg-[var(--empops-accent)] px-4 py-2 text-sm font-medium text-white hover:opacity-90 disabled:opacity-60"
            disabled={updateMutation.isPending}
          >
            {updateMutation.isPending ? 'Saving…' : 'Save settings'}
          </button>
          {updateMutation.isSuccess && (
            <span className="text-sm text-[var(--empops-accent)]">Saved</span>
          )}
          {updateMutation.isError && (
            <span className="text-sm text-red-700">{(updateMutation.error as Error).message}</span>
          )}
        </div>
      </form>
      <ImageUploadField
        label="Company logo"
        imageUrl={company.logo_url}
        onUpload={async (ids) => {
          await authFetch(`/companies/${companyId}/logo`, {
            method: 'PUT',
            body: JSON.stringify(ids),
          })
          void queryClient.invalidateQueries({ queryKey: ['company', companyId] })
          void queryClient.invalidateQueries({ queryKey: ['companies'] })
        }}
      />
      {company.code_to_join_company && (
        <p className="text-sm text-black/60">
          Join code:{' '}
          <code className="rounded bg-black/[0.06] px-1.5 py-0.5">
            {company.code_to_join_company}
          </code>
        </p>
      )}
    </section>
  )
}

function PositionsSection({ companyId }: { companyId: string }) {
  const queryClient = useQueryClient()
  const [editingId, setEditingId] = useState<string | null>(null)

  const positionsQuery = useQuery({
    queryKey: ['positions', companyId],
    queryFn: async () => {
      const res = await authFetch<Position[]>(`/companies/${companyId}/positions`)
      return res.data
    },
  })

  const createForm = useForm<PositionValues>({
    resolver: zodResolver(positionSchema),
    defaultValues: { title: '' },
  })

  const createMutation = useMutation({
    mutationFn: async (values: PositionValues) => {
      const res = await authFetch<Position>(`/companies/${companyId}/positions`, {
        method: 'POST',
        body: JSON.stringify(values),
      })
      return res.data
    },
    onSuccess: () => {
      createForm.reset({ title: '' })
      void queryClient.invalidateQueries({ queryKey: ['positions', companyId] })
    },
  })

  const updateMutation = useMutation({
    mutationFn: async ({ id, title }: { id: string; title: string }) => {
      const res = await authFetch<Position>(`/companies/${companyId}/positions/${id}`, {
        method: 'PATCH',
        body: JSON.stringify({ title }),
      })
      return res.data
    },
    onSuccess: () => {
      setEditingId(null)
      void queryClient.invalidateQueries({ queryKey: ['positions', companyId] })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      await authFetch(`/companies/${companyId}/positions/${id}`, { method: 'DELETE' })
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['positions', companyId] })
    },
  })

  return (
    <section className="space-y-3 rounded-2xl border border-black/10 bg-white/80 p-5 shadow-sm">
      <h2 className="text-lg font-semibold">Positions</h2>
      {positionsQuery.isLoading && <p className="text-sm text-black/60">Loading…</p>}
      <ul className="space-y-2">
        {positionsQuery.data?.map((p) => (
          <li key={p.id} className="flex items-center justify-between gap-2 rounded-lg bg-black/[0.03] px-3 py-2">
            {editingId === p.id ? (
              <EditableTitleRow
                initialValue={p.title}
                isSaving={updateMutation.isPending}
                onCancel={() => setEditingId(null)}
                onSave={(title) => updateMutation.mutate({ id: p.id, title })}
              />
            ) : (
              <>
                <span className="text-sm">{p.title}</span>
                <div className="flex gap-2">
                  <button
                    type="button"
                    onClick={() => setEditingId(p.id)}
                    className="rounded-lg border border-black/15 px-2 py-1 text-xs hover:bg-white"
                  >
                    Edit
                  </button>
                  <button
                    type="button"
                    onClick={() => deleteMutation.mutate(p.id)}
                    className="rounded-lg border border-black/15 px-2 py-1 text-xs text-red-700 hover:bg-white"
                  >
                    Delete
                  </button>
                </div>
              </>
            )}
          </li>
        ))}
      </ul>
      <form
        className="flex flex-wrap items-end gap-2"
        onSubmit={createForm.handleSubmit((values) => createMutation.mutate(values))}
      >
        <label className="block flex-1 space-y-1">
          <span className="text-xs text-black/60">New position title</span>
          <input
            className="w-full rounded-lg border border-black/15 bg-white px-3 py-2 text-sm outline-none focus:border-[var(--empops-accent)]"
            {...createForm.register('title')}
          />
        </label>
        <button
          type="submit"
          className="rounded-lg border border-black/15 px-3 py-2 text-sm hover:bg-black/[0.03] disabled:opacity-60"
          disabled={createMutation.isPending}
        >
          Add
        </button>
      </form>
      {createForm.formState.errors.title && (
        <p className="text-sm text-red-700">{createForm.formState.errors.title.message}</p>
      )}
    </section>
  )
}

function EmployeeStatusesSection({ companyId }: { companyId: string }) {
  const queryClient = useQueryClient()
  const [editingId, setEditingId] = useState<string | null>(null)

  const statusesQuery = useQuery({
    queryKey: ['employee-statuses', companyId],
    queryFn: async () => {
      const res = await authFetch<EmployeeStatus[]>(`/companies/${companyId}/employee-statuses`)
      return res.data
    },
  })

  const createForm = useForm<StatusValues>({
    resolver: zodResolver(statusSchema),
    defaultValues: { name: '', type: 'internal' },
  })

  const createMutation = useMutation({
    mutationFn: async (values: StatusValues) => {
      const res = await authFetch<EmployeeStatus>(`/companies/${companyId}/employee-statuses`, {
        method: 'POST',
        body: JSON.stringify(values),
      })
      return res.data
    },
    onSuccess: () => {
      createForm.reset({ name: '', type: 'internal' })
      void queryClient.invalidateQueries({ queryKey: ['employee-statuses', companyId] })
    },
  })

  const updateMutation = useMutation({
    mutationFn: async ({
      id,
      name,
      type,
    }: {
      id: string
      name: string
      type: EmployeeStatusType
    }) => {
      const res = await authFetch<EmployeeStatus>(
        `/companies/${companyId}/employee-statuses/${id}`,
        { method: 'PATCH', body: JSON.stringify({ name, type }) },
      )
      return res.data
    },
    onSuccess: () => {
      setEditingId(null)
      void queryClient.invalidateQueries({ queryKey: ['employee-statuses', companyId] })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      await authFetch(`/companies/${companyId}/employee-statuses/${id}`, { method: 'DELETE' })
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['employee-statuses', companyId] })
    },
  })

  return (
    <section className="space-y-3 rounded-2xl border border-black/10 bg-white/80 p-5 shadow-sm">
      <h2 className="text-lg font-semibold">Employee statuses</h2>
      {statusesQuery.isLoading && <p className="text-sm text-black/60">Loading…</p>}
      <ul className="space-y-2">
        {statusesQuery.data?.map((s) => (
          <li
            key={s.id}
            className="flex items-center justify-between gap-2 rounded-lg bg-black/[0.03] px-3 py-2"
          >
            {editingId === s.id ? (
              <EditableStatusRow
                initialName={s.name}
                initialType={s.type}
                isSaving={updateMutation.isPending}
                onCancel={() => setEditingId(null)}
                onSave={(name, type) => updateMutation.mutate({ id: s.id, name, type })}
              />
            ) : (
              <>
                <span className="text-sm">
                  {s.name} <span className="text-xs text-black/50">({s.type})</span>
                </span>
                <div className="flex gap-2">
                  <button
                    type="button"
                    onClick={() => setEditingId(s.id)}
                    className="rounded-lg border border-black/15 px-2 py-1 text-xs hover:bg-white"
                  >
                    Edit
                  </button>
                  <button
                    type="button"
                    onClick={() => deleteMutation.mutate(s.id)}
                    className="rounded-lg border border-black/15 px-2 py-1 text-xs text-red-700 hover:bg-white"
                  >
                    Delete
                  </button>
                </div>
              </>
            )}
          </li>
        ))}
      </ul>
      <form
        className="flex flex-wrap items-end gap-2"
        onSubmit={createForm.handleSubmit((values) => createMutation.mutate(values))}
      >
        <label className="block flex-1 space-y-1">
          <span className="text-xs text-black/60">New status name</span>
          <input
            className="w-full rounded-lg border border-black/15 bg-white px-3 py-2 text-sm outline-none focus:border-[var(--empops-accent)]"
            {...createForm.register('name')}
          />
        </label>
        <label className="block space-y-1">
          <span className="text-xs text-black/60">Type</span>
          <select
            className="rounded-lg border border-black/15 bg-white px-3 py-2 text-sm outline-none focus:border-[var(--empops-accent)]"
            {...createForm.register('type')}
          >
            <option value="internal">Internal</option>
            <option value="external">External</option>
          </select>
        </label>
        <button
          type="submit"
          className="rounded-lg border border-black/15 px-3 py-2 text-sm hover:bg-black/[0.03] disabled:opacity-60"
          disabled={createMutation.isPending}
        >
          Add
        </button>
      </form>
      {createForm.formState.errors.name && (
        <p className="text-sm text-red-700">{createForm.formState.errors.name.message}</p>
      )}
    </section>
  )
}

function EditableTitleRow({
  initialValue,
  isSaving,
  onCancel,
  onSave,
}: {
  initialValue: string
  isSaving: boolean
  onCancel: () => void
  onSave: (value: string) => void
}) {
  const [value, setValue] = useState(initialValue)
  return (
    <div className="flex flex-1 items-center gap-2">
      <input
        className="flex-1 rounded-lg border border-black/15 bg-white px-2 py-1 text-sm outline-none focus:border-[var(--empops-accent)]"
        value={value}
        onChange={(e) => setValue(e.target.value)}
      />
      <button
        type="button"
        onClick={() => onSave(value)}
        disabled={isSaving || value.trim() === ''}
        className="rounded-lg bg-[var(--empops-accent)] px-2 py-1 text-xs font-medium text-white hover:opacity-90 disabled:opacity-60"
      >
        Save
      </button>
      <button
        type="button"
        onClick={onCancel}
        className="rounded-lg border border-black/15 px-2 py-1 text-xs hover:bg-white"
      >
        Cancel
      </button>
    </div>
  )
}

function EditableStatusRow({
  initialName,
  initialType,
  isSaving,
  onCancel,
  onSave,
}: {
  initialName: string
  initialType: EmployeeStatusType
  isSaving: boolean
  onCancel: () => void
  onSave: (name: string, type: EmployeeStatusType) => void
}) {
  const [name, setName] = useState(initialName)
  const [type, setType] = useState<EmployeeStatusType>(initialType)
  return (
    <div className="flex flex-1 flex-wrap items-center gap-2">
      <input
        className="flex-1 rounded-lg border border-black/15 bg-white px-2 py-1 text-sm outline-none focus:border-[var(--empops-accent)]"
        value={name}
        onChange={(e) => setName(e.target.value)}
      />
      <select
        className="rounded-lg border border-black/15 bg-white px-2 py-1 text-sm outline-none focus:border-[var(--empops-accent)]"
        value={type}
        onChange={(e) => setType(e.target.value as EmployeeStatusType)}
      >
        <option value="internal">Internal</option>
        <option value="external">External</option>
      </select>
      <button
        type="button"
        onClick={() => onSave(name, type)}
        disabled={isSaving || name.trim() === ''}
        className="rounded-lg bg-[var(--empops-accent)] px-2 py-1 text-xs font-medium text-white hover:opacity-90 disabled:opacity-60"
      >
        Save
      </button>
      <button
        type="button"
        onClick={onCancel}
        className="rounded-lg border border-black/15 px-2 py-1 text-xs hover:bg-white"
      >
        Cancel
      </button>
    </div>
  )
}

export default function AdminlandPage() {
  const { companyId } = useParams<{ companyId: string }>()
  const { isAdmin, isHrOrAdmin } = useCompanyContext()

  if (!isHrOrAdmin || !companyId) {
    return <Navigate to={`/companies/${companyId}/employees`} replace />
  }

  return (
    <div className="space-y-6">
      <section>
        <h2 className="text-lg font-semibold">Adminland</h2>
        <p className="text-sm text-black/60">
          Manage company settings, positions, and employee statuses.
        </p>
      </section>

      {isAdmin && <CompanySettingsSection companyId={companyId} />}
      <CompanyNewsSection companyId={companyId} />
      <QuestionsSection companyId={companyId} />
      <ExpenseCategoriesSection companyId={companyId} />
      <WorkFromHomeAdminSection companyId={companyId} />
      <AccountantsSection companyId={companyId} />
      <PositionsSection companyId={companyId} />
      <EmployeeStatusesSection companyId={companyId} />
    </div>
  )
}
