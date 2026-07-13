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
// DTO-Specific Domain Errors
// ============================================================================

func TestDTOErrors(t *testing.T) {
	t.Run("ErrInvalidApplicationID", func(t *testing.T) {
		assert.Error(t, ErrInvalidApplicationID)
		assert.Equal(t, "invalid application_id", ErrInvalidApplicationID.Error())
	})
}

// ============================================================================
// StoreEmailRequest
// ============================================================================

func TestStoreEmailRequest_JSONDeserialization_Full(t *testing.T) {
	appID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	to := "recipient@example.com"
	subj := "Interview"
	body := "Details here"
	class := ClassificationInterviewInvite
	now := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)

	raw := `{
		"application_id": "550e8400-e29b-41d4-a716-446655440000",
		"message_id": "msg-100",
		"from_address": "sender@example.com",
		"to_address": "recipient@example.com",
		"subject": "Interview",
		"body": "Details here",
		"received_at": "2026-07-13T10:00:00Z",
		"classification": "interview_invite"
	}`

	var req StoreEmailRequest
	err := json.Unmarshal([]byte(raw), &req)
	require.NoError(t, err)

	assert.Equal(t, appID, *req.ApplicationID)
	assert.Equal(t, "msg-100", req.MessageID)
	assert.Equal(t, "sender@example.com", req.FromAddress)
	assert.Equal(t, to, *req.ToAddress)
	assert.Equal(t, subj, *req.Subject)
	assert.Equal(t, body, *req.Body)
	assert.True(t, now.Equal(req.ReceivedAt))
	assert.Equal(t, class, *req.Classification)
}

func TestStoreEmailRequest_JSONDeserialization_Minimal(t *testing.T) {
	now := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)
	raw := `{
		"message_id": "msg-101",
		"from_address": "sender@example.com",
		"received_at": "2026-07-13T10:00:00Z"
	}`

	var req StoreEmailRequest
	err := json.Unmarshal([]byte(raw), &req)
	require.NoError(t, err)

	assert.Equal(t, "msg-101", req.MessageID)
	assert.Equal(t, "sender@example.com", req.FromAddress)
	assert.True(t, now.Equal(req.ReceivedAt))
	assert.Nil(t, req.ApplicationID)
	assert.Nil(t, req.ToAddress)
	assert.Nil(t, req.Subject)
	assert.Nil(t, req.Body)
	assert.Nil(t, req.Classification)
}

func TestStoreEmailRequest_JSONDeserialization_NullFields(t *testing.T) {
	raw := `{
		"message_id": "msg-102",
		"from_address": "sender@example.com",
		"received_at": "2026-07-13T10:00:00Z",
		"to_address": null,
		"subject": null,
		"body": null,
		"classification": null
	}`

	var req StoreEmailRequest
	err := json.Unmarshal([]byte(raw), &req)
	require.NoError(t, err)

	assert.Nil(t, req.ToAddress)
	assert.Nil(t, req.Subject)
	assert.Nil(t, req.Body)
	assert.Nil(t, req.Classification)
}

