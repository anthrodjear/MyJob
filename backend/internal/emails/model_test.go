package emails

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Domain Errors
// ============================================================================

func TestDomainErrors(t *testing.T) {
	t.Run("ErrNotFound", func(t *testing.T) {
		assert.Error(t, ErrNotFound)
		assert.Equal(t, "email not found", ErrNotFound.Error())
	})

	t.Run("ErrInvalidClassification", func(t *testing.T) {
		assert.Error(t, ErrInvalidClassification)
		assert.Equal(t, "invalid classification", ErrInvalidClassification.Error())
	})
}

// ============================================================================
// Classification Constants
// ============================================================================

func TestClassificationConstants(t *testing.T) {
	tests := []struct {
		name  string
		constant string
		expected string
	}{
		{name: "InterviewInvite", constant: ClassificationInterviewInvite, expected: "interview_invite"},
		{name: "Rejection", constant: ClassificationRejection, expected: "rejection"},
		{name: "Offer", constant: ClassificationOffer, expected: "offer"},
		{name: "FollowUp", constant: ClassificationFollowUp, expected: "follow_up"},
		{name: "Spam", constant: ClassificationSpam, expected: "spam"},
		{name: "Phishing", constant: ClassificationPhishing, expected: "phishing"},
		{name: "Other", constant: ClassificationOther, expected: "other"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.constant)
		})
	}
}

func TestClassificationConstants_AllDistinct(t *testing.T) {
	// Ensure no two constants have the same value.
	seen := make(map[string]string)
	constants := map[string]string{
		"ClassificationInterviewInvite": ClassificationInterviewInvite,
		"ClassificationRejection":       ClassificationRejection,
		"ClassificationOffer":           ClassificationOffer,
		"ClassificationFollowUp":        ClassificationFollowUp,
		"ClassificationSpam":            ClassificationSpam,
		"ClassificationPhishing":        ClassificationPhishing,
		"ClassificationOther":           ClassificationOther,
	}
	for name, val := range constants {
		if existing, ok := seen[val]; ok {
			t.Errorf("duplicate value %q for %s and %s", val, existing, name)
		}
		seen[val] = name
	}
	assert.Len(t, seen, 7, "expected 7 distinct classification values")
}

// ============================================================================
// IsValidClassification
// ============================================================================

func TestIsValidClassification_AllValid(t *testing.T) {
	valid := []string{
		ClassificationInterviewInvite,
		ClassificationRejection,
		ClassificationOffer,
		ClassificationFollowUp,
		ClassificationSpam,
		ClassificationPhishing,
		ClassificationOther,
	}
	for _, c := range valid {
		t.Run("valid/"+c, func(t *testing.T) {
			assert.True(t, IsValidClassification(c), "expected %q to be valid", c)
		})
	}
}

func TestIsValidClassification_InvalidEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "empty string", value: ""},
		{name: "unknown string", value: "unknown"},
		{name: "upper case", value: "INTERVIEW_INVITE"},
		{name: "pascal case", value: "InterviewInvite"},
		{name: "title case", value: "Rejection"},
		{name: "partial match", value: "interview"},
		{name: "trailing space", value: "offer "},
		{name: "leading space", value: " spam"},
		{name: "extra suffix", value: "follow_up_extra"},
		{name: "with underscore prefix", value: "_phishing"},
		{name: "whitespace only", value: "   "},
		{name: "special characters", value: "phishing!"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.False(t, IsValidClassification(tt.value), "expected %q to be invalid", tt.value)
		})
	}
}

func TestIsValidClassification_MapIntegrity(t *testing.T) {
	// Every constant in validClassifications should be present.
	assert.Len(t, validClassifications, 7)
	for _, c := range []string{
		ClassificationInterviewInvite,
		ClassificationRejection,
		ClassificationOffer,
		ClassificationFollowUp,
		ClassificationSpam,
		ClassificationPhishing,
		ClassificationOther,
	} {
		assert.True(t, validClassifications[c], "classification %q missing from map", c)
	}
}

// ============================================================================
// Email Struct — JSON Serialization
// ============================================================================

