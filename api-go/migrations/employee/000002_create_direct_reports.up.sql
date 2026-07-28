CREATE TABLE IF NOT EXISTS direct_reports (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    manager_id UUID NOT NULL REFERENCES employees (id) ON DELETE CASCADE,
    employee_id UUID NOT NULL REFERENCES employees (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (manager_id, employee_id)
);

CREATE INDEX IF NOT EXISTS direct_reports_company_id_idx ON direct_reports (company_id);
CREATE INDEX IF NOT EXISTS direct_reports_manager_id_idx ON direct_reports (manager_id);
CREATE INDEX IF NOT EXISTS direct_reports_employee_id_idx ON direct_reports (employee_id);
