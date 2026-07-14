// DTOs (Data Transfer Types) for the activity domain.
//
// API surface:
//   - GET /activity-logs         → List activity logs with filters
//   - GET /activity-logs/:id     → Get single activity log
//
// Activity logs are write-only from the API perspective — other domains
// create events via Service.LogEvent(), not via HTTP endpoints.
// The API provides read access for the frontend activity feed.
package activity

import (
	"time"

	"github.com/google/uuid"
)

// ListFilterRequest maps GET /activity-logs query parameters.
// All fields are optional — zero values mean "no filter".
type ListFilterRequest struct {
	EntityType string `form:"entity_type" example:"application" enums:"job,application,resume,cover_letter,interview,email,approval"`
	EntityID   string `form:"entity_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	EventType  string `form:"event_type" example:"status_changed" enums:"created,updated,deleted,status_changed,scored,applied,email_received,interview_scheduled"`
	StartTime  string `form:"start_time" example:"2026-01-01T00:00:00Z"`
	EndTime    string `form:"end_time" example:"2026-12-31T23:59:59Z"`
	Limit      int    `form:"limit" example:"50" default:"20" minimum:"1" maximum:"100"`
	Offset     int    `form:"offset" example:"0" minimum:"0"`
}

// ToListFilter converts the request to a domain ListFilter.
// Parses entity_id (UUID) and time strings (RFC3339) into typed values.
// Returns ErrInvalidEntityID or ErrInvalidTimeRange for malformed input.
func (r *ListFilterRequest) ToListFilter() (ListFilter, error) {
	var entityID uuid.UUID
	if r.EntityID != "" {
		err := error(nil)
		entityID, err = uuid.Parse(r.EntityID)
		if err != nil {
			return ListFilter{}, ErrInvalidEntityID
		}
	}

	var startTime, endTime time.Time
	if r.StartTime != "" {
		t, err := time.Parse(time.RFC3339, r.StartTime)
		if err != nil {
			return ListFilter{}, ErrInvalidTimeRange
		}
		startTime = t
	}
	if r.EndTime != "" {
		t, err := time.Parse(time.RFC3339, r.EndTime)
		if err != nil {
			return ListFilter{}, ErrInvalidTimeRange
		}
		endTime = t
	}

	return ListFilter{
		EntityType: r.EntityType,
		EntityID:   entityID,
		EventType:  r.EventType,
		StartTime:  startTime,
		EndTime:    endTime,
		Limit:      r.Limit,
		Offset:     r.Offset,
	}, nil
}

// DetailsResponse is the API representation of activity details.
// Each event type populates this with relevant context:
//
//	EventJobDiscovered → {"title": "...", "company": "...", "source": "indeed"}
//	EventEmailClassified → {"category": "interview_invite", "confidence": 0.95}
//	EventError → {"error": "timeout", "component": "scoring"}
type DetailsResponse map[string]any

// ActivityResponse is the API response for a single activity log entry.
// Returned by GET /activity-logs and GET /activity-logs/:id.
type ActivityResponse struct {
	ID         uuid.UUID       `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	EventType  string          `json:"event_type" example:"status_changed" enums:"created,updated,deleted,status_changed,scored,applied,email_received,interview_scheduled"`
	EntityType string          `json:"entity_type" example:"application" enums:"job,application,resume,cover_letter,interview,email,approval"`
	EntityID   uuid.UUID       `json:"entity_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Details    DetailsResponse `json:"details"`
	CreatedAt  time.Time       `json:"created_at" example:"2026-06-19T14:30:00Z"`
}

// ActivityListResponse is the response for GET /activity-logs.
// Includes pagination metadata (total, limit, offset).
type ActivityListResponse struct {
	Activities []ActivityResponse `json:"activities"`
	Total      int64              `json:"total" example:"150"`
	Limit      int                `json:"limit" example:"20"`
	Offset     int                `json:"offset" example:"0"`
}

// ToActivityResponse converts a domain ActivityLog to an API ActivityResponse.
// Used by the handler to map database entities to HTTP responses.
func ToActivityResponse(a *ActivityLog) ActivityResponse {
	return ActivityResponse{
		ID:         a.ID,
		EventType:  a.EventType,
		EntityType: a.EntityType,
		EntityID:   a.EntityID,
		Details:    DetailsResponse(a.Details),
		CreatedAt:  a.CreatedAt,
	}
}
