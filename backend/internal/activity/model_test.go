package activity

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// Event type constants
// ──────────────────────────────────────────────

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name   string
		constant string
		want   string
	}{
		{"EventJobDiscovered", EventJobDiscovered, "job_discovered"},
		{"EventJobScored", EventJobScored, "job_scored"},
		{"EventApplicationCreated", EventApplicationCreated, "application_created"},
		{"EventApplicationSubmitted", EventApplicationSubmitted, "application_submitted"},
		{"EventApplicationStatusChanged", EventApplicationStatusChanged, "application_status_changed"},
		{"EventResumeGenerated", EventResumeGenerated, "resume_generated"},
		{"EventResumeTailored", EventResumeTailored, "resume_tailored"},
		{"EventCoverLetterGenerated", EventCoverLetterGenerated, "cover_letter_generated"},
		{"EventEmailReceived", EventEmailReceived, "email_received"},
		{"EventEmailClassified", EventEmailClassified, "email_classified"},
		{"EventInterviewStarted", EventInterviewStarted, "interview_started"},
		{"EventInterviewCompleted", EventInterviewCompleted, "interview_completed"},
		{"EventApprovalRequested", EventApprovalRequested, "approval_requested"},
		{"EventApprovalApproved", EventApprovalApproved, "approval_approved"},
		{"EventApprovalRejected", EventApprovalRejected, "approval_rejected"},
		{"EventEmbeddingCreated", EventEmbeddingCreated, "embedding_created"},
		{"EventSearchPerformed", EventSearchPerformed, "search_performed"},
		{"EventError", EventError, "error"},
		{"EventInfo", EventInfo, "info"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.constant,
				"%s should equal %q", tt.name, tt.want)
			assert.NotEmpty(t, tt.constant,
				"%s must not be empty", tt.name)
		})
	}
}

// ──────────────────────────────────────────────
// Domain errors
// ──────────────────────────────────────────────

func TestDomainErrors(t *testing.T) {
	tests := []struct {
		name  string
		err   error
		want  string
		isNil bool
	}{
		{
			name: "ErrNotFound",
			err:  ErrNotFound,
			want: "activity not found",
		},
		{
			name: "ErrInvalidEntityID",
			err:  ErrInvalidEntityID,
			want: "invalid entity_id",
		},
		{
			name: "ErrInvalidTimeRange",
			err:  ErrInvalidTimeRange,
			want: "invalid time range",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Error(t, tt.err, "%s should be an error", tt.name)
			assert.Equal(t, tt.want, tt.err.Error(),
				"%s.Error() should match", tt.name)

			// Verify sentinel comparison works
			switch tt.name {
			case "ErrNotFound":
				assert.True(t, errors.Is(tt.err, ErrNotFound))
			case "ErrInvalidEntityID":
				assert.True(t, errors.Is(tt.err, ErrInvalidEntityID))
			case "ErrInvalidTimeRange":
				assert.True(t, errors.Is(tt.err, ErrInvalidTimeRange))
			}
		})
	}
}

// ──────────────────────────────────────────────
// ActivityLog model
// ──────────────────────────────────────────────

func TestActivityLog_ZeroValues(t *testing.T) {
	var a ActivityLog

	assert.Equal(t, uuid.Nil, a.ID, "zero ID should be uuid.Nil")
	assert.Empty(t, a.EventType, "zero EventType should be empty")
	assert.Empty(t, a.EntityType, "zero EntityType should be empty")
	assert.Equal(t, uuid.Nil, a.EntityID, "zero EntityID should be uuid.Nil")
	assert.Nil(t, a.Details, "zero Details should be nil")
	assert.True(t, a.CreatedAt.IsZero(), "zero CreatedAt should be zero time")
}

func TestActivityLog_TableName(t *testing.T) {
	tests := []struct {
		name string
		log  ActivityLog
		want string
	}{
		{"zero value", ActivityLog{}, "activity_log"},
		{"populated", ActivityLog{EventType: "test"}, "activity_log"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.log.TableName())
		})
	}
}

func TestActivityLog_Populated(t *testing.T) {
	id := uuid.New()
	eid := uuid.New()
	now := time.Now().Truncate(time.Microsecond) // PostgreSQL truncates to micros

	a := ActivityLog{
		ID:         id,
		EventType:  EventJobDiscovered,
		EntityType: "jobs",
		EntityID:   eid,
		Details:    Details{"title": "Senior Engineer", "source": "linkedin"},
		CreatedAt:  now,
	}

	assert.Equal(t, id, a.ID)
	assert.Equal(t, EventJobDiscovered, a.EventType)
	assert.Equal(t, "jobs", a.EntityType)
	assert.Equal(t, eid, a.EntityID)
	assert.Equal(t, Details{"title": "Senior Engineer", "source": "linkedin"}, a.Details)
	assert.Equal(t, now, a.CreatedAt)
}

// ──────────────────────────────────────────────
// Details: Value (driver.Valuer)
// ──────────────────────────────────────────────

