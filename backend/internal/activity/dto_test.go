package activity

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// ListFilterRequest struct
// ──────────────────────────────────────────────

func TestListFilterRequest_Struct(t *testing.T) {
	t.Run("zero values", func(t *testing.T) {
		var req ListFilterRequest
		assert.Empty(t, req.EntityType)
		assert.Empty(t, req.EntityID)
		assert.Empty(t, req.EventType)
		assert.Empty(t, req.StartTime)
		assert.Empty(t, req.EndTime)
		assert.Zero(t, req.Limit)
		assert.Zero(t, req.Offset)
	})

	t.Run("populated", func(t *testing.T) {
		req := ListFilterRequest{
			EntityType: "jobs",
			EntityID:   "550e8400-e29b-41d4-a716-446655440000",
			EventType:  EventJobDiscovered,
			StartTime:  "2024-01-01T00:00:00Z",
			EndTime:    "2024-12-31T23:59:59Z",
			Limit:      50,
			Offset:     10,
		}
		// Verify form tags are present (compile-time check via field access)
		assert.Equal(t, "jobs", req.EntityType)
		assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", req.EntityID)
		assert.Equal(t, EventJobDiscovered, req.EventType)
		assert.Equal(t, "2024-01-01T00:00:00Z", req.StartTime)
		assert.Equal(t, "2024-12-31T23:59:59Z", req.EndTime)
		assert.Equal(t, 50, req.Limit)
		assert.Equal(t, 10, req.Offset)
	})
}

// ──────────────────────────────────────────────
// ToListFilter mapper
// ──────────────────────────────────────────────

func TestToListFilter(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	parsedUUID := uuid.MustParse(validUUID)

	startRFC3339 := "2024-01-15T10:30:00Z"
	parsedStart, _ := time.Parse(time.RFC3339, startRFC3339)

	endRFC3339 := "2024-06-30T18:00:00Z"
	parsedEnd, _ := time.Parse(time.RFC3339, endRFC3339)

	tests := []struct {
		name    string
		req     ListFilterRequest
		want    ListFilter
		wantErr error
	}{
		{
			name: "empty request / no filters",
			req:  ListFilterRequest{},
			want: ListFilter{
				EntityID:  uuid.Nil,
				StartTime: time.Time{},
				EndTime:   time.Time{},
			},
		},
		{
			name: "only entity_type",
			req: ListFilterRequest{
				EntityType: "applications",
			},
			want: ListFilter{
				EntityType: "applications",
				EntityID:   uuid.Nil,
			},
		},
		{
			name: "only event_type",
			req: ListFilterRequest{
				EventType: EventEmailClassified,
			},
			want: ListFilter{
				EventType: EventEmailClassified,
				EntityID:  uuid.Nil,
			},
		},
		{
			name: "valid entity_id",
			req: ListFilterRequest{
				EntityID: validUUID,
			},
			want: ListFilter{
				EntityID: parsedUUID,
			},
		},
		{
			name: "valid start_time",
			req: ListFilterRequest{
				StartTime: startRFC3339,
			},
			want: ListFilter{
				EntityID:  uuid.Nil,
				StartTime: parsedStart,
			},
		},
		{
			name: "valid end_time",
			req: ListFilterRequest{
				EndTime: endRFC3339,
			},
			want: ListFilter{
				EntityID: uuid.Nil,
				EndTime:  parsedEnd,
			},
		},
		{
			name: "both start and end time",
			req: ListFilterRequest{
				StartTime: startRFC3339,
				EndTime:   endRFC3339,
			},
			want: ListFilter{
				EntityID:  uuid.Nil,
				StartTime: parsedStart,
				EndTime:   parsedEnd,
			},
		},
		{
			name: "limit and offset preserved",
			req: ListFilterRequest{
				Limit:  25,
				Offset: 5,
			},
			want: ListFilter{
				EntityID: uuid.Nil,
				Limit:    25,
				Offset:   5,
			},
		},
		{
			name: "all fields populated",
			req: ListFilterRequest{
				EntityType: "jobs",
				EntityID:   validUUID,
				EventType:  EventJobDiscovered,
				StartTime:  startRFC3339,
				EndTime:    endRFC3339,
				Limit:      10,
				Offset:     0,
			},
			want: ListFilter{
				EntityType: "jobs",
				EntityID:   parsedUUID,
				EventType:  EventJobDiscovered,
				StartTime:  parsedStart,
				EndTime:    parsedEnd,
				Limit:      10,
				Offset:     0,
			},
		},
		{
			name: "invalid entity_id",
			req: ListFilterRequest{
				EntityID: "not-a-uuid",
			},
			wantErr: ErrInvalidEntityID,
		},
		{
			name: "invalid start_time",
			req: ListFilterRequest{
				StartTime: "2024/01/15 10:30",
			},
			wantErr: ErrInvalidTimeRange,
		},
		{
			name: "invalid end_time",
			req: ListFilterRequest{
				EndTime: "not a time",
			},
			wantErr: ErrInvalidTimeRange,
		},
		{
			name: "invalid time without timezone",
			req: ListFilterRequest{
				StartTime: "2024-01-15T10:30:00", // missing Z or offset
			},
			wantErr: ErrInvalidTimeRange,
		},
		{
			name: "uuid with extra hyphens",
			req: ListFilterRequest{
				EntityID: "550e8400-e29b-41d4-a716-446655440000-extra",
			},
			wantErr: ErrInvalidEntityID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.req.ToListFilter()

			if tt.wantErr != nil {
				assert.Error(t, err, "ToListFilter should return error")
				assert.ErrorIs(t, err, tt.wantErr,
					"error should be %v", tt.wantErr)
				assert.Zero(t, got, "result should be zero-value on error")
				return
			}

			require.NoError(t, err, "ToListFilter should not error")
			assert.Equal(t, tt.want, got,
				"ToListFilter result should match expected")
		})
	}
}

