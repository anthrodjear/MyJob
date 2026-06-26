-- 011_users_setup_fields.down.sql
-- Reverse the setup fields migration.

DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_users_username;
ALTER TABLE users DROP COLUMN IF EXISTS email;
ALTER TABLE users DROP COLUMN IF EXISTS username;
