import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { authFetch } from '@/lib/authFetch'
import type { CompanyInvoice, Flow, PtoCalendarDay, PtoPolicy } from '@/types/api'

export function InventoryAdminSection({ companyId }: { companyId: string }) {
  return (
    <section className="space-y-3 rounded-2xl border border-black/10 bg-white/80 p-5 shadow-sm">
      <h2 className="text-lg font-semibold">Hardware & software</h2>
      <p className="text-sm text-black/60">
        Track company assets and SaaS seat ownership.
      </p>
      <div className="flex flex-wrap gap-3">
        <Link
          to={`/companies/${companyId}/hardware`}
          className="rounded-lg border border-black/15 px-3 py-2 text-sm hover:border-[var(--empops-accent)]"
        >
          Manage hardware
        </Link>
        <Link
          to={`/companies/${companyId}/softwares`}
          className="rounded-lg border border-black/15 px-3 py-2 text-sm hover:border-[var(--empops-accent)]"
        >
          Manage software
        </Link>
      </div>
    </section>
  )
}

const cardClass = 'space-y-3 rounded-2xl border border-black/10 bg-white/80 p-5 shadow-sm'
const inputClass = 'rounded-lg border border-black/15 bg-white px-3 py-2 text-sm'

export function StepTenLinksSection({ companyId }: { companyId: string }) {
  return (
    <section className={cardClass}>
      <h2 className="text-lg font-semibold">Company spaces</h2>
      <p className="text-sm text-black/60">Open shared knowledge, conversations, and groups.</p>
      <div className="flex flex-wrap gap-2">
        {(['wikis', 'ama', 'groups'] as const).map((path) => <Link key={path} className="rounded-lg border border-black/15 px-3 py-2 text-sm capitalize hover:border-[var(--empops-accent)]" to={`/companies/${companyId}/${path}`}>{path === 'wikis' ? 'Wiki' : path.toUpperCase()}</Link>)}
      </div>
    </section>
  )
}

