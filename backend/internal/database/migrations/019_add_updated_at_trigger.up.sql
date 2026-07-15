-- High Priority: Add updated_at auto-update trigger function
-- PostgreSQL doesn't have ON UPDATE CURRENT_TIMESTAMP, so we need a trigger

-- 1. Create the trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

-- 2. Apply to all tables that have updated_at column
-- Core tables (from 001_initial)
DROP TRIGGER IF EXISTS update_updated_at ON profiles;
CREATE TRIGGER update_updated_at BEFORE UPDATE ON profiles
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_updated_at ON job_sources;
CREATE TRIGGER update_updated_at BEFORE UPDATE ON job_sources
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_updated_at ON jobs;
CREATE TRIGGER update_updated_at BEFORE UPDATE ON jobs
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_updated_at ON resumes;
CREATE TRIGGER update_updated_at BEFORE UPDATE ON resumes
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_updated_at ON cover_letters;
CREATE TRIGGER update_updated_at BEFORE UPDATE ON cover_letters
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_updated_at ON applications;
CREATE TRIGGER update_updated_at BEFORE UPDATE ON applications
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_updated_at ON tasks;
CREATE TRIGGER update_updated_at BEFORE UPDATE ON tasks
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_updated_at ON interviews;
CREATE TRIGGER update_updated_at BEFORE UPDATE ON interviews
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_updated_at ON users;
CREATE TRIGGER update_updated_at BEFORE UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_updated_at ON interview_sessions;
CREATE TRIGGER update_updated_at BEFORE UPDATE ON interview_sessions
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_updated_at ON system_config_overrides;
CREATE TRIGGER update_updated_at BEFORE UPDATE ON system_config_overrides
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_updated_at ON refresh_tokens;
CREATE TRIGGER update_updated_at BEFORE UPDATE ON refresh_tokens
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_updated_at ON password_reset_tokens;
CREATE TRIGGER update_updated_at BEFORE UPDATE ON password_reset_tokens
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
