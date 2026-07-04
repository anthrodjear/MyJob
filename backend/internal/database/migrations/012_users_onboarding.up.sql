-- 012_users_onboarding.up.sql
-- Add onboarding columns to track wizard completion state and progress.

ALTER TABLE users ADD COLUMN onboarding_completed_at TIMESTAMP NULL;
ALTER TABLE users ADD COLUMN onboarding_step VARCHAR(50) DEFAULT 'account';

CREATE INDEX idx_users_onboarding ON users (onboarding_completed_at);
