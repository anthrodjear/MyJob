-- ============================================
-- REFRESH TOKENS (JWT rotation)
-- ============================================
CREATE TABLE refresh_tokens (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id    VARCHAR(100) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL UNIQUE,                        -- SHA-256 hex of the token
    expires_at TIMESTAMPTZ NOT NULL,                                -- when the token expires
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),                  -- when the token was created
    revoked_at TIMESTAMPTZ,                                         -- NULL = active, non-NULL = revoked
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()                   -- last modification time
);

-- Fast cleanup of expired active tokens
CREATE INDEX idx_refresh_tokens_expires ON refresh_tokens(expires_at)
    WHERE revoked_at IS NULL;

-- Query active tokens by user (session listing, mass revocation)
CREATE INDEX idx_refresh_tokens_user ON refresh_tokens(user_id)
    WHERE revoked_at IS NULL;
