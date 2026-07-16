package resumes

import (
	"time"

	"github.com/google/uuid"
)

// --- Request DTOs ---

// CreateResumeRequest is the payload for POST /resumes.
type CreateResumeRequest struct {
	Name                string      `json:"name" binding:"required,min=2,max=100" example:"Senior Go Engineer Resume"`
	Specialization      string      `json:"specialization" binding:"required,min=2,max=100" example:"Backend Engineering"`
	TemplatePath        string      `json:"template_path" binding:"required" example:"templates/resume.tex"`
	FocusSkills         []string    `json:"focus_skills" binding:"required,min=1,max=20" example:"[\"Go\",\"PostgreSQL\",\"Kubernetes\",\"gRPC\"]"`
	HighlightExperience []uuid.UUID `json:"highlight_experience,omitempty" example:"[\"550e8400-e29b-41d4-a716-446655440000\"]"`
}

// UpdateResumeRequest is the payload for PUT /resumes/:id.
// Only client-writable fields are exposed — ID, Version, timestamps are never client-settable.
type UpdateResumeRequest struct {
	Name                string      `json:"name" binding:"required,min=2,max=100" example:"Senior Go Engineer Resume v2"`
	Specialization      string      `json:"specialization" binding:"required,min=2,max=100" example:"Backend Engineering"`
	TemplatePath        string      `json:"template_path" binding:"required" example:"templates/resume.tex"`
	FocusSkills         []string    `json:"focus_skills" binding:"required,min=1,max=20" example:"[\"Go\",\"PostgreSQL\",\"Kubernetes\",\"gRPC\",\"Redis\"]"`
	HighlightExperience []uuid.UUID `json:"highlight_experience,omitempty" example:"[\"550e8400-e29b-41d4-a716-446655440000\"]"`
}

// GenerateResumeContentRequest is the payload for POST /resumes/:id/generate.
// Triggers async LLM generation of structured resume content.
type GenerateResumeContentRequest struct {
	JobID           *uuid.UUID `json:"job_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440001"`          // optional: tailor for specific job
	JobTitle        string     `json:"job_title,omitempty" example:"Senior Go Engineer"`                         // optional: target job title
	JobRequirements string     `json:"job_requirements,omitempty" example:"5+ years Go, Kubernetes, PostgreSQL"` // optional: target job requirements
}

// UpdateResumeContentRequest is the payload for PUT /resumes/:id/content.
// Allows manual override of structured content.
type UpdateResumeContentRequest struct {
	Content ResumeContent `json:"content" binding:"required"`
}

// CreateCoverLetterRequest is the payload for POST /cover-letters.
// Creates an empty cover letter placeholder — use POST /cover-letters/:id/generate to fill content.
type CreateCoverLetterRequest struct {
	JobID    uuid.UUID  `json:"job_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440001"`
	ResumeID *uuid.UUID `json:"resume_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440002"`
}

// GenerateCoverLetterRequest is the payload for POST /cover-letters/:id/generate.
// Triggers LLM generation of cover letter content with job context.
type GenerateCoverLetterRequest struct {
	JobTitle        string     `json:"job_title" binding:"required,max=200" example:"Senior Go Engineer"`
	JobRequirements string     `json:"job_requirements" binding:"required,max=10000" example:"5+ years Go experience, Kubernetes, PostgreSQL, gRPC"`
	JobDescription  string     `json:"job_description" binding:"required,max=50000" example:"We are looking for a Senior Go Engineer..."`
	ResumeID        *uuid.UUID `json:"resume_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440002"` // override: use specific resume
}

// --- Response DTOs ---

// ResumeResponse is the API response for a single resume (list view — content omitted).
type ResumeResponse struct {
	ID                  uuid.UUID   `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name                string      `json:"name" example:"Senior Go Engineer Resume"`
	Specialization      string      `json:"specialization" example:"Backend Engineering"`
	TemplatePath        string      `json:"template_path" example:"templates/resume.tex"`
	FocusSkills         []string    `json:"focus_skills" example:"[\"Go\",\"PostgreSQL\",\"Kubernetes\",\"gRPC\"]"`
	HighlightExperience []uuid.UUID `json:"highlight_experience,omitempty" example:"[\"550e8400-e29b-41d4-a716-446655440001\"]"`
	HasContent          bool        `json:"has_content" example:"true"`
	PdfKey              *string     `json:"pdf_key,omitempty" example:"resumes/550e8400-e29b-41d4-a716-446655440000.pdf"`
	Version             int32       `json:"version" example:"3"`
	CreatedAt           time.Time   `json:"created_at" example:"2026-01-10T10:00:00Z"`
	UpdatedAt           time.Time   `json:"updated_at" example:"2026-01-15T14:30:00Z"`
}

// ResumeDetailResponse is the API response for a single resume with content.
type ResumeDetailResponse struct {
	ResumeResponse
	Content *ResumeContent `json:"content,omitempty"`
}

// ResumeListResponse is the API response for listing resumes.
type ResumeListResponse struct {
	Resumes []ResumeResponse `json:"resumes"`
	Total   int64            `json:"total" example:"15"`
	Limit   int              `json:"limit" example:"20"`
	Offset  int              `json:"offset" example:"0"`
}

// ResumeContentResponse wraps the structured content for API responses.
type ResumeContentResponse struct {
	ResumeID uuid.UUID     `json:"resume_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Version  int32         `json:"version" example:"3"`
	Content  ResumeContent `json:"content"`
}

