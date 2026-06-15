package resumes

import (
	"time"

	"github.com/google/uuid"
)

// Resume represents a resume version.
type Resume struct {
	ID                  uuid.UUID
	Name                string
	Specialization      string
	TemplatePath        string
	FocusSkills         []string
	HighlightExperience []uuid.UUID
	PdfKey              *string
	Version             int32
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// NewResume creates a new Resume with default values.
func NewResume(name, specialization, templatePath string, focusSkills []string) *Resume {
	now := time.Now().UTC()
	return &Resume{
		ID:                uuid.New(),
		Name:              name,
		Specialization:    specialization,
		TemplatePath:      templatePath,
		FocusSkills:       focusSkills,
		Version:           1,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

// CoverLetter represents a generated cover letter.
type CoverLetter struct {
	ID        uuid.UUID
	JobID     *uuid.UUID
	ResumeID  *uuid.UUID
	Content   string
	PdfKey    *string
	WordCount *int
	Version   int32
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewCoverLetter creates a new CoverLetter with default values.
func NewCoverLetter(content string) *CoverLetter {
	now := time.Now().UTC()
	return &CoverLetter{
		ID:        uuid.New(),
		Content:   content,
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
