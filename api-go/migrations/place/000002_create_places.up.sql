CREATE TABLE IF NOT EXISTS places (
    id UUID PRIMARY KEY,
    placable_id UUID NOT NULL,
    placable_type TEXT NOT NULL,
    street TEXT,
    city TEXT,
    province TEXT,
    postal_code TEXT,
    country_id UUID REFERENCES countries(id) ON DELETE SET NULL,
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    is_active BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS places_placable_idx ON places (placable_type, placable_id);
