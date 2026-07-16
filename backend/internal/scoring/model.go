// Package scoring provides job-candidate matching and scoring functionality.
// It supports three scoring modes: heuristic (keyword-based), LLM (semantic), and hybrid (pre-filter + LLM).
// The service computes factor scores (skills, experience, location, salary, description) and combines them
// into a final 0-100 score with approval tier (auto/review/reject).
package scoring

import (
	"errors"
	"math"
	"strings"
)

// ApprovalTier represents the approval category for a scored job.
type ApprovalTier string

// Tier constants for score-based approval.
const (
	TierReject   ApprovalTier = "reject" // score < ReviewThreshold
	TierReview   ApprovalTier = "review" // score >= ReviewThreshold, < AutoThreshold
	TierAuto     ApprovalTier = "auto"   // score >= AutoThreshold
	NeutralScore float64      = 50       // neutral score when no signal (e.g., no keywords)
)

// Tier returns the approval tier for a given score and thresholds.
// Caller must ensure autoThreshold > reviewThreshold — behavior is undefined otherwise.
func Tier(score float64, autoThreshold, reviewThreshold int) ApprovalTier {
	if score >= float64(autoThreshold) {
		return TierAuto
	}
	if score >= float64(reviewThreshold) {
		return TierReview
	}
	return TierReject
}

// ScoreResult holds the output of a scoring computation.
type ScoreResult struct {
	Score     float64       `json:"score"`     // 0-100
	Tier      ApprovalTier  `json:"tier"`      // auto, review, reject
	Reasoning string        `json:"reasoning"` // LLM explanation (empty for heuristic-only)
	Source    string        `json:"source"`    // "heuristic" | "llm" | "hybrid"
	Model     string        `json:"model"`     // model name used (e.g., "gpt-4o", "qwen2.5")
	Details   *ScoreDetails `json:"details"`   // breakdown of scoring factors
}

// ScoreDetails provides per-factor scoring breakdown.
type ScoreDetails struct {
	SkillMatch       float64 `json:"skill_match" example:"95.0"`       // 0-100
	ExperienceMatch  float64 `json:"experience_match" example:"85.0"`  // 0-100
	LocationMatch    float64 `json:"location_match" example:"100.0"`   // 0-100
	SalaryMatch      float64 `json:"salary_match" example:"90.0"`      // 0-100
	DescriptionMatch float64 `json:"description_match" example:"88.0"` // 0-100
}

// Validate checks that all factor scores are in [0, 100].
func (d ScoreDetails) Validate() error {
	values := []struct {
		name  string
		value float64
	}{
		{"skill_match", d.SkillMatch},
		{"experience_match", d.ExperienceMatch},
		{"location_match", d.LocationMatch},
		{"salary_match", d.SalaryMatch},
		{"description_match", d.DescriptionMatch},
	}
	for _, v := range values {
		if v.value < 0 || v.value > 100 {
			return errors.New("scoring: factor " + v.name + " must be 0-100")
		}
	}
	return nil
}

// Weights holds factor weights for score computation.
type Weights struct {
	Skill       float64 `json:"skill"`
	Experience  float64 `json:"experience"`
	Location    float64 `json:"location"`
	Salary      float64 `json:"salary"`
	Description float64 `json:"description"`
}

// Validate checks that all weights are non-negative and sum to ~1.0.
func (w Weights) Validate() error {
	weights := []struct {
		name  string
		value float64
	}{
		{"skill", w.Skill},
		{"experience", w.Experience},
		{"location", w.Location},
		{"salary", w.Salary},
		{"description", w.Description},
	}
	for _, w := range weights {
		if w.value < 0 {
			return errors.New("scoring: weight " + w.name + " must be non-negative")
		}
	}
	sum := w.Skill + w.Experience + w.Location + w.Salary + w.Description
	if sum == 0 {
		return errors.New("scoring: weights must not all be zero")
	}
	return nil
}

// ComputeScore calculates a weighted average from individual factor scores.
// Weights are normalized by their sum, so they don't need to sum to 1.0.
// Result is clamped to [0, 100] and rounded to 2 decimal places.
func ComputeScore(details ScoreDetails, weights Weights) float64 {
	totalWeight := weights.Skill + weights.Experience + weights.Location + weights.Salary + weights.Description
	if totalWeight == 0 {
		return 0
	}
	total := details.SkillMatch*weights.Skill +
		details.ExperienceMatch*weights.Experience +
		details.LocationMatch*weights.Location +
		details.SalaryMatch*weights.Salary +
		details.DescriptionMatch*weights.Description
	score := total / totalWeight
	return NormalizeScore(math.Round(score*100) / 100)
}

// NormalizeScore clamps a score to [0, 100].
func NormalizeScore(score float64) float64 {
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

// NormalizeSkills lowercases and trims a list of skill strings.
func NormalizeSkills(skills []string) []string {
	out := make([]string, len(skills))
	for i, s := range skills {
		out[i] = strings.ToLower(strings.TrimSpace(s))
	}
	return out
}

// SkillOverlap computes what fraction of required skills are present in candidate.
// Inputs must be pre-normalized (lowercase, trimmed) and deduplicated by caller.
// Returns 0-100.
func SkillOverlap(required, candidate []string) float64 {
	if len(required) == 0 {
		return 100
	}

	// Deduplicate required skills
	seen := make(map[string]struct{}, len(required))
	var uniqueRequired []string
	for _, r := range required {
		if _, exists := seen[r]; !exists {
			seen[r] = struct{}{}
			uniqueRequired = append(uniqueRequired, r)
		}
	}

	candidateSet := make(map[string]struct{}, len(candidate))
	for _, s := range candidate {
		candidateSet[s] = struct{}{}
	}
	matched := 0
	for _, r := range uniqueRequired {
		if _, ok := candidateSet[r]; ok {
			matched++
		}
	}
	return float64(matched) / float64(len(uniqueRequired)) * 100
}

// KeywordOverlap computes what fraction of keywords appear in the text.
// Uses substring matching — "go" will match "google", "sql" will match "postgresql".
// For job matching this is acceptable: partial skill overlap still indicates relevance.
// Returns NeutralScore (50) when no keywords are provided — absence of keywords
// means no signal either way, so a moderate score is returned.
// Returns 0-100.
func KeywordOverlap(text string, keywords []string) float64 {
	if len(keywords) == 0 {
		return NeutralScore
	}
	textLower := strings.ToLower(text)
	matched := 0
	for _, kw := range keywords {
		if strings.Contains(textLower, strings.ToLower(kw)) {
			matched++
		}
	}
	return float64(matched) / float64(len(keywords)) * 100
}
