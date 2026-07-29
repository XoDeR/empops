import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { authFetch } from '@/lib/authFetch'
import type { Country, Place } from '@/types/api'

type PlaceFormValues = {
  street: string
  city: string
  province: string
  postal_code: string
  country_id: string
  latitude: string
  longitude: string
}

const emptyForm: PlaceFormValues = {
  street: '',
  city: '',
  province: '',
  postal_code: '',
  country_id: '',
  latitude: '',
  longitude: '',
}

type PlacesSectionProps = {
  companyId: string
  employeeId: string
  canEdit: boolean
}

export function PlacesSection({ companyId, employeeId, canEdit }: PlacesSectionProps) {
  const qc = useQueryClient()
  const [showForm, setShowForm] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [form, setForm] = useState<PlaceFormValues>(emptyForm)
  const [error, setError] = useState<string | null>(null)

  const placesQuery = useQuery({
    queryKey: ['places', companyId, employeeId],
    queryFn: async () => {
      const res = await authFetch<Place[]>(
        `/companies/${companyId}/employees/${employeeId}/places`,
      )
      return res.data
    },
    enabled: Boolean(companyId && employeeId),
  })

  const countriesQuery = useQuery({
    queryKey: ['countries'],
    queryFn: async () => {
      const res = await authFetch<Country[]>('/countries')
      return res.data
    },
    enabled: showForm || Boolean(editingId),
  })

  const invalidate = () => {
    void qc.invalidateQueries({ queryKey: ['places', companyId, employeeId] })
  }

  const toPayload = (values: PlaceFormValues) => {
    const payload: Record<string, unknown> = {
      street: values.street || null,
      city: values.city || null,
      province: values.province || null,
      postal_code: values.postal_code || null,
      country_id: values.country_id || null,
    }
    if (values.latitude) payload.latitude = Number(values.latitude)
    if (values.longitude) payload.longitude = Number(values.longitude)
    return payload
  }

  const createPlace = useMutation({
    mutationFn: async (values: PlaceFormValues) => {
      await authFetch(`/companies/${companyId}/employees/${employeeId}/places`, {
        method: 'POST',
        body: JSON.stringify(toPayload(values)),
      })
    },
    onSuccess: () => {
      setError(null)
      setShowForm(false)
      setForm(emptyForm)
      invalidate()
    },
    onError: (e: Error) => setError(e.message),
  })

  const updatePlace = useMutation({
    mutationFn: async ({ placeId, values }: { placeId: string; values: PlaceFormValues }) => {
      await authFetch(`/companies/${companyId}/places/${placeId}`, {
        method: 'PATCH',
        body: JSON.stringify(toPayload(values)),
      })
    },
    onSuccess: () => {
      setError(null)
      setEditingId(null)
      setForm(emptyForm)
      invalidate()
    },
    onError: (e: Error) => setError(e.message),
  })

  const activatePlace = useMutation({
    mutationFn: async (placeId: string) => {
      await authFetch(`/companies/${companyId}/places/${placeId}/activate`, { method: 'PUT' })
    },
    onSuccess: invalidate,
    onError: (e: Error) => setError(e.message),
  })

  const deletePlace = useMutation({
    mutationFn: async (placeId: string) => {
      await authFetch(`/companies/${companyId}/places/${placeId}`, { method: 'DELETE' })
    },
    onSuccess: invalidate,
    onError: (e: Error) => setError(e.message),
  })

  const startEdit = (place: Place) => {
    setEditingId(place.id)
    setShowForm(false)
    setForm({
      street: place.street ?? '',
      city: place.city ?? '',
      province: place.province ?? '',
      postal_code: place.postal_code ?? '',
      country_id: place.country?.id ?? '',
      latitude: place.latitude != null ? String(place.latitude) : '',
      longitude: place.longitude != null ? String(place.longitude) : '',
    })
  }

  const field = (key: keyof PlaceFormValues, label: string, type = 'text') => (
    <label key={key} className="block space-y-1">
      <span className="text-xs text-black/55">{label}</span>
      <input
        type={type}
        className="w-full rounded-lg border border-black/15 bg-white px-3 py-1.5 text-sm"
        value={form[key]}
        onChange={(e) => setForm((f) => ({ ...f, [key]: e.target.value }))}
      />
    </label>
  )

  const formEl = (
    <form
      className="mt-3 grid gap-2 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault()
        if (editingId) {
          updatePlace.mutate({ placeId: editingId, values: form })
        } else {
          createPlace.mutate(form)
        }
      }}
    >
      {field('street', 'Street')}
      {field('city', 'City')}
      {field('province', 'Province / state')}
      {field('postal_code', 'Postal code')}
      <label className="block space-y-1 sm:col-span-2">
        <span className="text-xs text-black/55">Country</span>
        <select
          className="w-full rounded-lg border border-black/15 bg-white px-3 py-1.5 text-sm"
          value={form.country_id}
          onChange={(e) => setForm((f) => ({ ...f, country_id: e.target.value }))}
        >
          <option value="">—</option>
          {(countriesQuery.data ?? []).map((c) => (
            <option key={c.id} value={c.id}>
              {c.name}
            </option>
          ))}
        </select>
      </label>
      {field('latitude', 'Latitude', 'number')}
      {field('longitude', 'Longitude', 'number')}
      <div className="flex flex-wrap gap-2 sm:col-span-2">
        <button
          type="submit"
          className="rounded-lg bg-[var(--empops-accent)] px-3 py-1.5 text-sm text-white disabled:opacity-60"
          disabled={createPlace.isPending || updatePlace.isPending}
        >
          {editingId ? 'Save place' : 'Add place'}
        </button>
        <button
          type="button"
          className="rounded-lg border border-black/15 px-3 py-1.5 text-sm"
          onClick={() => {
            setShowForm(false)
            setEditingId(null)
            setForm(emptyForm)
          }}
        >
          Cancel
        </button>
      </div>
    </form>
  )

  return (
    <section className="rounded-2xl border border-black/10 bg-white/70 p-4">
      <div className="flex items-center justify-between gap-3">
        <h3 className="font-medium">Addresses</h3>
        {canEdit && !showForm && !editingId && (
          <button
            type="button"
            className="text-sm text-[var(--empops-accent)] hover:underline"
            onClick={() => setShowForm(true)}
          >
            Add address
          </button>
        )}
      </div>

      {error && <p className="mt-2 text-sm text-red-700">{error}</p>}

      {placesQuery.isLoading && <p className="mt-2 text-sm text-black/50">Loading…</p>}

      <ul className="mt-3 space-y-3 text-sm">
        {(placesQuery.data ?? []).map((place) => (
          <li
            key={place.id}
            className={`rounded-xl border p-3 ${
              place.is_active ? 'border-[var(--empops-accent)]/40 bg-[var(--empops-accent)]/5' : 'border-black/10'
            }`}
          >
            <div className="flex flex-wrap items-start justify-between gap-2">
              <div>
                {place.is_active && (
                  <span className="mb-1 inline-block rounded bg-[var(--empops-accent)]/15 px-1.5 py-0.5 text-xs font-medium text-[var(--empops-accent)]">
                    Active
                  </span>
                )}
                <p>{[place.street, place.city, place.province, place.postal_code].filter(Boolean).join(', ') || '—'}</p>
                {place.country && (
                  <p className="text-black/55">{place.country.name}</p>
                )}
                {(place.latitude != null || place.longitude != null) && (
                  <p className="text-xs text-black/45">
                    {place.latitude ?? '—'}, {place.longitude ?? '—'}
                  </p>
                )}
              </div>
              {canEdit && (
                <div className="flex flex-wrap gap-2 text-xs">
                  {!place.is_active && (
                    <button
                      type="button"
                      className="text-[var(--empops-accent)] hover:underline"
                      onClick={() => activatePlace.mutate(place.id)}
                    >
                      Set active
                    </button>
                  )}
                  <button
                    type="button"
                    className="hover:underline"
                    onClick={() => startEdit(place)}
                  >
                    Edit
                  </button>
                  <button
                    type="button"
                    className="text-red-700 hover:underline"
                    onClick={() => deletePlace.mutate(place.id)}
                  >
                    Delete
                  </button>
                </div>
              )}
            </div>
          </li>
        ))}
        {(placesQuery.data ?? []).length === 0 && !placesQuery.isLoading && (
          <li className="text-black/50">No addresses yet.</li>
        )}
      </ul>

      {canEdit && showForm && formEl}
      {canEdit && editingId && formEl}
    </section>
  )
}
