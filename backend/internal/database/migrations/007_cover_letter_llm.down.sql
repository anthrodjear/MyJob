-- Remove LLM traceability columns from cover_letters
ALTER TABLE cover_letters DROP COLUMN IF EXISTS model;
ALTER TABLE cover_letters DROP COLUMN IF EXISTS prompt_version;
ALTER TABLE cover_letters DROP COLUMN IF EXISTS resume_version;
ALTER TABLE cover_letters DROP COLUMN IF EXISTS strengths;
ALTER TABLE cover_letters DROP COLUMN IF EXISTS gaps;
ALTER TABLE cover_letters DROP COLUMN IF EXISTS job_title;

-- Restore pdf_path column name
ALTER TABLE cover_letters RENAME COLUMN pdf_key TO pdf_path;
