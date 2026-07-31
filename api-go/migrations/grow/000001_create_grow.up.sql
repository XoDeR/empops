ALTER TABLE companies
    ADD COLUMN IF NOT EXISTS e_coffee_enabled BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE IF NOT EXISTS morales (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    employee_id UUID NOT NULL REFERENCES employees (id) ON DELETE CASCADE,
    emotion SMALLINT NOT NULL,
    comment VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_morales_employee_created ON morales (employee_id, created_at);
CREATE INDEX IF NOT EXISTS idx_morales_company_created ON morales (company_id, created_at);

CREATE TABLE IF NOT EXISTS morale_company_histories (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    average DOUBLE PRECISION NOT NULL DEFAULT 0,
    number_of_employees INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_morale_company_histories_company ON morale_company_histories (company_id, created_at);

CREATE TABLE IF NOT EXISTS morale_team_histories (
    id UUID PRIMARY KEY,
    team_id UUID NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
    average DOUBLE PRECISION NOT NULL DEFAULT 0,
    number_of_team_members INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_morale_team_histories_team ON morale_team_histories (team_id, created_at);

CREATE TABLE IF NOT EXISTS one_on_one_entries (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    manager_id UUID NOT NULL REFERENCES employees (id) ON DELETE CASCADE,
    employee_id UUID NOT NULL REFERENCES employees (id) ON DELETE CASCADE,
    happened BOOLEAN NOT NULL DEFAULT FALSE,
    happened_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_one_on_one_manager ON one_on_one_entries (manager_id, happened);
CREATE INDEX IF NOT EXISTS idx_one_on_one_employee ON one_on_one_entries (employee_id, happened);

CREATE TABLE IF NOT EXISTS one_on_one_talking_points (
    id UUID PRIMARY KEY,
    one_on_one_entry_id UUID NOT NULL REFERENCES one_on_one_entries (id) ON DELETE CASCADE,
    description VARCHAR(255) NOT NULL,
    checked BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS one_on_one_action_items (
    id UUID PRIMARY KEY,
    one_on_one_entry_id UUID NOT NULL REFERENCES one_on_one_entries (id) ON DELETE CASCADE,
    description VARCHAR(255) NOT NULL,
    checked BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS one_on_one_notes (
    id UUID PRIMARY KEY,
    one_on_one_entry_id UUID NOT NULL REFERENCES one_on_one_entries (id) ON DELETE CASCADE,
    note TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS rate_your_manager_surveys (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    manager_id UUID REFERENCES employees (id) ON DELETE SET NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    valid_until_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_rym_surveys_company ON rate_your_manager_surveys (company_id, active);

CREATE TABLE IF NOT EXISTS rate_your_manager_answers (
    id UUID PRIMARY KEY,
    rate_your_manager_survey_id UUID NOT NULL REFERENCES rate_your_manager_surveys (id) ON DELETE CASCADE,
    employee_id UUID NOT NULL REFERENCES employees (id) ON DELETE CASCADE,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    rating VARCHAR(255),
    comment TEXT,
    reveal_identity_to_manager BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (rate_your_manager_survey_id, employee_id)
);
CREATE INDEX IF NOT EXISTS idx_rym_answers_employee ON rate_your_manager_answers (employee_id, active);

CREATE TABLE IF NOT EXISTS skills (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (company_id, name)
);

CREATE TABLE IF NOT EXISTS employee_skill (
    employee_id UUID NOT NULL REFERENCES employees (id) ON DELETE CASCADE,
    skill_id UUID NOT NULL REFERENCES skills (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (employee_id, skill_id)
);

CREATE TABLE IF NOT EXISTS e_coffees (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    batch_number INTEGER NOT NULL DEFAULT 1,
    active BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_e_coffees_company ON e_coffees (company_id, active);

CREATE TABLE IF NOT EXISTS e_coffee_matches (
    id UUID PRIMARY KEY,
    e_coffee_id UUID NOT NULL REFERENCES e_coffees (id) ON DELETE CASCADE,
    employee_id UUID NOT NULL REFERENCES employees (id) ON DELETE CASCADE,
    with_employee_id UUID NOT NULL REFERENCES employees (id) ON DELETE CASCADE,
    happened BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_e_coffee_matches_session ON e_coffee_matches (e_coffee_id, employee_id);

CREATE TABLE IF NOT EXISTS discipline_cases (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    employee_id UUID NOT NULL REFERENCES employees (id) ON DELETE CASCADE,
    opened_by_employee_id UUID REFERENCES employees (id) ON DELETE SET NULL,
    opened_by_employee_name VARCHAR(255),
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_discipline_cases_company ON discipline_cases (company_id, active);

CREATE TABLE IF NOT EXISTS discipline_events (
    id UUID PRIMARY KEY,
    discipline_case_id UUID NOT NULL REFERENCES discipline_cases (id) ON DELETE CASCADE,
    author_id UUID REFERENCES employees (id) ON DELETE SET NULL,
    author_name VARCHAR(255) NOT NULL,
    happened_at DATE NOT NULL,
    description TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_discipline_events_case ON discipline_events (discipline_case_id);