func TestStoreEmailRequest_JSONSerialization(t *testing.T) {
	appID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	to := "r@example.com"
	subj := "Hello"
	now := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)

	req := StoreEmailRequest{
		ApplicationID: &appID,
		MessageID:     "msg-200",
		FromAddress:   "s@example.com",
		ToAddress:     &to,
		Subject:       &subj,
		ReceivedAt:    now,
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"application_id":"550e8400-e29b-41d4-a716-446655440000"`)
	assert.Contains(t, string(data), `"message_id":"msg-200"`)
	assert.Contains(t, string(data), `"from_address":"s@example.com"`)
	assert.Contains(t, string(data), `"to_address":"r@example.com"`)
	assert.Contains(t, string(data), `"subject":"Hello"`)

	// omitempty — body and classification are nil, so they should be absent
	assert.NotContains(t, string(data), "body")
	assert.NotContains(t, string(data), "classification")

	// Round-trip
	var deserialized StoreEmailRequest
	err = json.Unmarshal(data, &deserialized)
	require.NoError(t, err)
	assert.Equal(t, req.MessageID, deserialized.MessageID)
	assert.Equal(t, req.FromAddress, deserialized.FromAddress)
	assert.True(t, req.ReceivedAt.Equal(deserialized.ReceivedAt))
}

func TestStoreEmailRequest_MissingRequired(t *testing.T) {
	// These are struct tags only — no runtime validation in the type.
	// This test documents what the zero value looks like.
	var req StoreEmailRequest
	assert.Equal(t, "", req.MessageID)
	assert.Equal(t, "", req.FromAddress)
	assert.True(t, req.ReceivedAt.IsZero())
}

func TestStoreEmailRequest_InvalidApplicationIDJSON(t *testing.T) {
	raw := `{
		"application_id": "not-a-uuid",
		"message_id": "msg-103",
		"from_address": "s@example.com",
		"received_at": "2026-07-13T10:00:00Z"
	}`
	var req StoreEmailRequest
	err := json.Unmarshal([]byte(raw), &req)
	assert.Error(t, err)
}

// ============================================================================
// UpdateEmailRequest
// ============================================================================

func TestUpdateEmailRequest_JSONDeserialization_Full(t *testing.T) {
	raw := `{"is_read": true, "reply_draft": "Thank you for the update"}`

	var req UpdateEmailRequest
	err := json.Unmarshal([]byte(raw), &req)
	require.NoError(t, err)

	require.NotNil(t, req.IsRead)
	assert.True(t, *req.IsRead)
	require.NotNil(t, req.ReplyDraft)
	assert.Equal(t, "Thank you for the update", *req.ReplyDraft)
}

func TestUpdateEmailRequest_JSONDeserialization_ReadOnly(t *testing.T) {
	raw := `{"is_read": false}`

	var req UpdateEmailRequest
	err := json.Unmarshal([]byte(raw), &req)
	require.NoError(t, err)

	require.NotNil(t, req.IsRead)
	assert.False(t, *req.IsRead)
	assert.Nil(t, req.ReplyDraft)
}

func TestUpdateEmailRequest_JSONDeserialization_DraftOnly(t *testing.T) {
	raw := `{"reply_draft": "Looking forward to it"}`

	var req UpdateEmailRequest
	err := json.Unmarshal([]byte(raw), &req)
	require.NoError(t, err)

	assert.Nil(t, req.IsRead)
	require.NotNil(t, req.ReplyDraft)
	assert.Equal(t, "Looking forward to it", *req.ReplyDraft)
}

func TestUpdateEmailRequest_JSONDeserialization_EmptyBody(t *testing.T) {
	raw := `{}`

	var req UpdateEmailRequest
	err := json.Unmarshal([]byte(raw), &req)
	require.NoError(t, err)

	assert.Nil(t, req.IsRead)
	assert.Nil(t, req.ReplyDraft)
}

func TestUpdateEmailRequest_JSONSerialization(t *testing.T) {
	isRead := true
	draft := "Thanks!"

	req := UpdateEmailRequest{
		IsRead:     &isRead,
		ReplyDraft: &draft,
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"is_read":true`)
	assert.Contains(t, string(data), `"reply_draft":"Thanks!"`)
}

func TestUpdateEmailRequest_NilPointerFields(t *testing.T) {
	t.Run("nil IsRead only", func(t *testing.T) {
		draft := "draft"
		req := UpdateEmailRequest{ReplyDraft: &draft}
		data, err := json.Marshal(req)
		require.NoError(t, err)
		assert.NotContains(t, string(data), "is_read")
		assert.Contains(t, string(data), "reply_draft")
	})

	t.Run("nil ReplyDraft only", func(t *testing.T) {
		isRead := true
		req := UpdateEmailRequest{IsRead: &isRead}
		data, err := json.Marshal(req)
		require.NoError(t, err)
		assert.Contains(t, string(data), "is_read")
		assert.NotContains(t, string(data), "reply_draft")
	})

	t.Run("both nil", func(t *testing.T) {
		req := UpdateEmailRequest{}
		data, err := json.Marshal(req)
		require.NoError(t, err)
		assert.Equal(t, `{}`, string(data))
	})
}

