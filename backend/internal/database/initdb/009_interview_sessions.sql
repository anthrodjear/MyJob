-- ============================================
-- INTERVIEW SESSIONS (voice interview tracking)
-- ============================================
CREATE TABLE interview_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    application_id UUID REFERENCES applications(id) ON DELETE CASCADE NOT NULL,
    mode VARCHAR(50) NOT NULL CHECK (mode IN ('assist', 'autonomous')),
    status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'starting', 'active', 'completed', 'failed', 'cancelled')),
    external_session_id VARCHAR(255),
    provider VARCHAR(100) NOT NULL DEFAULT '',
    model VARCHAR(100) NOT NULL DEFAULT '',
    transcript JSONB DEFAULT '[]'::jsonb,
    score DOUBLE PRECISION,
    feedback JSONB,
    started_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_interview_sessions_application ON interview_sessions(application_id);
CREATE INDEX idx_interview_sessions_status ON interview_sessions(status);
CREATE INDEX idx_interview_sessions_external_session ON interview_sessions(external_session_id) WHERE external_session_id IS NOT NULL;
CREATE INDEX idx_interview_sessions_created ON interview_sessions(created_at);
