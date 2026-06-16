DROP INDEX IF EXISTS idx_resume_versions_resume_id;
DROP TABLE IF EXISTS resume_versions;
ALTER TABLE resumes DROP COLUMN IF EXISTS content;