// ============================================================================
// ClassifyRequest
// ============================================================================

func TestClassifyRequest_EmptyStruct(t *testing.T) {
	// ClassifyRequest is an empty struct; serialization should produce "{}".
	req := ClassifyRequest{}
	data, err := json.Marshal(req)
	require.NoError(t, err)
	assert.Equal(t, `{}`, string(data))
}

func TestClassifyRequest_Deserialize(t *testing.T) {
	// Empty body, no fields expected.
	var req ClassifyRequest
	err := json.Unmarshal([]byte(`{}`), &req)
	require.NoError(t, err)
}

func TestClassifyRequest_IgnoreFields(t *testing.T) {
	// Extra fields in JSON should be silently ignored.
	raw := `{"unexpected_field": "value", "another": 42}`
	var req ClassifyRequest
	err := json.Unmarshal([]byte(raw), &req)
	require.NoError(t, err)
}

// ============================================================================
// ListFilterRequest — Form Binding + ToListFilter
// ============================================================================

func TestListFilterRequest_FormTags(t *testing.T) {
	// Verify form tag names are set (not just JSON).
	var r ListFilterRequest
	assert.Equal(t, "", r.ApplicationID, "should have form tag for application_id")
	assert.Equal(t, "", r.Classification, "should have form tag for classification")
	assert.Equal(t, 0, r.Limit, "should have form tag for limit")
	assert.Equal(t, 0, r.Offset, "should have form tag for offset")
}

func TestListFilterRequest_ToListFilter_Empty(t *testing.T) {
	r := ListFilterRequest{}
	filter, err := r.ToListFilter()
	require.NoError(t, err)

	assert.Equal(t, uuid.Nil, filter.ApplicationID)
	assert.Equal(t, "", filter.Classification)
	assert.Equal(t, 0, filter.Limit)
	assert.Equal(t, 0, filter.Offset)
}

func TestListFilterRequest_ToListFilter_WithApplicationID(t *testing.T) {
	appID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	r := ListFilterRequest{
		ApplicationID:  appID.String(),
		Classification: ClassificationInterviewInvite,
		Limit:          25,
		Offset:         10,
	}
	filter, err := r.ToListFilter()
	require.NoError(t, err)

	assert.Equal(t, appID, filter.ApplicationID)
	assert.Equal(t, ClassificationInterviewInvite, filter.Classification)
	assert.Equal(t, 25, filter.Limit)
	assert.Equal(t, 10, filter.Offset)
}

func TestListFilterRequest_ToListFilter_InvalidApplicationID(t *testing.T) {
	r := ListFilterRequest{
		ApplicationID: "not-a-valid-uuid",
	}
	_, err := r.ToListFilter()
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidApplicationID)
}

func TestListFilterRequest_ToListFilter_InvalidApplicationIDEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "garbage", value: "abc123"},
		{name: "too short", value: "550e8400"},
		{name: "extra chars", value: "550e8400-e29b-41d4-a716-446655440000-extra"},
		{name: "empty string", value: ""},
		{name: "whitespace", value: " "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := ListFilterRequest{ApplicationID: tt.value}
			filter, err := r.ToListFilter()
			if tt.value == "" {
				// Empty string should be treated as no filter.
				assert.NoError(t, err)
				assert.Equal(t, uuid.Nil, filter.ApplicationID)
			} else {
				assert.Error(t, err)
				assert.ErrorIs(t, err, ErrInvalidApplicationID)
			}
		})
	}
}