export function PtoPoliciesSection({ companyId }: { companyId: string }) {
  const qc = useQueryClient()
  const [year, setYear] = useState(new Date().getFullYear())
  const [selected, setSelected] = useState<PtoPolicy | null>(null)
  const [month, setMonth] = useState(`${new Date().getFullYear()}-01`)
  const [defaults, setDefaults] = useState({ holidays: 20, sick: 5, pto: 0 })
  const policies = useQuery({ queryKey: ['pto-policies', companyId], queryFn: async () => (await authFetch<PtoPolicy[]>(`/companies/${companyId}/pto-policies`)).data })
  const calendar = useQuery({ queryKey: ['pto-calendar', companyId, selected?.id], queryFn: async () => (await authFetch<PtoCalendarDay[]>(`/companies/${companyId}/pto-policies/${selected?.id}/calendar`)).data, enabled: Boolean(selected) })
  const create = useMutation({ mutationFn: () => authFetch(`/companies/${companyId}/pto-policies`, { method: 'POST', body: JSON.stringify({ year, default_amount_of_allowed_holidays: defaults.holidays, default_amount_of_sick_days: defaults.sick, default_amount_of_pto_days: defaults.pto }) }), onSuccess: () => void qc.invalidateQueries({ queryKey: ['pto-policies', companyId] }) })
  const save = useMutation({ mutationFn: () => authFetch(`/companies/${companyId}/pto-policies/${selected?.id}`, { method: 'PATCH', body: JSON.stringify({ default_amount_of_allowed_holidays: defaults.holidays, default_amount_of_sick_days: defaults.sick, default_amount_of_pto_days: defaults.pto }) }), onSuccess: () => void qc.invalidateQueries({ queryKey: ['pto-policies', companyId] }) })
  const toggle = useMutation({ mutationFn: (day: PtoCalendarDay) => authFetch(`/companies/${companyId}/pto-policies/${selected?.id}/calendar/${day.day}`, { method: 'PATCH', body: JSON.stringify({ is_worked: !day.is_worked }) }), onSuccess: () => void qc.invalidateQueries({ queryKey: ['pto-calendar', companyId, selected?.id] }) })
  const choose = (policy: PtoPolicy) => {
    setSelected(policy); setMonth(`${policy.year}-01`)
    setDefaults({ holidays: Number(policy.default_amount_of_allowed_holidays), sick: Number(policy.default_amount_of_sick_days), pto: Number(policy.default_amount_of_pto_days) })
  }
  return (
    <section className={cardClass}>
      <h2 className="text-lg font-semibold">PTO policies</h2>
      <form className="flex flex-wrap gap-2" onSubmit={(e) => { e.preventDefault(); create.mutate() }}>
        <input aria-label="Policy year" type="number" min="2000" max="9999" className={inputClass} value={year} onChange={(e) => setYear(Number(e.target.value))} />
        <button className="rounded-lg border border-black/15 px-3 py-2 text-sm">Create year</button>
      </form>
      <div className="flex flex-wrap gap-2">{(policies.data ?? []).map((p) => <button type="button" key={p.id} onClick={() => choose(p)} className={`rounded-lg border px-3 py-2 text-sm ${selected?.id === p.id ? 'border-[var(--empops-accent)] bg-[var(--empops-accent)]/10' : 'border-black/15'}`}>{p.year} · {p.total_worked_days} work days</button>)}</div>
      {selected && <div className="space-y-3 border-t border-black/10 pt-3">
        <div className="grid gap-2 sm:grid-cols-3">{([['holidays', 'Holiday days'], ['sick', 'Sick days'], ['pto', 'PTO days']] as const).map(([key, label]) => <label key={key} className="space-y-1 text-xs text-black/60">{label}<input type="number" min="0" step="0.5" className={`${inputClass} w-full text-black`} value={defaults[key]} onChange={(e) => setDefaults((p) => ({ ...p, [key]: Number(e.target.value) }))} /></label>)}</div>
        <button type="button" className="rounded-lg bg-[var(--empops-accent)] px-3 py-2 text-sm text-white" onClick={() => save.mutate()}>Save defaults</button>
        <div><input aria-label="Calendar month" type="month" className={inputClass} value={month} min={`${selected.year}-01`} max={`${selected.year}-12`} onChange={(e) => setMonth(e.target.value)} /><div className="mt-2 grid grid-cols-4 gap-1 sm:grid-cols-7">{(calendar.data ?? []).filter((d) => d.day.startsWith(month)).map((d) => <button type="button" key={d.day} title={d.is_worked ? 'Worked day' : 'Day off'} onClick={() => toggle.mutate(d)} className={`rounded p-2 text-xs ${d.is_worked ? 'bg-green-100 text-green-800' : 'bg-black/[0.06] text-black/55'}`}>{Number(d.day.slice(-2))}</button>)}</div></div>
      </div>}
    </section>
  )
}

