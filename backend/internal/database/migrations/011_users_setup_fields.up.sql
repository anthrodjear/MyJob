-- 011_users_setup_fields.up.sql
-- Add username and email columns for the setup flow.
-- Defaults ensure existing rows are unaffected.

ALTER TABLE users ADD COLUMN username VARCHAR(100) DEFAULT '';
ALTER TABLE users ADD COLUMN email VARCHAR(255) DEFAULT '';

-- Partial unique indexes: only enforce uniqueness when the field is non-empty.
-- Empty string (default for existing rows) does not trigger the constraint.
CREATE UNIQUE INDEX idx_users_username ON users (username) WHERE username != '';
CREATE UNIQUE INDEX idx_users_email ON users (email) WHERE email != '';
