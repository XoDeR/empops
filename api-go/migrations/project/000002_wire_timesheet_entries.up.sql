ALTER TABLE time_tracking_entries DROP CONSTRAINT IF EXISTS time_entry_day_unique;
ALTER TABLE time_tracking_entries
    ADD COLUMN IF NOT EXISTS project_id UUID REFERENCES projects (id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS project_task_id UUID REFERENCES project_tasks (id) ON DELETE SET NULL;

CREATE UNIQUE INDEX IF NOT EXISTS time_entry_adhoc_unique
    ON time_tracking_entries (timesheet_id, happened_at)
    WHERE project_task_id IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS time_entry_task_unique
    ON time_tracking_entries (timesheet_id, happened_at, project_task_id)
    WHERE project_task_id IS NOT NULL;
