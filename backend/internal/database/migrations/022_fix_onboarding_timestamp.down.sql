-- Down migration: Revert to TIMESTAMP (no timezone)

ALTER TABLE users 
ALTER COLUMN onboarding_completed_at TYPE TIMESTAMP
USING onboarding_completed_at AT TIME ZONE 'UTC';
