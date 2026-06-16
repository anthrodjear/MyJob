package resumes

import (
	"time"

	"github.com/google/uuid"
)

// --- Request DTOs ---

// CreateResumeRequest is the payload for POST /resumes.
type CreateResumeRequest struct {
	Name                string      `json:"name" binding:"required,min=2,max=100"`
	Specialization      string      `json:"specialization" binding:"required,min=2,max=100"`
	TemplatePath        string      `json:"template_path" binding:"required"`
	FocusSkills         []string    `json:"focus_skills" binding:"required,min=1,max=20"`
	HighlightExperience []uuid.UUID `json:"highlight_experience,omitempty"`
}

// UpdateResumeRequest is the payload for PUT /resumes/:id.
// Only client-writable fields are exposed — ID, Version, timestamps are never client-settable.
type UpdateResumeRequest struct {
	Name                string      `json:"name" binding:"required,min=2,max=100"`
	Specialization      string      `json:"specialization" binding:"required,min=2,max=100"`
	TemplatePath        string      `json:"template_path" binding:"required"`
	FocusSkills         []string    `json:"focus_skills" binding:"required,min=1,max=20"`
	HighlightExperience []uuid.UUID `json:"highlight_experience,omitempty"`
}

// GenerateResumeContentRequest is the payload for POST /resumes/:id/generate.
// Triggers async LLM generation of structured resume content.
type GenerateResumeContentRequest struct {
	JobID        *uuid.UUID `json:"job_id,omitempty"`        // optional: tailor for specific job
	JobTitle     string     `json:"job_title,omitempty"`     // optional: target job title
	JobRequirements string `json:"job_requirements,omitempty"` // optional: target job requirements
}

// UpdateResumeContentRequest is the payload for PUT /resumes/:id/content.
// Allows manual override of structured content.
type UpdateResumeContentRequest struct {
	Content ResumeContent `json:"content" binding:"required"`
}

// GenerateCoverLetterRequest is the payload for POST /cover-letters.
type GenerateCoverLetterRequest struct {
	JobID    uuid.UUID  `json:"job_id" binding:"required"`
	ResumeID *uuid.UUID `json:"resume_id"`
}

// --- Response DTOs ---

