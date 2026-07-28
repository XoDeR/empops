CREATE TABLE IF NOT EXISTS activity_logs (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    event VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    subject_type VARCHAR(255),
    subject_id UUID,
    causer_type VARCHAR(255),
    causer_id UUID,
    properties JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS activity_logs_company_id_created_at_idx
    ON activity_logs (company_id, created_at DESC);
