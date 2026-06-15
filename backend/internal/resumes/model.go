package resumes

import (
	"time"

	"github.com/google/uuid"
)

// TableName returns the table name for the Resume model.
func (Resume) TableName() string {
	return "resumes"
}

// Resume represents a resume version.
type Resume struct {
	ID                 uuid.UUID  `db:"id" json:"id"`
	Name               string     `db:"name" json:"name"`
	Specialization     string     `db:"specialization" json:"specialization"`
	TemplatePath       string     `db:"template_path" json:"template_path"`
	FocusSkills        []string   `db:"focus_skills" json:"focus_skills"`
	HighlightExperience []uuid.UUID `db:"highlight_experience" json:"highlight_experience,omitempty"`
	PdfPath            *string    `db:"pdf_path" json:"pdf_path,omitempty"`
	Version            int        `db:"version" json:"version"`
	CreatedAt          time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt          time.Time  `db:"updated_at" json:"updated_at"`
}

// TableName returns the table name for the CoverLetter model.
func (CoverLetter) TableName() string {
	return "cover_letters"
}

// CoverLetter represents a generated cover letter.
type CoverLetter struct {
	ID        uuid.UUID  `db:"id" json:"id"`
	JobID     *uuid.UUID `db:"job_id" json:"job_id,omitempty"`
	ResumeID  *uuid.UUID `db:"resume_id" json:"resume_id,omitempty"`
	Content   string     `db:"content" json:"content"`
	PdfPath   *string    `db:"pdf_path" json:"pdf_path,omitempty"`
	WordCount *int       `db:"word_count" json:"word_count,omitempty"`
	Version   int        `db:"version" json:"version"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
}
