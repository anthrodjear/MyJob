-- Complementary to 016: after resume_version is converted to INT by 016,
-- enforce NOT NULL with a safe default of 1 (baseline resume version).
-- Guarded so it is a no-op if 016 has not yet converted the column,
-- and idempotent so re-running is safe. Removes the previous crashing
-- add-column/copy/drop/rename dance.

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'cover_letter_versions'
          AND column_name = 'resume_version'
          AND data_type = 'integer'
    ) THEN
        -- Backfill any NULLs to the baseline version 1 before enforcing NOT NULL
        UPDATE cover_letter_versions SET resume_version = 1 WHERE resume_version IS NULL;

        -- Set default 1 (only if not already set)
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_name = 'cover_letter_versions'
              AND column_name = 'resume_version'
              AND column_default = '1'
        ) THEN
            ALTER TABLE cover_letter_versions ALTER COLUMN resume_version SET DEFAULT 1;
        END IF;

        -- Enforce NOT NULL (only if currently nullable)
        IF EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_name = 'cover_letter_versions'
              AND column_name = 'resume_version'
              AND is_nullable = 'YES'
        ) THEN
            ALTER TABLE cover_letter_versions ALTER COLUMN resume_version SET NOT NULL;
        END IF;
    END IF;
END $$;
