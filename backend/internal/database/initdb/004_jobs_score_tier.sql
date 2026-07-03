-- Add score tier and scored_at to jobs table
ALTER TABLE jobs ADD COLUMN score_tier VARCHAR(20);
ALTER TABLE jobs ADD COLUMN scored_at TIMESTAMPTZ;

CREATE INDEX idx_jobs_score_tier ON jobs(score_tier);