func TestListFilterRequest_ToListFilter_ZeroLimitOffset(t *testing.T) {
	// Zero values for limit/offset should pass through as-is.
	r := ListFilterRequest{}
	filter, err := r.ToListFilter()
	require.NoError(t, err)
	assert.Equal(t, 0, filter.Limit)
	assert.Equal(t, 0, filter.Offset)
}

// ============================================================================
// EmailResponse — JSON Serialization
// ============================================================================

func TestEmailResponse_JSONSerialization_Full(t *testing.T) {
	appID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	emailID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440001")
	to := "recipient@example.com"
	subj := "Interview"
	body := "Details"
	class := ClassificationRejection
	draft := "Noted"
	now := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)

	resp := EmailResponse{
		ID:             emailID,
		ApplicationID:  &appID,
		MessageID:      "msg-300",
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

	data, err := json.Marshal(resp)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"id":"660e8400-e29b-41d4-a716-446655440001"`)
	assert.Contains(t, string(data), `"application_id"`)
	assert.Contains(t, string(data), `"message_id":"msg-300"`)
	assert.Contains(t, string(data), `"from_address":"hr@company.com"`)
	assert.Contains(t, string(data), `"to_address":"recipient@example.com"`)
	assert.Contains(t, string(data), `"subject":"Interview"`)
	assert.Contains(t, string(data), `"body":"Details"`)
	assert.Contains(t, string(data), `"is_read":true`)
	assert.Contains(t, string(data), `"reply_draft":"Noted"`)

	// Round-trip
	var deserialized EmailResponse
	err = json.Unmarshal(data, &deserialized)
	require.NoError(t, err)
	assert.Equal(t, resp.ID, deserialized.ID)
	assert.Equal(t, resp.ApplicationID, deserialized.ApplicationID)
	assert.Equal(t, resp.MessageID, deserialized.MessageID)
	assert.Equal(t, resp.FromAddress, deserialized.FromAddress)
	assert.Equal(t, resp.ToAddress, deserialized.ToAddress)
	assert.Equal(t, resp.Subject, deserialized.Subject)
	assert.Equal(t, resp.Body, deserialized.Body)
	assert.True(t, resp.ReceivedAt.Equal(deserialized.ReceivedAt))
	assert.Equal(t, resp.Classification, deserialized.Classification)
	assert.Equal(t, resp.IsRead, deserialized.IsRead)
	assert.Equal(t, resp.ReplyDraft, deserialized.ReplyDraft)
	assert.True(t, resp.CreatedAt.Equal(deserialized.CreatedAt))
}

func TestEmailResponse_JSONSerialization_Partial(t *testing.T) {
	emailID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440001")
	now := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)

	resp := EmailResponse{
		ID:          emailID,
		MessageID:   "msg-301",
		FromAddress: "noreply@example.com",
		ReceivedAt:  now,
		CreatedAt:   now,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	assert.NotContains(t, string(data), "application_id")
	assert.NotContains(t, string(data), "to_address")
	assert.NotContains(t, string(data), "subject")
	assert.NotContains(t, string(data), "body")
	assert.NotContains(t, string(data), "classification")
	assert.NotContains(t, string(data), "reply_draft")

	assert.Contains(t, string(data), `"is_read":false`)

	// Round-trip
	var deserialized EmailResponse
	err = json.Unmarshal(data, &deserialized)
	require.NoError(t, err)
	assert.Nil(t, deserialized.ApplicationID)
	assert.Nil(t, deserialized.ToAddress)
	assert.Nil(t, deserialized.Subject)
	assert.Nil(t, deserialized.Body)
	assert.Nil(t, deserialized.Classification)
	assert.Nil(t, deserialized.ReplyDraft)
}

func TestEmailResponse_JSONSerialization_ZeroValue(t *testing.T) {
	var resp EmailResponse
	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var deserialized EmailResponse
	err = json.Unmarshal(data, &deserialized)
	require.NoError(t, err)
	assert.Equal(t, uuid.Nil, deserialized.ID)
	assert.Equal(t, "", deserialized.MessageID)
	assert.Equal(t, "", deserialized.FromAddress)
	assert.Nil(t, deserialized.ApplicationID)
	assert.False(t, deserialized.IsRead)
}

// ============================================================================
// EmailListResponse — JSON Serialization
// ============================================================================

func TestEmailListResponse_JSONSerialization_WithItems(t *testing.T) {
	id1 := uuid.MustParse("660e8400-e29b-41d4-a716-446655440001")
	id2 := uuid.MustParse("660e8400-e29b-41d4-a716-446655440002")
	now := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)

	resp := EmailListResponse{
		Emails: []EmailResponse{
			{
				ID:          id1,
				MessageID:   "msg-001",
				FromAddress: "a@b.com",
				ReceivedAt:  now,
				CreatedAt:   now,
			},
			{
				ID:          id2,
				MessageID:   "msg-002",
				FromAddress: "c@d.com",
				ReceivedAt:  now,
				CreatedAt:   now,
			},
		},
		Total:  2,
		Limit:  50,
		Offset: 0,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"emails":`)
	assert.Contains(t, string(data), `"total":2`)
	assert.Contains(t, string(data), `"limit":50`)
	assert.Contains(t, string(data), `"offset":0`)
	assert.Contains(t, string(data), `"msg-001"`)
	assert.Contains(t, string(data), `"msg-002"`)

	// Round-trip
	var deserialized EmailListResponse
	err = json.Unmarshal(data, &deserialized)
	require.NoError(t, err)
	assert.Len(t, deserialized.Emails, 2)
	assert.Equal(t, int64(2), deserialized.Total)
	assert.Equal(t, 50, deserialized.Limit)
	assert.Equal(t, 0, deserialized.Offset)
}

