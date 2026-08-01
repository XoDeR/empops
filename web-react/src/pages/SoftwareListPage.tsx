import { Link, Navigate, useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { authFetch } from '@/lib/authFetch'
import { useCompanyContext } from '@/routes/CompanyLayout'
import type { Software } from '@/types/api'

export default function SoftwareListPage() {
  const { companyId } = useParams<{ companyId: string }>()
  const { isHrOrAdmin, company } = useCompanyContext()
  const qc = useQueryClient()
  const [name, setName] = useState('')
  const [productKey, setProductKey] = useState('')
  const [seats, setSeats] = useState('1')
  const [purchaseAmount, setPurchaseAmount] = useState('')
  const [currency, setCurrency] = useState(company.currency || 'USD')
  const [error, setError] = useState<string | null>(null)

  const listQuery = useQuery({
    queryKey: ['softwares', companyId],
    queryFn: async () => {
      const res = await authFetch<Software[]>(`/companies/${companyId}/softwares`)
      return res.data
    },
    enabled: Boolean(companyId) && isHrOrAdmin,
  })

  const create = useMutation({
    mutationFn: async () => {
      const amount = purchaseAmount.trim() ? Number(purchaseAmount) : null
      await authFetch(`/companies/${companyId}/softwares`, {
        method: 'POST',
        body: JSON.stringify({
          name,
          product_key: productKey,
          seats: Number(seats),
          purchase_amount: amount,
          currency: amount ? currency.toUpperCase() : null,
        }),
      })
    },
    onSuccess: () => {
      setName('')
      setProductKey('')
      setSeats('1')
      setPurchaseAmount('')
      setError(null)
      void qc.invalidateQueries({ queryKey: ['softwares', companyId] })
    },
    onError: (e: Error) => setError(e.message),
  })

  if (!isHrOrAdmin) {
    return <Navigate to={`/companies/${companyId}/dashboard/me`} replace />
  }

  return (
    <div className="space-y-6">
      <div>
        <Link
          to={`/companies/${companyId}/adminland`}
          className="text-sm text-black/55 hover:text-black"
        >
          ← Adminland
        </Link>
        <h2 className="mt-2 text-xl font-semibold">Software licenses</h2>
        <p className="text-sm text-black/55">Seat pools, purchase amounts, and assignments.</p>
      </div>

      <section className="rounded-2xl border border-black/10 bg-white/70 p-4 space-y-3">
        <h3 className="font-medium">Add software</h3>
        {error && <p className="text-sm text-red-600">{error}</p>}
        <form
          className="grid gap-2 sm:grid-cols-2"
          onSubmit={(e) => {
            e.preventDefault()
            create.mutate()
          }}
        >
          <input
            required
            className="rounded-lg border border-black/15 px-3 py-2 text-sm"
            placeholder="Name"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
          <input
            required
            className="rounded-lg border border-black/15 px-3 py-2 text-sm"
            placeholder="Product key"
            value={productKey}
            onChange={(e) => setProductKey(e.target.value)}
          />
          <input
            required
            type="number"
            min={1}
            className="rounded-lg border border-black/15 px-3 py-2 text-sm"
            placeholder="Seats"
            value={seats}
            onChange={(e) => setSeats(e.target.value)}
          />
          <input
            type="number"
            min={1}
            className="rounded-lg border border-black/15 px-3 py-2 text-sm"
            placeholder="Purchase amount (minor units)"
            value={purchaseAmount}
            onChange={(e) => setPurchaseAmount(e.target.value)}
          />
          <input
            className="rounded-lg border border-black/15 px-3 py-2 text-sm"
            placeholder="Currency"
            value={currency}
            onChange={(e) => setCurrency(e.target.value)}
          />
          <button
            type="submit"
            disabled={create.isPending}
            className="rounded-lg bg-[var(--empops-accent)] px-3 py-2 text-sm text-white sm:col-span-2"
          >
            Create
          </button>
        </form>
      </section>

      <section className="rounded-2xl border border-black/10 bg-white/70 p-4">
        {listQuery.isLoading && <p className="text-sm text-black/60">Loading…</p>}
        <ul className="divide-y divide-black/5">
          {(listQuery.data ?? []).map((s) => (
            <li key={s.id} className="flex flex-wrap items-center justify-between gap-2 py-3">
              <div>
                <Link
                  to={`/companies/${companyId}/softwares/${s.id}`}
                  className="font-medium hover:underline"
                >
                  {s.name}
                </Link>
                <p className="text-xs text-black/50">
                  {s.seats_used ?? 0}/{s.seats} seats used
                  {s.purchase_amount != null && s.currency
                    ? ` · ${s.purchase_amount} ${s.currency}`
                    : ''}
                  {s.converted_purchase_amount != null && s.converted_to_currency
                    ? ` → ${s.converted_purchase_amount} ${s.converted_to_currency}`
                    : ''}
                </p>
              </div>
              <Link
                to={`/companies/${companyId}/softwares/${s.id}`}
                className="text-sm text-[var(--empops-accent)] hover:underline"
              >
                Open
              </Link>
            </li>
          ))}
        </ul>
        {!listQuery.isLoading && (listQuery.data ?? []).length === 0 && (
          <p className="text-sm text-black/50">No software licenses yet.</p>
        )}
      </section>
    </div>
  )
}
