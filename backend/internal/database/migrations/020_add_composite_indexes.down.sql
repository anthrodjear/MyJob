-- Down migration: Drop all composite indexes added in 020

DROP INDEX IF EXISTS idx_applications_status_created;
DROP INDEX IF EXISTS idx_applications_job_status;
DROP INDEX IF EXISTS idx_jobs_status_match_score;
DROP INDEX IF EXISTS idx_jobs_source_scraped;
DROP INDEX IF EXISTS idx_tasks_type_status_scheduled;
DROP INDEX IF EXISTS idx_tasks_completed_at;
DROP INDEX IF EXISTS idx_cover_letters_job_resume;
DROP INDEX IF EXISTS idx_interview_sessions_app_status;
