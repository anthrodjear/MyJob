package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

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
			j.description, j.requirements, j.url, j.application_url, j.company_url, j.source,
			j.posted_at, j.scraped_at, j.match_score, j.match_details,
			j.score_tier, j.scored_at, j.scoring_reasoning, j.scoring_model, j.scoring_source,
			j.saved, j.metadata,
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
			j.description, j.requirements, j.url, j.application_url, j.company_url, j.source,
			j.posted_at, j.scraped_at, j.match_score, j.match_details,
			j.score_tier, j.scored_at, j.scoring_reasoning, j.scoring_model, j.scoring_source,
			j.saved, j.metadata,
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
			description, requirements, url, application_url, company_url, source,
			posted_at, scraped_at, match_score, match_details,
			status, created_at, updated_at
		) VALUES (
			:id, :source_id, :external_id, :title, :company, :location,
			:remote_type, :salary_min, :salary_max, :salary_currency,
			:description, :requirements, :url, :application_url, :company_url, :source,
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
				description, requirements, url, application_url, company_url, source,
				posted_at, scraped_at, match_score, match_details,
				status, created_at, updated_at
			) VALUES (
				:id, :source_id, :external_id, :title, :company, :location,
				:remote_type, :salary_min, :salary_max, :salary_currency,
				:description, :requirements, :url, :application_url, :company_url, :source,
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

// GetSourceNameByID returns the source name for a given job_sources UUID.
func (r *Repository) GetSourceNameByID(ctx context.Context, id uuid.UUID) (string, error) {
	var name string
	err := r.db.GetContext(ctx, &name, `SELECT name FROM job_sources WHERE id = $1`, id)
	if err != nil {
		return "", fmt.Errorf("jobs: get source name: %w", err)
	}
	return name, nil
}

// GetSourceBaseURLByID returns the base_url for a given job_sources UUID.
func (r *Repository) GetSourceBaseURLByID(ctx context.Context, id uuid.UUID) (string, error) {
	var baseURL string
	err := r.db.GetContext(ctx, &baseURL, `SELECT base_url FROM job_sources WHERE id = $1`, id)
	if err != nil {
		return "", fmt.Errorf("jobs: get source base_url: %w", err)
	}
	return baseURL, nil
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

// SetSaved toggles the saved flag on a job. Returns ErrNoRowsAffected
// when the job does not exist so the caller can return 404.
func (r *Repository) SetSaved(ctx context.Context, id uuid.UUID, saved bool) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE jobs
		SET saved = $1, updated_at = NOW()
		WHERE id = $2
	`, saved, id)
	if err != nil {
		return fmt.Errorf("jobs: set saved: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("jobs: set saved rows: %w", err)
	}
	if rows == 0 {
		return ErrNoRowsAffected
	}
	return nil
}

// Delete removes a job by ID. Returns ErrNoRowsAffected if the job
// doesn't exist (so the handler can decide to return 404 or treat the
// call as idempotent).
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM jobs WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("jobs: delete: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("jobs: delete rows: %w", err)
	}
	if rows == 0 {
		return ErrNoRowsAffected
	}
	return nil
}

// FindSimilar returns up to `limit` jobs whose title shares keywords with
// the source job's title. Uses simple ILIKE matching on individual title
// tokens — adequate for the basic "similar jobs" affordance the frontend
// needs; a vector similarity search via the embeddings table would be a
// stronger replacement later.
func (r *Repository) FindSimilar(ctx context.Context, id uuid.UUID, limit int) ([]Job, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	source, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Extract keywords (length >= 3) from the source title. We ILIKE-match
	// each keyword against other titles with an OR group, capped at limit.
	keywords := extractTitleKeywords(source.Title)
	if len(keywords) == 0 {
		return []Job{}, nil
	}

	conditions := make([]string, 0, len(keywords))
	args := make([]interface{}, 0, len(keywords)+2)
	for _, kw := range keywords {
		conditions = append(conditions, fmt.Sprintf("j.title ILIKE $%d", len(args)+1))
		args = append(args, "%"+kw+"%")
	}
	args = append(args, id, limit)

	where := "WHERE j.id <> $" + fmt.Sprintf("%d", len(args)-1) +
		" AND (" + strings.Join(conditions, " OR ") + ")"

	query := `
		SELECT
			j.id, j.source_id, j.external_id, j.title, j.company, j.location,
			j.remote_type, j.salary_min, j.salary_max, j.salary_currency,
			j.description, j.requirements, j.url, j.application_url, j.company_url, j.source,
			j.posted_at, j.scraped_at, j.match_score, j.match_details,
			j.score_tier, j.scored_at, j.scoring_reasoning, j.scoring_model, j.scoring_source,
			j.saved, j.metadata,
			j.status, j.created_at, j.updated_at,
			s.name as source_name
		FROM jobs j
		LEFT JOIN job_sources s ON j.source_id = s.id
	` + where + `
		ORDER BY j.scraped_at DESC
		LIMIT $` + fmt.Sprintf("%d", len(args))

	var similar []Job
	if err := r.db.SelectContext(ctx, &similar, query, args...); err != nil {
		return nil, fmt.Errorf("jobs: find similar: %w", err)
	}
	return similar, nil
}

// extractTitleKeywords pulls out alpha-numeric tokens of length >= 3 from
// a job title. Used by FindSimilar to build a coarse ILIKE predicate.
func extractTitleKeywords(title string) []string {
	parts := strings.FieldsFunc(title, func(r rune) bool {
		switch r {
		case ' ', '\t', '\n', '\r', ',', '.', '/', '\\', '-', '(', ')', '[', ']', ':', ';', '|', '+', '&':
			return true
		}
		return false
	})
	seen := make(map[string]struct{}, len(parts))
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if len(p) < 3 {
			continue
		}
		lower := strings.ToLower(p)
		if _, ok := seen[lower]; ok {
			continue
		}
		seen[lower] = struct{}{}
		out = append(out, lower)
	}
	return out
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
