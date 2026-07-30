ALTER TABLE companies
    ADD COLUMN IF NOT EXISTS work_from_home_enabled BOOLEAN NOT NULL DEFAULT true;
