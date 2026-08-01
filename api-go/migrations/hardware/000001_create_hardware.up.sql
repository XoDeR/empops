CREATE TABLE IF NOT EXISTS hardware (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    employee_id UUID REFERENCES employees (id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    serial_number VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_hardware_company_employee ON hardware (company_id, employee_id);

CREATE TABLE IF NOT EXISTS softwares (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    product_key TEXT,
    seats INTEGER NOT NULL CHECK (seats >= 1),
    website VARCHAR(255),
    licensed_to_name VARCHAR(255),
    licensed_to_email_address VARCHAR(255),
    order_number VARCHAR(255),
    purchase_amount BIGINT,
    currency VARCHAR(3),
    converted_purchase_amount BIGINT,
    converted_to_currency VARCHAR(3),
    converted_at TIMESTAMPTZ,
    exchange_rate NUMERIC(18, 8),
    purchased_at DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_softwares_company ON softwares (company_id);

CREATE TABLE IF NOT EXISTS employee_software (
    employee_id UUID NOT NULL REFERENCES employees (id) ON DELETE CASCADE,
    software_id UUID NOT NULL REFERENCES softwares (id) ON DELETE CASCADE,
    product_key TEXT,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (employee_id, software_id)
);
