import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { authFetch } from '@/lib/authFetch'
import { useCompanyContext } from '@/routes/CompanyLayout'
import type { Employee, ExpenseCategory } from '@/types/api'

export function ExpenseCategoriesSection({ companyId }: { companyId: string }) {
  const qc = useQueryClient()
  const [name, setName] = useState('')
  const [error, setError] = useState<string | null>(null)

  const list = useQuery({
    queryKey: ['expense-categories', companyId],
    queryFn: async () => {
      const res = await authFetch<ExpenseCategory[]>(`/companies/${companyId}/expense-categories`)
      return res.data
    },
  })

  const create = useMutation({
    mutationFn: async () => {
      await authFetch(`/companies/${companyId}/expense-categories`, {
        method: 'POST',
        body: JSON.stringify({ name }),
      })
    },
    onSuccess: () => {
      setName('')
      setError(null)
      void qc.invalidateQueries({ queryKey: ['expense-categories', companyId] })
    },
    onError: (e: Error) => setError(e.message),
  })

  const destroy = useMutation({
    mutationFn: async (categoryId: string) => {
      await authFetch(`/companies/${companyId}/expense-categories/${categoryId}`, {
        method: 'DELETE',
      })
    },
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['expense-categories', companyId] }),
  })

  return (
    <section className="space-y-3 rounded-2xl border border-black/10 bg-white/80 p-5 shadow-sm">
      <h2 className="text-lg font-semibold">Expense categories</h2>
      <ul className="space-y-2 text-sm">
        {(list.data ?? []).map((c) => (
          <li key={c.id} className="flex items-center justify-between gap-2">
            <span>{c.name}</span>
            <button
              type="button"
              className="rounded-lg border border-black/15 px-2 py-1 text-xs hover:bg-white"
              onClick={() => destroy.mutate(c.id)}
            >
              Delete
            </button>
          </li>
        ))}
      </ul>
      <div className="flex flex-wrap gap-2">
        <input
          className="min-w-[12rem] flex-1 rounded-lg border border-black/15 bg-white px-3 py-2 text-sm"
          placeholder="New category"
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
        <button
          type="button"
          disabled={!name.trim() || create.isPending}
          className="rounded-lg bg-[var(--empops-accent)] px-3 py-2 text-sm text-white disabled:opacity-60"
          onClick={() => create.mutate()}
        >
          Add
        </button>
      </div>
      {error && <p className="text-sm text-red-700">{error}</p>}
    </section>
  )
}

export function WorkFromHomeAdminSection({ companyId }: { companyId: string }) {
  const { company } = useCompanyContext()
  const qc = useQueryClient()

  const setting = useQuery({
    queryKey: ['wfh-setting', companyId],
    queryFn: async () => {
      const res = await authFetch<{ enabled: boolean }>(`/companies/${companyId}/work-from-home`)
      return res.data
    },
    initialData: { enabled: company.work_from_home_enabled ?? true },
  })

  const toggle = useMutation({
    mutationFn: async (enabled: boolean) => {
      const res = await authFetch<{ enabled: boolean }>(`/companies/${companyId}/work-from-home`, {
        method: 'PATCH',
        body: JSON.stringify({ enabled }),
      })
      return res.data
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['wfh-setting', companyId] })
      void qc.invalidateQueries({ queryKey: ['company', companyId] })
    },
  })

  const enabled = setting.data?.enabled ?? true

  return (
    <section className="space-y-3 rounded-2xl border border-black/10 bg-white/80 p-5 shadow-sm">
      <h2 className="text-lg font-semibold">Work from home</h2>
      <p className="text-sm text-black/60">
        When enabled, employees can mark days as work-from-home on the Me dashboard.
      </p>
      <button
        type="button"
        className="rounded-lg border border-black/15 px-3 py-2 text-sm hover:bg-white disabled:opacity-60"
        disabled={toggle.isPending}
        onClick={() => toggle.mutate(!enabled)}
      >
        {enabled ? 'Disable WFH tracking' : 'Enable WFH tracking'}
      </button>
      <p className="text-xs text-black/50">Status: {enabled ? 'enabled' : 'disabled'}</p>
    </section>
  )
}

export function AccountantsSection({ companyId }: { companyId: string }) {
  const qc = useQueryClient()
  const [error, setError] = useState<string | null>(null)

  const employees = useQuery({
    queryKey: ['employees', companyId],
    queryFn: async () => {
      const res = await authFetch<Employee[]>(`/companies/${companyId}/employees`)
      return res.data
    },
  })

  const setAccountant = useMutation({
    mutationFn: async ({ employeeId, grant }: { employeeId: string; grant: boolean }) => {
      await authFetch(`/companies/${companyId}/employees/${employeeId}/accountant`, {
        method: grant ? 'POST' : 'DELETE',
      })
    },
    onSuccess: () => {
      setError(null)
      void qc.invalidateQueries({ queryKey: ['employees', companyId] })
    },
    onError: (e: Error) => setError(e.message),
  })

  const accountants = (employees.data ?? []).filter((e) => e.roles.includes('accountant'))
  const candidates = (employees.data ?? []).filter((e) => !e.roles.includes('accountant'))

  return (
    <section className="space-y-3 rounded-2xl border border-black/10 bg-white/80 p-5 shadow-sm">
      <h2 className="text-lg font-semibold">Accountants</h2>
      <p className="text-sm text-black/60">
        Accountants finalize expenses after manager approval.
      </p>
      <ul className="space-y-2 text-sm">
        {accountants.map((e) => (
          <li key={e.id} className="flex items-center justify-between gap-2">
            <span>
              {e.first_name} {e.last_name}
            </span>
            <button
              type="button"
              className="rounded-lg border border-black/15 px-2 py-1 text-xs hover:bg-white"
              onClick={() => setAccountant.mutate({ employeeId: e.id, grant: false })}
            >
              Revoke
            </button>
          </li>
        ))}
        {accountants.length === 0 && (
          <li className="text-black/50">No accountants assigned.</li>
        )}
      </ul>
      <div className="flex flex-wrap gap-2">
        <select
          className="min-w-[12rem] flex-1 rounded-lg border border-black/15 bg-white px-3 py-2 text-sm"
          defaultValue=""
          id="grant-accountant"
          onChange={(e) => {
            const employeeId = e.target.value
            if (!employeeId) return
            setAccountant.mutate({ employeeId, grant: true })
            e.target.value = ''
          }}
        >
          <option value="">Grant accountant role…</option>
          {candidates.map((e) => (
            <option key={e.id} value={e.id}>
              {e.first_name} {e.last_name}
            </option>
          ))}
        </select>
      </div>
      {error && <p className="text-sm text-red-700">{error}</p>}
    </section>
  )
}