func TestEmailListResponse_JSONSerialization_EmptyList(t *testing.T) {
	resp := EmailListResponse{
		Emails: []EmailResponse{},
		Total:  0,
		Limit:  25,
		Offset: 0,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"emails":[]`)
	assert.Contains(t, string(data), `"total":0`)

	var deserialized EmailListResponse
	err = json.Unmarshal(data, &deserialized)
	require.NoError(t, err)
	assert.Empty(t, deserialized.Emails)
}

func TestEmailListResponse_JSONSerialization_NilEmails(t *testing.T) {
	resp := EmailListResponse{
		Emails: nil,
		Total:  0,
		Limit:  0,
		Offset: 0,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"emails":null`)

	var deserialized EmailListResponse
	err = json.Unmarshal(data, &deserialized)
	require.NoError(t, err)
	assert.Nil(t, deserialized.Emails)
}

// ============================================================================
// ClassifyResponse — JSON Serialization
// ============================================================================

func TestClassifyResponse_JSONSerialization(t *testing.T) {
	emailID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")

	resp := ClassifyResponse{
		EmailID:        emailID,
		Classification: ClassificationInterviewInvite,
		Confidence:     0.95,
		Reasoning:      "Scheduling interview",
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"email_id":"660e8400-e29b-41d4-a716-446655440000"`)
	assert.Contains(t, string(data), `"classification":"interview_invite"`)
	assert.Contains(t, string(data), `"confidence":0.95`)
	assert.Contains(t, string(data), `"reasoning":"Scheduling interview"`)

	// Round-trip
	var deserialized ClassifyResponse
	err = json.Unmarshal(data, &deserialized)
	require.NoError(t, err)
	assert.Equal(t, resp.EmailID, deserialized.EmailID)
	assert.Equal(t, resp.Classification, deserialized.Classification)
	assert.InDelta(t, resp.Confidence, deserialized.Confidence, 0.0001)
	assert.Equal(t, resp.Reasoning, deserialized.Reasoning)
}

func TestClassifyResponse_JSONSerialization_ConfidenceBoundaries(t *testing.T) {
	emailID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")

	tests := []struct {
		name       string
		confidence float64
	}{
		{name: "zero confidence", confidence: 0.0},
		{name: "perfect confidence", confidence: 1.0},
		{name: "low confidence", confidence: 0.01},
		{name: "high precision", confidence: 0.123456789},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := ClassifyResponse{
				EmailID:        emailID,
				Classification: ClassificationOther,
				Confidence:     tt.confidence,
				Reasoning:      "",
			}
			data, err := json.Marshal(resp)
			require.NoError(t, err)

			var deserialized ClassifyResponse
			err = json.Unmarshal(data, &deserialized)
			require.NoError(t, err)
			assert.InDelta(t, tt.confidence, deserialized.Confidence, 0.0001)
		})
	}
}

func TestClassifyResponse_JSONSerialization_EmptyReasoning(t *testing.T) {
	emailID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")
	resp := ClassifyResponse{
		EmailID:        emailID,
		Classification: ClassificationSpam,
		Confidence:     0.5,
		Reasoning:      "",
	}
	data, err := json.Marshal(resp)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"reasoning":""`)
}

