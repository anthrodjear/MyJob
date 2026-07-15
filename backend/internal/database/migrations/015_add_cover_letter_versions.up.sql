-- ============================================
-- COVER LETTER VERSIONS
-- ============================================
CREATE TABLE IF NOT EXISTS cover_letter_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    cover_letter_id UUID NOT NULL REFERENCES cover_letters(id) ON DELETE CASCADE,
    version INT NOT NULL,
    content TEXT NOT NULL,
    model VARCHAR(100),
    prompt_version VARCHAR(50),
    resume_version VARCHAR(50),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (cover_letter_id, version)
);

CREATE INDEX IF NOT EXISTS idx_cover_letter_versions_cover_letter_id ON cover_letter_versions(cover_letter_id);
