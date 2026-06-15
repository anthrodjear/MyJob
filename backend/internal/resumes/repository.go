package resumes

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// Domain errors for resumes.
var (
	ErrNotFound = errors.New("resume not found")
)

// Repository handles database operations for resumes and cover letters.
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates a new resumes repository.
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// --- Resume methods ---

// GetByID fetches a resume by ID.
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Resume, error) {
	var resume Resume
	err := r.db.GetContext(ctx, &resume,
		`SELECT id, name, specialization, template_path, focus_skills, highlight_experience,
		        pdf_key, version, created_at, updated_at
		 FROM resumes WHERE id = $1`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get resume by id: %w", err)
	}
	return &resume, nil
}

// List returns resumes ordered by created_at DESC.
func (r *Repository) List(ctx context.Context, limit, offset int) ([]*Resume, int64, error) {
	var total int64
	if err := r.db.GetContext(ctx, &total, "SELECT COUNT(*) FROM resumes"); err != nil {
		return nil, 0, fmt.Errorf("count resumes: %w", err)
	}

	var resumes []*Resume
	err := r.db.SelectContext(ctx, &resumes,
		`SELECT id, name, specialization, template_path, focus_skills, highlight_experience,
		        pdf_key, version, created_at, updated_at
		 FROM resumes ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list resumes: %w", err)
	}
	return resumes, total, nil
}

// Create inserts a new resume and returns the DB-assigned values.
func (r *Repository) Create(ctx context.Context, resume *Resume) error {
	return r.db.QueryRowxContext(ctx,
		`INSERT INTO resumes (id, name, specialization, template_path, focus_skills, highlight_experience,
		                      pdf_key, version, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 RETURNING version, created_at, updated_at`,
		resume.ID, resume.Name, resume.Specialization, resume.TemplatePath,
		pq.StringArray(resume.FocusSkills), pq.Array(resume.HighlightExperience),
		resume.PdfKey, resume.Version, resume.CreatedAt, resume.UpdatedAt,
	).Scan(&resume.Version, &resume.CreatedAt, &resume.UpdatedAt)
}

// Update updates an existing resume with optimistic locking.
func (r *Repository) Update(ctx context.Context, resume *Resume) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE resumes SET name = $1, specialization = $2, template_path = $3,
		        focus_skills = $4, highlight_experience = $5, pdf_key = $6,
		        version = version + 1, updated_at = NOW()
		 WHERE id = $7 AND version = $8`,
		resume.Name, resume.Specialization, resume.TemplatePath,
		pq.StringArray(resume.FocusSkills), pq.Array(resume.HighlightExperience),
		resume.PdfKey, resume.ID, resume.Version)
	if err != nil {
		return fmt.Errorf("update resume: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	// Fetch updated values
	return r.db.QueryRowContext(ctx,
		`SELECT version, updated_at FROM resumes WHERE id = $1`, resume.ID,
	).Scan(&resume.Version, &resume.UpdatedAt)
}

// Delete deletes a resume by ID.
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM resumes WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete resume: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// --- Cover Letter methods ---

// GetCoverLetterByID fetches a cover letter by ID.
func (r *Repository) GetCoverLetterByID(ctx context.Context, id uuid.UUID) (*CoverLetter, error) {
	var cl CoverLetter
	err := r.db.GetContext(ctx, &cl,
		`SELECT id, job_id, resume_id, content, pdf_key, word_count, version, created_at, updated_at
		 FROM cover_letters WHERE id = $1`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get cover letter by id: %w", err)
	}
	return &cl, nil
}

// ListCoverLetters returns cover letters ordered by created_at DESC.
func (r *Repository) ListCoverLetters(ctx context.Context, limit, offset int) ([]*CoverLetter, int64, error) {
	var total int64
	if err := r.db.GetContext(ctx, &total, "SELECT COUNT(*) FROM cover_letters"); err != nil {
		return nil, 0, fmt.Errorf("count cover letters: %w", err)
	}

	var letters []*CoverLetter
	err := r.db.SelectContext(ctx, &letters,
		`SELECT id, job_id, resume_id, content, pdf_key, word_count, version, created_at, updated_at
		 FROM cover_letters ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list cover letters: %w", err)
	}
	return letters, total, nil
}

// CreateCoverLetter inserts a new cover letter and returns the DB-assigned values.
func (r *Repository) CreateCoverLetter(ctx context.Context, cl *CoverLetter) error {
	return r.db.QueryRowxContext(ctx,
		`INSERT INTO cover_letters (id, job_id, resume_id, content, pdf_key, word_count, version, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING version, created_at, updated_at`,
		cl.ID, cl.JobID, cl.ResumeID, cl.Content, cl.PdfKey, cl.WordCount,
		cl.Version, cl.CreatedAt, cl.UpdatedAt,
	).Scan(&cl.Version, &cl.CreatedAt, &cl.UpdatedAt)
}

// DeleteCoverLetter deletes a cover letter by ID.
func (r *Repository) DeleteCoverLetter(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM cover_letters WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete cover letter: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}
