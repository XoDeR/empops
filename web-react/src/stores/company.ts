import { create } from 'zustand'
import { persist } from 'zustand/middleware'

type CompanyState = {
  selectedCompanyId: string | null
  setSelectedCompanyId: (id: string | null) => void
}

export const useCompanyStore = create<CompanyState>()(
  persist(
    (set) => ({
      selectedCompanyId: null,
      setSelectedCompanyId: (id) => set({ selectedCompanyId: id }),
    }),
    { name: 'empops-company' },
  ),
)
