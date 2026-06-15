package resumes

import (
	"time"

	"github.com/google/uuid"
)

// --- Request DTOs ---

// GenerateResumeRequest is the payload for POST /resumes/generate.
type GenerateResumeRequest struct {
	Name                string     `json:"name" binding:"required,min=2,max=100"`
	Specialization      string     `json:"specialization" binding:"required,min=2,max=100"`
	TemplatePath        string     `json:"template_path" binding:"required"`
	FocusSkills         []string   `json:"focus_skills" binding:"required,min=1,max=20"`
	HighlightExperience []uuid.UUID `json:"highlight_experience,omitempty"`
}

// GenerateCoverLetterRequest is the payload for POST /cover-letters/generate.
type GenerateCoverLetterRequest struct {
	JobID    uuid.UUID  `json:"job_id" binding:"required"`
	ResumeID *uuid.UUID `json:"resume_id"`
}

// --- Response DTOs ---

// ResumeResponse is the API response for a single resume.
type ResumeResponse struct {
	ID                 uuid.UUID  `json:"id"`
	Name               string     `json:"name"`
	Specialization     string     `json:"specialization"`
	TemplatePath       string     `json:"template_path"`
	FocusSkills        []string   `json:"focus_skills"`
	HighlightExperience []uuid.UUID `json:"highlight_experience,omitempty"`
	PdfURL             *string    `json:"pdf_url,omitempty"`
	Version            int32      `json:"version"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// ResumeListResponse is the API response for listing resumes.
type ResumeListResponse struct {
	Resumes []ResumeResponse `json:"resumes"`
	Total   int64            `json:"total"`
	Limit   int              `json:"limit"`
	Offset  int              `json:"offset"`
}

// CoverLetterResponse is the API response for a single cover letter.
type CoverLetterResponse struct {
	ID        uuid.UUID  `json:"id"`
	JobID     *uuid.UUID `json:"job_id,omitempty"`
	ResumeID  *uuid.UUID `json:"resume_id,omitempty"`
	Content   string     `json:"content"`
	PdfURL    *string    `json:"pdf_url,omitempty"`
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

// ToResponse converts a Resume domain model to an API response.
func ToResponse(r *Resume) ResumeResponse {
	return ResumeResponse{
		ID:                 r.ID,
		Name:               r.Name,
		Specialization:     r.Specialization,
		TemplatePath:       r.TemplatePath,
		FocusSkills:        r.FocusSkills,
		HighlightExperience: r.HighlightExperience,
		PdfURL:             r.PdfKey,
		Version:            r.Version,
		CreatedAt:          r.CreatedAt,
		UpdatedAt:          r.UpdatedAt,
	}
}

// ToResponses converts a slice of Resume domain models to API responses.
func ToResponses(resumes []*Resume) []ResumeResponse {
	responses := make([]ResumeResponse, len(resumes))
	for i, r := range resumes {
		responses[i] = ToResponse(r)
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
		PdfURL:    cl.PdfKey,
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
