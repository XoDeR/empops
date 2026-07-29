-- Countries reference data for places.

CREATE TABLE IF NOT EXISTS countries (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    code CHAR(2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS countries_code_key ON countries (code) WHERE code IS NOT NULL;

INSERT INTO countries (id, name, code) VALUES
    (gen_random_uuid(), 'United States', 'US'),
    (gen_random_uuid(), 'United Kingdom', 'GB'),
    (gen_random_uuid(), 'Canada', 'CA'),
    (gen_random_uuid(), 'Germany', 'DE'),
    (gen_random_uuid(), 'France', 'FR'),
    (gen_random_uuid(), 'Netherlands', 'NL'),
    (gen_random_uuid(), 'Belgium', 'BE'),
    (gen_random_uuid(), 'Spain', 'ES'),
    (gen_random_uuid(), 'Italy', 'IT'),
    (gen_random_uuid(), 'Switzerland', 'CH'),
    (gen_random_uuid(), 'Austria', 'AT'),
    (gen_random_uuid(), 'Poland', 'PL'),
    (gen_random_uuid(), 'Sweden', 'SE'),
    (gen_random_uuid(), 'Norway', 'NO'),
    (gen_random_uuid(), 'Denmark', 'DK'),
    (gen_random_uuid(), 'Ireland', 'IE'),
    (gen_random_uuid(), 'Portugal', 'PT'),
    (gen_random_uuid(), 'Australia', 'AU'),
    (gen_random_uuid(), 'New Zealand', 'NZ'),
    (gen_random_uuid(), 'Romania', 'RO')
ON CONFLICT (name) DO NOTHING;
