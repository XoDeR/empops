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
  project_id?: string | null
  project_task_id?: string | null
  project_name?: string | null
  project_task_title?: string | null
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
  | {
      type: 'morale_today'
      data: {
        logged: boolean
        morale: Morale | null
      }
    }
  | {
      type: 'one_on_one_current'
      data: { entries: OneOnOneEntry[] }
    }
  | {
      type: 'rate_manager_pending'
      data: { answers: RateYourManagerAnswer[] }
    }
  | {
      type: 'e_coffee_current'
      data: { match: ECoffeeMatch | null }
    }
  | {
      type: 'one_on_ones_open'
      data: { count: number; entries: OneOnOneEntry[] }
    }
  | {
      type: 'discipline_active'
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

export type ProjectStatus =
  | 'created'
  | 'started'
  | 'paused'
  | 'cancelled'
  | 'closed'

export type TeamSummary = {
  id: string
  name: string
}

export type ProjectSummary = {
  id: string
  name: string
  code?: string | null
  short_code?: string | null
  status: string
  emoji?: string | null
}

export type ProjectTaskSummary = {
  id: string
  title: string
  completed: boolean
  project_task_list_id?: string | null
}

export type Project = {
  id: string
  company_id: string
  name: string
  code: string | null
  short_code: string | null
  emoji: string | null
  summary: string | null
  description: string | null
  status: ProjectStatus
  completed: boolean
  project_lead_id: string | null
  lead: EmployeeSummary | null
  started_at: string | null
  planned_finished_at: string | null
  actually_finished_at: string | null
  members: EmployeeSummary[]
  teams: TeamSummary[]
  member_count: number
}

export type ProjectLink = {
  id: string
  project_id: string
  type: string
  label: string | null
  url: string
}

export type ProjectStatusUpdate = {
  id: string
  project_id: string
  author_id: string | null
  author_name: string | null
  title: string
  status: string
  description: string
  created_at?: string
}

export type ProjectFile = {
  id: number
  file_name: string
  mime_type: string | null
  size: number
  url: string
}

export type Comment = {
  id: string
  company_id: string
  author_id: string | null
  author_name: string
  content: string
  created_at?: string
}

export type ProjectMessage = {
  id: string
  project_id: string
  author_id: string | null
  author_name: string | null
  title: string
  content: string
  comments?: Comment[]
  created_at?: string
}

export type ProjectDecision = {
  id: string
  project_id: string
  author_id: string | null
  author_name: string | null
  title: string
  decided_at: string | null
  deciders: EmployeeSummary[]
}

export type ProjectTask = {
  id: string
  project_id: string
  project_task_list_id: string | null
  author_id: string | null
  assignee_id: string | null
  assignee?: EmployeeSummary | null
  title: string
  description: string | null
  completed: boolean
  completed_at: string | null
  comments?: Comment[]
}

export type ProjectTaskList = {
  id: string
  project_id: string
  author_id: string | null
  title: string
  description: string | null
  tasks: ProjectTask[]
}

export type IssueType = {
  id: string
  company_id: string
  name: string
  icon: string | null
}

export type ProjectIssue = {
  id: string
  project_id: string
  project_board_id: string | null
  reporter_id: string | null
  issue_type_id: string | null
  issue_type?: IssueType | null
  is_separator: boolean
  id_in_project: number
  key: string
  slug: string
  title: string
  description: string | null
  story_points: number | null
  position?: number | null
  assignees: EmployeeSummary[]
}

export type ProjectSprint = {
  id: string
  project_id: string
  project_board_id: string | null
  name: string
  active: boolean
  position?: number | null
  started_at: string | null
  completed_at: string | null
  issues?: ProjectIssue[]
}

export type ProjectBoard = {
  id: string
  project_id: string
  name: string
  sprints?: ProjectSprint[]
}

export type RecruitingStage = {
  id: string
  name: string
  position: number
}

export type RecruitingStageTemplate = {
  id: string
  company_id: string
  name: string
  stages: RecruitingStage[]
}

export type JobOpening = {
  id: string
  company_id: string
  title: string
  description: string
  slug: string
  reference_number: string | null
  position_id: string
  position?: { id: string; title: string } | null
  recruiting_stage_template_id: string | null
  team_id: string | null
  active: boolean
  fulfilled: boolean
  page_views: number
  activated_at: string | null
  fulfilled_at: string | null
  sponsors: EmployeeSummary[]
}

export type CandidateStageNote = {
  id: string
  author_id: string | null
  author_name: string
  note: string
  created_at?: string
}

export type CandidateStageParticipant = {
  id: string
  participant_id: string
  participant_name: string
  participated: boolean
}

export type CandidateStage = {
  id: string
  stage_name: string
  stage_position: number
  status: 'pending' | 'passed' | 'rejected'
  decider_id: string | null
  decider_name: string | null
  decided_at: string | null
  notes?: CandidateStageNote[]
  participants?: CandidateStageParticipant[]
}

export type CandidateFile = {
  id: number
  file_name: string
  mime_type: string | null
  size: number
  url: string
}

export type Candidate = {
  id: string
  job_opening_id: string
  name: string
  email: string
  uuid: string
  url: string | null
  desired_salary: string | null
  notes: string | null
  application_completed: boolean
  rejected: boolean
  employee_id: string | null
  employee_name: string | null
  created_at?: string
  stages?: CandidateStage[]
  files?: CandidateFile[]
}

export type PublicJobCompany = {
  slug: string
  name: string
  openings_count: number
}

export type PublicJobOpening = {
  title: string
  slug: string
  reference_number: string | null
  description?: string
  company?: { slug: string; name: string }
}

export type EmployeeImportResult = {
  created: number
  errors: { row: number; message: string }[]
}

export type Morale = {
  id: string
  employee_id: string
  emotion: 1 | 2 | 3
  comment: string | null
  created_at: string
}

export type MoraleHistoryPoint = {
  id: string
  average: number
  number_of_employees?: number
  number_of_team_members?: number
  created_at: string
}

export type OneOnOneChecklistItem = {
  id: string
  description: string
  checked: boolean
}

export type OneOnOneNote = {
  id: string
  note: string
  created_at?: string
}

export type OneOnOneEntry = {
  id: string
  company_id?: string
  manager: EmployeeSummary
  employee: EmployeeSummary
  happened: boolean
  happened_at: string | null
  talking_points: OneOnOneChecklistItem[]
  action_items: OneOnOneChecklistItem[]
  notes: OneOnOneNote[]
  created_at?: string
}

export type RateYourManagerAnswer = {
  id: string
  survey_id: string
  employee?: EmployeeSummary
  manager: EmployeeSummary | null
  active: boolean
  rating: 'bad' | 'average' | 'good' | null
  comment: string | null
  reveal_identity_to_manager: boolean
  valid_until_at: string | null
  survey_active?: boolean
}

export type Skill = {
  id: string
  name: string
  employees_count?: number
}

export type ECoffeeMatch = {
  id: string
  e_coffee_id: string
  batch_number: number
  employee: EmployeeSummary
  with_employee: EmployeeSummary
  happened: boolean
}

export type DisciplineEvent = {
  id: string
  author_name: string
  author?: EmployeeSummary | null
  happened_at: string
  description: string
  files: { id: number; file_name: string; mime_type: string; size: number; url: string }[]
  created_at?: string
}

export type DisciplineCase = {
  id: string
  employee: EmployeeSummary
  opened_by?: EmployeeSummary | null
  opened_by_employee_name: string | null
  active: boolean
  created_at: string
  events?: DisciplineEvent[]
}

export type Hardware = {
  id: string
  company_id: string
  name: string
  serial_number: string | null
  employee_id: string | null
  employee: EmployeeSummary | null
  created_at?: string
  updated_at?: string
}

export type Software = {
  id: string
  company_id: string
  name: string
  product_key: string | null
  seats: number
  seats_used?: number
  remaining_seats?: number
  website: string | null
  licensed_to_name: string | null
  licensed_to_email_address: string | null
  order_number: string | null
  purchase_amount: number | null
  currency: string | null
  converted_purchase_amount: number | null
  converted_to_currency: string | null
  converted_at: string | null
  exchange_rate: number | null
  purchased_at: string | null
  employees?: EmployeeSummary[]
  files?: ProjectFile[]
  created_at?: string
  updated_at?: string
}

