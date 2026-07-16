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

// resumeColumns lists all resume columns for SELECT queries.
const resumeColumns = `id, name, specialization, template_path, focus_skills, highlight_experience,
                       content, pdf_key, version, created_at, updated_at`

// coverLetterColumns lists all cover letter columns for SELECT queries.
const coverLetterColumns = `id, job_id, resume_id, job_title, content, model, prompt_version,
                            resume_version, pdf_key, strengths, gaps, word_count, version,
                            created_at, updated_at`

// Repository defines the interface for resume and cover letter data access.
// Used for testability — mock this in unit tests.
type Repository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*Resume, error)
	List(ctx context.Context, limit, offset int) ([]*Resume, int64, error)
	Create(ctx context.Context, resume *Resume) error
	Update(ctx context.Context, resume *Resume) error
	UpdateContent(ctx context.Context, id uuid.UUID, content ResumeContentDB, currentVersion int32) (int32, error)
	UpdatePdfKey(ctx context.Context, id uuid.UUID, pdfKey string) error
	Delete(ctx context.Context, id uuid.UUID) error
	SaveVersion(ctx context.Context, v *ResumeVersion) error
	GetVersions(ctx context.Context, resumeID uuid.UUID) ([]*ResumeVersion, error)
	GetVersion(ctx context.Context, resumeID uuid.UUID, version int32) (*ResumeVersion, error)
	GetCoverLetterByID(ctx context.Context, id uuid.UUID) (*CoverLetter, error)
	ListCoverLetters(ctx context.Context, limit, offset int) ([]*CoverLetter, int64, error)
	CreateCoverLetter(ctx context.Context, cl *CoverLetter) error
	UpdateCoverLetterContent(ctx context.Context, id uuid.UUID, content string, model, promptVersion *string, resumeVersion *int32, strengths, gaps *StringSliceDB, wordCount *int, currentVersion int32) (int32, error)
	UpdateCoverLetterPdfKey(ctx context.Context, id uuid.UUID, pdfKey string) error
	DeleteCoverLetter(ctx context.Context, id uuid.UUID) error
	SaveCoverLetterVersion(ctx context.Context, v *CoverLetterVersion) error
	GetCoverLetterVersions(ctx context.Context, coverLetterID uuid.UUID) ([]*CoverLetterVersion, error)
	GetCoverLetterVersion(ctx context.Context, coverLetterID uuid.UUID, version int32) (*CoverLetterVersion, error)
}

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	db *sqlx.DB
}

// NewRepository creates a new resumes repository.
func NewRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// --- Resume methods ---

// GetByID fetches a resume by ID including content.
func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*Resume, error) {
	var resume Resume
	err := r.db.GetContext(ctx, &resume,
		`SELECT `+resumeColumns+`
		 FROM resumes WHERE id = $1`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get resume by id: %w", err)
	}
	return &resume, nil
}

// List returns resumes ordered by created_at DESC (without content for list view).
func (r *PostgresRepository) List(ctx context.Context, limit, offset int) ([]*Resume, int64, error) {
	var total int64
	if err := r.db.GetContext(ctx, &total, "SELECT COUNT(*) FROM resumes"); err != nil {
		return nil, 0, fmt.Errorf("count resumes: %w", err)
	}

	var resumes []*Resume
	err := r.db.SelectContext(ctx, &resumes,
		`SELECT id, name, specialization, template_path, focus_skills, highlight_experience,
		        content, pdf_key, version, created_at, updated_at
		 FROM resumes ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list resumes: %w", err)
	}
	return resumes, total, nil
}

// Create inserts a new resume and returns the DB-assigned values.
func (r *PostgresRepository) Create(ctx context.Context, resume *Resume) error {
	return r.db.QueryRowxContext(ctx,
		`INSERT INTO resumes (id, name, specialization, template_path, focus_skills, highlight_experience,
		                      content, pdf_key, version, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 RETURNING version, created_at, updated_at`,
		resume.ID, resume.Name, resume.Specialization, resume.TemplatePath,
		pq.StringArray(resume.FocusSkills), pq.Array(resume.HighlightExperience),
		ResumeContentDB(resume.Content), resume.PdfKey, resume.Version,
		resume.CreatedAt, resume.UpdatedAt,
	).Scan(&resume.Version, &resume.CreatedAt, &resume.UpdatedAt)
}

// Update updates an existing resume with optimistic locking.
func (r *PostgresRepository) Update(ctx context.Context, resume *Resume) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE resumes SET name = $1, specialization = $2, template_path = $3,
		        focus_skills = $4, highlight_experience = $5, content = $6,
		        pdf_key = $7, version = version + 1, updated_at = NOW()
		 WHERE id = $8 AND version = $9`,
		resume.Name, resume.Specialization, resume.TemplatePath,
		pq.StringArray(resume.FocusSkills), pq.Array(resume.HighlightExperience),
		ResumeContentDB(resume.Content), resume.PdfKey, resume.ID, resume.Version)
	if err != nil {
		return fmt.Errorf("update resume: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		// Distinguish: resume not found vs version conflict
		var exists bool
		err := r.db.QueryRowContext(ctx,
			"SELECT EXISTS(SELECT 1 FROM resumes WHERE id = $1)", resume.ID,
		).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check resume exists: %w", err)
		}
		if !exists {
			return ErrNotFound
		}
		return ErrVersionConflict
	}
	// Fetch updated values
	return r.db.QueryRowContext(ctx,
		`SELECT version, updated_at FROM resumes WHERE id = $1`, resume.ID,
	).Scan(&resume.Version, &resume.UpdatedAt)
}

// UpdateContent updates only the content and bumps the version.
// Returns the new version number.
func (r *PostgresRepository) UpdateContent(ctx context.Context, id uuid.UUID, content ResumeContentDB, currentVersion int32) (int32, error) {
	var newVersion int32
	err := r.db.QueryRowContext(ctx,
		`UPDATE resumes SET content = $1, version = version + 1, updated_at = NOW()
		 WHERE id = $2 AND version = $3
		 RETURNING version`,
		content, id, currentVersion,
	).Scan(&newVersion)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Distinguish: resume not found vs version conflict
			var exists bool
			if existsErr := r.db.QueryRowContext(ctx,
				"SELECT EXISTS(SELECT 1 FROM resumes WHERE id = $1)", id,
			).Scan(&exists); existsErr != nil {
				return 0, fmt.Errorf("check resume exists: %w", existsErr)
			}
			if !exists {
				return 0, ErrNotFound
			}
			return 0, ErrVersionConflict
		}
		return 0, fmt.Errorf("update content: %w", err)
	}
	return newVersion, nil
}