// ToListFilter on nil receiver — panics is expected for nil pointer.
func TestToListFilter_NilReceiver(t *testing.T) {
	var req *ListFilterRequest
	assert.Panics(t, func() {
		_, _ = req.ToListFilter()
	}, "ToListFilter on nil receiver should panic")
}

// ──────────────────────────────────────────────
// ActivityResponse struct
// ──────────────────────────────────────────────

func TestActivityResponse_Struct(t *testing.T) {
	t.Run("zero values", func(t *testing.T) {
		var resp ActivityResponse
		assert.Equal(t, uuid.Nil, resp.ID)
		assert.Empty(t, resp.EventType)
		assert.Empty(t, resp.EntityType)
		assert.Equal(t, uuid.Nil, resp.EntityID)
		assert.Nil(t, resp.Details)
		assert.True(t, resp.CreatedAt.IsZero())
	})

	t.Run("populated", func(t *testing.T) {
		id := uuid.New()
		eid := uuid.New()
		now := time.Now()

		resp := ActivityResponse{
			ID:         id,
			EventType:  EventInterviewCompleted,
			EntityType: "interviews",
			EntityID:   eid,
			Details:    DetailsResponse{"duration_sec": float64(1800)},
			CreatedAt:  now,
		}
		assert.Equal(t, id, resp.ID)
		assert.Equal(t, EventInterviewCompleted, resp.EventType)
		assert.Equal(t, "interviews", resp.EntityType)
		assert.Equal(t, eid, resp.EntityID)
		assert.Equal(t, DetailsResponse{"duration_sec": float64(1800)}, resp.Details)
		assert.Equal(t, now, resp.CreatedAt)
	})
}

// ──────────────────────────────────────────────
// ActivityListResponse struct
// ──────────────────────────────────────────────