// ResumeResponse is the API response for a single resume (list view — content omitted).
type ResumeResponse struct {
	ID                  uuid.UUID  `json:"id"`
	Name                string     `json:"name"`
	Specialization      string     `json:"specialization"`
	TemplatePath        string     `json:"template_path"`
	FocusSkills         []string   `json:"focus_skills"`
	HighlightExperience []uuid.UUID `json:"highlight_experience,omitempty"`
	HasContent          bool       `json:"has_content"`
	PdfKey              *string    `json:"pdf_key,omitempty"`
	Version             int32      `json:"version"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// ResumeDetailResponse is the API response for a single resume with content.
type ResumeDetailResponse struct {
	ResumeResponse
	Content *ResumeContent `json:"content,omitempty"`
}

// ResumeListResponse is the API response for listing resumes.
type ResumeListResponse struct {
	Resumes []ResumeResponse `json:"resumes"`
	Total   int64            `json:"total"`
	Limit   int              `json:"limit"`
	Offset  int              `json:"offset"`
}

// ResumeContentResponse wraps the structured content for API responses.
type ResumeContentResponse struct {
	ResumeID uuid.UUID      `json:"resume_id"`
	Version  int32          `json:"version"`
	Content  ResumeContent  `json:"content"`
}

// ResumeVersionResponse represents a historical version of resume content.
type ResumeVersionResponse struct {
	ID        uuid.UUID      `json:"id"`
	ResumeID  uuid.UUID      `json:"resume_id"`
	Version   int32          `json:"version"`
	Content   ResumeContent  `json:"content"`
	PdfKey    *string        `json:"pdf_key,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

// ResumeVersionListResponse is the API response for listing versions.
type ResumeVersionListResponse struct {
	Versions []ResumeVersionResponse `json:"versions"`
}

// CoverLetterResponse is the API response for a single cover letter.
type CoverLetterResponse struct {
	ID        uuid.UUID  `json:"id"`
	JobID     *uuid.UUID `json:"job_id,omitempty"`
	ResumeID  *uuid.UUID `json:"resume_id,omitempty"`
	Content   string     `json:"content"`
	PdfKey    *string    `json:"pdf_key,omitempty"`
	WordCount *int       `json:"word_count,omitempty"`
	Version   int32      `json:"version"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// CoverLetterListResponse is the API response for listing cover letters.
type CoverLetterListResponse struct {
	CoverLetters []CoverLetterResponse `json:"cover_letters"`
	Total        int64                 `json:"total"`
	Limit        int                   `json:"limit"`
	Offset       int                   `json:"offset"`
}

// --- Mappers ---

// ToResponse converts a Resume domain model to a list-view API response (no content).
func ToResponse(r *Resume) ResumeResponse {
	return ResumeResponse{
		ID:                  r.ID,
		Name:                r.Name,
		Specialization:      r.Specialization,
		TemplatePath:        r.TemplatePath,
		FocusSkills:         r.FocusSkills,
		HighlightExperience: r.HighlightExperience,
		HasContent:          hasContent(r.Content),
		PdfKey:              r.PdfKey,
		Version:             r.Version,
		CreatedAt:           r.CreatedAt,
		UpdatedAt:           r.UpdatedAt,
	}
}

// ToDetailResponse converts a Resume domain model to a detail API response (with content).
func ToDetailResponse(r *Resume) ResumeDetailResponse {
	resp := ResumeDetailResponse{
		ResumeResponse: ToResponse(r),
	}
	if hasContent(r.Content) {
		c := ResumeContent(r.Content)
		resp.Content = &c
	}
	return resp
}

// ToResponses converts a slice of Resume domain models to API responses.
func ToResponses(resumes []*Resume) []ResumeResponse {
	responses := make([]ResumeResponse, len(resumes))
	for i, r := range resumes {
		responses[i] = ToResponse(r)
	}
	return responses
}

// ToVersionResponse converts a ResumeVersion domain model to an API response.
func ToVersionResponse(v *ResumeVersion) ResumeVersionResponse {
	return ResumeVersionResponse{
		ID:        v.ID,
		ResumeID:  v.ResumeID,
		Version:   v.Version,
		Content:   ResumeContent(v.Content),
		PdfKey:    v.PdfKey,
		CreatedAt: v.CreatedAt,
	}
}

// ToVersionResponses converts a slice of ResumeVersion domain models to API responses.
func ToVersionResponses(versions []*ResumeVersion) []ResumeVersionResponse {
	responses := make([]ResumeVersionResponse, len(versions))
	for i, v := range versions {
		responses[i] = ToVersionResponse(v)
	}
	return responses
}

// ToCoverLetterResponse converts a CoverLetter domain model to an API response.
func ToCoverLetterResponse(cl *CoverLetter) CoverLetterResponse {
	return CoverLetterResponse{
		ID:        cl.ID,
		JobID:     cl.JobID,
		ResumeID:  cl.ResumeID,
		Content:   cl.Content,
		PdfKey:    cl.PdfKey,
		WordCount: cl.WordCount,
		Version:   cl.Version,
		CreatedAt: cl.CreatedAt,
		UpdatedAt: cl.UpdatedAt,
	}
}

// ToCoverLetterResponses converts a slice of CoverLetter domain models to API responses.
func ToCoverLetterResponses(letters []*CoverLetter) []CoverLetterResponse {
	responses := make([]CoverLetterResponse, len(letters))
	for i, cl := range letters {
		responses[i] = ToCoverLetterResponse(cl)
	}
	return responses
}

// hasContent checks if a ResumeContentDB has meaningful content.
func hasContent(c ResumeContentDB) bool {
	return len(c.Skills) > 0 || len(c.Experience) > 0 || c.Summary != ""
}