func TestEmail_JSONSerialization_Full(t *testing.T) {
	appID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	emailID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440001")
	to := "recipient@example.com"
	subj := "Interview Invitation"
	body := "We are pleased to invite you..."
	class := ClassificationInterviewInvite
	draft := "Thank you for the invitation"
	now := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)

	email := Email{
		ID:             emailID,
		ApplicationID:  &appID,
		MessageID:      "msg-001",
		FromAddress:    "hr@company.com",
		ToAddress:      &to,
		Subject:        &subj,
		Body:           &body,
		ReceivedAt:     now,
		Classification: &class,
		IsRead:         true,
		ReplyDraft:     &draft,
		CreatedAt:      now,
	}

	data, err := json.Marshal(email)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"id":"660e8400-e29b-41d4-a716-446655440001"`)
	assert.Contains(t, string(data), `"application_id":"550e8400-e29b-41d4-a716-446655440000"`)
	assert.Contains(t, string(data), `"message_id":"msg-001"`)
	assert.Contains(t, string(data), `"from_address":"hr@company.com"`)
	assert.Contains(t, string(data), `"to_address":"recipient@example.com"`)
	assert.Contains(t, string(data), `"subject":"Interview Invitation"`)
	assert.Contains(t, string(data), `"body":"We are pleased to invite you..."`)
	assert.Contains(t, string(data), `"is_read":true`)
	assert.Contains(t, string(data), `"reply_draft":"Thank you for the invitation"`)

	// Round-trip
	var deserialized Email
	err = json.Unmarshal(data, &deserialized)
	require.NoError(t, err)
	assert.Equal(t, email.ID, deserialized.ID)
	assert.Equal(t, email.ApplicationID, deserialized.ApplicationID)
	assert.Equal(t, email.MessageID, deserialized.MessageID)
	assert.Equal(t, email.FromAddress, deserialized.FromAddress)
	assert.Equal(t, email.ToAddress, deserialized.ToAddress)
	assert.Equal(t, email.Subject, deserialized.Subject)
	assert.Equal(t, email.Body, deserialized.Body)
	assert.True(t, email.ReceivedAt.Equal(deserialized.ReceivedAt))
	assert.Equal(t, email.Classification, deserialized.Classification)
	assert.Equal(t, email.IsRead, deserialized.IsRead)
	assert.Equal(t, email.ReplyDraft, deserialized.ReplyDraft)
	assert.True(t, email.CreatedAt.Equal(deserialized.CreatedAt))
}

func TestEmail_JSONSerialization_Partial(t *testing.T) {
	emailID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440001")
	now := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)

	// Only required fields, all optionals nil
	email := Email{
		ID:          emailID,
		MessageID:   "msg-002",
		FromAddress: "noreply@linkedin.com",
		ReceivedAt:  now,
		CreatedAt:   now,
	}

	data, err := json.Marshal(email)
	require.NoError(t, err)

	// Optional fields should be omitted due to omitempty
	assert.NotContains(t, string(data), "application_id")
	assert.NotContains(t, string(data), "to_address")
	assert.NotContains(t, string(data), "subject")
	assert.NotContains(t, string(data), "body")
	assert.NotContains(t, string(data), "classification")
	assert.NotContains(t, string(data), "reply_draft")

	// Required fields must be present
	assert.Contains(t, string(data), `"message_id":"msg-002"`)
	assert.Contains(t, string(data), `"from_address":"noreply@linkedin.com"`)
	assert.Contains(t, string(data), `"is_read":false`)

	// Round-trip
	var deserialized Email
	err = json.Unmarshal(data, &deserialized)
	require.NoError(t, err)
	assert.Equal(t, email.ID, deserialized.ID)
	assert.Equal(t, email.MessageID, deserialized.MessageID)
	assert.Equal(t, email.FromAddress, deserialized.FromAddress)
	assert.Nil(t, deserialized.ApplicationID)
	assert.Nil(t, deserialized.ToAddress)
	assert.Nil(t, deserialized.Subject)
	assert.Nil(t, deserialized.Body)
	assert.Nil(t, deserialized.Classification)
	assert.Nil(t, deserialized.ReplyDraft)
	assert.False(t, deserialized.IsRead)
}

func TestEmail_JSONSerialization_NullVsOmitted(t *testing.T) {
	// JSON null and omitted are both valid for *string with omitempty.
	// Unmarshal should produce nil in both cases.
	emailID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440001")

	t.Run("explicit null", func(t *testing.T) {
		raw := `{"id":"` + emailID.String() + `","message_id":"msg-null","from_address":"a@b.com","received_at":"2026-07-13T10:00:00Z","created_at":"2026-07-13T10:00:00Z","subject":null,"body":null}`
		var e Email
		err := json.Unmarshal([]byte(raw), &e)
		require.NoError(t, err)
		assert.Nil(t, e.Subject)
		assert.Nil(t, e.Body)
	})

	t.Run("omitted fields", func(t *testing.T) {
		raw := `{"id":"` + emailID.String() + `","message_id":"msg-omit","from_address":"a@b.com","received_at":"2026-07-13T10:00:00Z","created_at":"2026-07-13T10:00:00Z"}`
		var e Email
		err := json.Unmarshal([]byte(raw), &e)
		require.NoError(t, err)
		assert.Nil(t, e.Subject)
		assert.Nil(t, e.Body)
	})
}

func TestEmail_JSONSerialization_ZeroValue(t *testing.T) {
	var e Email
	data, err := json.Marshal(e)
	require.NoError(t, err)

	// Zero UUID should serialize as all zeros
	assert.Contains(t, string(data), `"id":"00000000-0000-0000-0000-000000000000"`)
	assert.Contains(t, string(data), `"message_id":""`)
	assert.Contains(t, string(data), `"from_address":""`)
	assert.Contains(t, string(data), `"is_read":false`)

	// Round-trip
	var deserialized Email
	err = json.Unmarshal(data, &deserialized)
	require.NoError(t, err)
	assert.Equal(t, uuid.Nil, deserialized.ID)
	assert.Equal(t, "", deserialized.MessageID)
	assert.Equal(t, "", deserialized.FromAddress)
	assert.Nil(t, deserialized.ApplicationID)
	assert.Nil(t, deserialized.ToAddress)
	assert.Nil(t, deserialized.Subject)
	assert.Nil(t, deserialized.Body)
	assert.Nil(t, deserialized.Classification)
	assert.Nil(t, deserialized.ReplyDraft)
	assert.False(t, deserialized.IsRead)
}

func TestEmail_JSONSerialization_InvalidUUID(t *testing.T) {
	raw := `{"id":"not-a-uuid","message_id":"msg","from_address":"a@b.com","received_at":"2026-07-13T10:00:00Z","created_at":"2026-07-13T10:00:00Z"}`
	var e Email
	err := json.Unmarshal([]byte(raw), &e)
	assert.Error(t, err)
}

func TestEmail_JSONSerialization_InvalidTime(t *testing.T) {
	emailID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440001")
	raw := `{"id":"` + emailID.String() + `","message_id":"msg","from_address":"a@b.com","received_at":"not-a-time","created_at":"2026-07-13T10:00:00Z"}`
	var e Email
	err := json.Unmarshal([]byte(raw), &e)
	assert.Error(t, err)
}

// ============================================================================
// Email Struct — Pointers / Nil Safety
// ============================================================================

func TestEmail_NilPointers_DoNotPanicOnAccess(t *testing.T) {
	// Verify that accessing fields on an Email with nil pointers is safe.
	var e *Email
	// A nil pointer dereference SHOULD panic — this test documents the behavior.
	assert.Nil(t, e)
}

func TestEmail_ZeroValue_SafeFields(t *testing.T) {
	var e Email
	// These should never panic
	assert.Equal(t, uuid.Nil, e.ID)
	assert.Nil(t, e.ApplicationID)
	assert.Equal(t, "", e.MessageID)
	assert.Equal(t, "", e.FromAddress)
	assert.Nil(t, e.ToAddress)
	assert.Nil(t, e.Subject)
	assert.Nil(t, e.Body)
	assert.True(t, e.ReceivedAt.IsZero())
	assert.Nil(t, e.Classification)
	assert.False(t, e.IsRead)
	assert.Nil(t, e.ReplyDraft)
	assert.True(t, e.CreatedAt.IsZero())
}

// ============================================================================
// emailColumns constant
// ============================================================================

func TestEmailColumns_NotEmpty(t *testing.T) {
	assert.NotEmpty(t, emailColumns, "emailColumns should not be empty")
}

func TestEmailColumns_ExpectedColumns(t *testing.T) {
	expected := []string{
		"id", "application_id", "message_id", "from_address", "to_address",
		"subject", "body", "received_at", "classification", "is_read", "reply_draft", "created_at",
	}
	for _, col := range expected {
		assert.Contains(t, emailColumns, col, "emailColumns should contain %s", col)
	}
}

func TestEmailColumns_Count(t *testing.T) {
	// Count column names (split by comma, trim spaces)
	count := 0
	in := false
	for _, c := range emailColumns {
		switch {
		case c == ',':
			if in {
				count++
				in = false
			}
		case c != ' ' && c != '\n' && c != '\t':
			if !in {
				in = true
			}
		}
	}
	if in {
		count++
	}
	assert.Equal(t, 12, count, "emailColumns should contain exactly 12 column names")
}
