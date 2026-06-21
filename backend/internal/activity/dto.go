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
	EntityType string `form:"entity_type"`
	EntityID   string `form:"entity_id"`
	EventType  string `form:"event_type"`
	StartTime  string `form:"start_time"`
	EndTime    string `form:"end_time"`
	Limit      int    `form:"limit"`
	Offset     int    `form:"offset"`
}

// ToListFilter converts the request to a domain ListFilter.
// Parses entity_id (UUID) and time strings (RFC3339) into typed values.
// Returns ErrInvalidEntityID or ErrInvalidTimeRange for malformed input.
func (r *ListFilterRequest) ToListFilter() (ListFilter, error) {
	var entityID uuid.UUID
	if r.EntityID != "" {
		var err error
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

// ActivityResponse is the API response for a single activity log entry.
// Returned by GET /activity-logs and GET /activity-logs/:id.
type ActivityResponse struct {
	ID         uuid.UUID `json:"id"`
	EventType  string    `json:"event_type"`
	EntityType string    `json:"entity_type"`
	EntityID   uuid.UUID `json:"entity_id"`
	Details    Details   `json:"details"`
	CreatedAt  time.Time `json:"created_at"`
}

// ActivityListResponse is the response for GET /activity-logs.
// Includes pagination metadata (total, limit, offset).
type ActivityListResponse struct {
	Activities []ActivityResponse `json:"activities"`
	Total      int64              `json:"total"`
	Limit      int                `json:"limit"`
	Offset     int                `json:"offset"`
}

// ToActivityResponse converts a domain ActivityLog to an API ActivityResponse.
// Used by the handler to map database entities to HTTP responses.
func ToActivityResponse(a *ActivityLog) ActivityResponse {
	return ActivityResponse{
		ID:         a.ID,
		EventType:  a.EventType,
		EntityType: a.EntityType,
		EntityID:   a.EntityID,
		Details:    a.Details,
		CreatedAt:  a.CreatedAt,
	}
}
