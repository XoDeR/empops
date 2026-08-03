CREATE TABLE IF NOT EXISTS company_pto_policies (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    year SMALLINT NOT NULL,
    total_worked_days SMALLINT NOT NULL DEFAULT 0,
    default_amount_of_allowed_holidays NUMERIC(8, 2) NOT NULL DEFAULT 0,
    default_amount_of_sick_days NUMERIC(8, 2) NOT NULL DEFAULT 0,
    default_amount_of_pto_days NUMERIC(8, 2) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (company_id, year)
);

CREATE TABLE IF NOT EXISTS company_calendars (
    id UUID PRIMARY KEY,
    company_pto_policy_id UUID NOT NULL REFERENCES company_pto_policies (id) ON DELETE CASCADE,
    day DATE NOT NULL,
    day_of_week SMALLINT NOT NULL,
    day_of_year SMALLINT NOT NULL,
    is_worked BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (company_pto_policy_id, day)
);

ALTER TABLE employees
    ADD COLUMN IF NOT EXISTS holiday_balance NUMERIC(10, 4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS amount_of_allowed_holidays NUMERIC(8, 2),
    ADD COLUMN IF NOT EXISTS amount_of_sick_days NUMERIC(8, 2),
    ADD COLUMN IF NOT EXISTS amount_of_pto_days NUMERIC(8, 2);

CREATE TABLE IF NOT EXISTS employee_planned_holidays (
    id UUID PRIMARY KEY,
    employee_id UUID NOT NULL REFERENCES employees (id) ON DELETE CASCADE,
    planned_date DATE NOT NULL,
    type VARCHAR(32) NOT NULL,
    full BOOLEAN NOT NULL DEFAULT true,
    actually_taken BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (employee_id, planned_date, type)
);
CREATE INDEX IF NOT EXISTS idx_planned_holidays_date ON employee_planned_holidays (planned_date);

CREATE TABLE IF NOT EXISTS employee_daily_calendar_entries (
    id UUID PRIMARY KEY,
    employee_id UUID NOT NULL REFERENCES employees (id) ON DELETE CASCADE,
    log_date DATE NOT NULL,
    new_balance NUMERIC(10, 4) NOT NULL,
    daily_accrued_amount NUMERIC(10, 6) NOT NULL,
    current_holidays_per_year NUMERIC(8, 2),
    default_amount_of_allowed_holidays_in_company NUMERIC(8, 2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (employee_id, log_date)
);

CREATE TABLE IF NOT EXISTS flows (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_flows_company_type ON flows (company_id, type);

CREATE TABLE IF NOT EXISTS flow_steps (
    id UUID PRIMARY KEY,
    flow_id UUID NOT NULL REFERENCES flows (id) ON DELETE CASCADE,
    number INTEGER NOT NULL DEFAULT 0,
    unit_of_time VARCHAR(16) NOT NULL DEFAULT 'days',
    modifier VARCHAR(16) NOT NULL,
    real_number_of_days INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_flow_steps_flow ON flow_steps (flow_id, real_number_of_days);

CREATE TABLE IF NOT EXISTS flow_actions (
    id UUID PRIMARY KEY,
    step_id UUID NOT NULL REFERENCES flow_steps (id) ON DELETE CASCADE,
    type VARCHAR(64) NOT NULL,
    recipient VARCHAR(64) NOT NULL,
    specific_recipient_information TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS flow_action_runs (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    flow_action_id UUID NOT NULL REFERENCES flow_actions (id) ON DELETE CASCADE,
    employee_id UUID NOT NULL REFERENCES employees (id) ON DELETE CASCADE,
    due_on DATE NOT NULL,
    executed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (flow_action_id, employee_id, due_on)
);
CREATE INDEX IF NOT EXISTS idx_flow_action_runs_due ON flow_action_runs (company_id, due_on, executed_at);

CREATE TABLE IF NOT EXISTS wikis (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_wikis_company ON wikis (company_id);

CREATE TABLE IF NOT EXISTS wiki_pages (
    id UUID PRIMARY KEY,
    wiki_id UUID NOT NULL REFERENCES wikis (id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    content TEXT,
    pageviews_counter INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_wiki_pages_wiki ON wiki_pages (wiki_id);

CREATE TABLE IF NOT EXISTS wiki_page_revisions (
    id UUID PRIMARY KEY,
    page_id UUID NOT NULL REFERENCES wiki_pages (id) ON DELETE CASCADE,
    employee_id UUID REFERENCES employees (id) ON DELETE SET NULL,
    employee_name VARCHAR(255) NOT NULL,
    title VARCHAR(255) NOT NULL,
    content TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_wiki_page_revisions_page ON wiki_page_revisions (page_id);

CREATE TABLE IF NOT EXISTS ask_me_anything_sessions (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    happened_at DATE NOT NULL,
    active BOOLEAN NOT NULL DEFAULT false,
    theme VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_ama_sessions_company ON ask_me_anything_sessions (company_id, active);

CREATE TABLE IF NOT EXISTS ask_me_anything_questions (
    id UUID PRIMARY KEY,
    ask_me_anything_session_id UUID NOT NULL REFERENCES ask_me_anything_sessions (id) ON DELETE CASCADE,
    employee_id UUID REFERENCES employees (id) ON DELETE SET NULL,
    question TEXT NOT NULL,
    answered BOOLEAN NOT NULL DEFAULT false,
    anonymous BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_ama_questions_session ON ask_me_anything_questions (ask_me_anything_session_id);

CREATE TABLE IF NOT EXISTS groups (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    mission TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_groups_company ON groups (company_id);

CREATE TABLE IF NOT EXISTS employee_group (
    employee_id UUID NOT NULL REFERENCES employees (id) ON DELETE CASCADE,
    group_id UUID NOT NULL REFERENCES groups (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (employee_id, group_id)
);

CREATE TABLE IF NOT EXISTS meetings (
    id UUID PRIMARY KEY,
    group_id UUID NOT NULL REFERENCES groups (id) ON DELETE CASCADE,
    happened BOOLEAN NOT NULL DEFAULT false,
    happened_at DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_meetings_group ON meetings (group_id);

CREATE TABLE IF NOT EXISTS employee_meeting (
    employee_id UUID NOT NULL REFERENCES employees (id) ON DELETE CASCADE,
    meeting_id UUID NOT NULL REFERENCES meetings (id) ON DELETE CASCADE,
    was_a_guest BOOLEAN NOT NULL DEFAULT false,
    attended BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (employee_id, meeting_id)
);

CREATE TABLE IF NOT EXISTS agenda_items (
    id UUID PRIMARY KEY,
    meeting_id UUID NOT NULL REFERENCES meetings (id) ON DELETE CASCADE,
    position INTEGER NOT NULL DEFAULT 0,
    checked BOOLEAN NOT NULL DEFAULT false,
    summary VARCHAR(255) NOT NULL,
    description TEXT,
    presented_by_id UUID REFERENCES employees (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_agenda_items_meeting ON agenda_items (meeting_id, position);

CREATE TABLE IF NOT EXISTS meeting_decisions (
    id UUID PRIMARY KEY,
    agenda_item_id UUID NOT NULL REFERENCES agenda_items (id) ON DELETE CASCADE,
    description TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_meeting_decisions_item ON meeting_decisions (agenda_item_id);

CREATE TABLE IF NOT EXISTS company_daily_usage_history (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    number_of_active_employees INTEGER NOT NULL DEFAULT 0,
    logged_on DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (company_id, logged_on)
);

CREATE TABLE IF NOT EXISTS company_usage_history_details (
    id UUID PRIMARY KEY,
    usage_history_id UUID NOT NULL REFERENCES company_daily_usage_history (id) ON DELETE CASCADE,
    employee_id UUID REFERENCES employees (id) ON DELETE SET NULL,
    employee_name VARCHAR(255),
    employee_email VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_usage_details_history ON company_usage_history_details (usage_history_id);

CREATE TABLE IF NOT EXISTS company_invoices (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    usage_history_id UUID REFERENCES company_daily_usage_history (id) ON DELETE SET NULL,
    sent_to_customer BOOLEAN NOT NULL DEFAULT false,
    customer_has_paid BOOLEAN NOT NULL DEFAULT false,
    email_address_invoice_sent_to VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_company_invoices_company ON company_invoices (company_id);
