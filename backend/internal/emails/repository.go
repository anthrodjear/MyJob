// Repository handles database operations for emails.
//
// Responsibilities:
//   - CRUD operations on the emails table
//   - Filtering and pagination for list queries
//   - Upsert by message_id (deduplication)
//
// This file contains NO business logic. It translates SQL errors to
// domain errors (ErrNotFound) for the service layer.
//
// Rules followed:
//   - No SELECT * — columns listed explicitly
//   - Parameterized queries only — no string interpolation
//   - Errors wrapped with context for debugging
package emails

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ============================================================================
// Domain Errors
// ============================================================================
//
// ErrNotFound is defined in model.go (same package).

// ============================================================================
// Repository
// ============================================================================

// Repository handles database operations for emails.
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates a new emails repository.
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// ---------------------------------------------------------------------------
// Queries: Read
// ---------------------------------------------------------------------------

// GetByID fetches an email by ID.
// Returns ErrNotFound if no matching row exists.
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Email, error) {
	var e Email
	err := r.db.GetContext(ctx, &e,
		`SELECT `+emailColumns+`
		 FROM emails WHERE id = $1`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get email by id: %w", err)
	}
	return &e, nil
}

// GetByMessageID fetches an email by its external message ID.
// Returns ErrNotFound if no matching row exists.
func (r *Repository) GetByMessageID(ctx context.Context, messageID string) (*Email, error) {
	var e Email
	err := r.db.GetContext(ctx, &e,
		`SELECT `+emailColumns+`
		 FROM emails WHERE message_id = $1`, messageID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get email by message_id: %w", err)
	}
	return &e, nil
}

// ListFilter defines filters for listing emails.
type ListFilter struct {
	ApplicationID  uuid.UUID
	Classification string
	Limit          int
	Offset         int
}

// buildWhere builds the WHERE clause and args from a filter.
func (f ListFilter) buildWhere() (string, []interface{}, int) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if f.ApplicationID != uuid.Nil {
		conditions = append(conditions, fmt.Sprintf("application_id = $%d", argIdx))
		args = append(args, f.ApplicationID)
		argIdx++
	}
	if f.Classification != "" {
		conditions = append(conditions, fmt.Sprintf("classification = $%d", argIdx))
		args = append(args, f.Classification)
		argIdx++
	}

	if len(conditions) == 0 {
		return "", nil, argIdx
	}
	return " WHERE " + strings.Join(conditions, " AND "), args, argIdx
}

// List returns emails matching the filter with total count.
func (r *Repository) List(ctx context.Context, filter ListFilter) ([]Email, int64, error) {
	where, args, argIdx := filter.buildWhere()

	// Count total
	var total int64
	countQuery := "SELECT COUNT(*) FROM emails" + where
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("count emails: %w", err)
	}

	// Fetch page
	// LIMIT/OFFSET use fmt.Sprintf with parameter index because sqlx doesn't support
	// parameterized LIMIT/OFFSET clauses. The argIdx tracks the next parameter position.
	query := `SELECT ` + emailColumns + ` FROM emails` + where + " ORDER BY received_at DESC"
	if filter.Limit > 0 {
		args = append(args, filter.Limit)
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		argIdx++
	}
	if filter.Offset > 0 {
		args = append(args, filter.Offset)
		query += fmt.Sprintf(" OFFSET $%d", argIdx)
	}

	var emails []Email
	if err := r.db.SelectContext(ctx, &emails, query, args...); err != nil {
		return nil, 0, fmt.Errorf("list emails: %w", err)
	}
	return emails, total, nil
}

// ---------------------------------------------------------------------------
// Queries: Write
// ---------------------------------------------------------------------------

// Upsert inserts or updates an email by message_id.
// If an email with the same message_id exists, updates its fields.
// Returns the email ID (existing or newly created).
func (r *Repository) Upsert(ctx context.Context, e *Email) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRowxContext(ctx,
		`INSERT INTO emails (id, application_id, message_id, from_address, to_address,
		                     subject, body, received_at, classification, is_read, reply_draft)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 ON CONFLICT (message_id) DO UPDATE
			SET application_id  = COALESCE(EXCLUDED.application_id, emails.application_id),
				from_address    = EXCLUDED.from_address,
				to_address      = COALESCE(EXCLUDED.to_address, emails.to_address),
				subject         = COALESCE(EXCLUDED.subject, emails.subject),
				body            = COALESCE(EXCLUDED.body, emails.body),
				classification  = COALESCE(EXCLUDED.classification, emails.classification)
		 RETURNING id`,
		e.ID, e.ApplicationID, e.MessageID, e.FromAddress, e.ToAddress,
		e.Subject, e.Body, e.ReceivedAt, e.Classification, e.IsRead, e.ReplyDraft,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("upsert email: %w", err)
	}
	return id, nil
}

// execAndCheckRows executes an UPDATE query and returns ErrNotFound if no rows affected.
// Used by update methods to avoid code duplication.
func (r *Repository) execAndCheckRows(ctx context.Context, operation string, query string, args ...interface{}) error {
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", operation, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: get rows affected: %w", operation, err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateReadStatus updates the is_read flag for an email.
func (r *Repository) UpdateReadStatus(ctx context.Context, id uuid.UUID, isRead bool) error {
	return r.execAndCheckRows(ctx, "update read status",
		`UPDATE emails SET is_read = $1 WHERE id = $2`, isRead, id)
}

// UpdateClassification updates the classification for an email.
func (r *Repository) UpdateClassification(ctx context.Context, id uuid.UUID, classification string) error {
	return r.execAndCheckRows(ctx, "update classification",
		`UPDATE emails SET classification = $1 WHERE id = $2`, classification, id)
}

// UpdateReplyDraft updates the reply draft text for an email.
func (r *Repository) UpdateReplyDraft(ctx context.Context, id uuid.UUID, draft *string) error {
	return r.execAndCheckRows(ctx, "update reply draft",
		`UPDATE emails SET reply_draft = $1 WHERE id = $2`, draft, id)
}