// UpdatePdfKey updates only the PDF storage key.
func (r *PostgresRepository) UpdatePdfKey(ctx context.Context, id uuid.UUID, pdfKey string) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE resumes SET pdf_key = $1, updated_at = NOW() WHERE id = $2`,
		pdfKey, id)
	if err != nil {
		return fmt.Errorf("update pdf key: %w", err)
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

// Delete deletes a resume by ID.
func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
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

// --- Version methods ---

// SaveVersion inserts a versioned snapshot of resume content.
func (r *PostgresRepository) SaveVersion(ctx context.Context, v *ResumeVersion) error {
	return r.db.QueryRowxContext(ctx,
		`INSERT INTO resume_versions (id, resume_id, content, version, pdf_key, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING created_at`,
		v.ID, v.ResumeID, v.Content, v.Version, v.PdfKey, v.CreatedAt,
	).Scan(&v.CreatedAt)
}

// GetVersions returns all versions for a resume, newest first.
func (r *PostgresRepository) GetVersions(ctx context.Context, resumeID uuid.UUID) ([]*ResumeVersion, error) {
	var versions []*ResumeVersion
	err := r.db.SelectContext(ctx, &versions,
		`SELECT id, resume_id, content, version, pdf_key, created_at
		 FROM resume_versions
		 WHERE resume_id = $1
		 ORDER BY version DESC`, resumeID)
	if err != nil {
		return nil, fmt.Errorf("get versions: %w", err)
	}
	return versions, nil
}

// GetVersion returns a specific version of a resume.
func (r *PostgresRepository) GetVersion(ctx context.Context, resumeID uuid.UUID, version int32) (*ResumeVersion, error) {
	var v ResumeVersion
	err := r.db.GetContext(ctx, &v,
		`SELECT id, resume_id, content, version, pdf_key, created_at
		 FROM resume_versions
		 WHERE resume_id = $1 AND version = $2`, resumeID, version)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get version: %w", err)
	}
	return &v, nil
}

// --- Cover Letter methods ---

// GetCoverLetterByID fetches a cover letter by ID.
func (r *PostgresRepository) GetCoverLetterByID(ctx context.Context, id uuid.UUID) (*CoverLetter, error) {
	var cl CoverLetter
	err := r.db.GetContext(ctx, &cl,
		`SELECT `+coverLetterColumns+`
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
func (r *PostgresRepository) ListCoverLetters(ctx context.Context, limit, offset int) ([]*CoverLetter, int64, error) {
	var total int64
	if err := r.db.GetContext(ctx, &total, "SELECT COUNT(*) FROM cover_letters"); err != nil {
		return nil, 0, fmt.Errorf("count cover letters: %w", err)
	}

	var letters []*CoverLetter
	err := r.db.SelectContext(ctx, &letters,
		`SELECT `+coverLetterColumns+`
		 FROM cover_letters ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list cover letters: %w", err)
	}
	return letters, total, nil
}

// CreateCoverLetter inserts a new cover letter and returns the DB-assigned values.
func (r *PostgresRepository) CreateCoverLetter(ctx context.Context, cl *CoverLetter) error {
	return r.db.QueryRowxContext(ctx,
		`INSERT INTO cover_letters (id, job_id, resume_id, job_title, content, model, prompt_version,
		                           resume_version, pdf_key, strengths, gaps, word_count, version,
		                           created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		 RETURNING version, created_at, updated_at`,
		cl.ID, cl.JobID, cl.ResumeID, cl.JobTitle, cl.Content, cl.Model, cl.PromptVersion,
		cl.ResumeVersion, cl.PdfKey, cl.Strengths, cl.Gaps,
		cl.WordCount, cl.Version, cl.CreatedAt, cl.UpdatedAt,
	).Scan(&cl.Version, &cl.CreatedAt, &cl.UpdatedAt)
}

// UpdateCoverLetterContent updates content and LLM traceability fields with optimistic locking.
// Returns the new version number.
func (r *PostgresRepository) UpdateCoverLetterContent(ctx context.Context, id uuid.UUID, content string, model, promptVersion *string, resumeVersion *int32, strengths, gaps *StringSliceDB, wordCount *int, currentVersion int32) (int32, error) {
	var newVersion int32
	err := r.db.QueryRowContext(ctx,
		`UPDATE cover_letters SET content = $1, model = $2, prompt_version = $3,
		        resume_version = $4, strengths = $5, gaps = $6, word_count = $7,
		        version = version + 1, updated_at = NOW()
		 WHERE id = $8 AND version = $9
		 RETURNING version`,
		content, model, promptVersion, resumeVersion,
		strengths, gaps, wordCount, id, currentVersion,
	).Scan(&newVersion)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Distinguish: not found vs version conflict
			var exists bool
			if existsErr := r.db.QueryRowContext(ctx,
				"SELECT EXISTS(SELECT 1 FROM cover_letters WHERE id = $1)", id,
			).Scan(&exists); existsErr != nil {
				return 0, fmt.Errorf("check cover letter exists: %w", existsErr)
			}
			if !exists {
				return 0, ErrNotFound
			}
			return 0, ErrVersionConflict
		}
		return 0, fmt.Errorf("update cover letter content: %w", err)
	}
	return newVersion, nil
}

// UpdateCoverLetterPdfKey updates only the PDF storage key.
func (r *PostgresRepository) UpdateCoverLetterPdfKey(ctx context.Context, id uuid.UUID, pdfKey string) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE cover_letters SET pdf_key = $1, updated_at = NOW() WHERE id = $2`,
		pdfKey, id)
	if err != nil {
		return fmt.Errorf("update cover letter pdf key: %w", err)
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

// DeleteCoverLetter deletes a cover letter by ID.
func (r *PostgresRepository) DeleteCoverLetter(ctx context.Context, id uuid.UUID) error {
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

// --- Cover Letter Version methods ---

// SaveCoverLetterVersion inserts a versioned snapshot of cover letter content.
func (r *PostgresRepository) SaveCoverLetterVersion(ctx context.Context, v *CoverLetterVersion) error {
	return r.db.QueryRowxContext(ctx,
		`INSERT INTO cover_letter_versions (id, cover_letter_id, content, version, model, prompt_version, resume_version, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING created_at`,
		v.ID, v.CoverLetterID, v.Content, v.Version, v.Model, v.PromptVersion, v.ResumeVersion, v.CreatedAt,
	).Scan(&v.CreatedAt)
}

// GetCoverLetterVersions returns all versions for a cover letter, newest first.
func (r *PostgresRepository) GetCoverLetterVersions(ctx context.Context, coverLetterID uuid.UUID) ([]*CoverLetterVersion, error) {
	var versions []*CoverLetterVersion
	err := r.db.SelectContext(ctx, &versions,
		`SELECT id, cover_letter_id, content, version, model, prompt_version, resume_version, created_at
		 FROM cover_letter_versions
		 WHERE cover_letter_id = $1
		 ORDER BY version DESC`, coverLetterID)
	if err != nil {
		return nil, fmt.Errorf("get cover letter versions: %w", err)
	}
	return versions, nil
}

// GetCoverLetterVersion returns a specific version of a cover letter.
func (r *PostgresRepository) GetCoverLetterVersion(ctx context.Context, coverLetterID uuid.UUID, version int32) (*CoverLetterVersion, error) {
	var v CoverLetterVersion
	err := r.db.GetContext(ctx, &v,
		`SELECT id, cover_letter_id, content, version, model, prompt_version, resume_version, created_at
		 FROM cover_letter_versions
		 WHERE cover_letter_id = $1 AND version = $2`, coverLetterID, version)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get cover letter version: %w", err)
	}
	return &v, nil
}
