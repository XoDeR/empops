import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { authFetch } from '@/lib/authFetch'
import type { AppNotification, NotificationList } from '@/types/api'

function formatAction(n: AppNotification): string {
  if (n.action === 'employee_attached_to_recent_ship') {
    const title = String(n.objects.ship_title ?? 'a ship')
    const author = String(n.objects.author_name ?? 'Someone')
    return `${author} attached you to “${title}”`
  }
  return n.action.replaceAll('_', ' ')
}

export function NotificationBell({ companyId }: { companyId: string }) {
  const qc = useQueryClient()
  const [open, setOpen] = useState(false)

  const listQuery = useQuery({
    queryKey: ['notifications', companyId],
    queryFn: async () => {
      const res = await authFetch<NotificationList>(`/companies/${companyId}/notifications`)
      return res.data
    },
    enabled: Boolean(companyId),
    refetchInterval: 60_000,
  })

  const markRead = useMutation({
    mutationFn: async () => {
      await authFetch(`/companies/${companyId}/notifications/read`, {
        method: 'POST',
        body: JSON.stringify({}),
      })
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['notifications', companyId] })
      void qc.invalidateQueries({ queryKey: ['dashboard', companyId, 'me'] })
    },
  })

  const unread = listQuery.data?.unread_count ?? 0
  const items = listQuery.data?.items ?? []

  return (
    <div className="relative">
      <button
        type="button"
        className="relative rounded-lg border border-black/10 px-2.5 py-1.5 text-sm hover:bg-black/[0.04]"
        onClick={() => setOpen((v) => !v)}
        aria-label="Notifications"
      >
        Alerts
        {unread > 0 && (
          <span className="ml-1 inline-flex min-w-5 items-center justify-center rounded-full bg-[var(--empops-accent)] px-1.5 text-[10px] font-semibold text-white">
            {unread}
          </span>
        )}
      </button>
      {open && (
        <div className="absolute right-0 z-20 mt-2 w-80 rounded-xl border border-black/10 bg-white p-3 shadow-lg">
          <div className="mb-2 flex items-center justify-between gap-2">
            <p className="text-sm font-medium">Notifications</p>
            <button
              type="button"
              className="text-xs text-[var(--empops-accent)] hover:underline disabled:opacity-50"
              disabled={unread === 0 || markRead.isPending}
              onClick={() => markRead.mutate()}
            >
              Mark all read
            </button>
          </div>
          <ul className="max-h-72 space-y-2 overflow-y-auto">
            {items.length === 0 && (
              <li className="text-sm text-black/50">No notifications yet.</li>
            )}
            {items.map((n) => (
              <li
                key={n.id}
                className={`rounded-lg px-2 py-1.5 text-sm ${n.read ? 'text-black/55' : 'bg-black/[0.03] text-black/80'}`}
              >
                {formatAction(n)}
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  )
}
