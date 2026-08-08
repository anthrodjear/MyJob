-- ============================================
-- 024: Add UNIQUE constraint on embeddings(source_type, source_id)
-- ============================================
-- The ON CONFLICT (source_type, source_id) clause in the embedding_generate
-- worker handler requires a unique constraint or unique index. Without it,
-- every upsert fails at runtime with:
--   "there is no unique or exclusion constraint matching the ON CONFLICT specification"
-- This migration deduplicates existing rows first (keeping the most recent
-- per source_type/source_id), then adds the unique index.

BEGIN;

-- Deduplicate: keep the most recent row per (source_type, source_id)
-- Uses ctid because the table has no surrogate order column we can rely on;
-- pairs with the later ORDER BY created_at DESC to pick the latest.
DELETE FROM embeddings a
USING (
    SELECT ctid, ROW_NUMBER() OVER (
        PARTITION BY source_type, source_id
        ORDER BY created_at DESC, ctid DESC
    ) AS rn
    FROM embeddings
) b
WHERE a.ctid = b.ctid AND b.rn > 1;

-- Now safe to enforce uniqueness
CREATE UNIQUE INDEX IF NOT EXISTS idx_embeddings_source_unique
    ON embeddings(source_type, source_id);

COMMIT;
