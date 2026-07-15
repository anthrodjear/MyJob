-- Reverse only the changes made by 021.up: drop NOT NULL and the DEFAULT 1.
-- The INT type conversion itself is owned by 016 and reverted there.

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'cover_letter_versions'
          AND column_name = 'resume_version'
          AND data_type = 'integer'
    ) THEN
        ALTER TABLE cover_letter_versions ALTER COLUMN resume_version DROP NOT NULL;
        ALTER TABLE cover_letter_versions ALTER COLUMN resume_version DROP DEFAULT;
    END IF;
END $$;
