import { Link } from 'react-router-dom'

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
