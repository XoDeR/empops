CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    project_lead_id UUID REFERENCES employees (id) ON DELETE SET NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'created',
    completed BOOLEAN NOT NULL DEFAULT FALSE,
    name VARCHAR(255) NOT NULL,
    code VARCHAR(255),
    short_code VARCHAR(64),
    emoji VARCHAR(16),
    summary VARCHAR(255),
    description TEXT,
    started_at TIMESTAMPTZ,
    planned_finished_at TIMESTAMPTZ,
    actually_finished_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_projects_company_status ON projects (company_id, status);

CREATE TABLE IF NOT EXISTS employee_project (
    employee_id UUID NOT NULL REFERENCES employees (id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    role VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (employee_id, project_id)
);

CREATE TABLE IF NOT EXISTS project_team (
    project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    team_id UUID NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, team_id)
);

CREATE TABLE IF NOT EXISTS project_links (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    type VARCHAR(255) NOT NULL,
    label VARCHAR(255),
    url VARCHAR(2048) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS project_statuses (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    author_id UUID REFERENCES employees (id) ON DELETE SET NULL,
    title VARCHAR(255) NOT NULL,
    status VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS project_messages (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    author_id UUID REFERENCES employees (id) ON DELETE SET NULL,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS project_decisions (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    author_id UUID REFERENCES employees (id) ON DELETE SET NULL,
    title VARCHAR(255) NOT NULL,
    decided_at DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS project_decision_deciders (
    project_decision_id UUID NOT NULL REFERENCES project_decisions (id) ON DELETE CASCADE,
    employee_id UUID NOT NULL REFERENCES employees (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (project_decision_id, employee_id)
);

CREATE TABLE IF NOT EXISTS project_task_lists (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    author_id UUID REFERENCES employees (id) ON DELETE SET NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS project_tasks (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    project_task_list_id UUID REFERENCES project_task_lists (id) ON DELETE CASCADE,
    author_id UUID REFERENCES employees (id) ON DELETE SET NULL,
    assignee_id UUID REFERENCES employees (id) ON DELETE SET NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    completed BOOLEAN NOT NULL DEFAULT FALSE,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS comments (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    author_id UUID REFERENCES employees (id) ON DELETE SET NULL,
    author_name VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    commentable_id UUID,
    commentable_type VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_comments_commentable ON comments (commentable_type, commentable_id);

CREATE TABLE IF NOT EXISTS issue_types (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    icon VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (company_id, name)
);

CREATE TABLE IF NOT EXISTS project_boards (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS project_sprints (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    project_board_id UUID REFERENCES project_boards (id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    active BOOLEAN NOT NULL DEFAULT FALSE,
    position INTEGER,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS project_issues (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    project_board_id UUID REFERENCES project_boards (id) ON DELETE SET NULL,
    reporter_id UUID REFERENCES employees (id) ON DELETE SET NULL,
    issue_type_id UUID REFERENCES issue_types (id) ON DELETE SET NULL,
    is_separator BOOLEAN NOT NULL DEFAULT FALSE,
    id_in_project INTEGER NOT NULL,
    key VARCHAR(64) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    story_points INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (project_id, id_in_project),
    UNIQUE (project_id, key)
);

CREATE TABLE IF NOT EXISTS project_issue_project_sprint (
    project_issue_id UUID NOT NULL REFERENCES project_issues (id) ON DELETE CASCADE,
    project_sprint_id UUID NOT NULL REFERENCES project_sprints (id) ON DELETE CASCADE,
    position INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (project_issue_id, project_sprint_id)
);

CREATE TABLE IF NOT EXISTS project_labels (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS project_issue_project_label (
    project_issue_id UUID NOT NULL REFERENCES project_issues (id) ON DELETE CASCADE,
    project_label_id UUID NOT NULL REFERENCES project_labels (id) ON DELETE CASCADE,
    PRIMARY KEY (project_issue_id, project_label_id)
);

CREATE TABLE IF NOT EXISTS project_issue_assignees (
    project_issue_id UUID NOT NULL REFERENCES project_issues (id) ON DELETE CASCADE,
    employee_id UUID NOT NULL REFERENCES employees (id) ON DELETE CASCADE,
    PRIMARY KEY (project_issue_id, employee_id)
);
