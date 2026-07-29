import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { authFetch } from '@/lib/authFetch'
import type { CompanyNews, Question } from '@/types/api'

export function CompanyNewsSection({ companyId }: { companyId: string }) {
  const qc = useQueryClient()
  const [title, setTitle] = useState('')
  const [content, setContent] = useState('')
  const [error, setError] = useState<string | null>(null)

  const listQuery = useQuery({
    queryKey: ['company-news', companyId],
    queryFn: async () => {
      const res = await authFetch<CompanyNews[]>(`/companies/${companyId}/news`)
      return res.data
    },
  })

  const create = useMutation({
    mutationFn: async () => {
      await authFetch(`/companies/${companyId}/news`, {
        method: 'POST',
        body: JSON.stringify({ title, content }),
      })
    },
    onSuccess: () => {
      setTitle('')
      setContent('')
      setError(null)
      void qc.invalidateQueries({ queryKey: ['company-news', companyId] })
    },
    onError: (e: Error) => setError(e.message),
  })

  const destroy = useMutation({
    mutationFn: async (newsId: string) => {
      await authFetch(`/companies/${companyId}/news/${newsId}`, { method: 'DELETE' })
    },
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['company-news', companyId] }),
    onError: (e: Error) => setError(e.message),
  })

  return (
    <section className="space-y-3 rounded-2xl border border-black/10 bg-white/70 p-4">
      <h3 className="font-medium">Company news</h3>
      {error && <p className="text-sm text-red-700">{error}</p>}
      <ul className="space-y-2">
        {(listQuery.data ?? []).map((n) => (
          <li key={n.id} className="flex items-start justify-between gap-2 rounded-lg border border-black/5 p-3 text-sm">
            <div>
              <p className="font-medium">{n.title}</p>
              <p className="mt-1 whitespace-pre-wrap text-black/70">{n.content}</p>
              <p className="mt-1 text-xs text-black/45">by {n.author_name}</p>
            </div>
            <button
              type="button"
              className="text-xs text-red-700 hover:underline"
              onClick={() => destroy.mutate(n.id)}
            >
              Delete
            </button>
          </li>
        ))}
        {(listQuery.data ?? []).length === 0 && (
          <li className="text-sm text-black/50">No company news yet.</li>
        )}
      </ul>
      <div className="space-y-2 border-t border-black/5 pt-3">
        <input
          className="w-full rounded-lg border border-black/15 bg-white px-3 py-2 text-sm"
          placeholder="Title"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
        />
        <textarea
          className="w-full rounded-lg border border-black/15 bg-white px-3 py-2 text-sm"
          rows={2}
          placeholder="Announcement…"
          value={content}
          onChange={(e) => setContent(e.target.value)}
        />
        <button
          type="button"
          disabled={!title.trim() || !content.trim() || create.isPending}
          className="rounded-lg bg-[var(--empops-accent)] px-3 py-1.5 text-sm text-white disabled:opacity-60"
          onClick={() => create.mutate()}
        >
          Publish
        </button>
      </div>
    </section>
  )
}

export function QuestionsSection({ companyId }: { companyId: string }) {
  const qc = useQueryClient()
  const [title, setTitle] = useState('')
  const [error, setError] = useState<string | null>(null)

  const listQuery = useQuery({
    queryKey: ['questions', companyId],
    queryFn: async () => {
      const res = await authFetch<Question[]>(`/companies/${companyId}/questions`)
      return res.data
    },
  })

  const create = useMutation({
    mutationFn: async () => {
      await authFetch(`/companies/${companyId}/questions`, {
        method: 'POST',
        body: JSON.stringify({ title }),
      })
    },
    onSuccess: () => {
      setTitle('')
      setError(null)
      void qc.invalidateQueries({ queryKey: ['questions', companyId] })
    },
    onError: (e: Error) => setError(e.message),
  })

  const activate = useMutation({
    mutationFn: async (questionId: string) => {
      await authFetch(`/companies/${companyId}/questions/${questionId}/activate`, {
        method: 'PUT',
      })
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['questions', companyId] })
      void qc.invalidateQueries({ queryKey: ['dashboard', companyId, 'me'] })
    },
    onError: (e: Error) => setError(e.message),
  })

  const deactivate = useMutation({
    mutationFn: async (questionId: string) => {
      await authFetch(`/companies/${companyId}/questions/${questionId}/deactivate`, {
        method: 'PUT',
      })
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['questions', companyId] })
      void qc.invalidateQueries({ queryKey: ['dashboard', companyId, 'me'] })
    },
    onError: (e: Error) => setError(e.message),
  })

  const destroy = useMutation({
    mutationFn: async (questionId: string) => {
      await authFetch(`/companies/${companyId}/questions/${questionId}`, { method: 'DELETE' })
    },
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['questions', companyId] }),
    onError: (e: Error) => setError(e.message),
  })

  return (
    <section className="space-y-3 rounded-2xl border border-black/10 bg-white/70 p-4">
      <h3 className="font-medium">Q&A questions</h3>
      {error && <p className="text-sm text-red-700">{error}</p>}
      <ul className="space-y-2">
        {(listQuery.data ?? []).map((q) => (
          <li key={q.id} className="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-black/5 p-3 text-sm">
            <div>
              <p className="font-medium">
                {q.title}
                {q.active && (
                  <span className="ml-2 text-xs font-normal text-emerald-700">Active</span>
                )}
              </p>
              <p className="text-xs text-black/45">{q.answer_count ?? 0} answers</p>
            </div>
            <div className="flex gap-2">
              {q.active ? (
                <button
                  type="button"
                  className="text-xs text-black/60 hover:underline"
                  onClick={() => deactivate.mutate(q.id)}
                >
                  Deactivate
                </button>
              ) : (
                <button
                  type="button"
                  className="text-xs text-[var(--empops-accent)] hover:underline"
                  onClick={() => activate.mutate(q.id)}
                >
                  Activate
                </button>
              )}
              <button
                type="button"
                className="text-xs text-red-700 hover:underline"
                onClick={() => destroy.mutate(q.id)}
              >
                Delete
              </button>
            </div>
          </li>
        ))}
        {(listQuery.data ?? []).length === 0 && (
          <li className="text-sm text-black/50">No questions yet.</li>
        )}
      </ul>
      <div className="flex flex-wrap gap-2 border-t border-black/5 pt-3">
        <input
          className="min-w-[16rem] flex-1 rounded-lg border border-black/15 bg-white px-3 py-2 text-sm"
          placeholder="New question…"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
        />
        <button
          type="button"
          disabled={!title.trim() || create.isPending}
          className="rounded-lg bg-[var(--empops-accent)] px-3 py-1.5 text-sm text-white disabled:opacity-60"
          onClick={() => create.mutate()}
        >
          Add
        </button>
      </div>
    </section>
  )
}
