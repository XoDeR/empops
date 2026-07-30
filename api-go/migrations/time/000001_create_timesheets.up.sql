CREATE TABLE IF NOT EXISTS timesheets (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    employee_id UUID NOT NULL REFERENCES employees (id) ON DELETE CASCADE,
    started_at DATE NOT NULL,
    ended_at DATE NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'open',
    approver_id UUID REFERENCES employees (id) ON DELETE SET NULL,
    approved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (employee_id, started_at)
);
CREATE INDEX IF NOT EXISTS idx_timesheets_company_status_week
    ON timesheets (company_id, status, started_at);

CREATE TABLE IF NOT EXISTS time_tracking_entries (
    id UUID PRIMARY KEY,
    timesheet_id UUID NOT NULL REFERENCES timesheets (id) ON DELETE CASCADE,
    employee_id UUID NOT NULL REFERENCES employees (id) ON DELETE CASCADE,
    duration SMALLINT NOT NULL CHECK (duration BETWEEN 1 AND 1440),
    happened_at DATE NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT time_entry_day_unique UNIQUE (timesheet_id, employee_id, happened_at)
);

CREATE TABLE IF NOT EXISTS employee_work_from_home (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    employee_id UUID NOT NULL REFERENCES employees (id) ON DELETE CASCADE,
    date DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (employee_id, date)
);
CREATE INDEX IF NOT EXISTS idx_employee_wfh_company_date
    ON employee_work_from_home (company_id, date);
