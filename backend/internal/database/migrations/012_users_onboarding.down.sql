-- 012_users_onboarding.down.sql
-- Reverse the onboarding columns migration.

DROP INDEX IF EXISTS idx_users_onboarding;
ALTER TABLE users DROP COLUMN IF EXISTS onboarding_step;
ALTER TABLE users DROP COLUMN IF EXISTS onboarding_completed_at;
