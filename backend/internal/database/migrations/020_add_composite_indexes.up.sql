-- High Priority: Add missing composite indexes for common query patterns

-- 1. applications: list by status, newest first
CREATE INDEX IF NOT EXISTS idx_applications_status_created ON applications(status, created_at DESC);

-- 2. applications: find apps for a job by status
CREATE INDEX IF NOT EXISTS idx_applications_job_status ON applications(job_id, status);

-- 3. jobs: high-scoring discovered jobs (for dashboard)
CREATE INDEX IF NOT EXISTS idx_jobs_status_match_score ON jobs(status, match_score DESC);

-- 4. jobs: latest jobs from a source
CREATE INDEX IF NOT EXISTS idx_jobs_source_scraped ON jobs(source_id, scraped_at DESC);

-- 5. tasks: worker polling (type + status + scheduled)
CREATE INDEX IF NOT EXISTS idx_tasks_type_status_scheduled ON tasks(type, status, scheduled_at);

-- 5b. tasks: cleanup old completed/failed tasks
CREATE INDEX IF NOT EXISTS idx_tasks_completed_at ON tasks(completed_at) WHERE status IN ('completed', 'failed');

-- 6. cover_letters: find by job + resume combo
CREATE INDEX IF NOT EXISTS idx_cover_letters_job_resume ON cover_letters(job_id, resume_id);

-- 7. interview_sessions: interviews by application + status
CREATE INDEX IF NOT EXISTS idx_interview_sessions_app_status ON interview_sessions(application_id, status);
