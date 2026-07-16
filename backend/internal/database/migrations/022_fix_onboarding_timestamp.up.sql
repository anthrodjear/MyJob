-- Fix: users.onboarding_completed_at should be TIMESTAMPTZ not TIMESTAMP
-- All other timestamps in the schema use TIMESTAMPTZ for timezone awareness

ALTER TABLE users 
ALTER COLUMN onboarding_completed_at TYPE TIMESTAMPTZ
USING onboarding_completed_at AT TIME ZONE 'UTC';
