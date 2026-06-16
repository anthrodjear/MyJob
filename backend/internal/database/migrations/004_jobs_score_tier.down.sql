ALTER TABLE jobs DROP COLUMN score_tier;
ALTER TABLE jobs DROP COLUMN scored_at;
DROP INDEX IF EXISTS idx_jobs_score_tier;
