-- Add LLM traceability columns to cover_letters
ALTER TABLE cover_letters ADD COLUMN IF NOT EXISTS model VARCHAR(100);
ALTER TABLE cover_letters ADD COLUMN IF NOT EXISTS prompt_version VARCHAR(50);
ALTER TABLE cover_letters ADD COLUMN IF NOT EXISTS resume_version INT;
ALTER TABLE cover_letters ADD COLUMN IF NOT EXISTS strengths JSONB;
ALTER TABLE cover_letters ADD COLUMN IF NOT EXISTS gaps JSONB;
ALTER TABLE cover_letters ADD COLUMN IF NOT EXISTS job_title VARCHAR(200);

-- Rename pdf_path to pdf_key for consistency with resumes table
ALTER TABLE cover_letters RENAME COLUMN pdf_path TO pdf_key;
