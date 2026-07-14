package profile

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// Skill Proficiency Constants
// ============================================================================
// Immutable enum-like values for skill proficiency levels.
// Mirrors the Speaker constant pattern from interviews/model.go.

const (
	SkillBeginner     = "beginner"
	SkillIntermediate = "intermediate"
	SkillAdvanced     = "advanced"
	SkillExpert       = "expert"
)

// ValidSkillProficiencies is the set of allowed proficiency values.
var ValidSkillProficiencies = map[string]bool{
	SkillBeginner:     true,
	SkillIntermediate: true,
	SkillAdvanced:     true,
	SkillExpert:       true,
}

// ============================================================================
// Database Row Model
// ============================================================================

// Profile is a database row model. For API responses, use a DTO in dto.go.
// The flexible JSONB fields live in ProfileData and are marshalled/unmarshalled
// automatically via the Data field.
//
// Schema: profiles(id UUID PK, data JSONB, version INT, created_at, updated_at)
type Profile struct {
	ID        uuid.UUID   `db:"id"`
	Data      ProfileData `db:"data"`
	Version   int         `db:"version"`
	CreatedAt time.Time   `db:"created_at"`
	UpdatedAt time.Time   `db:"updated_at"`
}

// ============================================================================
// JSONB Document Model
// ============================================================================

// ProfileData is the JSONB payload stored in the profiles.data column.
// Structured into sub-groups to avoid a flat "everything bag" that grows
// unbounded. Each sub-struct has a single responsibility.
type ProfileData struct {
	Preferences ProfilePreferences `json:"preferences"`
	Skills      []Skill            `json:"skills,omitempty"`
	Education   []Education        `json:"education,omitempty"`
	Links       ProfileLinks       `json:"links,omitempty"`
}

// ProfilePreferences holds all job-search and resume generation preferences.
// Grouped together because they all influence how the agent searches,
// scores, and generates application materials.
type ProfilePreferences struct {
	// --- Job targeting ---
	TargetTitles    []string `json:"target_titles,omitempty"`
	TargetLocations []string `json:"target_locations,omitempty"`
	RemoteOnly      bool     `json:"remote_only,omitempty"`
	MinSalary       *int     `json:"min_salary,omitempty"`
	MaxSalary       *int     `json:"max_salary,omitempty"`

	// --- Work authorization ---
	// Free-text: "US Citizen", "H1B", "OPT", "Green Card", etc.
	// Kept as string because visa categories are not enumerable.
	WorkAuthorization string `json:"work_authorization,omitempty"`

	// --- Experience ---
	// Pointer (*int) so 0 is distinguishable from unset.
	YearsExperience *int `json:"years_experience,omitempty"`

	// --- Resume generation ---
	ResumeTone  string `json:"resume_tone,omitempty"`
	ResumeStyle string `json:"resume_style,omitempty"`

	// --- Application behavior ---
	// AutoApplyThreshold: score (0-100) above which applications are
	// submitted without human review. nil = require manual approval always.
	AutoApplyThreshold *int   `json:"auto_apply_threshold,omitempty"`
	CoverLetterStyle   string `json:"cover_letter_style,omitempty"`
}

// ProfileLinks holds external profile URLs.
// Separated because they are reference data, not preferences.
type ProfileLinks struct {
	LinkedIn  string `json:"linkedin,omitempty"`
	GitHub    string `json:"github,omitempty"`
	Portfolio string `json:"portfolio,omitempty"`
}

// Skill represents a single skill entry in the profile.
// Proficiency is constrained to the constants defined above.
type Skill struct {
	Name        string `json:"name"`
	Proficiency string `json:"proficiency,omitempty"`
	Years       int    `json:"years,omitempty"`
}

// Education represents a single education entry.
// Uses int years instead of string dates to enforce valid ranges.
type Education struct {
	Institution string `json:"institution"`
	Degree      string `json:"degree"`
	Field       string `json:"field,omitempty"`
	Description string `json:"description,omitempty"`
	StartYear   int    `json:"start_year,omitempty"`
	EndYear     int    `json:"end_year,omitempty"`
	GPA         string `json:"gpa,omitempty"`
}

// ============================================================================
// Validation
// ============================================================================

