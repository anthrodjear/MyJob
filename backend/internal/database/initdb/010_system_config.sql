-- ============================================
-- SYSTEM CONFIG OVERRIDES
-- ============================================
-- Runtime configuration overrides that merge on top of application.yaml
-- Key uses dot-notation (e.g., "scoring.auto_threshold")
-- Value is JSONB for flexibility (numbers, strings, booleans, objects)
-- Category distinguishes "runtime" vs "operational" settings
CREATE TABLE system_config_overrides (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    key VARCHAR(255) UNIQUE NOT NULL,
    value JSONB NOT NULL,
    category VARCHAR(50) NOT NULL DEFAULT 'runtime',
    description TEXT,
    updated_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_config_key ON system_config_overrides(key);
CREATE INDEX idx_config_category ON system_config_overrides(category);