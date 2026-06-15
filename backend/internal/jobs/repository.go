package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ErrNoRowsAffected is returned when an UPDATE affects zero rows (resource not found).
var ErrNoRowsAffected = errors.New("jobs: no rows affected")

// Repository provides database access for jobs.
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates a new jobs repository.
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// GetByID retrieves a job by ID.
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Job, error) {
	var job Job
	err := r.db.GetContext(ctx, &job, `
		SELECT
			j.id, j.source_id, j.external_id, j.title, j.company, j.location,
			j.remote_type, j.salary_min, j.salary_max, j.salary_currency,
			j.description, j.requirements, j.url, j.company_url,
			j.posted_at, j.scraped_at, j.match_score, j.match_details,
			j.status, j.created_at, j.updated_at,
			s.name as source_name
		FROM jobs j
		LEFT JOIN job_sources s ON j.source_id = s.id
		WHERE j.id = $1
	`, id)
	if err != nil {
		return nil, fmt.Errorf("jobs: get by id: %w", err)
	}
	return &job, nil
}

// List retrieves jobs with filtering and pagination.
// Returns both the job slice and total count for pagination.
func (r *Repository) List(ctx context.Context, filter ListFilter) ([]Job, int, error) {
	whereClause, args := filter.buildWhere()

	// Count total
	var total int
	countQuery := `SELECT COUNT(*) FROM jobs j ` + whereClause
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("jobs: count: %w", err)
	}

	// List jobs
	query := `
		SELECT
			j.id, j.source_id, j.external_id, j.title, j.company, j.location,
			j.remote_type, j.salary_min, j.salary_max, j.salary_currency,
			j.description, j.requirements, j.url, j.company_url,
			j.posted_at, j.scraped_at, j.match_score, j.match_details,
			j.status, j.created_at, j.updated_at,
			s.name as source_name
		FROM jobs j
		LEFT JOIN job_sources s ON j.source_id = s.id
	` + whereClause + `
		ORDER BY j.scraped_at DESC
		LIMIT $` + fmt.Sprintf("%d", len(args)+1) + ` OFFSET $` + fmt.Sprintf("%d", len(args)+2)

	args = append(args, filter.Limit, filter.Offset)

	var jobs []Job
	err = r.db.SelectContext(ctx, &jobs, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("jobs: list: %w", err)
	}

	return jobs, total, nil
}

// Create inserts a single job.
func (r *Repository) Create(ctx context.Context, job *Job) error {
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO jobs (
			id, source_id, external_id, title, company, location,
			remote_type, salary_min, salary_max, salary_currency,
			description, requirements, url, company_url,
			posted_at, scraped_at, match_score, match_details,
			status, created_at, updated_at
		) VALUES (
			:id, :source_id, :external_id, :title, :company, :location,
			:remote_type, :salary_min, :salary_max, :salary_currency,
			:description, :requirements, :url, :company_url,
			:posted_at, :scraped_at, :match_score, :match_details,
			:status, :created_at, :updated_at
		)
		ON CONFLICT (source_id, external_id) DO NOTHING
	`, job)
	if err != nil {
		return fmt.Errorf("jobs: create: %w", err)
	}
	return nil
}

// BulkCreate inserts multiple jobs in a single transaction.
// Uses ON CONFLICT to skip duplicates.
func (r *Repository) BulkCreate(ctx context.Context, jobs []*Job) (int, error) {
	if len(jobs) == 0 {
		return 0, nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("jobs: bulk create begin: %w", err)
	}
	defer tx.Rollback()

	imported := 0
	for _, job := range jobs {
		result, err := tx.NamedExecContext(ctx, `
			INSERT INTO jobs (
				id, source_id, external_id, title, company, location,
				remote_type, salary_min, salary_max, salary_currency,
				description, requirements, url, company_url,
				posted_at, scraped_at, match_score, match_details,
				status, created_at, updated_at
			) VALUES (
				:id, :source_id, :external_id, :title, :company, :location,
				:remote_type, :salary_min, :salary_max, :salary_currency,
				:description, :requirements, :url, :company_url,
				:posted_at, :scraped_at, :match_score, :match_details,
				:status, :created_at, :updated_at
			)
			ON CONFLICT (source_id, external_id) DO NOTHING
		`, job)
		if err != nil {
			return 0, fmt.Errorf("jobs: bulk create exec: %w", err)
		}

		rows, err := result.RowsAffected()
		if err != nil {
			return 0, fmt.Errorf("jobs: bulk create rows: %w", err)
		}
		imported += int(rows)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("jobs: bulk create commit: %w", err)
	}

	return imported, nil
}

// ExistsBySourceAndExternalID checks if a job already exists.
func (r *Repository) ExistsBySourceAndExternalID(ctx context.Context, sourceID uuid.UUID, externalID string) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists, `
		SELECT EXISTS(
			SELECT 1 FROM jobs
			WHERE source_id = $1 AND external_id = $2
		)
	`, sourceID, externalID)
	if err != nil {
		return false, fmt.Errorf("jobs: exists: %w", err)
	}
	return exists, nil
}

// UpdateStatus updates a job's status.
func (r *Repository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE jobs
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`, status, id)
	if err != nil {
		return fmt.Errorf("jobs: update status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("jobs: update status rows: %w", err)
	}
	if rows == 0 {
		return ErrNoRowsAffected
	}

	return nil
}

// UpdateMatchScore updates a job's match score and details.
func (r *Repository) UpdateMatchScore(ctx context.Context, id uuid.UUID, score float64, details json.RawMessage) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE jobs
		SET match_score = $1, match_details = $2, updated_at = NOW()
		WHERE id = $3
	`, score, details, id)
	if err != nil {
		return fmt.Errorf("jobs: update match score: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("jobs: update match score rows: %w", err)
	}
	if rows == 0 {
		return ErrNoRowsAffected
	}

	return nil
}

// ListFilter holds the filter criteria for listing jobs.
type ListFilter struct {
	Status   string
	Company  string
	SourceID uuid.UUID
	MinScore float64
	Limit    int
	Offset   int
}

// buildWhere constructs the WHERE clause and arguments for filtering.
func (f *ListFilter) buildWhere() (string, []interface{}) {
	where := "WHERE 1=1"
	args := []interface{}{}

	if f.Status != "" {
		where += " AND j.status = $" + fmt.Sprintf("%d", len(args)+1)
		args = append(args, f.Status)
	}
	if f.Company != "" {
		where += " AND j.company ILIKE $" + fmt.Sprintf("%d", len(args)+1)
		args = append(args, "%"+f.Company+"%")
	}
	if f.SourceID != uuid.Nil {
		where += " AND j.source_id = $" + fmt.Sprintf("%d", len(args)+1)
		args = append(args, f.SourceID)
	}
	if f.MinScore > 0 {
		where += " AND j.match_score >= $" + fmt.Sprintf("%d", len(args)+1)
		args = append(args, f.MinScore)
	}

	return where, args
}
