ALTER TABLE employees
    ADD COLUMN IF NOT EXISTS consecutive_worklog_missed INTEGER NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS worklogs (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    employee_id UUID NOT NULL REFERENCES employees (id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    logged_on DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (employee_id, logged_on)
);

CREATE INDEX IF NOT EXISTS worklogs_company_id_idx ON worklogs (company_id);
CREATE INDEX IF NOT EXISTS worklogs_employee_id_logged_on_idx ON worklogs (employee_id, logged_on DESC);
