-- ============================================
-- 025: Add saved flag + metadata JSONB to jobs
-- ============================================
-- The frontend calls PATCH /jobs/:id/save to toggle a per-user "saved"
-- bit on jobs, and may use the metadata column for client-side annotations.
-- Both columns are nullable with safe defaults so backfilling existing
-- rows is a no-op.

BEGIN;

ALTER TABLE jobs
    ADD COLUMN IF NOT EXISTS saved BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}'::jsonb;

CREATE INDEX IF NOT EXISTS idx_jobs_saved ON jobs(saved) WHERE saved = TRUE;

COMMIT;