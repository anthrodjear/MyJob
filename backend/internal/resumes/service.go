package resumes

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Domain errors for resumes service.
var (
	ErrInvalidInput = errors.New("invalid input")
)

// Service handles resume and cover letter business logic.
type Service struct {
	repo   *Repository
	logger *zap.Logger
}

// NewService creates a new resumes service.
func NewService(repo *Repository, logger *zap.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger.Named("resumes"),
	}
}

// --- Resume methods ---

// GetByID returns a resume by ID.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Resume, error) {
	resume, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get resume: %w", err)
	}
	return resume, nil
}

// List returns resumes with pagination.
func (s *Service) List(ctx context.Context, limit, offset int) ([]*Resume, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.List(ctx, limit, offset)
}

// Create creates a new resume.
func (s *Service) Create(ctx context.Context, req GenerateResumeRequest) (*Resume, error) {
	// Validate input (defensive — handler also validates)
	if strings.TrimSpace(req.Name) == "" {
		return nil, fmt.Errorf("name: %w", ErrInvalidInput)
	}
	if strings.TrimSpace(req.Specialization) == "" {
		return nil, fmt.Errorf("specialization: %w", ErrInvalidInput)
	}
	if strings.TrimSpace(req.TemplatePath) == "" {
		return nil, fmt.Errorf("template_path: %w", ErrInvalidInput)
	}
	if len(req.FocusSkills) == 0 {
		return nil, fmt.Errorf("focus_skills: %w", ErrInvalidInput)
	}

	resume := NewResume(req.Name, req.Specialization, req.TemplatePath, req.FocusSkills)
	if len(req.HighlightExperience) > 0 {
		resume.HighlightExperience = req.HighlightExperience
	}

	if err := s.repo.Create(ctx, resume); err != nil {
		return nil, fmt.Errorf("create resume: %w", err)
	}

	s.logger.Info("resume created",
		zap.String("id", resume.ID.String()),
		zap.String("name", resume.Name),
	)
	return resume, nil
}

// Update updates an existing resume.
func (s *Service) Update(ctx context.Context, resume *Resume) error {
	if err := s.repo.Update(ctx, resume); err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("update resume: %w", err)
	}
	s.logger.Info("resume updated", zap.String("id", resume.ID.String()))
	return nil
}

// Delete deletes a resume by ID.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("delete resume: %w", err)
	}
	s.logger.Info("resume deleted", zap.String("id", id.String()))
	return nil
}

// --- Cover Letter methods ---

// GetCoverLetterByID returns a cover letter by ID.
func (s *Service) GetCoverLetterByID(ctx context.Context, id uuid.UUID) (*CoverLetter, error) {
	cl, err := s.repo.GetCoverLetterByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get cover letter: %w", err)
	}
	return cl, nil
}

// ListCoverLetters returns cover letters with pagination.
func (s *Service) ListCoverLetters(ctx context.Context, limit, offset int) ([]*CoverLetter, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListCoverLetters(ctx, limit, offset)
}

// CreateCoverLetter creates a new cover letter with placeholder content.
// The actual content is generated asynchronously by a worker task.
func (s *Service) CreateCoverLetter(ctx context.Context, req GenerateCoverLetterRequest) (*CoverLetter, error) {
	cl := NewCoverLetter("")
	cl.JobID = &req.JobID
	if req.ResumeID != nil {
		cl.ResumeID = req.ResumeID
	}

	if err := s.repo.CreateCoverLetter(ctx, cl); err != nil {
		return nil, fmt.Errorf("create cover letter: %w", err)
	}

	s.logger.Info("cover letter created",
		zap.String("id", cl.ID.String()),
		zap.String("job_id", req.JobID.String()),
	)
	return cl, nil
}

// DeleteCoverLetter deletes a cover letter by ID.
func (s *Service) DeleteCoverLetter(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.DeleteCoverLetter(ctx, id); err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("delete cover letter: %w", err)
	}
	s.logger.Info("cover letter deleted", zap.String("id", id.String()))
	return nil
}