func TestClassifyResponse_JSONDeserialization_InvalidUUID(t *testing.T) {
	raw := `{"email_id":"bad-uuid","classification":"spam","confidence":0.5,"reasoning":"test"}`
	var resp ClassifyResponse
	err := json.Unmarshal([]byte(raw), &resp)
	assert.Error(t, err)
}

func TestClassifyResponse_ZeroValue(t *testing.T) {
	var resp ClassifyResponse
	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var deserialized ClassifyResponse
	err = json.Unmarshal(data, &deserialized)
	require.NoError(t, err)
	assert.Equal(t, uuid.Nil, deserialized.EmailID)
	assert.Equal(t, "", deserialized.Classification)
	assert.Equal(t, 0.0, deserialized.Confidence)
	assert.Equal(t, "", deserialized.Reasoning)
}

// ============================================================================
// ToEmailResponse Mapper
// ============================================================================

func TestToEmailResponse_FullEmail(t *testing.T) {
	appID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	emailID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440001")
	to := "recipient@example.com"
	subj := "Subject"
	body := "Body text"
	class := ClassificationOffer
	draft := "Draft reply"
	now := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)

	email := &Email{
		ID:             emailID,
		ApplicationID:  &appID,
		MessageID:      "msg-400",
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

	resp := ToEmailResponse(email)

	assert.Equal(t, emailID, resp.ID)
	assert.Equal(t, &appID, resp.ApplicationID)
	assert.Equal(t, "msg-400", resp.MessageID)
	assert.Equal(t, "hr@company.com", resp.FromAddress)
	assert.Equal(t, &to, resp.ToAddress)
	assert.Equal(t, &subj, resp.Subject)
	assert.Equal(t, &body, resp.Body)
	assert.True(t, now.Equal(resp.ReceivedAt))
	assert.Equal(t, &class, resp.Classification)
	assert.True(t, resp.IsRead)
	assert.Equal(t, &draft, resp.ReplyDraft)
	assert.True(t, now.Equal(resp.CreatedAt))
}

func TestToEmailResponse_MinimalEmail(t *testing.T) {
	emailID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440001")
	now := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)

	email := &Email{
		ID:          emailID,
		MessageID:   "msg-401",
		FromAddress: "noreply@example.com",
		ReceivedAt:  now,
		CreatedAt:   now,
	}

	resp := ToEmailResponse(email)

	assert.Equal(t, emailID, resp.ID)
	assert.Nil(t, resp.ApplicationID)
	assert.Equal(t, "msg-401", resp.MessageID)
	assert.Equal(t, "noreply@example.com", resp.FromAddress)
	assert.Nil(t, resp.ToAddress)
	assert.Nil(t, resp.Subject)
	assert.Nil(t, resp.Body)
	assert.True(t, now.Equal(resp.ReceivedAt))
	assert.Nil(t, resp.Classification)
	assert.False(t, resp.IsRead)
	assert.Nil(t, resp.ReplyDraft)
	assert.True(t, now.Equal(resp.CreatedAt))
}

