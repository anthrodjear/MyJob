-- Down migration: Drop the added FK indexes
-- Reverse order of creation

DROP INDEX IF EXISTS idx_emails_application_id;
CREATE INDEX IF NOT EXISTS idx_emails_application ON emails(application_id);

DROP INDEX IF EXISTS idx_interviews_application_id;
DROP INDEX IF EXISTS idx_cover_letters_resume_id;
DROP INDEX IF EXISTS idx_cover_letters_job_id;
DROP INDEX IF EXISTS idx_approval_requests_application_id;
