package scoring

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScoreResponse_Fields(t *testing.T) {
	jobID := uuid.New()
	details := &ScoreDetails{
		SkillMatch:       90,
		ExperienceMatch:  85,
		LocationMatch:    70,
		SalaryMatch:      80,
		DescriptionMatch: 75,
	}

	resp := ScoreResponse{
		JobID:      jobID,
		Score:      87.5,
		Tier:       TierReview,
		Reasoning:  "Strong skill match but location not ideal",
		Source:     "hybrid",
		Model:      "gpt-4o",
		Confidence: 0.92,
		Strengths:  []string{"Strong technical background", "Relevant experience"},
		Gaps:       []string{"No cloud certification"},
		Details:    details,
	}

	assert.Equal(t, jobID, resp.JobID)
	assert.Equal(t, 87.5, resp.Score)
	assert.Equal(t, TierReview, resp.Tier)
	assert.Equal(t, "Strong skill match but location not ideal", resp.Reasoning)
	assert.Equal(t, "hybrid", resp.Source)
	assert.Equal(t, "gpt-4o", resp.Model)
	assert.Equal(t, 0.92, resp.Confidence)
	assert.Equal(t, []string{"Strong technical background", "Relevant experience"}, resp.Strengths)
	assert.Equal(t, []string{"No cloud certification"}, resp.Gaps)
	require.NotNil(t, resp.Details)
	assert.Equal(t, 90.0, resp.Details.SkillMatch)
}

func TestScoreResponse_EmptyOptionalFields(t *testing.T) {
	jobID := uuid.New()
	resp := ScoreResponse{
		JobID: jobID,
		Score: 85.0,
		Tier:  TierReview,
	}

	assert.Equal(t, jobID, resp.JobID)
	assert.Equal(t, 85.0, resp.Score)
	assert.Equal(t, TierReview, resp.Tier)
	assert.Empty(t, resp.Reasoning)
	assert.Empty(t, resp.Source)
	assert.Empty(t, resp.Model)
	assert.Equal(t, 0.0, resp.Confidence)
	assert.Nil(t, resp.Strengths)
	assert.Nil(t, resp.Gaps)
	assert.Nil(t, resp.Details)
}

func TestScoreResponse_JSONOmitEmpty(t *testing.T) {
	jobID := uuid.New()
	resp := ScoreResponse{
		JobID: jobID,
		Score: 95.0,
		Tier:  TierAuto,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	// Verify optional fields are omitted when empty
	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.Equal(t, jobID.String(), raw["job_id"])
	assert.Equal(t, 95.0, raw["score"])
	assert.Equal(t, "auto", raw["tier"])
	assert.NotContains(t, raw, "reasoning")
	assert.NotContains(t, raw, "source")
	assert.NotContains(t, raw, "model")
	assert.NotContains(t, raw, "confidence")
	assert.NotContains(t, raw, "strengths")
	assert.NotContains(t, raw, "gaps")
	assert.NotContains(t, raw, "details")
}

func TestScoreResponse_JSONRoundTrip(t *testing.T) {
	jobID := uuid.New()
	details := &ScoreDetails{
		SkillMatch:       100,
		ExperienceMatch:  80,
		LocationMatch:    60,
		SalaryMatch:      40,
		DescriptionMatch: 20,
	}

	resp := ScoreResponse{
		JobID:      jobID,
		Score:      82.5,
		Tier:       TierReview,
		Reasoning:  "Good match overall",
		Source:     "llm",
		Model:      "gpt-4o",
		Confidence: 0.88,
		Strengths:  []string{"Skills", "Experience"},
		Gaps:       []string{"Salary"},
		Details:    details,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded ScoreResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, resp.JobID, decoded.JobID)
	assert.Equal(t, resp.Score, decoded.Score)
	assert.Equal(t, resp.Tier, decoded.Tier)
	assert.Equal(t, resp.Reasoning, decoded.Reasoning)
	assert.Equal(t, resp.Source, decoded.Source)
	assert.Equal(t, resp.Model, decoded.Model)
	assert.Equal(t, resp.Confidence, decoded.Confidence)
	assert.Equal(t, resp.Strengths, decoded.Strengths)
	assert.Equal(t, resp.Gaps, decoded.Gaps)
	require.NotNil(t, decoded.Details)
	assert.Equal(t, resp.Details.SkillMatch, decoded.Details.SkillMatch)
}

func TestScoreResponse_AllTiers(t *testing.T) {
	jobID := uuid.New()

	tests := []struct {
		name  string
		tier  ApprovalTier
		score float64
	}{
		{"auto tier", TierAuto, 95},
		{"review tier", TierReview, 85},
		{"reject tier", TierReject, 40},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := ScoreResponse{
				JobID: jobID,
				Score: tt.score,
				Tier:  tt.tier,
			}
			assert.Equal(t, jobID, resp.JobID)
			assert.Equal(t, tt.score, resp.Score)
			assert.Equal(t, tt.tier, resp.Tier)
		})
	}
}

