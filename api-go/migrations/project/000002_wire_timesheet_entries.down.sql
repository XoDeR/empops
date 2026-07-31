DROP INDEX IF EXISTS time_entry_task_unique;
DROP INDEX IF EXISTS time_entry_adhoc_unique;
ALTER TABLE time_tracking_entries DROP COLUMN IF EXISTS project_task_id;
ALTER TABLE time_tracking_entries DROP COLUMN IF EXISTS project_id;
ALTER TABLE time_tracking_entries
    ADD CONSTRAINT time_entry_day_unique UNIQUE (timesheet_id, employee_id, happened_at);
