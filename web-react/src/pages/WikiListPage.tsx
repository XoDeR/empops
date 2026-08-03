import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { authFetch } from '@/lib/authFetch'
import type { Wiki } from '@/types/api'

export default function WikiListPage() {
  const { companyId } = useParams<{ companyId: string }>()
  const [title, setTitle] = useState('')
  const qc = useQueryClient()
  const query = useQuery({
    queryKey: ['wikis', companyId],
    queryFn: async () => (await authFetch<Wiki[]>(`/companies/${companyId}/wikis`)).data,
    enabled: Boolean(companyId),
  })
  const create = useMutation({
    mutationFn: () =>
      authFetch<Wiki>(`/companies/${companyId}/wikis`, {
        method: 'POST',
        body: JSON.stringify({ title }),
      }),
    onSuccess: () => {
      setTitle('')
      void qc.invalidateQueries({ queryKey: ['wikis', companyId] })
    },
  })

  return (
    <div className="space-y-6">
      <header>
        <h2 className="text-xl font-semibold">Wiki</h2>
        <p className="text-sm text-black/55">Shared company knowledge.</p>
      </header>
      <form
        className="flex gap-2 rounded-2xl border border-black/10 bg-white/70 p-4"
        onSubmit={(e) => {
          e.preventDefault()
          if (title.trim()) create.mutate()
        }}
      >
        <input className="flex-1 rounded-lg border border-black/15 px-3 py-2 text-sm" placeholder="New wiki title" value={title} onChange={(e) => setTitle(e.target.value)} />
        <button disabled={create.isPending} className="rounded-lg bg-[var(--empops-accent)] px-4 py-2 text-sm text-white disabled:opacity-60">Create</button>
      </form>
      {query.isError && <p className="text-sm text-red-700">{(query.error as Error).message}</p>}
      <section className="rounded-2xl border border-black/10 bg-white/70 p-4">
        {query.isLoading && <p className="text-sm text-black/55">Loading…</p>}
        <ul className="divide-y divide-black/5">
          {(query.data ?? []).map((wiki) => (
            <li key={wiki.id} className="flex items-center justify-between py-3">
              <div>
                <Link className="font-medium hover:underline" to={`/companies/${companyId}/wikis/${wiki.id}`}>{wiki.title}</Link>
                <p className="text-xs text-black/50">{wiki.pages?.length ?? 0} pages</p>
              </div>
              <Link className="text-sm text-[var(--empops-accent)]" to={`/companies/${companyId}/wikis/${wiki.id}`}>Open</Link>
            </li>
          ))}
        </ul>
        {!query.isLoading && !query.data?.length && <p className="text-sm text-black/50">No wikis yet.</p>}
      </section>
    </div>
  )
}
