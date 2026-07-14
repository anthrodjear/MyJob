package tasks

import (
	"context"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// taskColumns lists all columns for SELECT queries to avoid SELECT *.
const taskColumns = `
	id, type, status, params, result, error, attempts, max_attempts,
	priority, scheduled_at, started_at, completed_at, created_at, updated_at
`

// Repository handles database operations for tasks.
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates a new tasks repository.
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new task into the database.
func (r *Repository) Create(ctx context.Context, task *Task) error {
	query := `
		INSERT INTO tasks (id, type, status, params, max_attempts, priority, scheduled_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	now := time.Now()
	if task.ScheduledAt.IsZero() {
		task.ScheduledAt = now
	}
	_, err := r.db.ExecContext(ctx, query,
		task.ID, task.Type, task.Status, task.Params,
		task.MaxAttempts, task.Priority, task.ScheduledAt, now, now,
	)
	return err
}

// GetByID retrieves a task by its ID.
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Task, error) {
	var task Task
	query := `SELECT ` + taskColumns + ` FROM tasks WHERE id = $1`
	err := r.db.GetContext(ctx, &task, query, id)
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// Update updates a task's mutable fields.
func (r *Repository) Update(ctx context.Context, task *Task) error {
	query := `
		UPDATE tasks
		SET status = $2, result = $3, error = $4, attempts = $5,
		    started_at = $6, completed_at = $7, updated_at = $8
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query,
		task.ID, task.Status, task.Result, task.Error,
		task.Attempts, task.StartedAt, task.CompletedAt, time.Now(),
	)
	return err
}

// List returns tasks filtered by status and/or type, ordered by priority desc, scheduled_at asc.
func (r *Repository) List(ctx context.Context, status, taskType string, limit, offset int) ([]Task, int, error) {
	var tasks []Task
	var total int

	// Count query
	countQuery := `SELECT COUNT(*) FROM tasks WHERE 1=1`
	countArgs := []interface{}{}
	argIdx := 1

	if status != "" {
		countQuery += ` AND status = ` + ph(argIdx)
		countArgs = append(countArgs, status)
		argIdx++
	}
	if taskType != "" {
		countQuery += ` AND type = ` + ph(argIdx)
		countArgs = append(countArgs, taskType)
	}

	err := r.db.GetContext(ctx, &total, countQuery, countArgs...)
	if err != nil {
		return nil, 0, err
	}

	// Data query
	dataQuery := `SELECT ` + taskColumns + ` FROM tasks WHERE 1=1`
	dataArgs := []interface{}{}
	argIdx = 1

	if status != "" {
		dataQuery += ` AND status = ` + ph(argIdx)
		dataArgs = append(dataArgs, status)
		argIdx++
	}
	if taskType != "" {
		dataQuery += ` AND type = ` + ph(argIdx)
		dataArgs = append(dataArgs, taskType)
		argIdx++
	}

	dataQuery += ` ORDER BY priority DESC, scheduled_at ASC`
	dataQuery += ` LIMIT ` + ph(argIdx)
	dataArgs = append(dataArgs, limit)
	argIdx++
	dataQuery += ` OFFSET ` + ph(argIdx)
	dataArgs = append(dataArgs, offset)

	err = r.db.SelectContext(ctx, &tasks, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

// GetPendingByType returns the next pending task of a given type (for worker polling).
func (r *Repository) GetPendingByType(ctx context.Context, taskType string) (*Task, error) {
	var task Task
	query := `
	
		SELECT ` + taskColumns + ` FROM tasks
		WHERE status = $1 AND type = $2 AND scheduled_at <= NOW()
		ORDER BY priority DESC, scheduled_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	`
	err := r.db.GetContext(ctx, &task, query, StatusPending, taskType)
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// ph returns a SQL placeholder like "$1", "$2", etc.
func ph(i int) string {
	return "$" + strconv.Itoa(i)
}
