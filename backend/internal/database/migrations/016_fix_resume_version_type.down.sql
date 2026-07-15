-- Down migration: Revert resume_version to VARCHAR(50)
-- Only run if column exists and is INT

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'cover_letter_versions' 
        AND column_name = 'resume_version' 
        AND data_type = 'integer'
    ) THEN
        ALTER TABLE cover_letter_versions 
        ALTER COLUMN resume_version TYPE VARCHAR(50) USING resume_version::text;
    END IF;
END $$;