export function FlowsSection({ companyId }: { companyId: string }) {
  const qc = useQueryClient()
  const [name, setName] = useState('')
  const [type, setType] = useState('onboarding')
  const [selectedId, setSelectedId] = useState('')
  const [step, setStep] = useState({ number: 1, unit_of_time: 'days', modifier: 'after' })
  const [action, setAction] = useState<Record<string, { type: string; recipient: string }>>({})
  const list = useQuery({ queryKey: ['flows', companyId], queryFn: async () => (await authFetch<Flow[]>(`/companies/${companyId}/flows`)).data })
  const detail = useQuery({ queryKey: ['flow', companyId, selectedId], queryFn: async () => (await authFetch<Flow>(`/companies/${companyId}/flows/${selectedId}`)).data, enabled: Boolean(selectedId) })
  const create = useMutation({ mutationFn: () => authFetch(`/companies/${companyId}/flows`, { method: 'POST', body: JSON.stringify({ name, type }) }), onSuccess: () => { setName(''); void qc.invalidateQueries({ queryKey: ['flows', companyId] }) } })
  const addStep = useMutation({ mutationFn: () => authFetch(`/companies/${companyId}/flows/${selectedId}/steps`, { method: 'POST', body: JSON.stringify(step) }), onSuccess: () => void qc.invalidateQueries({ queryKey: ['flow', companyId, selectedId] }) })
  const addAction = useMutation({ mutationFn: ({ stepId, value }: { stepId: string; value: { type: string; recipient: string } }) => authFetch(`/companies/${companyId}/flows/${selectedId}/steps/${stepId}/actions`, { method: 'POST', body: JSON.stringify({ ...value, specific_recipient_information: null }) }), onSuccess: (_, v) => { setAction((p) => ({ ...p, [v.stepId]: { type: '', recipient: '' } })); void qc.invalidateQueries({ queryKey: ['flow', companyId, selectedId] }) } })
  return (
    <section className={cardClass}>
      <h2 className="text-lg font-semibold">Flows</h2>
      <form className="flex flex-wrap gap-2" onSubmit={(e) => { e.preventDefault(); if (name.trim()) create.mutate() }}><input className={`${inputClass} flex-1`} placeholder="Flow name" value={name} onChange={(e) => setName(e.target.value)} /><input className={inputClass} placeholder="Type" value={type} onChange={(e) => setType(e.target.value)} /><button className="rounded-lg border border-black/15 px-3 py-2 text-sm">Create</button></form>
      <select className={`${inputClass} w-full`} value={selectedId} onChange={(e) => setSelectedId(e.target.value)}><option value="">Select a flow…</option>{(list.data ?? []).map((f) => <option key={f.id} value={f.id}>{f.name} ({f.type})</option>)}</select>
      {detail.data && <div className="space-y-3 border-t border-black/10 pt-3"><h3 className="font-medium">{detail.data.name}</h3>{(detail.data.steps ?? []).map((s) => <div key={s.id} className="rounded-lg bg-black/[0.03] p-3 text-sm"><p className="font-medium">{s.modifier} {s.number} {s.unit_of_time}</p><ul className="my-2 pl-4">{(s.actions ?? []).map((a) => <li key={a.id}>• {a.type} → {a.recipient}</li>)}</ul><form className="flex gap-2" onSubmit={(e) => { e.preventDefault(); const value = action[s.id]; if (value?.type && value.recipient) addAction.mutate({ stepId: s.id, value }) }}><input className={`${inputClass} min-w-0 flex-1`} placeholder="Action type" value={action[s.id]?.type ?? ''} onChange={(e) => setAction((p) => ({ ...p, [s.id]: { type: e.target.value, recipient: p[s.id]?.recipient ?? '' } }))} /><input className={`${inputClass} min-w-0 flex-1`} placeholder="Recipient" value={action[s.id]?.recipient ?? ''} onChange={(e) => setAction((p) => ({ ...p, [s.id]: { type: p[s.id]?.type ?? '', recipient: e.target.value } }))} /><button className="rounded-lg border border-black/15 px-2">Add</button></form></div>)}<form className="flex flex-wrap gap-2" onSubmit={(e) => { e.preventDefault(); addStep.mutate() }}><input type="number" min="0" className={`${inputClass} w-24`} value={step.number} onChange={(e) => setStep((p) => ({ ...p, number: Number(e.target.value) }))} /><select className={inputClass} value={step.modifier} onChange={(e) => setStep((p) => ({ ...p, modifier: e.target.value }))}><option value="before">before</option><option value="after">after</option></select><input className={inputClass} value={step.unit_of_time} onChange={(e) => setStep((p) => ({ ...p, unit_of_time: e.target.value }))} /><button className="rounded-lg bg-[var(--empops-accent)] px-3 py-2 text-sm text-white">Add step</button></form></div>}
    </section>
  )
}

export function BillingInvoicesSection({ companyId }: { companyId: string }) {
  const query = useQuery({ queryKey: ['invoices', companyId], queryFn: async () => (await authFetch<CompanyInvoice[]>(`/companies/${companyId}/invoices`)).data })
  return <section className={cardClass}><h2 className="text-lg font-semibold">Billing invoices</h2>{query.isLoading && <p className="text-sm text-black/55">Loading…</p>}<ul className="space-y-2 text-sm">{(query.data ?? []).map((i) => <li key={i.id} className="flex flex-wrap justify-between gap-2 rounded-lg bg-black/[0.03] p-3"><span>{i.logged_on ?? i.created_at?.slice(0, 10) ?? i.id}</span><span>{i.number_of_active_employees != null ? `${i.number_of_active_employees} employees · ` : ''}{i.customer_has_paid ? 'Paid' : i.sent_to_customer ? 'Sent' : 'Draft'}</span></li>)}</ul>{!query.isLoading && !query.data?.length && <p className="text-sm text-black/50">No invoices yet.</p>}</section>
}
