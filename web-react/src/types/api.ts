export type ApiEnvelope<T> = {
  success: boolean
  message: string
  data: T
  error: unknown
  timestamp: string
}

export type AuthUser = {
  id: string
  email: string
  name: string
}

export type TokenPayload = {
  access_token: string
  refresh_token: string
  expires_in: number
  token_type: string
  user: AuthUser
}

export type CompanyRole = 'administrator' | 'hr' | 'employee'

export type CompanyMembership = {
  id: string
  name: string
  slug: string
  currency: string
  code_to_join_company?: string
  employee_id: string
  roles: string[]
}

export type Position = {
  id: string
  company_id: string
  title: string
}

export type EmployeeStatusType = 'internal' | 'external'

export type EmployeeStatus = {
  id: string
  company_id: string
  name: string
  type: EmployeeStatusType
}

export type EmployeeSummary = {
  id: string
  first_name: string
  last_name: string
  email: string
}

export type TeamRef = {
  id: string
  name: string
}

export type Employee = {
  id: string
  company_id: string
  user_id: string | null
  email: string
  first_name: string
  last_name: string
  hired_at: string | null
  locked: boolean
  position: { id: string; title: string } | null
  status: { id: string; name: string; type: EmployeeStatusType } | null
  roles: string[]
  manager?: EmployeeSummary | null
  managers?: EmployeeSummary[]
  teams?: TeamRef[]
  is_manager?: boolean
  invitation_link?: string | null
  invitation_url?: string | null
}

export type Team = {
  id: string
  company_id: string
  name: string
  description: string | null
  leader: EmployeeSummary | null
  members: EmployeeSummary[]
  member_count: number
}

export type DashboardShell = {
  view: 'me' | 'team' | 'manager' | 'hr'
  widgets: unknown[]
  flags: {
    is_manager: boolean
    can_manage_hr: boolean
    is_admin: boolean
  }
}
