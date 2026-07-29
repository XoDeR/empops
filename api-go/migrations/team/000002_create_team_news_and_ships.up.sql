CREATE TABLE IF NOT EXISTS team_news (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    team_id UUID NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
    author_id UUID REFERENCES employees (id) ON DELETE SET NULL,
    author_name VARCHAR(255) NOT NULL,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS team_news_team_id_idx ON team_news (team_id, created_at DESC);

CREATE TABLE IF NOT EXISTS ships (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    team_id UUID NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
    author_id UUID REFERENCES employees (id) ON DELETE SET NULL,
    author_name VARCHAR(255) NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS ships_team_id_idx ON ships (team_id, created_at DESC);

CREATE TABLE IF NOT EXISTS employee_ship (
    employee_id UUID NOT NULL REFERENCES employees (id) ON DELETE CASCADE,
    ship_id UUID NOT NULL REFERENCES ships (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (employee_id, ship_id)
);
