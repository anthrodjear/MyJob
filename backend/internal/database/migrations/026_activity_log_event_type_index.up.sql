-- ============================================
-- 026: Index on activity_log.event_type
-- ============================================
-- The activity feed handler exposes GET /activity-logs?event_type=...,
-- but the activity_log table has no index on event_type, causing
-- sequential scans on every filtered query.

CREATE INDEX IF NOT EXISTS idx_activity_log_event_type ON activity_log(event_type);
