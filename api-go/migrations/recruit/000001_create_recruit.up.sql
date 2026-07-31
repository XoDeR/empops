CREATE TABLE IF NOT EXISTS recruiting_stage_templates (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_recruiting_stage_templates_company ON recruiting_stage_templates (company_id);

CREATE TABLE IF NOT EXISTS recruiting_stages (
    id UUID PRIMARY KEY,
    recruiting_stage_template_id UUID NOT NULL REFERENCES recruiting_stage_templates (id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    position INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_recruiting_stages_template ON recruiting_stages (recruiting_stage_template_id);

CREATE TABLE IF NOT EXISTS job_openings (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    position_id UUID NOT NULL REFERENCES positions (id) ON DELETE CASCADE,
    recruiting_stage_template_id UUID REFERENCES recruiting_stage_templates (id) ON DELETE SET NULL,
    team_id UUID REFERENCES teams (id) ON DELETE SET NULL,
    fulfilled_by_candidate_id UUID,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    slug VARCHAR(255) NOT NULL,
    reference_number VARCHAR(255),
    active BOOLEAN NOT NULL DEFAULT FALSE,
    fulfilled BOOLEAN NOT NULL DEFAULT FALSE,
    page_views INTEGER NOT NULL DEFAULT 0,
    activated_at TIMESTAMPTZ,
    fulfilled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (company_id, slug)
);
CREATE INDEX IF NOT EXISTS idx_job_openings_company_active ON job_openings (company_id, active, fulfilled);

CREATE TABLE IF NOT EXISTS job_opening_sponsor (
    job_opening_id UUID NOT NULL REFERENCES job_openings (id) ON DELETE CASCADE,
    employee_id UUID NOT NULL REFERENCES employees (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (job_opening_id, employee_id)
);

CREATE TABLE IF NOT EXISTS candidates (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    job_opening_id UUID NOT NULL REFERENCES job_openings (id) ON DELETE CASCADE,
    employee_id UUID REFERENCES employees (id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    uuid UUID NOT NULL UNIQUE,
    url VARCHAR(2048),
    desired_salary VARCHAR(255),
    notes TEXT,
    application_completed BOOLEAN NOT NULL DEFAULT FALSE,
    rejected BOOLEAN NOT NULL DEFAULT FALSE,
    employee_name VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_candidates_opening ON candidates (job_opening_id, application_completed, rejected);

ALTER TABLE job_openings
    ADD CONSTRAINT job_openings_fulfilled_by_candidate_id_fkey
    FOREIGN KEY (fulfilled_by_candidate_id) REFERENCES candidates (id) ON DELETE SET NULL;

CREATE TABLE IF NOT EXISTS candidate_stages (
    id UUID PRIMARY KEY,
    candidate_id UUID NOT NULL REFERENCES candidates (id) ON DELETE CASCADE,
    decider_id UUID REFERENCES employees (id) ON DELETE SET NULL,
    stage_name VARCHAR(255) NOT NULL,
    stage_position INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    decider_name VARCHAR(255),
    decided_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_candidate_stages_candidate ON candidate_stages (candidate_id, stage_position);

CREATE TABLE IF NOT EXISTS candidate_stage_notes (
    id UUID PRIMARY KEY,
    candidate_stage_id UUID NOT NULL REFERENCES candidate_stages (id) ON DELETE CASCADE,
    author_id UUID REFERENCES employees (id) ON DELETE SET NULL,
    author_name VARCHAR(255) NOT NULL,
    note TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS candidate_stage_participants (
    id UUID PRIMARY KEY,
    candidate_stage_id UUID NOT NULL REFERENCES candidate_stages (id) ON DELETE CASCADE,
    participant_id UUID NOT NULL REFERENCES employees (id) ON DELETE CASCADE,
    participant_name VARCHAR(255) NOT NULL,
    participated BOOLEAN NOT NULL DEFAULT FALSE,
    participated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (candidate_stage_id, participant_id)
);
