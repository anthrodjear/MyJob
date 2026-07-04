-- Add reasoning and model to jobs table for LLM scoring metadata
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS scoring_reasoning TEXT;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS scoring_model VARCHAR(100);
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS scoring_source VARCHAR(20);
