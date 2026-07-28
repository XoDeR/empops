import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useParams } from 'react-router-dom'
import { authFetch } from '@/lib/authFetch'
import { useCompanyContext } from '@/routes/CompanyLayout'
import type { CompanyRole, Employee, EmployeeStatus, Position } from '@/types/api'

const createEmployeeSchema = z.object({
  email: z.string().email(),
  first_name: z.string().min(1, 'Required'),
  last_name: z.string().min(1, 'Required'),
  hired_at: z.string().optional(),
  position_id: z.string().optional(),
  employee_status_id: z.string().optional(),
  role: z.enum(['administrator', 'hr', 'employee']),
})
type CreateEmployeeValues = z.infer<typeof createEmployeeSchema>

const editEmployeeSchema = z.object({
  first_name: z.string().min(1, 'Required'),
  last_name: z.string().min(1, 'Required'),
  email: z.string().email().optional(),
  hired_at: z.string().optional(),
  position_id: z.string().optional(),
  employee_status_id: z.string().optional(),
  role: z.enum(['administrator', 'hr', 'employee']).optional(),
})
type EditEmployeeValues = z.infer<typeof editEmployeeSchema>

function EmployeeEditForm({
  employee,
  canFull,
  positions,
  statuses,
  onSubmit,
  onCancel,
  isSaving,
}: {
  employee: Employee
  canFull: boolean
  positions: Position[]
  statuses: EmployeeStatus[]
  onSubmit: (payload: Record<string, unknown>) => void
  onCancel: () => void
  isSaving: boolean
}) {
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<EditEmployeeValues>({
    resolver: zodResolver(editEmployeeSchema),
    defaultValues: {
      first_name: employee.first_name,
      last_name: employee.last_name,
      email: employee.email,
      hired_at: employee.hired_at ?? '',
      position_id: employee.position?.id ?? '',
      employee_status_id: employee.status?.id ?? '',
      role: (employee.roles[0] as CompanyRole | undefined) ?? 'employee',
    },
  })

  const submit = handleSubmit((values) => {
    if (canFull) {
      onSubmit({
        first_name: values.first_name,
        last_name: values.last_name,
        email: values.email,
        hired_at: values.hired_at || null,
        position_id: values.position_id || null,
        employee_status_id: values.employee_status_id || null,
        role: values.role,
      })
    } else {
      onSubmit({
        first_name: values.first_name,
        last_name: values.last_name,
      })
    }
  })

  return (
    <form onSubmit={submit} className="grid gap-3 rounded-lg bg-black/[0.03] p-3 sm:grid-cols-2">
      <label className="block space-y-1">
        <span className="text-xs text-black/60">First name</span>
        <input
          className="w-full rounded-lg border border-black/15 bg-white px-2 py-1.5 text-sm outline-none focus:border-[var(--empops-accent)]"
          {...register('first_name')}
        />
        {errors.first_name && (
          <span className="text-xs text-red-700">{errors.first_name.message}</span>
        )}
      </label>
      <label className="block space-y-1">
        <span className="text-xs text-black/60">Last name</span>
        <input
          className="w-full rounded-lg border border-black/15 bg-white px-2 py-1.5 text-sm outline-none focus:border-[var(--empops-accent)]"
          {...register('last_name')}
        />
        {errors.last_name && (
          <span className="text-xs text-red-700">{errors.last_name.message}</span>
        )}
      </label>

      {canFull && (
        <>
          <label className="block space-y-1">
            <span className="text-xs text-black/60">Email</span>
            <input
              className="w-full rounded-lg border border-black/15 bg-white px-2 py-1.5 text-sm outline-none focus:border-[var(--empops-accent)]"
              type="email"
              {...register('email')}
            />
            {errors.email && <span className="text-xs text-red-700">{errors.email.message}</span>}
          </label>
          <label className="block space-y-1">
            <span className="text-xs text-black/60">Hired at</span>
            <input
              className="w-full rounded-lg border border-black/15 bg-white px-2 py-1.5 text-sm outline-none focus:border-[var(--empops-accent)]"
              type="date"
              {...register('hired_at')}
            />
          </label>
          <label className="block space-y-1">
            <span className="text-xs text-black/60">Position</span>
            <select
              className="w-full rounded-lg border border-black/15 bg-white px-2 py-1.5 text-sm outline-none focus:border-[var(--empops-accent)]"
              {...register('position_id')}
            >
              <option value="">None</option>
              {positions.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.title}
                </option>
              ))}
            </select>
          </label>
          <label className="block space-y-1">
            <span className="text-xs text-black/60">Status</span>
            <select
              className="w-full rounded-lg border border-black/15 bg-white px-2 py-1.5 text-sm outline-none focus:border-[var(--empops-accent)]"
              {...register('employee_status_id')}
            >
              <option value="">None</option>
              {statuses.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name}
                </option>
              ))}
            </select>
          </label>
          <label className="block space-y-1">
            <span className="text-xs text-black/60">Role</span>
            <select
              className="w-full rounded-lg border border-black/15 bg-white px-2 py-1.5 text-sm outline-none focus:border-[var(--empops-accent)]"
              {...register('role')}
            >
              <option value="employee">Employee</option>
              <option value="hr">HR</option>
              <option value="administrator">Administrator</option>
            </select>
          </label>
        </>
      )}

      <div className="col-span-full flex gap-2 pt-1">
        <button
          type="submit"
          className="rounded-lg bg-[var(--empops-accent)] px-3 py-1.5 text-sm font-medium text-white hover:opacity-90 disabled:opacity-60"
          disabled={isSaving}
        >
          {isSaving ? 'Saving…' : 'Save'}
        </button>
        <button
          type="button"
          onClick={onCancel}
          className="rounded-lg border border-black/15 px-3 py-1.5 text-sm hover:bg-black/[0.03]"
        >
          Cancel
        </button>
      </div>
    </form>
  )
}