// ResumeVersionResponse represents a historical version of resume content.
type ResumeVersionResponse struct {
	ID        uuid.UUID     `json:"id" example:"550e8400-e29b-41d4-a716-446655440003"`
	ResumeID  uuid.UUID     `json:"resume_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Version   int32         `json:"version" example:"2"`
	Content   ResumeContent `json:"content"`
	PdfKey    *string       `json:"pdf_key,omitempty" example:"resumes/550e8400-e29b-41d4-a716-446655440000_v2.pdf"`
	CreatedAt time.Time     `json:"created_at" example:"2026-01-12T08:00:00Z"`
}

// ResumeVersionListResponse is the API response for listing versions.
type ResumeVersionListResponse struct {
	Versions []ResumeVersionResponse `json:"versions"`
}

// CoverLetterResponse is the API response for a single cover letter.
type CoverLetterResponse struct {
	ID            uuid.UUID      `json:"id" example:"550e8400-e29b-41d4-a716-446655440004"`
	JobID         *uuid.UUID     `json:"job_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440001"`
	ResumeID      *uuid.UUID     `json:"resume_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440002"`
	JobTitle      *string        `json:"job_title,omitempty" example:"Senior Go Engineer"`
	Content       string         `json:"content" example:"Dear Hiring Manager,\n\nI am writing to express my interest..."`
	Model         *string        `json:"model,omitempty" example:"qwen2.5:latest"`
	PromptVersion *string        `json:"prompt_version,omitempty" example:"v1.2"`
	ResumeVersion *int32         `json:"resume_version,omitempty" example:"3"`
	Strengths     *StringSliceDB `json:"strengths,omitempty" example:"[\"Strong Go experience\",\"Kubernetes expertise\"]"`
	Gaps          *StringSliceDB `json:"gaps,omitempty" example:"[\"No direct PostgreSQL experience\"]"`
	PdfKey        *string        `json:"pdf_key,omitempty" example:"cover-letters/550e8400-e29b-41d4-a716-446655440004.pdf"`
	WordCount     *int           `json:"word_count,omitempty" example:"350"`
	Version       int32          `json:"version" example:"1"`
	CreatedAt     time.Time      `json:"created_at" example:"2026-01-20T10:00:00Z"`
	UpdatedAt     time.Time      `json:"updated_at" example:"2026-01-20T10:00:00Z"`
}

// UpdateCoverLetterContentRequest is the payload for PUT /cover-letters/:id/content.
// Allows manual override of cover letter content.
type UpdateCoverLetterContentRequest struct {
	Content string `json:"content" binding:"required,min=10,max=50000" example:"Dear Hiring Manager,\n\nI am writing to express my interest..."`
}

// CoverLetterListResponse is the API response for listing cover letters.
type CoverLetterListResponse struct {
	CoverLetters []CoverLetterResponse `json:"cover_letters"`
	Total        int64                 `json:"total" example:"8"`
	Limit        int                   `json:"limit" example:"20"`
	Offset       int                   `json:"offset" example:"0"`
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
		ID:            cl.ID,
		JobID:         cl.JobID,
		ResumeID:      cl.ResumeID,
		JobTitle:      cl.JobTitle,
		Content:       cl.Content,
		Model:         cl.Model,
		PromptVersion: cl.PromptVersion,
		ResumeVersion: cl.ResumeVersion,
		Strengths:     cl.Strengths,
		Gaps:          cl.Gaps,
		PdfKey:        cl.PdfKey,
		WordCount:     cl.WordCount,
		Version:       cl.Version,
		CreatedAt:     cl.CreatedAt,
		UpdatedAt:     cl.UpdatedAt,
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

// --- Cover Letter Version DTOs ---

// CoverLetterVersionResponse represents a historical version of cover letter content.
type CoverLetterVersionResponse struct {
	ID            uuid.UUID `json:"id" example:"550e8400-e29b-41d4-a716-446655440003"`
	CoverLetterID uuid.UUID `json:"cover_letter_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Version       int32     `json:"version" example:"2"`
	Content       string    `json:"content" example:"Dear Hiring Manager,\n\nI am writing to express my interest..."`
	Model         *string   `json:"model,omitempty" example:"qwen2.5:latest"`
	PromptVersion *string   `json:"prompt_version,omitempty" example:"v1.2"`
	ResumeVersion *int32    `json:"resume_version,omitempty" example:"3"`
	CreatedAt     time.Time `json:"created_at" example:"2026-01-20T10:00:00Z"`
}

// CoverLetterVersionListResponse is the API response for listing cover letter versions.
type CoverLetterVersionListResponse struct {
	Versions []CoverLetterVersionResponse `json:"versions"`
}

// ToCoverLetterVersionResponse converts a CoverLetterVersion domain model to an API response.
func ToCoverLetterVersionResponse(v *CoverLetterVersion) CoverLetterVersionResponse {
	return CoverLetterVersionResponse{
		ID:            v.ID,
		CoverLetterID: v.CoverLetterID,
		Version:       v.Version,
		Content:       v.Content,
		Model:         v.Model,
		PromptVersion: v.PromptVersion,
		ResumeVersion: v.ResumeVersion,
		CreatedAt:     v.CreatedAt,
	}
}

// ToCoverLetterVersionResponses converts a slice of CoverLetterVersion domain models to API responses.
func ToCoverLetterVersionResponses(versions []*CoverLetterVersion) []CoverLetterVersionResponse {
	responses := make([]CoverLetterVersionResponse, len(versions))
	for i, v := range versions {
		responses[i] = ToCoverLetterVersionResponse(v)
	}
	return responses
}