func TestActivityListResponse_Struct(t *testing.T) {
	t.Run("zero values", func(t *testing.T) {
		var list ActivityListResponse
		assert.Nil(t, list.Activities)
		assert.Zero(t, list.Total)
		assert.Zero(t, list.Limit)
		assert.Zero(t, list.Offset)
	})

	t.Run("populated", func(t *testing.T) {
		id1 := uuid.New()
		id2 := uuid.New()
		eid := uuid.New()

		list := ActivityListResponse{
			Activities: []ActivityResponse{
				{
					ID:         id1,
					EventType:  EventJobDiscovered,
					EntityType: "jobs",
					EntityID:   eid,
					Details:    DetailsResponse{"title": "Engineer"},
				},
				{
					ID:         id2,
					EventType:  EventJobScored,
					EntityType: "jobs",
					EntityID:   eid,
					Details:    DetailsResponse{"score": float64(85)},
				},
			},
			Total:  42,
			Limit:  20,
			Offset: 0,
		}

		assert.Len(t, list.Activities, 2)
		assert.Equal(t, id1, list.Activities[0].ID)
		assert.Equal(t, id2, list.Activities[1].ID)
		assert.Equal(t, int64(42), list.Total)
		assert.Equal(t, 20, list.Limit)
		assert.Equal(t, 0, list.Offset)
	})

	t.Run("empty activities slice", func(t *testing.T) {
		list := ActivityListResponse{
			Activities: []ActivityResponse{},
			Total:      0,
			Limit:      20,
			Offset:     0,
		}
		assert.Empty(t, list.Activities)
		assert.NotNil(t, list.Activities, "empty slice should not be nil")
	})
}

// ──────────────────────────────────────────────
// ToActivityResponse mapper
// ──────────────────────────────────────────────

func TestToActivityResponse(t *testing.T) {
	id := uuid.New()
	eid := uuid.New()
	now := time.Now().Truncate(time.Microsecond)

	a := &ActivityLog{
		ID:         id,
		EventType:  EventApprovalApproved,
		EntityType: "applications",
		EntityID:   eid,
		Details:    Details{"approved_by": "human"},
		CreatedAt:  now,
	}

	resp := ToActivityResponse(a)

	assert.Equal(t, a.ID, resp.ID)
	assert.Equal(t, a.EventType, resp.EventType)
	assert.Equal(t, a.EntityType, resp.EntityType)
	assert.Equal(t, a.EntityID, resp.EntityID)
	assert.Equal(t, DetailsResponse(a.Details), resp.Details)
	assert.Equal(t, a.CreatedAt, resp.CreatedAt)

	// Verify it's a value copy, not a shared reference
	a.EventType = "mutated"
	assert.NotEqual(t, a.EventType, resp.EventType,
		"ToActivityResponse should return a copy, not a reference")
}

func TestToActivityResponse_WithNilDetails(t *testing.T) {
	a := &ActivityLog{
		ID:         uuid.New(),
		EventType:  EventInfo,
		EntityType: "system",
		EntityID:   uuid.Nil,
		Details:    nil,
		CreatedAt:  time.Now(),
	}

	resp := ToActivityResponse(a)
	assert.Nil(t, resp.Details, "nil Details should remain nil in response")
}

func TestToActivityResponse_WithEmptyDetails(t *testing.T) {
	a := &ActivityLog{
		ID:         uuid.New(),
		EventType:  EventInfo,
		EntityType: "system",
		EntityID:   uuid.Nil,
		Details:    Details{},
		CreatedAt:  time.Now(),
	}

	resp := ToActivityResponse(a)
	assert.NotNil(t, resp.Details, "empty Details should be non-nil map")
	assert.Empty(t, resp.Details, "empty Details should have zero entries")
}

func TestToActivityResponse_ZeroActivityLog(t *testing.T) {
	a := &ActivityLog{}
	resp := ToActivityResponse(a)

	assert.Equal(t, uuid.Nil, resp.ID)
	assert.Empty(t, resp.EventType)
	assert.Empty(t, resp.EntityType)
	assert.Equal(t, uuid.Nil, resp.EntityID)
	assert.Nil(t, resp.Details)
	assert.True(t, resp.CreatedAt.IsZero())
}

func TestToActivityResponse_NilPointer(t *testing.T) {
	assert.Panics(t, func() {
		ToActivityResponse(nil)
	}, "ToActivityResponse should panic on nil *ActivityLog")
}
