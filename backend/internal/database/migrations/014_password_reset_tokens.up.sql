-- ============================================
-- PASSWORD RESET TOKENS (UP)
-- ============================================
-- For local-first single-user app: password reset via reset token
-- Token is generated on request, shown to user, expires in 1 hour
CREATE TABLE password_reset_tokens (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id    VARCHAR(100) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL UNIQUE,                        -- SHA-256 hex of the token
    expires_at TIMESTAMPTZ NOT NULL,                                -- when the token expires
    used_at    TIMESTAMPTZ,                                         -- NULL = unused, non-NULL = consumed
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),                  -- when the token was created
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()                   -- last modification time
);

-- Fast cleanup of expired unused tokens
CREATE INDEX idx_password_reset_tokens_expires ON password_reset_tokens(expires_at)
    WHERE used_at IS NULL;

-- Query tokens by user (for debugging/admin)
CREATE INDEX idx_password_reset_tokens_user ON password_reset_tokens(user_id)
    WHERE used_at IS NULL;
    