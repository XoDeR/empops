import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { authFetch } from '@/lib/authFetch'
import type { Wiki, WikiPage, WikiPageRevision } from '@/types/api'

export default function WikiDetailPage() {
  const { companyId, wikiId } = useParams<{ companyId: string; wikiId: string }>()
  const qc = useQueryClient()
  const [selectedId, setSelectedId] = useState('')
  const [newTitle, setNewTitle] = useState('')
  const [title, setTitle] = useState('')
  const [content, setContent] = useState('')
  const wiki = useQuery({
    queryKey: ['wiki', companyId, wikiId],
    queryFn: async () => (await authFetch<Wiki>(`/companies/${companyId}/wikis/${wikiId}`)).data,
    enabled: Boolean(companyId && wikiId),
  })
  const page = useQuery({
    queryKey: ['wiki-page', companyId, wikiId, selectedId],
    queryFn: async () => (await authFetch<WikiPage>(`/companies/${companyId}/wikis/${wikiId}/pages/${selectedId}`)).data,
    enabled: Boolean(selectedId),
  })
  const revisions = useQuery({
    queryKey: ['wiki-revisions', companyId, wikiId, selectedId],
    queryFn: async () => (await authFetch<WikiPageRevision[]>(`/companies/${companyId}/wikis/${wikiId}/pages/${selectedId}/revisions`)).data,
    enabled: Boolean(selectedId),
  })
  useEffect(() => {
    if (page.data) {
      setTitle(page.data.title)
      setContent(page.data.content ?? '')
    }
  }, [page.data])
  const create = useMutation({
    mutationFn: () => authFetch(`/companies/${companyId}/wikis/${wikiId}/pages`, { method: 'POST', body: JSON.stringify({ title: newTitle, content: '' }) }),
    onSuccess: () => {
      setNewTitle('')
      void qc.invalidateQueries({ queryKey: ['wiki', companyId, wikiId] })
    },
  })
  const save = useMutation({
    mutationFn: () => authFetch(`/companies/${companyId}/wikis/${wikiId}/pages/${selectedId}`, { method: 'PATCH', body: JSON.stringify({ title, content }) }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['wiki-page', companyId, wikiId, selectedId] })
      void qc.invalidateQueries({ queryKey: ['wiki-revisions', companyId, wikiId, selectedId] })
      void qc.invalidateQueries({ queryKey: ['wiki', companyId, wikiId] })
    },
  })

  if (wiki.isLoading) return <p className="text-black/55">Loading wiki…</p>
  if (!wiki.data) return <p className="text-red-700">Wiki not found.</p>
  return (
    <div className="space-y-6">
      <header>
        <Link to={`/companies/${companyId}/wikis`} className="text-sm text-black/50">← Wiki</Link>
        <h2 className="mt-2 text-xl font-semibold">{wiki.data.title}</h2>
      </header>
      <div className="grid gap-5 md:grid-cols-[15rem_1fr]">
        <aside className="space-y-3 rounded-2xl border border-black/10 bg-white/70 p-4">
          <h3 className="font-medium">Pages</h3>
          <ul className="space-y-1">
            {(wiki.data.pages ?? []).map((item) => (
              <li key={item.id}><button type="button" onClick={() => setSelectedId(item.id)} className={`w-full rounded-lg px-2 py-1 text-left text-sm ${selectedId === item.id ? 'bg-[var(--empops-accent)]/10 font-medium' : 'hover:bg-black/[0.03]'}`}>{item.title}</button></li>
            ))}
          </ul>
          <form className="space-y-2 border-t border-black/10 pt-3" onSubmit={(e) => { e.preventDefault(); if (newTitle.trim()) create.mutate() }}>
            <input className="w-full rounded-lg border border-black/15 px-2 py-1.5 text-sm" placeholder="Page title" value={newTitle} onChange={(e) => setNewTitle(e.target.value)} />
            <button className="text-sm text-[var(--empops-accent)]">Create page</button>
          </form>
        </aside>
        <main className="space-y-4">
          {!selectedId && <div className="rounded-2xl border border-black/10 bg-white/70 p-5 text-sm text-black/55">Select a page to read or edit.</div>}
          {selectedId && (
            <>
              <form className="space-y-3 rounded-2xl border border-black/10 bg-white/70 p-4" onSubmit={(e) => { e.preventDefault(); save.mutate() }}>
                <input className="w-full rounded-lg border border-black/15 px-3 py-2 font-medium" value={title} onChange={(e) => setTitle(e.target.value)} />
                <textarea className="min-h-72 w-full rounded-lg border border-black/15 px-3 py-2 font-mono text-sm" aria-label="Markdown content" value={content} onChange={(e) => setContent(e.target.value)} />
                <button disabled={save.isPending} className="rounded-lg bg-[var(--empops-accent)] px-4 py-2 text-sm text-white disabled:opacity-60">{save.isPending ? 'Saving…' : 'Save revision'}</button>
              </form>
              <section className="rounded-2xl border border-black/10 bg-white/70 p-4">
                <h3 className="font-medium">Revisions</h3>
                <ul className="mt-2 space-y-2 text-sm">
                  {(revisions.data ?? []).map((r) => <li key={r.id} className="rounded-lg bg-black/[0.03] p-2"><span className="font-medium">{r.title}</span><span className="text-black/50"> · {r.employee_name}{r.created_at ? ` · ${new Date(r.created_at).toLocaleString()}` : ''}</span></li>)}
                </ul>
                {!revisions.isLoading && !revisions.data?.length && <p className="mt-2 text-sm text-black/50">No revisions yet.</p>}
              </section>
            </>
          )}
        </main>
      </div>
    </div>
  )
}
