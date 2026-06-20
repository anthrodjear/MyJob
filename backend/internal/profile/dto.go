// DTOs (Data Transfer Types) for the profile domain.
//
// The profile is a singleton resource — one profile per user.
// API surfaces are simple: GET (read), PUT (replace), PATCH (partial update).
//
// Request DTOs define the API contract for incoming payloads.
// Response DTOs define the API contract for outgoing payloads.
// Mappers convert between domain models and response DTOs.
//
// This file contains NO business logic. Validation happens here
// (binding tags) and in the service layer (business rules).
package profile

import (
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Request DTOs
// ---------------------------------------------------------------------------

// UpdateProfileRequest is the payload for PUT /api/v1/profile.
//
// Replaces the entire profile data. The client is expected to fetch
// the current profile first, modify it, and PUT the full result.
//
// Example:
//
//	{
//	  "preferences": {
//	    "target_titles": ["Backend Engineer", "Platform Engineer"],
//	    "target_locations": ["Remote", "New York"],
//	    "remote_only": true,
//	    "min_salary": 150000,
//	    "work_authorization": "US Citizen"
//	  },
//	  "skills": [
//	    {"name": "Go", "proficiency": "advanced", "years": 5},
//	    {"name": "PostgreSQL", "proficiency": "intermediate", "years": 3}
//	  ],
//	  "education": [
//	    {"institution": "MIT", "degree": "BS", "field": "CS", "start_year": 2015, "end_year": 2019}
//	  ],
//	  "links": {
//	    "linkedin": "https://linkedin.com/in/example",
//	    "github": "https://github.com/example"
//	  }
//	}
type UpdateProfileRequest struct {
	Preferences ProfilePreferences `json:"preferences"`
	Skills      []Skill            `json:"skills,omitempty"`
	Education   []Education        `json:"education,omitempty"`
	Links       ProfileLinks       `json:"links,omitempty"`
}

// PatchProfileRequest is the payload for PATCH /api/v1/profile.
//
// Partially updates profile data. Only provided fields are merged
// into the existing profile. Nil pointer fields are ignored (don't change).
// Non-nil pointer fields overwrite the current value.
// Slice fields: nil = don't change, non-nil (even empty) = replace.
//
// Example (disable remote-only, update one skill):
//
//	{
//	  "preferences": {
//	    "remote_only": false
//	  },
//	  "skills": [
//	    {"name": "Go", "proficiency": "expert", "years": 6}
//	  ]
//	}
type PatchProfileRequest struct {
	Preferences *PatchPreferences `json:"preferences,omitempty"`
	Skills      *[]Skill          `json:"skills,omitempty"`
	Education   *[]Education      `json:"education,omitempty"`
	Links       *PatchLinks       `json:"links,omitempty"`
}

// PatchPreferences is the preferences sub-object for PATCH requests.
// All fields are pointers so nil means "don't change" and non-nil means "set this".
// This solves the bool/int zero-value ambiguity.
type PatchPreferences struct {
	TargetTitles    *[]string `json:"target_titles,omitempty"`
	TargetLocations *[]string `json:"target_locations,omitempty"`
	RemoteOnly      *bool     `json:"remote_only,omitempty"`
	MinSalary       *int      `json:"min_salary,omitempty"`
	MaxSalary       *int      `json:"max_salary,omitempty"`
	WorkAuthorization *string `json:"work_authorization,omitempty"`
	YearsExperience *int      `json:"years_experience,omitempty"`
	ResumeTone      *string   `json:"resume_tone,omitempty"`
	ResumeStyle     *string   `json:"resume_style,omitempty"`
	AutoApplyThreshold *int   `json:"auto_apply_threshold,omitempty"`
	CoverLetterStyle   *string `json:"cover_letter_style,omitempty"`
}

// PatchLinks is the links sub-object for PATCH requests.
type PatchLinks struct {
	LinkedIn  *string `json:"linkedin,omitempty"`
	GitHub    *string `json:"github,omitempty"`
	Portfolio *string `json:"portfolio,omitempty"`
}

// ---------------------------------------------------------------------------
// Response DTOs
// ---------------------------------------------------------------------------

// ProfileResponse is the API response for GET /api/v1/profile.
//
// Returns the full profile with embedded stats. The version is NOT
// in the JSON body — it is returned as an ETag header for use with
// If-Match on PUT/PATCH requests (optimistic concurrency).
//
// Example response:
//
//	{
//	  "id": "550e8400-e29b-41d4-a716-446655440000",
//	  "data": { "preferences": {...}, "skills": [...], ... },
//	  "stats": { "skill_count": 5, "education_count": 2, ... },
//	  "created_at": "2026-06-19T10:00:00Z",
//	  "updated_at": "2026-06-20T14:30:00Z"
//	}
type ProfileResponse struct {
	ID        uuid.UUID             `json:"id"`
	Data      ProfileData           `json:"data"`
	Stats     ProfileStatsResponse  `json:"stats"`
	CreatedAt time.Time             `json:"created_at"`
	UpdatedAt time.Time             `json:"updated_at"`
}

// ProfileStatsResponse is embedded in ProfileResponse.
// Computed on every GET — cheap for a single-user local system.
type ProfileStatsResponse struct {
	SkillCount     int  `json:"skill_count"`
	EducationCount int  `json:"education_count"`
	HasResumePrefs bool `json:"has_resume_preferences"`
	HasLinks       bool `json:"has_links"`
}

// ---------------------------------------------------------------------------
// Mappers
// ---------------------------------------------------------------------------

// ToResponse converts a domain Profile to an API ProfileResponse.
// Includes computed stats — no separate /stats endpoint needed.
func ToResponse(p *Profile) ProfileResponse {
	return ProfileResponse{
		ID:        p.ID,
		Data:      p.Data,
		Stats:     ToStatsResponse(p),
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

// ToStatsResponse derives aggregate stats from a Profile.
func ToStatsResponse(p *Profile) ProfileStatsResponse {
	return ProfileStatsResponse{
		SkillCount:     len(p.Data.Skills),
		EducationCount: len(p.Data.Education),
		HasResumePrefs: p.Data.Preferences.ResumeTone != "" || p.Data.Preferences.ResumeStyle != "",
		HasLinks:       p.Data.Links.LinkedIn != "" || p.Data.Links.GitHub != "" || p.Data.Links.Portfolio != "",
	}
}
