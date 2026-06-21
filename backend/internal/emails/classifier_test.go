package emails

import (
	"testing"
)

// --- parseClassifyOutput ---

func TestParseClassifyOutput_ValidJSON(t *testing.T) {
	input := `{"category":"interview_invite","confidence":0.95,"reasoning":"Scheduling interview"}`
	result, err := parseClassifyOutput(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Category != "interview_invite" {
		t.Errorf("expected category=interview_invite, got %s", result.Category)
	}
	if result.Confidence != 0.95 {
		t.Errorf("expected confidence=0.95, got %f", result.Confidence)
	}
	if result.Reasoning != "Scheduling interview" {
		t.Errorf("expected reasoning='Scheduling interview', got %s", result.Reasoning)
	}
}

func TestParseClassifyOutput_WrappedInCodeFence(t *testing.T) {
	input := "Here's the classification:\n```json\n{\"category\":\"rejection\",\"confidence\":0.88,\"reasoning\":\"Declined\"}\n```\nHope that helps!"
	result, err := parseClassifyOutput(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Category != "rejection" {
		t.Errorf("expected category=rejection, got %s", result.Category)
	}
}

func TestParseClassifyOutput_CodeFenceWithoutLangTag(t *testing.T) {
	input := "```\n{\"category\":\"offer\",\"confidence\":0.99,\"reasoning\":\"Job offer\"}\n```"
	result, err := parseClassifyOutput(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Category != "offer" {
		t.Errorf("expected category=offer, got %s", result.Category)
	}
}

func TestParseClassifyOutput_LeadingTrailingWhitespace(t *testing.T) {
	input := "  \n  {\"category\":\"spam\",\"confidence\":0.7,\"reasoning\":\"Marketing\"}  \n  "
	result, err := parseClassifyOutput(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Category != "spam" {
		t.Errorf("expected category=spam, got %s", result.Category)
	}
}

func TestParseClassifyOutput_InvalidJSON(t *testing.T) {
	input := "this is not json at all"
	_, err := parseClassifyOutput(input)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestParseClassifyOutput_EmptyString(t *testing.T) {
	_, err := parseClassifyOutput("")
	if err == nil {
		t.Error("expected error for empty string, got nil")
	}
}

func TestParseClassifyOutput_CodeFenceWithInvalidJSON(t *testing.T) {
	input := "```json\nnot json\n```"
	_, err := parseClassifyOutput(input)
	if err == nil {
		t.Error("expected error for invalid JSON inside code fence, got nil")
	}
}

// --- truncate ---

func TestTruncate_ShortString(t *testing.T) {
	result := truncate("hello", 10)
	if result != "hello" {
		t.Errorf("expected 'hello', got %q", result)
	}
}

func TestTruncate_ExactlyMaxLen(t *testing.T) {
	result := truncate("12345", 5)
	if result != "12345" {
		t.Errorf("expected '12345', got %q", result)
	}
}

func TestTruncate_LongString(t *testing.T) {
	result := truncate("hello world", 5)
	if result != "hello... (truncated)" {
		t.Errorf("expected 'hello... (truncated)', got %q", result)
	}
}

func TestTruncate_EmptyString(t *testing.T) {
	result := truncate("", 10)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

// --- IsValidClassification ---

func TestIsValidClassification_Valid(t *testing.T) {
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
		if !IsValidClassification(c) {
			t.Errorf("expected %q to be valid", c)
		}
	}
}

func TestIsValidClassification_Invalid(t *testing.T) {
	invalid := []string{"", "unknown", "INTERVIEW_INVITE", "Rejection"}
	for _, c := range invalid {
		if IsValidClassification(c) {
			t.Errorf("expected %q to be invalid", c)
		}
	}
}