func TestScoreBreakdownResponse_Fields(t *testing.T) {
	jobID := uuid.New()
	details := ScoreDetails{
		SkillMatch:       88,
		ExperienceMatch:  76,
		LocationMatch:    100,
		SalaryMatch:      60,
		DescriptionMatch: 82,
	}
	weights := WeightsResponse{
		Skill:       0.35,
		Experience:  0.25,
		Location:    0.15,
		Salary:      0.15,
		Description: 0.10,
	}

	resp := ScoreBreakdownResponse{
		ScoreResponse: ScoreResponse{
			JobID:     jobID,
			Score:     80.0,
			Tier:      TierReview,
			Reasoning: "Detailed breakdown",
			Source:    "hybrid",
		},
		Details: details,
		Weights: weights,
	}

	// Embedded fields
	assert.Equal(t, jobID, resp.JobID)
	assert.Equal(t, 80.0, resp.Score)
	assert.Equal(t, TierReview, resp.Tier)
	assert.Equal(t, "Detailed breakdown", resp.Reasoning)
	assert.Equal(t, "hybrid", resp.Source)

	// Details (value, not pointer)
	assert.Equal(t, 88.0, resp.Details.SkillMatch)
	assert.Equal(t, 76.0, resp.Details.ExperienceMatch)
	assert.Equal(t, 100.0, resp.Details.LocationMatch)
	assert.Equal(t, 60.0, resp.Details.SalaryMatch)
	assert.Equal(t, 82.0, resp.Details.DescriptionMatch)

	// Weights
	assert.Equal(t, 0.35, resp.Weights.Skill)
	assert.Equal(t, 0.25, resp.Weights.Experience)
	assert.Equal(t, 0.15, resp.Weights.Location)
	assert.Equal(t, 0.15, resp.Weights.Salary)
	assert.Equal(t, 0.10, resp.Weights.Description)
}

func TestScoreBreakdownResponse_JSONRoundTrip(t *testing.T) {
	jobID := uuid.New()
	resp := ScoreBreakdownResponse{
		ScoreResponse: ScoreResponse{
			JobID: jobID,
			Score: 75.0,
			Tier:  TierReview,
		},
		Details: ScoreDetails{
			SkillMatch:       80,
			ExperienceMatch:  70,
			LocationMatch:    90,
			SalaryMatch:      50,
			DescriptionMatch: 65,
		},
		Weights: WeightsResponse{
			Skill:       0.4,
			Experience:  0.3,
			Location:    0.1,
			Salary:      0.1,
			Description: 0.1,
		},
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded ScoreBreakdownResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, resp.JobID, decoded.JobID)
	assert.Equal(t, resp.Score, decoded.Score)
	assert.Equal(t, resp.Tier, decoded.Tier)
	assert.Equal(t, resp.Details.SkillMatch, decoded.Details.SkillMatch)
	assert.Equal(t, resp.Weights.Skill, decoded.Weights.Skill)
}

func TestWeightsResponse_Fields(t *testing.T) {
	w := WeightsResponse{
		Skill:       0.4,
		Experience:  0.3,
		Location:    0.1,
		Salary:      0.15,
		Description: 0.05,
	}

	assert.Equal(t, 0.4, w.Skill)
	assert.Equal(t, 0.3, w.Experience)
	assert.Equal(t, 0.1, w.Location)
	assert.Equal(t, 0.15, w.Salary)
	assert.Equal(t, 0.05, w.Description)
}

