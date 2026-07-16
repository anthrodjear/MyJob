-- High Priority: Add missing foreign key indexes
-- These indexes are required for JOIN performance and FK constraint checks
-- Per PostgreSQL docs: FK referencing columns should be indexed

-- 1. approval_requests.application_id (FK to applications.id)
CREATE INDEX IF NOT EXISTS idx_approval_requests_application_id 
ON approval_requests(application_id);

-- 2. cover_letters.job_id (FK to jobs.id)
CREATE INDEX IF NOT EXISTS idx_cover_letters_job_id 
ON cover_letters(job_id);

-- 3. cover_letters.resume_id (FK to resumes.id)
CREATE INDEX IF NOT EXISTS idx_cover_letters_resume_id 
ON cover_letters(resume_id);

-- 4. interviews.application_id (FK to applications.id)
CREATE INDEX IF NOT EXISTS idx_interviews_application_id 
ON interviews(application_id);

-- 5. Fix emails index name typo: idx_emails_application -> idx_emails_application_id
DROP INDEX IF EXISTS idx_emails_application;
CREATE INDEX IF NOT EXISTS idx_emails_application_id 
ON emails(application_id);
