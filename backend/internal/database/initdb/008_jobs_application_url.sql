-- Add application_url and source columns to jobs table.
-- application_url: where the user actually applies (resolved from apply buttons/redirects).
-- source: scraper identifier (greenhouse, lever, remoteok, indeed, custom).

ALTER TABLE jobs ADD COLUMN application_url TEXT NOT NULL DEFAULT '';
ALTER TABLE jobs ADD COLUMN source TEXT NOT NULL DEFAULT 'custom';

-- Index on source for filtering by scraper
CREATE INDEX idx_jobs_source ON jobs (source);