// Validate checks ProfileData for internal consistency.
// Called before persisting to prevent invalid data from entering JSONB.
func (pd ProfileData) Validate() error {
	// Salary range: min must be <= max when both are set.
	if pd.Preferences.MinSalary != nil && pd.Preferences.MaxSalary != nil {
		if *pd.Preferences.MinSalary > *pd.Preferences.MaxSalary {
			return fmt.Errorf("profile: min_salary (%d) must be <= max_salary (%d)",
				*pd.Preferences.MinSalary, *pd.Preferences.MaxSalary)
		}
	}

	// Auto-apply threshold: must be 0-100 when set.
	if pd.Preferences.AutoApplyThreshold != nil {
		t := *pd.Preferences.AutoApplyThreshold
		if t < 0 || t > 100 {
			return fmt.Errorf("profile: auto_apply_threshold must be 0-100, got %d", t)
		}
	}

	// Skills: proficiency must be a known value.
	for i, s := range pd.Skills {
		if s.Proficiency != "" && !ValidSkillProficiencies[s.Proficiency] {
			return fmt.Errorf("profile: skills[%d].proficiency %q is not valid (must be one of: beginner, intermediate, advanced, expert)",
				i, s.Proficiency)
		}
	}

	// Education: year ranges must be sensible.
	for i, e := range pd.Education {
		if e.StartYear != 0 && e.EndYear != 0 {
			if e.StartYear > e.EndYear {
				return fmt.Errorf("profile: education[%d]: start_year (%d) must be <= end_year (%d)",
					i, e.StartYear, e.EndYear)
			}
		}
		if e.StartYear != 0 && (e.StartYear < 1900 || e.StartYear > 2100) {
			return fmt.Errorf("profile: education[%d]: start_year %d is out of reasonable range", i, e.StartYear)
		}
	}

	return nil
}

// ============================================================================
// Mutation Rules (PATCH merge logic)
// ============================================================================

// ApplyPatch merges a PatchProfileRequest into the current ProfileData.
// This is the single source of truth for PATCH semantics.
//
// Rules:
//   - Nil pointer fields in patch → current value unchanged
//   - Non-nil pointer fields → overwrite current value
//   - Slices: nil = don't change, non-nil (even empty) = replace
//   - bool: *false is distinguishable from nil (not provided)
//
// The domain owns this mutation logic — the repository only reads and writes.
func (pd *ProfileData) ApplyPatch(patch PatchProfileRequest) {
	// --- Preferences ---
	if patch.Preferences != nil {
		p := patch.Preferences
		if p.TargetTitles != nil {
			pd.Preferences.TargetTitles = *p.TargetTitles
		}
		if p.TargetLocations != nil {
			pd.Preferences.TargetLocations = *p.TargetLocations
		}
		if p.RemoteOnly != nil {
			pd.Preferences.RemoteOnly = *p.RemoteOnly
		}
		if p.MinSalary != nil {
			pd.Preferences.MinSalary = p.MinSalary
		}
		if p.MaxSalary != nil {
			pd.Preferences.MaxSalary = p.MaxSalary
		}
		if p.WorkAuthorization != nil {
			pd.Preferences.WorkAuthorization = *p.WorkAuthorization
		}
		if p.YearsExperience != nil {
			pd.Preferences.YearsExperience = p.YearsExperience
		}
		if p.ResumeTone != nil {
			pd.Preferences.ResumeTone = *p.ResumeTone
		}
		if p.ResumeStyle != nil {
			pd.Preferences.ResumeStyle = *p.ResumeStyle
		}
		if p.AutoApplyThreshold != nil {
			pd.Preferences.AutoApplyThreshold = p.AutoApplyThreshold
		}
		if p.CoverLetterStyle != nil {
			pd.Preferences.CoverLetterStyle = *p.CoverLetterStyle
		}
	}

	// --- Skills: nil = don't change, non-nil = replace (even if empty) ---
	if patch.Skills != nil {
		pd.Skills = *patch.Skills
	}

	// --- Education: nil = don't change, non-nil = replace ---
	if patch.Education != nil {
		pd.Education = *patch.Education
	}

	// --- Links ---
	if patch.Links != nil {
		l := patch.Links
		if l.LinkedIn != nil {
			pd.Links.LinkedIn = *l.LinkedIn
		}
		if l.GitHub != nil {
			pd.Links.GitHub = *l.GitHub
		}
		if l.Portfolio != nil {
			pd.Links.Portfolio = *l.Portfolio
		}
	}
}

// ============================================================================
// driver.Valuer / sql.Scanner (JSONB serialization)
// ============================================================================

// Value implements driver.Valuer so ProfileData can be persisted to JSONB.
func (pd ProfileData) Value() (driver.Value, error) {
	b, err := json.Marshal(pd)
	if err != nil {
		return nil, fmt.Errorf("profile: marshal data: %w", err)
	}
	return b, nil
}

// Scan implements sql.Scanner so ProfileData can be read from JSONB.
// Handles both []byte (most PostgreSQL drivers) and string (some drivers).
func (pd *ProfileData) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("profile: scan data: unsupported type %T", src)
	}
	if err := json.Unmarshal(data, pd); err != nil {
		return fmt.Errorf("profile: unmarshal data: %w", err)
	}
	return nil
}
