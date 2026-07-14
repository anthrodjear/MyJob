-- ============================================
-- PASSWORD RESET TOKENS (DOWN)
-- ============================================
DROP INDEX IF EXISTS idx_password_reset_tokens_user;
DROP INDEX IF EXISTS idx_password_reset_tokens_expires;
DROP TABLE IF EXISTS password_reset_tokens;
