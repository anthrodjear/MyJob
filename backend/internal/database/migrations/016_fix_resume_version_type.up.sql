-- Critical Fix: cover_letter_versions.resume_version type mismatch (VARCHAR → INT)
-- This fixes the data integrity issue where resume_version was VARCHAR(50) but references resumes.version (INT)

-- First, ensure the column can be safely cast (no non-numeric values)
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'cover_letter_versions' 
        AND column_name = 'resume_version' 
        AND data_type = 'character varying'
    ) THEN
        -- Verify all existing values are valid integers
        IF EXISTS (
            SELECT 1 FROM cover_letter_versions 
            WHERE resume_version IS NOT NULL 
            AND resume_version ~ '^\d+$' = FALSE
        ) THEN
            RAISE EXCEPTION 'Cannot convert resume_version: non-numeric values exist';
        END IF;

        -- Safe to convert
        ALTER TABLE cover_letter_versions 
        ALTER COLUMN resume_version TYPE INT USING resume_version::int;
    END IF;
END $$;
