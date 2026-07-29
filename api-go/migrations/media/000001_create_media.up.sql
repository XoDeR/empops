CREATE TABLE IF NOT EXISTS temporary_uploads (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS media (
    id BIGSERIAL PRIMARY KEY,
    model_type TEXT NOT NULL,
    model_id TEXT NOT NULL,
    collection_name TEXT NOT NULL,
    name TEXT NOT NULL,
    file_name TEXT NOT NULL,
    mime_type TEXT,
    disk TEXT NOT NULL DEFAULT 'local',
    size BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS media_model_idx ON media (model_type, model_id);