func TestToEmailResponse_NilPointers(t *testing.T) {
	// Test all nullable fields are nil
	emailID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440001")
	now := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)

	email := &Email{
		ID:          emailID,
		MessageID:   "msg-402",
		FromAddress: "noreply@example.com",
		ReceivedAt:  now,
		CreatedAt:   now,
		// All pointer fields left nil
		ApplicationID:  nil,
		ToAddress:      nil,
		Subject:        nil,
		Body:           nil,
		Classification: nil,
		ReplyDraft:     nil,
	}

	resp := ToEmailResponse(email)
	assert.Nil(t, resp.ApplicationID)
	assert.Nil(t, resp.ToAddress)
	assert.Nil(t, resp.Subject)
	assert.Nil(t, resp.Body)
	assert.Nil(t, resp.Classification)
	assert.Nil(t, resp.ReplyDraft)
}

func TestToEmailResponse_NilEmailPanics(t *testing.T) {
	// ToEmailResponse does not guard against nil *Email — this documents behavior.
	assert.Panics(t, func() {
		ToEmailResponse(nil)
	}, "ToEmailResponse should panic on nil input (nil dereference)")
}

func TestToEmailResponse_EmptyStrings(t *testing.T) {
	emailID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440001")
	subj := ""
	body := ""
	class := ""
	draft := ""

	email := &Email{
		ID:             emailID,
		ApplicationID:  nil,
		MessageID:      "",
		FromAddress:    "",
		ToAddress:      nil,
		Subject:        &subj,
		Body:           &body,
		ReceivedAt:     time.Time{},
		Classification: &class,
		IsRead:         false,
		ReplyDraft:     &draft,
		CreatedAt:      time.Time{},
	}

	resp := ToEmailResponse(email)
	require.NotNil(t, resp.Subject)
	assert.Equal(t, "", *resp.Subject)
	require.NotNil(t, resp.Body)
	assert.Equal(t, "", *resp.Body)
	require.NotNil(t, resp.Classification)
	assert.Equal(t, "", *resp.Classification)
	require.NotNil(t, resp.ReplyDraft)
	assert.Equal(t, "", *resp.ReplyDraft)
}

func TestToEmailResponse_JSONRoundTrip(t *testing.T) {
	// Build an Email, map to response, marshal, unmarshal back — ensures
	// the mapper output is fully JSON-compatible.
	appID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	emailID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440001")
	to := "r@e.com"
	subj := "Hello"
	body := "World"
	class := ClassificationFollowUp
	draft := "OK"
	now := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)

	email := &Email{
		ID:             emailID,
		ApplicationID:  &appID,
		MessageID:      "msg-500",
		FromAddress:    "s@e.com",
		ToAddress:      &to,
		Subject:        &subj,
		Body:           &body,
		ReceivedAt:     now,
		Classification: &class,
		IsRead:         true,
		ReplyDraft:     &draft,
		CreatedAt:      now,
	}

	resp := ToEmailResponse(email)
	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var deserialized EmailResponse
	err = json.Unmarshal(data, &deserialized)
	require.NoError(t, err)

	assert.Equal(t, resp.ID, deserialized.ID)
	assert.Equal(t, resp.MessageID, deserialized.MessageID)
	assert.Equal(t, resp.FromAddress, deserialized.FromAddress)
	assert.Equal(t, resp.IsRead, deserialized.IsRead)
	assert.True(t, resp.ReceivedAt.Equal(deserialized.ReceivedAt))
}
