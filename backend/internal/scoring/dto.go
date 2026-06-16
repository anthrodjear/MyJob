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
	JobID      uuid.UUID      `json:"job_id"`
	Score      float64        `json:"score"`
	Tier       ApprovalTier   `json:"tier"`
	Reasoning  string         `json:"reasoning,omitempty"`
	Source     string         `json:"source,omitempty"`      // "heuristic" | "llm" | "hybrid"
	Model      string         `json:"model,omitempty"`       // model name used (e.g., "gpt-4o", "qwen2.5")
	Confidence float64        `json:"confidence,omitempty"`  // 0-1 confidence in score
	Strengths  []string       `json:"strengths,omitempty"`   // extracted from reasoning
	Gaps       []string       `json:"gaps,omitempty"`        // extracted from reasoning
	Details    *ScoreDetails  `json:"details,omitempty"`
}

// ScoreBreakdownResponse provides detailed scoring breakdown with weights used.
type ScoreBreakdownResponse struct {
	ScoreResponse
	Details ScoreDetails    `json:"details"`
	Weights WeightsResponse `json:"weights"`
}

// WeightsResponse shows the weights used for scoring.
type WeightsResponse struct {
	Skill       float64 `json:"skill"`
	Experience  float64 `json:"experience"`
	Location    float64 `json:"location"`
	Salary      float64 `json:"salary"`
	Description float64 `json:"description"`
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
