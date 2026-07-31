DROP TABLE IF EXISTS candidate_stage_participants;
DROP TABLE IF EXISTS candidate_stage_notes;
DROP TABLE IF EXISTS candidate_stages;
ALTER TABLE IF EXISTS job_openings DROP CONSTRAINT IF EXISTS job_openings_fulfilled_by_candidate_id_fkey;
DROP TABLE IF EXISTS candidates;
DROP TABLE IF EXISTS job_opening_sponsor;
DROP TABLE IF EXISTS job_openings;
DROP TABLE IF EXISTS recruiting_stages;
DROP TABLE IF EXISTS recruiting_stage_templates;