func TestWeightsResponse_ZeroValues(t *testing.T) {
	w := WeightsResponse{}
	assert.Equal(t, 0.0, w.Skill)
	assert.Equal(t, 0.0, w.Experience)
	assert.Equal(t, 0.0, w.Location)
	assert.Equal(t, 0.0, w.Salary)
	assert.Equal(t, 0.0, w.Description)
}

func TestWeightsResponse_JSONTags(t *testing.T) {
	w := WeightsResponse{
		Skill:       0.5,
		Experience:  0.3,
		Location:    0.1,
		Salary:      0.05,
		Description: 0.05,
	}

	data, err := json.Marshal(w)
	require.NoError(t, err)

	var decoded WeightsResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, w, decoded)
}

func TestNewScoreResponse(t *testing.T) {
	jobID := uuid.New()
	details := &ScoreDetails{
		SkillMatch:       90,
		ExperienceMatch:  80,
		LocationMatch:    70,
		SalaryMatch:      60,
		DescriptionMatch: 50,
	}

	t.Run("auto tier", func(t *testing.T) {
		resp := NewScoreResponse(jobID, 96, details, 95, 80)
		assert.Equal(t, jobID, resp.JobID)
		assert.Equal(t, 96.0, resp.Score)
		assert.Equal(t, TierAuto, resp.Tier)
		require.NotNil(t, resp.Details)
		assert.Equal(t, 90.0, resp.Details.SkillMatch)
	})

	t.Run("review tier", func(t *testing.T) {
		resp := NewScoreResponse(jobID, 85, details, 95, 80)
		assert.Equal(t, jobID, resp.JobID)
		assert.Equal(t, 85.0, resp.Score)
		assert.Equal(t, TierReview, resp.Tier)
	})

	t.Run("reject tier", func(t *testing.T) {
		resp := NewScoreResponse(jobID, 40, details, 95, 80)
		assert.Equal(t, jobID, resp.JobID)
		assert.Equal(t, 40.0, resp.Score)
		assert.Equal(t, TierReject, resp.Tier)
	})
}

func TestNewScoreResponse_NormalizesScore(t *testing.T) {
	jobID := uuid.New()

	t.Run("negative score clamped to 0", func(t *testing.T) {
		resp := NewScoreResponse(jobID, -10, nil, 95, 80)
		assert.Equal(t, 0.0, resp.Score)
		assert.Equal(t, TierReject, resp.Tier)
	})

	t.Run("over 100 clamped to 100", func(t *testing.T) {
		resp := NewScoreResponse(jobID, 150, nil, 95, 80)
		assert.Equal(t, 100.0, resp.Score)
		assert.Equal(t, TierAuto, resp.Tier)
	})

	t.Run("exact boundary zero", func(t *testing.T) {
		resp := NewScoreResponse(jobID, 0, nil, 95, 80)
		assert.Equal(t, 0.0, resp.Score)
		assert.Equal(t, TierReject, resp.Tier)
	})
}

func TestNewScoreResponse_NilDetails(t *testing.T) {
	jobID := uuid.New()
	resp := NewScoreResponse(jobID, 85.5, nil, 95, 80)
	assert.Equal(t, jobID, resp.JobID)
	assert.Equal(t, 85.5, resp.Score)
	assert.Equal(t, TierReview, resp.Tier)
	assert.Nil(t, resp.Details)
	assert.Empty(t, resp.Reasoning)
	assert.Empty(t, resp.Source)
	assert.Empty(t, resp.Model)
}

func TestNewScoreResponse_EdgeThresholds(t *testing.T) {
	jobID := uuid.New()

	tests := []struct {
		name            string
		score           float64
		autoThreshold   int
		reviewThreshold int
		wantTier        ApprovalTier
	}{
		{"exactly auto threshold", 95, 95, 80, TierAuto},
		{"exactly review threshold", 80, 95, 80, TierReview},
		{"one below review threshold", 79, 95, 80, TierReject},
		{"thresholds are equal auto", 90, 90, 90, TierAuto},
		{"thresholds equal reject", 89, 90, 90, TierReject},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := NewScoreResponse(jobID, tt.score, nil, tt.autoThreshold, tt.reviewThreshold)
			assert.Equal(t, tt.wantTier, resp.Tier)
		})
	}
}
