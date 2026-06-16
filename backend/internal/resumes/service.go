package resumes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Service handles resume and cover letter business logic.
type Service struct {
	repo   Repository
	llm    ResumeGenerator
	clgen  CoverLetterGenerator
	logger *zap.Logger
}

// NewService creates a new resumes service.
func NewService(repo Repository, llm ResumeGenerator, clgen CoverLetterGenerator, logger *zap.Logger) *Service {
	return &Service{
		repo:   repo,
		llm:    llm,
		clgen:  clgen,
		logger: logger.Named("resumes"),
	}
}

// getResume is a helper that fetches a resume and translates errors.
// Extracted to avoid repeating GetByID + error handling across every method.
func (s *Service) getResume(ctx context.Context, id uuid.UUID) (*Resume, error) {
	resume, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get resume: %w", err)
	}
	return resume, nil
}

// --- Resume CRUD methods ---

// GetByID returns a resume by ID.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Resume, error) {
	return s.getResume(ctx, id)
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

// Create creates a new resume with empty content.
func (s *Service) Create(ctx context.Context, req CreateResumeRequest) (*Resume, error) {
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
		if errors.Is(err, ErrVersionConflict) {
			return ErrVersionConflict
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

// --- Content generation methods ---

// GenerateContent triggers LLM-based resume content generation.
// The generated content is stored in the database and a version snapshot is created.
func (s *Service) GenerateContent(ctx context.Context, resumeID uuid.UUID, req GenerateResumeContentRequest) (*ResumeContent, int32, error) {
	resume, err := s.getResume(ctx, resumeID)
	if err != nil {
		return nil, 0, fmt.Errorf("generate content: %w", err)
	}

	// Build profile from resume for the LLM
	profile := buildProfileFromResume(resume)

	// Generate content via LLM
	content, err := s.llm.GenerateContent(ctx, profile, req.JobTitle, req.JobRequirements)
	if err != nil {
		return nil, 0, fmt.Errorf("llm generation: %w", err)
	}

	// Validate generated content
	if err := validateContent(content); err != nil {
		return nil, 0, fmt.Errorf("invalid generated content: %w", err)
	}

	// Save version snapshot before updating
	if snapErr := s.saveVersionSnapshot(ctx, resume); snapErr != nil {
		s.logger.Warn("version snapshot failed (non-fatal)", zap.Error(snapErr))
	}

	// Update resume content
	contentDB := ResumeContentDB(*content)
	newVersion, err := s.repo.UpdateContent(ctx, resumeID, contentDB, resume.Version)
	if err != nil {
		return nil, 0, fmt.Errorf("update content: %w", err)
	}
	resume.Version = newVersion
	resume.Content = contentDB

	s.logger.Info("resume content generated",
		zap.String("id", resumeID.String()),
		zap.Int32("version", newVersion),
		zap.String("model", s.llm.ModelName()),
	)

	return content, newVersion, nil
}

// UpdateContent manually overrides the resume content.
func (s *Service) UpdateContent(ctx context.Context, resumeID uuid.UUID, content ResumeContent) (*ResumeContent, int32, error) {
	resume, err := s.getResume(ctx, resumeID)
	if err != nil {
		return nil, 0, fmt.Errorf("update content: %w", err)
	}

	// Save version snapshot before updating
	if snapErr := s.saveVersionSnapshot(ctx, resume); snapErr != nil {
		s.logger.Warn("version snapshot failed (non-fatal)", zap.Error(snapErr))
	}

	contentDB := ResumeContentDB(content)
	newVersion, err := s.repo.UpdateContent(ctx, resumeID, contentDB, resume.Version)
	if err != nil {
		return nil, 0, fmt.Errorf("update content: %w", err)
	}
	resume.Version = newVersion
	resume.Content = contentDB

	s.logger.Info("resume content updated",
		zap.String("id", resumeID.String()),
		zap.Int32("version", newVersion),
	)

	return &content, newVersion, nil
}

// GetContent returns the structured content and current version for a resume.
func (s *Service) GetContent(ctx context.Context, resumeID uuid.UUID) (*ResumeContent, int32, error) {
	resume, err := s.getResume(ctx, resumeID)
	if err != nil {
		return nil, 0, fmt.Errorf("get content: %w", err)
	}
	if !hasContent(resume.Content) {
		return nil, resume.Version, ErrNoContent
	}
	c := ResumeContent(resume.Content)
	return &c, resume.Version, nil
}

// GetVersions returns the version history for a resume.
func (s *Service) GetVersions(ctx context.Context, resumeID uuid.UUID) ([]*ResumeVersion, error) {
	// Verify resume exists
	if _, err := s.getResume(ctx, resumeID); err != nil {
		return nil, fmt.Errorf("get versions: %w", err)
	}
	return s.repo.GetVersions(ctx, resumeID)
}

// GetVersion returns a specific historical version.
func (s *Service) GetVersion(ctx context.Context, resumeID uuid.UUID, version int32) (*ResumeVersion, error) {
	v, err := s.repo.GetVersion(ctx, resumeID, version)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get version: %w", err)
	}
	return v, nil
}

// saveVersionSnapshot creates a version snapshot before content changes.
// Returns an error so callers can decide whether to treat failure as fatal.
func (s *Service) saveVersionSnapshot(ctx context.Context, resume *Resume) error {
	v := &ResumeVersion{
		ID:        uuid.New(),
		ResumeID:  resume.ID,
		Content:   resume.Content,
		Version:   resume.Version,
		PdfKey:    resume.PdfKey,
		CreatedAt: resume.UpdatedAt,
	}
	return s.repo.SaveVersion(ctx, v)
}

// buildProfileFromResume constructs a profile map from a resume for the LLM prompt.
func buildProfileFromResume(r *Resume) map[string]any {
	c := ResumeContent(r.Content)
	return map[string]any{
		"Name":           r.Name,
		"Specialization": r.Specialization,
		"FocusSkills":    strings.Join(r.FocusSkills, ", "),
		"Summary":        c.Summary,
		"Skills":         strings.Join(c.Skills, ", "),
		"Experience":     formatContentExperience(c.Experience),
		"Projects":       formatContentProjects(c.Projects),
		"Education":      formatContentEducation(c.Education),
		"Certifications": strings.Join(c.Certifications, ", "),
		"Languages":      formatContentLanguages(c.Languages),
		"Links":          formatContentLinks(c.Links),
		"CareerGoals":    strings.Join(r.FocusSkills, ", "),
	}
}

func formatContentExperience(exp []ExperienceEntry) string {
	var parts []string
	for _, e := range exp {
		parts = append(parts, fmt.Sprintf("%s at %s (%s-%s)", e.Title, e.Company, e.StartDate, e.EndDate))
	}
	return strings.Join(parts, "; ")
}

func formatContentProjects(projects []ProjectEntry) string {
	var parts []string
	for _, p := range projects {
		parts = append(parts, fmt.Sprintf("%s: %s", p.Name, p.Description))
	}
	return strings.Join(parts, "; ")
}

func formatContentEducation(edu []EducationEntry) string {
	var parts []string
	for _, e := range edu {
		parts = append(parts, fmt.Sprintf("%s %s in %s", e.Degree, e.Field, e.Institution))
	}
	return strings.Join(parts, "; ")
}

func formatContentLanguages(langs []LanguageEntry) string {
	var parts []string
	for _, l := range langs {
		parts = append(parts, fmt.Sprintf("%s (%s)", l.Language, l.Proficiency))
	}
	return strings.Join(parts, ", ")
}

func formatContentLinks(links []LinkEntry) string {
	var parts []string
	for _, l := range links {
		parts = append(parts, fmt.Sprintf("%s: %s", l.Type, l.URL))
	}
	return strings.Join(parts, ", ")
}

// validateContent checks that generated content has minimum required fields.
func validateContent(c *ResumeContent) error {
	if c == nil {
		return fmt.Errorf("content is nil")
	}
	if len(c.Skills) == 0 {
		return fmt.Errorf("skills cannot be empty")
	}
	if c.Summary == "" {
		return fmt.Errorf("summary cannot be empty")
	}
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

// CreateCoverLetter creates a new cover letter placeholder.
// Use GenerateCoverLetter to fill content via LLM.
func (s *Service) CreateCoverLetter(ctx context.Context, req CreateCoverLetterRequest) (*CoverLetter, error) {
	cl := NewCoverLetter()
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

// GenerateCoverLetter triggers LLM-based cover letter generation.
// Returns the generated content and new version.
func (s *Service) GenerateCoverLetter(ctx context.Context, clID uuid.UUID, req GenerateCoverLetterRequest) (*CoverLetter, error) {
	cl, err := s.getCoverLetter(ctx, clID)
	if err != nil {
		return nil, fmt.Errorf("generate cover letter: %w", err)
	}

	// Determine resume to use
	var resumeContent *ResumeContent
	resumeVersion := (*int32)(nil)
	resumeID := cl.ResumeID
	if req.ResumeID != nil {
		resumeID = req.ResumeID
	}

	if resumeID != nil {
		resume, err := s.getResume(ctx, *resumeID)
		if err != nil {
			return nil, fmt.Errorf("get resume for cover letter: %w", err)
		}
		if hasContent(resume.Content) {
			c := ResumeContent(resume.Content)
			resumeContent = &c
			resumeVersion = &resume.Version
		}
	}

	// Generate via LLM
	result, err := s.clgen.GenerateContent(ctx, req.JobTitle, req.JobRequirements, req.JobDescription, resumeContent)
	if err != nil {
		return nil, fmt.Errorf("llm cover letter generation: %w", err)
	}

	// Compute word count
	wordCount := len(strings.Fields(result.Content))

	// Convert []string to StringSliceDB for JSONB storage
	strengths := StringSliceDB(result.Strengths)
	gaps := StringSliceDB(result.Gaps)

	// Update cover letter with generated content + traceability
	modelName := s.clgen.ModelName()
	newVersion, err := s.repo.UpdateCoverLetterContent(ctx, clID, result.Content,
		&modelName, nil, resumeVersion, &strengths, &gaps, &wordCount, cl.Version)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}
		if errors.Is(err, ErrVersionConflict) {
			return nil, ErrVersionConflict
		}
		return nil, fmt.Errorf("update cover letter content: %w", err)
	}

	cl.Content = result.Content
	cl.Model = &modelName
	cl.ResumeVersion = resumeVersion
	cl.Strengths = &strengths
	cl.Gaps = &gaps
	cl.Version = newVersion
	cl.WordCount = &wordCount

	s.logger.Info("cover letter generated",
		zap.String("id", clID.String()),
		zap.Int32("version", newVersion),
		zap.String("model", modelName),
		zap.Int("word_count", wordCount),
	)

	return cl, nil
}

// UpdateCoverLetterContent manually overrides the cover letter content.
func (s *Service) UpdateCoverLetterContent(ctx context.Context, clID uuid.UUID, content string) (*CoverLetter, error) {
	cl, err := s.getCoverLetter(ctx, clID)
	if err != nil {
		return nil, fmt.Errorf("update cover letter content: %w", err)
	}

	wordCount := len(strings.Fields(content))
	newVersion, err := s.repo.UpdateCoverLetterContent(ctx, clID, content,
		cl.Model, cl.PromptVersion, cl.ResumeVersion, cl.Strengths, cl.Gaps, &wordCount, cl.Version)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}
		if errors.Is(err, ErrVersionConflict) {
			return nil, ErrVersionConflict
		}
		return nil, fmt.Errorf("update cover letter content: %w", err)
	}

	cl.Content = content
	cl.Version = newVersion
	cl.WordCount = &wordCount

	s.logger.Info("cover letter content updated",
		zap.String("id", clID.String()),
		zap.Int32("version", newVersion),
	)

	return cl, nil
}

// getCoverLetter is a helper that fetches a cover letter and translates errors.
func (s *Service) getCoverLetter(ctx context.Context, id uuid.UUID) (*CoverLetter, error) {
	cl, err := s.repo.GetCoverLetterByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get cover letter: %w", err)
	}
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

// UnmarshalContent is a helper to unmarshal raw JSON bytes into ResumeContent.
func UnmarshalContent(data []byte) (ResumeContent, error) {
	var c ResumeContent
	if err := json.Unmarshal(data, &c); err != nil {
		return ResumeContent{}, fmt.Errorf("unmarshal content: %w", err)
	}
	return c, nil
}
