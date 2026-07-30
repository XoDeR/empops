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
  work_from_home_enabled?: boolean
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

export type TimesheetStatus = 'open' | 'ready_to_submit' | 'approved' | 'rejected'

export type TimesheetEntry = {
  id: string
  timesheet_id: string
  employee_id: string
  duration: number
  happened_at: string
  description: string | null
}

export type Timesheet = {
  id: string
  company_id: string
  employee_id: string
  employee?: EmployeeSummary | null
  started_at: string
  ended_at: string
  status: TimesheetStatus
  approved_at: string | null
  approver_id: string | null
  approver_name?: string | null
  entries: TimesheetEntry[]
  total_duration: number
}

export type ExpenseStatus =
  | 'created'
  | 'manager_approval'
  | 'accounting_approval'
  | 'rejected_by_manager'
  | 'rejected_by_accounting'
  | 'accepted'

export type ExpenseCategory = {
  id: string
  company_id: string
  name: string
}

export type Expense = {
  id: string
  company_id: string
  employee_id: string | null
  employee_name: string | null
  expense_category_id: string | null
  category: ExpenseCategory | null
  status: ExpenseStatus
  title: string
  amount: number
  currency: string
  converted_amount: number | null
  converted_to_currency: string | null
  converted_at: string | null
  exchange_rate: number | null
  description: string | null
  expensed_at: string
  manager_approver_id: string | null
  manager_approver_name: string | null
  manager_approver_approved_at: string | null
  manager_rejection_explanation: string | null
  accounting_approver_id: string | null
  accounting_approver_name: string | null
  accounting_approver_approved_at: string | null
  accounting_rejection_explanation: string | null
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
  | {
      type: 'timesheet_current_week'
      data: Timesheet | null
    }
  | {
      type: 'wfh_today'
      data: { work_from_home: boolean }
    }
  | {
      type: 'pending_timesheets'
      data: { count: number }
    }
  | {
      type: 'pending_expenses'
      data: { count: number }
    }
  | {
      type: 'pending_accounting_expenses'
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
  view: 'me' | 'team' | 'manager' | 'hr' | 'accountant'
  widgets: DashboardWidget[]
  flags: {
    is_manager: boolean
    can_manage_hr: boolean
    is_admin: boolean
    is_accountant?: boolean
  }
}
