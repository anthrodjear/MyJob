package activity

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository defines data access for activity logs.
// Extracted as an interface for testability — mock this in unit tests.
//
// The repository is intentionally minimal: append-only writes and filtered reads.
// No update or delete methods — activity logs are immutable once created.
type Repository interface {
	// GetByID returns a single activity log by its primary key.
	// Returns ErrNotFound if no matching row exists.
	GetByID(ctx context.Context, id uuid.UUID) (*ActivityLog, error)

	// List returns activity logs matching the filter, ordered by created_at DESC.
	// Returns the matching logs and total count (before pagination).
	List(ctx context.Context, filter ListFilter) ([]ActivityLog, int64, error)

	// Create inserts a new activity log.
	// Sets ID and CreatedAt if zero values are provided.
	Create(ctx context.Context, a *ActivityLog) error
}

// PostgresRepository implements Repository with sqlx.
// All queries use the shared activityColumns constant (no SELECT *).
type PostgresRepository struct {
	db *sqlx.DB
}

// NewRepository creates a new PostgresRepository.
func NewRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// GetByID fetches an activity log by ID.
// Translates sql.ErrNoRows to the domain ErrNotFound sentinel.
func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*ActivityLog, error) {
	var a ActivityLog
	err := r.db.GetContext(ctx, &a,
		`SELECT `+activityColumns+`
		 FROM activity_log WHERE id = $1`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get activity by id: %w", err)
	}
	return &a, nil
}

// ListFilter defines filters for listing activity logs.
// Zero values mean "no filter" — all fields are optional.
// EntityType and EventType are exact-match strings.
// EntityID filters to a specific entity UUID.
// StartTime and EndTime define a time range (inclusive).
type ListFilter struct {
	EntityType string
	EntityID   uuid.UUID
	EventType  string
	StartTime  time.Time
	EndTime    time.Time
	Limit      int
	Offset     int
}

// buildWhere constructs the WHERE clause, args, and next arg index from a filter.
// Uses parameterized queries ($1, $2, ...) to prevent SQL injection.
// Returns empty string when no filters are applied (matches all rows).
func (f ListFilter) buildWhere() (string, []interface{}, int) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if f.EntityType != "" {
		conditions = append(conditions, fmt.Sprintf("entity_type = $%d", argIdx))
		args = append(args, f.EntityType)
		argIdx++
	}
	if f.EntityID != uuid.Nil {
		conditions = append(conditions, fmt.Sprintf("entity_id = $%d", argIdx))
		args = append(args, f.EntityID)
		argIdx++
	}
	if f.EventType != "" {
		conditions = append(conditions, fmt.Sprintf("event_type = $%d", argIdx))
		args = append(args, f.EventType)
		argIdx++
	}
	if !f.StartTime.IsZero() {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, f.StartTime)
		argIdx++
	}
	if !f.EndTime.IsZero() {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIdx))
		args = append(args, f.EndTime)
		argIdx++
	}

	if len(conditions) == 0 {
		return "", nil, argIdx
	}
	return " WHERE " + strings.Join(conditions, " AND "), args, argIdx
}

// List returns activity logs matching the filter with total count.
// Results are ordered by created_at DESC (newest first).
// Count is computed separately to support pagination metadata.
func (r *PostgresRepository) List(ctx context.Context, filter ListFilter) ([]ActivityLog, int64, error) {
	where, args, argIdx := filter.buildWhere()

	// Count total matching rows (before pagination)
	var total int64
	countQuery := "SELECT COUNT(*) FROM activity_log" + where
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("count activities: %w", err)
	}

	// Fetch paginated page
	query := `SELECT ` + activityColumns + ` FROM activity_log` + where + " ORDER BY created_at DESC"
	if filter.Limit > 0 {
		args = append(args, filter.Limit)
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		argIdx++
	}
	if filter.Offset > 0 {
		args = append(args, filter.Offset)
		query += fmt.Sprintf(" OFFSET $%d", argIdx)
	}

	var activities []ActivityLog
	if err := r.db.SelectContext(ctx, &activities, query, args...); err != nil {
		return nil, 0, fmt.Errorf("list activities: %w", err)
	}
	return activities, total, nil
}

// Create inserts a new activity log.
// Auto-generates ID (uuid.New()) and CreatedAt (time.Now()) if zero values.
// This is an append-only operation — no update or delete methods exist.
func (r *PostgresRepository) Create(ctx context.Context, a *ActivityLog) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now()
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO activity_log (id, event_type, entity_type, entity_id, details, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		a.ID, a.EventType, a.EntityType, a.EntityID, a.Details, a.CreatedAt)
	if err != nil {
		return fmt.Errorf("create activity: %w", err)
	}
	return nil
}
