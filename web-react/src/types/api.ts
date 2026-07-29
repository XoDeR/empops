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
  logo_url?: string | null
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
  avatar_url?: string | null
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

export type Country = {
  id: string
  name: string
  code: string
}

export type Place = {
  id: string
  street: string | null
  city: string | null
  province: string | null
  postal_code: string | null
  country: { id: string; name: string; code: string } | null
  latitude: number | null
  longitude: number | null
  is_active: boolean
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

export type DashboardWidget =
  | {
      type: 'worklog_today'
      data: {
        logged: boolean
        worklog: Worklog | null
        consecutive_missed: number
      }
    }
  | {
      type: 'active_question'
      data: { id: string; title: string; answered: boolean } | null
    }
  | {
      type: 'unread_notifications'
      data: { count: number }
    }

export type Worklog = {
  id: string
  company_id: string
  employee_id: string
  content: string
  logged_on: string
  created_at?: string
  updated_at?: string
  employee?: EmployeeSummary | null
}

export type CompanyNews = {
  id: string
  company_id: string
  author_id: string | null
  author_name: string
  title: string
  content: string
  created_at?: string
  updated_at?: string
}

export type TeamNews = {
  id: string
  company_id: string
  team_id: string
  author_id: string | null
  author_name: string
  title: string
  content: string
  created_at?: string
  updated_at?: string
}

export type Ship = {
  id: string
  company_id: string
  team_id: string
  author_id: string | null
  author_name: string
  title: string
  description: string | null
  employees: EmployeeSummary[]
  created_at?: string
  updated_at?: string
}

export type AppNotification = {
  id: string
  company_id: string
  employee_id: string
  action: string
  objects: Record<string, unknown>
  read: boolean
  read_at: string | null
  created_at?: string
  updated_at?: string
}

export type NotificationList = {
  items: AppNotification[]
  unread_count: number
}

export type Answer = {
  id: string
  question_id: string
  employee_id: string
  body: string
  employee?: EmployeeSummary | null
  created_at?: string
  updated_at?: string
}

export type Question = {
  id: string
  company_id: string
  title: string
  active: boolean
  activated_at?: string | null
  deactivated_at?: string | null
  answer_count?: number
  answers?: Answer[]
  my_answer?: Answer | null
  answered?: boolean
  created_at?: string
  updated_at?: string
}

export type DashboardShell = {
  view: 'me' | 'team' | 'manager' | 'hr'
  widgets: DashboardWidget[]
  flags: {
    is_manager: boolean
    can_manage_hr: boolean
    is_admin: boolean
  }
}
