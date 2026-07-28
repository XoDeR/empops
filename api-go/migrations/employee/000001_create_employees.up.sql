CREATE TABLE IF NOT EXISTS positions (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (company_id, title)
);

CREATE TABLE IF NOT EXISTS employee_statuses (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(32) NOT NULL DEFAULT 'internal',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (company_id, name)
);

CREATE TABLE IF NOT EXISTS employees (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    user_id UUID REFERENCES users (id) ON DELETE SET NULL,
    email VARCHAR(255) NOT NULL,
    first_name VARCHAR(255) NOT NULL,
    last_name VARCHAR(255) NOT NULL,
    hired_at DATE,
    position_id UUID REFERENCES positions (id) ON DELETE SET NULL,
    employee_status_id UUID REFERENCES employee_statuses (id) ON DELETE SET NULL,
    invitation_link UUID UNIQUE,
    invitation_used_at TIMESTAMPTZ,
    locked BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (company_id, email),
    UNIQUE (company_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_employees_user_id ON employees (user_id);

CREATE TABLE IF NOT EXISTS employee_roles (
    employee_id UUID NOT NULL REFERENCES employees (id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
    PRIMARY KEY (employee_id, role_id)
);
