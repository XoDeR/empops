CREATE TABLE IF NOT EXISTS company_news (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    author_id UUID REFERENCES employees (id) ON DELETE SET NULL,
    author_name VARCHAR(255) NOT NULL,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS company_news_company_id_idx ON company_news (company_id, created_at DESC);

CREATE TABLE IF NOT EXISTS questions (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT false,
    activated_at TIMESTAMPTZ,
    deactivated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS questions_company_id_idx ON questions (company_id);
CREATE UNIQUE INDEX IF NOT EXISTS questions_company_active_idx ON questions (company_id) WHERE active;

CREATE TABLE IF NOT EXISTS answers (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    question_id UUID NOT NULL REFERENCES questions (id) ON DELETE CASCADE,
    employee_id UUID NOT NULL REFERENCES employees (id) ON DELETE CASCADE,
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (question_id, employee_id)
);

CREATE INDEX IF NOT EXISTS answers_question_id_idx ON answers (question_id);