export default function EmployeesPage() {
  const { companyId } = useParams<{ companyId: string }>()
  const { company, isHrOrAdmin } = useCompanyContext()
  const queryClient = useQueryClient()
  const [editingId, setEditingId] = useState<string | null>(null)
  const [inviteResult, setInviteResult] = useState<{ employeeId: string; code: string } | null>(
    null,
  )

  const employeesQuery = useQuery({
    queryKey: ['employees', companyId],
    queryFn: async () => {
      const res = await authFetch<Employee[]>(`/companies/${companyId}/employees`)
      return res.data
    },
    enabled: Boolean(companyId),
  })

  const positionsQuery = useQuery({
    queryKey: ['positions', companyId],
    queryFn: async () => {
      const res = await authFetch<Position[]>(`/companies/${companyId}/positions`)
      return res.data
    },
    enabled: Boolean(companyId) && isHrOrAdmin,
  })

  const statusesQuery = useQuery({
    queryKey: ['employee-statuses', companyId],
    queryFn: async () => {
      const res = await authFetch<EmployeeStatus[]>(`/companies/${companyId}/employee-statuses`)
      return res.data
    },
    enabled: Boolean(companyId) && isHrOrAdmin,
  })

  const createForm = useForm<CreateEmployeeValues>({
    resolver: zodResolver(createEmployeeSchema),
    defaultValues: {
      email: '',
      first_name: '',
      last_name: '',
      hired_at: '',
      position_id: '',
      employee_status_id: '',
      role: 'employee',
    },
  })

  const createMutation = useMutation({
    mutationFn: async (values: CreateEmployeeValues) => {
      const payload: Record<string, unknown> = {
        email: values.email,
        first_name: values.first_name,
        last_name: values.last_name,
        role: values.role,
      }
      if (values.hired_at) payload.hired_at = values.hired_at
      if (values.position_id) payload.position_id = values.position_id
      if (values.employee_status_id) payload.employee_status_id = values.employee_status_id

      const res = await authFetch<Employee>(`/companies/${companyId}/employees`, {
        method: 'POST',
        body: JSON.stringify(payload),
      })
      return res.data
    },
    onSuccess: () => {
      createForm.reset({
        email: '',
        first_name: '',
        last_name: '',
        hired_at: '',
        position_id: '',
        employee_status_id: '',
        role: 'employee',
      })
      void queryClient.invalidateQueries({ queryKey: ['employees', companyId] })
    },
  })

  const updateMutation = useMutation({
    mutationFn: async ({
      employeeId,
      payload,
    }: {
      employeeId: string
      payload: Record<string, unknown>
    }) => {
      const res = await authFetch<Employee>(
        `/companies/${companyId}/employees/${employeeId}`,
        { method: 'PATCH', body: JSON.stringify(payload) },
      )
      return res.data
    },
    onSuccess: () => {
      setEditingId(null)
      void queryClient.invalidateQueries({ queryKey: ['employees', companyId] })
    },
  })

  const inviteMutation = useMutation({
    mutationFn: async (employeeId: string) => {
      const res = await authFetch<Employee>(
        `/companies/${companyId}/employees/${employeeId}/invite`,
        { method: 'POST' },
      )
      return res.data
    },
    onSuccess: (data) => {
      setInviteResult({ employeeId: data.id, code: data.invitation_link ?? '' })
      void queryClient.invalidateQueries({ queryKey: ['employees', companyId] })
    },
  })

  const currentEmployeeId = company.employee_id

  return (
    <div className="space-y-6">
      <section>
        <h2 className="text-lg font-semibold">Employee directory</h2>
        <p className="text-sm text-black/60">
          {isHrOrAdmin
            ? 'Create, invite, and edit employees in this company.'
            : 'You can update your own name below.'}
        </p>
      </section>

      {employeesQuery.isLoading && <p className="text-black/60">Loading…</p>}
      {employeesQuery.isError && (
        <p className="text-red-700">{(employeesQuery.error as Error).message}</p>
      )}

      <div className="space-y-3">
        {employeesQuery.data?.map((employee) => {
          const isSelf = employee.id === currentEmployeeId
          const canFull = isHrOrAdmin
          const canEdit = canFull || isSelf
          const isEditing = editingId === employee.id

          return (
            <div
              key={employee.id}
              className="rounded-2xl border border-black/10 bg-white/80 p-4 shadow-sm"
            >
              {isEditing ? (
                <EmployeeEditForm
                  employee={employee}
                  canFull={canFull}
                  positions={positionsQuery.data ?? []}
                  statuses={statusesQuery.data ?? []}
                  isSaving={updateMutation.isPending}
                  onCancel={() => setEditingId(null)}
                  onSubmit={(payload) =>
                    updateMutation.mutate({ employeeId: employee.id, payload })
                  }
                />
              ) : (
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div>
                    <p className="font-semibold">
                      {employee.first_name} {employee.last_name}
                      {isSelf && <span className="ml-2 text-xs text-black/50">(you)</span>}
                      {employee.locked && (
                        <span className="ml-2 rounded-full bg-red-100 px-2 py-0.5 text-xs text-red-700">
                          locked
                        </span>
                      )}
                    </p>
                    <p className="text-sm text-black/60">{employee.email}</p>
                    <p className="mt-1 flex flex-wrap gap-1 text-xs text-black/60">
                      {employee.position && (
                        <span className="rounded-full bg-black/[0.06] px-2 py-0.5">
                          {employee.position.title}
                        </span>
                      )}
                      {employee.status && (
                        <span className="rounded-full bg-black/[0.06] px-2 py-0.5">
                          {employee.status.name}
                        </span>
                      )}
                      {employee.roles.map((r) => (
                        <span
                          key={r}
                          className="rounded-full bg-[var(--empops-accent)]/10 px-2 py-0.5 text-[var(--empops-accent)]"
                        >
                          {r}
                        </span>
                      ))}
                      {!employee.user_id && (
                        <span className="rounded-full bg-amber-100 px-2 py-0.5 text-amber-700">
                          not yet linked
                        </span>
                      )}
                    </p>
                    {inviteResult?.employeeId === employee.id && (
                      <p className="mt-2 text-xs text-black/70">
                        Invite code:{' '}
                        <code className="rounded bg-black/[0.06] px-1.5 py-0.5">
                          {inviteResult.code}
                        </code>
                      </p>
                    )}
                  </div>
                  <div className="flex shrink-0 gap-2">
                    {canEdit && (
                      <button
                        type="button"
                        onClick={() => setEditingId(employee.id)}
                        className="rounded-lg border border-black/15 px-3 py-1.5 text-sm hover:bg-black/[0.03]"
                      >
                        Edit
                      </button>
                    )}
                    {canFull && !employee.user_id && (
                      <button
                        type="button"
                        onClick={() => inviteMutation.mutate(employee.id)}
                        disabled={inviteMutation.isPending}
                        className="rounded-lg border border-black/15 px-3 py-1.5 text-sm hover:bg-black/[0.03] disabled:opacity-60"
                      >
                        Invite
                      </button>
                    )}
                  </div>
                </div>
              )}
            </div>
          )
        })}
      </div>

      {isHrOrAdmin && (
        <form
          className="space-y-3 rounded-2xl border border-black/10 bg-white/80 p-5 shadow-sm"
          onSubmit={createForm.handleSubmit((values) => createMutation.mutate(values))}
        >
          <h2 className="text-lg font-semibold">Add employee</h2>
          <div className="grid gap-3 sm:grid-cols-2">
            <label className="block space-y-1">
              <span className="text-sm">Email</span>
              <input
                className="w-full rounded-lg border border-black/15 bg-white px-3 py-2 outline-none focus:border-[var(--empops-accent)]"
                type="email"
                {...createForm.register('email')}
              />
              {createForm.formState.errors.email && (
                <span className="text-sm text-red-700">
                  {createForm.formState.errors.email.message}
                </span>
              )}
            </label>
            <label className="block space-y-1">
              <span className="text-sm">Role</span>
              <select
                className="w-full rounded-lg border border-black/15 bg-white px-3 py-2 outline-none focus:border-[var(--empops-accent)]"
                {...createForm.register('role')}
              >
                <option value="employee">Employee</option>
                <option value="hr">HR</option>
                <option value="administrator">Administrator</option>
              </select>
            </label>
            <label className="block space-y-1">
              <span className="text-sm">First name</span>
              <input
                className="w-full rounded-lg border border-black/15 bg-white px-3 py-2 outline-none focus:border-[var(--empops-accent)]"
                {...createForm.register('first_name')}
              />
              {createForm.formState.errors.first_name && (
                <span className="text-sm text-red-700">
                  {createForm.formState.errors.first_name.message}
                </span>
              )}
            </label>
            <label className="block space-y-1">
              <span className="text-sm">Last name</span>
              <input
                className="w-full rounded-lg border border-black/15 bg-white px-3 py-2 outline-none focus:border-[var(--empops-accent)]"
                {...createForm.register('last_name')}
              />
              {createForm.formState.errors.last_name && (
                <span className="text-sm text-red-700">
                  {createForm.formState.errors.last_name.message}
                </span>
              )}
            </label>
            <label className="block space-y-1">
              <span className="text-sm">Hired at (optional)</span>
              <input
                className="w-full rounded-lg border border-black/15 bg-white px-3 py-2 outline-none focus:border-[var(--empops-accent)]"
                type="date"
                {...createForm.register('hired_at')}
              />
            </label>
            <label className="block space-y-1">
              <span className="text-sm">Position (optional)</span>
              <select
                className="w-full rounded-lg border border-black/15 bg-white px-3 py-2 outline-none focus:border-[var(--empops-accent)]"
                {...createForm.register('position_id')}
              >
                <option value="">None</option>
                {positionsQuery.data?.map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.title}
                  </option>
                ))}
              </select>
            </label>
            <label className="block space-y-1">
              <span className="text-sm">Status (optional)</span>
              <select
                className="w-full rounded-lg border border-black/15 bg-white px-3 py-2 outline-none focus:border-[var(--empops-accent)]"
                {...createForm.register('employee_status_id')}
              >
                <option value="">None</option>
                {statusesQuery.data?.map((s) => (
                  <option key={s.id} value={s.id}>
                    {s.name}
                  </option>
                ))}
              </select>
            </label>
          </div>
          {createMutation.isError && (
            <p className="text-sm text-red-700">{(createMutation.error as Error).message}</p>
          )}
          <button
            type="submit"
            className="rounded-lg bg-[var(--empops-accent)] px-4 py-2.5 font-medium text-white hover:opacity-90 disabled:opacity-60"
            disabled={createMutation.isPending}
          >
            {createMutation.isPending ? 'Adding…' : 'Add employee'}
          </button>
        </form>
      )}
    </div>
  )
}
