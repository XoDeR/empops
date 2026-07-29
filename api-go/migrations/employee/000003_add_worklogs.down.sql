DROP TABLE IF EXISTS worklogs;

ALTER TABLE employees DROP COLUMN IF EXISTS consecutive_worklog_missed;