func TestDetailsValue(t *testing.T) {
	tests := []struct {
		name    string
		details Details
		wantNil bool
		wantJSON string
	}{
		{
			name:    "nil",
			details: nil,
			wantNil: true,
		},
		{
			name:    "empty map",
			details: Details{},
			wantNil: false,
			wantJSON: `{}`,
		},
		{
			name:    "string value",
			details: Details{"key": "value"},
			wantNil: false,
			wantJSON: `{"key":"value"}`,
		},
		{
			name:    "numeric value",
			details: Details{"score": float64(0.95)},
			wantNil: false,
			wantJSON: `{"score":0.95}`,
		},
		{
			name: "nested object",
			details: Details{
				"user": map[string]any{
					"name": "Alice",
					"age":  float64(30),
				},
			},
			wantNil:  false,
			wantJSON: `{"user":{"age":30,"name":"Alice"}}`,
		},
		{
			name: "array value",
			details: Details{
				"tags": []any{"go", "testing"},
			},
			wantNil:  false,
			wantJSON: `{"tags":["go","testing"]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := tt.details.Value()
			require.NoError(t, err, "Value() should not error")

			if tt.wantNil {
				assert.Nil(t, v, "Value() should return nil for nil/empty Details")
				return
			}

			b, ok := v.([]byte)
			require.True(t, ok, "Value() should return []byte for non-nil Details")
			assert.JSONEq(t, tt.wantJSON, string(b),
				"Value() should produce correct JSON")
		})
	}
}

// ──────────────────────────────────────────────
// Details: Scan (sql.Scanner)
// ──────────────────────────────────────────────

func TestDetailsScan(t *testing.T) {
	tests := []struct {
		name      string
		src       any
		wantNil   bool
		want      Details
		wantErr   bool
		errContains string
	}{
		{
			name:    "nil src",
			src:     nil,
			wantNil: true,
		},
		{
			name: "byte slice valid JSON",
			src:  []byte(`{"key":"value"}`),
			want: Details{"key": "value"},
		},
		{
			name: "string valid JSON",
			src:  `{"count":42}`,
			want: Details{"count": float64(42)},
		},
		{
			name:    "empty byte slice",
			src:     []byte{},
			wantNil: true,
		},
		{
			name:    "empty string",
			src:     "",
			wantNil: true,
		},
		{
			name:      "unsupported type int",
			src:       42,
			wantErr:   true,
			errContains: "unsupported type int",
		},
		{
			name:      "unsupported type bool",
			src:       true,
			wantErr:   true,
			errContains: "unsupported type bool",
		},
		{
			name:      "invalid JSON",
			src:       []byte(`{invalid}`),
			wantErr:   true,
			errContains: "unmarshal",
		},
		{
			name: "nested JSON",
			src:  []byte(`{"nested":{"a":1,"b":"two"},"arr":[1,2,3]}`),
			want: Details{
				"nested": map[string]any{"a": float64(1), "b": "two"},
				"arr":    []any{float64(1), float64(2), float64(3)},
			},
		},
		{
			name:    "null literal byte slice",
			src:     []byte(`null`),
			wantNil: true, // json.Unmarshal into nil map stays nil
		},
		{
			name: "JSON with special chars",
			src:  []byte(`{"msg":"hello \"world\"\nnext line"}`),
			want: Details{"msg": "hello \"world\"\nnext line"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d Details
			err := d.Scan(tt.src)

			if tt.wantErr {
				assert.Error(t, err, "Scan should return error")
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains,
						"error message should contain expected text")
				}
				return
			}

			require.NoError(t, err, "Scan should not error")

			if tt.wantNil {
				assert.Nil(t, d, "Details should be nil")
				return
			}

			assert.Equal(t, tt.want, d, "Details should match expected")
		})
	}
}

// Scan with non-nil pointer receiver (Details already allocated).
// json.Unmarshal merges into existing maps rather than replacing them.
func TestDetailsScan_OnPopulatedTarget(t *testing.T) {
	d := Details{"existing": "value"}
	err := d.Scan([]byte(`{"new":"data"}`))
	require.NoError(t, err)
	assert.Equal(t, Details{"existing": "value", "new": "data"}, d,
		"json.Unmarshal merges into existing map — "+
			"existing keys are preserved, new keys are added")
}

// Scan null onto populated target produces nil Details.
func TestDetailsScan_NullOverwrites(t *testing.T) {
	d := Details{"existing": "value"}
	err := d.Scan(nil)
	require.NoError(t, err)
	assert.Nil(t, d, "Scan(nil) should set Details to nil")

	// Also test empty byte slice
	d = Details{"existing": "value"}
	err = d.Scan([]byte{})
	require.NoError(t, err)
	assert.Nil(t, d, "Scan([]byte{}) should set Details to nil")
}

// ──────────────────────────────────────────────
// activityColumns constant
// ──────────────────────────────────────────────

func TestActivityColumns(t *testing.T) {
	// The constant is a raw string literal with leading/trailing whitespace.
	// Verify structural content rather than exact formatting.
	assert.NotContains(t, activityColumns, "*",
		"activityColumns must not use SELECT *")
	assert.Contains(t, activityColumns, "id")
	assert.Contains(t, activityColumns, "event_type")
	assert.Contains(t, activityColumns, "entity_type")
	assert.Contains(t, activityColumns, "entity_id")
	assert.Contains(t, activityColumns, "details")
	assert.Contains(t, activityColumns, "created_at")

	// Verify whitespace boundaries
	assert.Contains(t, activityColumns, "\n\tid")
	assert.Contains(t, activityColumns, "created_at\n")

	// Strip whitespace/newlines and verify column order
	got := strings.ReplaceAll(strings.ReplaceAll(activityColumns, "\n", ""), "\t", "")
	const want = "id, event_type, entity_type, entity_id, details, created_at"
	assert.Equal(t, want, got,
		"activityColumns columns (ignoring whitespace) should match schema")
}
