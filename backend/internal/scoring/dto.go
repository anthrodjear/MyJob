// Package scoring provides job-candidate matching and scoring functionality.
// It supports three scoring modes: heuristic (keyword-based), LLM (semantic), and hybrid (pre-filter + LLM).
// The service computes factor scores (skills, experience, location, salary, description) and combines them
// into a final 0-100 score with approval tier (auto/review/reject).
package scoring

import (
	"github.com/google/uuid"
)

// ScoreResponse is the API response for a scored job.
type ScoreResponse struct {
	JobID      uuid.UUID     `json:"job_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Score      float64       `json:"score" example:"92.5"`
	Tier       ApprovalTier  `json:"tier" example:"AUTO" enums:"AUTO,REVIEW,REJECT"`
	Reasoning  string        `json:"reasoning,omitempty" example:"Strong match on Go and Kubernetes experience. 8 years backend experience aligns well with senior role requirements."`
	Source     string        `json:"source,omitempty" example:"hybrid" enums:"heuristic,llm,hybrid"`
	Model      string        `json:"model,omitempty" example:"qwen2.5:latest"`
	Confidence float64       `json:"confidence,omitempty" example:"0.9" minimum:"0" maximum:"1"`
	Strengths  []string      `json:"strengths,omitempty" example:"[\"Strong match\",\"Experienced candidate\",\"Proficient in required skills\"]"`
	Gaps       []string      `json:"gaps,omitempty" example:"[\"Missing cloud certification\"]"`
	Details    *ScoreDetails `json:"details,omitempty"`
}

// ScoreBreakdownResponse provides detailed scoring breakdown with weights used.
type ScoreBreakdownResponse struct {
	ScoreResponse
	Details ScoreDetails    `json:"details"`
	Weights WeightsResponse `json:"weights"`
}

// WeightsResponse shows the weights used for scoring.
type WeightsResponse struct {
	Skill       float64 `json:"skill" example:"0.35"`
	Experience  float64 `json:"experience" example:"0.25"`
	Location    float64 `json:"location" example:"0.10"`
	Salary      float64 `json:"salary" example:"0.15"`
	Description float64 `json:"description" example:"0.15"`
}

// NewScoreResponse creates a ScoreResponse with normalized score and tier.
func NewScoreResponse(jobID uuid.UUID, score float64, details *ScoreDetails, autoThreshold, reviewThreshold int) ScoreResponse {
	return ScoreResponse{
		JobID:   jobID,
		Score:   NormalizeScore(score),
		Tier:    Tier(score, autoThreshold, reviewThreshold),
		Details: details,
	}
}
