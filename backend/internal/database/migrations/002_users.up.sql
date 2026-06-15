-- ============================================
-- USERS (for local auth)
-- ============================================
CREATE TABLE users (
    id VARCHAR(100) PRIMARY KEY DEFAULT 'local-user',
    password_hash VARCHAR(255) NOT NULL,
    session_version INT NOT NULL DEFAULT 1,
    last_login_at TIMESTAMPTZ,
    password_changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed initial user from config (run once on first boot)
-- This is handled by application startup, not migration
