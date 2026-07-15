-- Down migration: Remove updated_at trigger function and triggers

-- Drop all triggers
DROP TRIGGER IF EXISTS update_updated_at ON profiles;
DROP TRIGGER IF EXISTS update_updated_at ON job_sources;
DROP TRIGGER IF EXISTS update_updated_at ON jobs;
DROP TRIGGER IF EXISTS update_updated_at ON resumes;
DROP TRIGGER IF EXISTS update_updated_at ON cover_letters;
DROP TRIGGER IF EXISTS update_updated_at ON applications;
DROP TRIGGER IF EXISTS update_updated_at ON tasks;
DROP TRIGGER IF EXISTS update_updated_at ON interviews;
DROP TRIGGER IF EXISTS update_updated_at ON users;
DROP TRIGGER IF EXISTS update_updated_at ON interview_sessions;
DROP TRIGGER IF EXISTS update_updated_at ON system_config_overrides;
DROP TRIGGER IF EXISTS update_updated_at ON refresh_tokens;
DROP TRIGGER IF EXISTS update_updated_at ON password_reset_tokens;

-- Drop the function
DROP FUNCTION IF EXISTS update_updated_at_column();
