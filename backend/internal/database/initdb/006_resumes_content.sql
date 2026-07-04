-- Add structured content column to resumes and create resume_versions table
ALTER TABLE resumes ADD COLUMN IF NOT EXISTS content JSONB;

-- Versioned resume content for history and rollback
CREATE TABLE IF NOT EXISTS resume_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resume_id UUID NOT NULL REFERENCES resumes(id) ON DELETE CASCADE,
    content JSONB NOT NULL,
    version INT NOT NULL,
    pdf_key VARCHAR(500),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(resume_id, version)
);

CREATE INDEX IF NOT EXISTS idx_resume_versions_resume_id ON resume_versions(resume_id);
