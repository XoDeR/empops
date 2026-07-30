CREATE TABLE IF NOT EXISTS expense_categories (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (company_id, name)
);

CREATE TABLE IF NOT EXISTS expenses (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    employee_id UUID REFERENCES employees (id) ON DELETE SET NULL,
    expense_category_id UUID REFERENCES expense_categories (id) ON DELETE SET NULL,
    status VARCHAR(32) NOT NULL,
    title VARCHAR(255) NOT NULL,
    amount BIGINT NOT NULL CHECK (amount > 0),
    currency VARCHAR(3) NOT NULL,
    converted_amount BIGINT,
    converted_to_currency VARCHAR(3),
    converted_at TIMESTAMPTZ,
    exchange_rate NUMERIC(18,8),
    description TEXT,
    expensed_at DATE NOT NULL,
    manager_approver_id UUID REFERENCES employees (id) ON DELETE SET NULL,
    manager_approver_approved_at TIMESTAMPTZ,
    manager_rejection_explanation TEXT,
    accounting_approver_id UUID REFERENCES employees (id) ON DELETE SET NULL,
    accounting_approver_approved_at TIMESTAMPTZ,
    accounting_rejection_explanation TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_expenses_company_status_created
    ON expenses (company_id, status, created_at);
CREATE INDEX IF NOT EXISTS idx_expenses_employee_status
    ON expenses (employee_id, status);
