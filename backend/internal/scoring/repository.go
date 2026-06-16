// Package scoring provides job-candidate matching and scoring functionality.
// It supports three scoring modes: heuristic (keyword-based), LLM (semantic), and hybrid (pre-filter + LLM).
// The service computes factor scores (skills, experience, location, salary, description) and combines them
// into a final 0-100 score with approval tier (auto/review/reject).
package scoring

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var (
	ErrNotFound = errors.New("scoring: not found")
)

// Repository defines data access for scoring.
// Extracted for testability — mock this in unit tests.
type Repository interface {
	GetJob(ctx context.Context, id uuid.UUID) (JobData, error)
	GetProfile(ctx context.Context) (Profile, error)
	PersistScore(ctx context.Context, jobID uuid.UUID, score float64, tier string, details json.RawMessage, reasoning, model, source string) error
}

// PostgresRepository implements Repository with sqlx.
type PostgresRepository struct {
	db *sqlx.DB
}

// NewRepository creates a new PostgresRepository.
func NewRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// jobData holds the job fields needed for scoring.
type JobData struct {
	ID               uuid.UUID       `db:"id"`
	Title            string          `db:"title"`
	Company          string          `db:"company"`
	Description      string          `db:"description"`
	Requirements     string          `db:"requirements"`
	Location         string          `db:"location"`
	RemoteType       string          `db:"remote_type"`
	SalaryMin        int             `db:"salary_min"`
	SalaryMax        int             `db:"salary_max"`
	MatchScore       float64         `db:"match_score"`
	ScoreTier        string          `db:"score_tier"`
	MatchDetails     json.RawMessage `db:"match_details"`
	ScoringReasoning string          `db:"scoring_reasoning"`
	ScoringModel     string          `db:"scoring_model"`
	ScoringSource    string          `db:"scoring_source"`
}

// GetJob fetches job data by ID.
func (r *PostgresRepository) GetJob(ctx context.Context, id uuid.UUID) (JobData, error) {
	var job JobData
	err := r.db.QueryRowxContext(ctx,
		"SELECT id, title, company, description, requirements, location, remote_type, salary_min, salary_max, match_score, score_tier, match_details, scoring_reasoning, scoring_model, scoring_source FROM jobs WHERE id = $1",
		id,
	).StructScan(&job)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return job, ErrNotFound
		}
		return job, err
	}
	return job, nil
}

// GetProfile fetches the user's master profile.
// Single-user local-first app: one profile, no user_id column.
func (r *PostgresRepository) GetProfile(ctx context.Context) (Profile, error) {
	var profile Profile
	var data json.RawMessage
	err := r.db.QueryRowxContext(ctx,
		"SELECT data FROM profiles LIMIT 1",
	).Scan(&data)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return profile, nil
		}
		return profile, err
	}
	if err := json.Unmarshal(data, &profile); err != nil {
		return profile, fmt.Errorf("unmarshal profile: %w", err)
	}
	return profile, nil
}

// PersistScore saves the score, tier, and details to the jobs table.
func (r *PostgresRepository) PersistScore(ctx context.Context, jobID uuid.UUID, score float64, tier string, details json.RawMessage, reasoning, model, source string) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE jobs
		 SET match_score = $1, score_tier = $2, match_details = $3, scored_at = NOW(), updated_at = NOW(),
		     scoring_reasoning = $4, scoring_model = $5, scoring_source = $6
		 WHERE id = $7`,
		score, tier, details, reasoning, model, source, jobID,
	)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
