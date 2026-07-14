// Package scoring provides job-candidate matching and scoring functionality.
// It supports three scoring modes: heuristic (keyword-based), LLM (semantic), and hybrid (pre-filter + LLM).
// The service computes factor scores (skills, experience, location, salary, description) and combines them
// into a final 0-100 score with approval tier (auto/review/reject).
package scoring

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"backend/internal/config"
)

// Profile represents the user's master profile data.
type Profile struct {
	Skills          []string            `json:"skills"`
	Experience      []ProfileExperience `json:"experience"`
	Preferences     ProfilePreferences  `json:"preferences"`
	Specializations []string            `json:"specializations"` // e.g., "Backend Engineering", "Cloud Infrastructure"
	Industries      []string            `json:"industries"`      // e.g., "Fintech", "SaaS"
	CareerGoals     []string            `json:"career_goals"`    // e.g., "Senior Backend Engineer"
}

// ProfileExperience represents a work experience entry.
type ProfileExperience struct {
	Title      string   `json:"title"`
	Company    string   `json:"company"`
	SkillsUsed []string `json:"skills_used"`
}

// ProfilePreferences holds location and salary preferences.
type ProfilePreferences struct {
	PreferredLocations []string `json:"preferred_locations"`
	RemoteOnly         bool     `json:"remote_only"`
	SalaryMin          int      `json:"salary_min"`
	SalaryMax          int      `json:"salary_max"`
}

// Service handles scoring business logic.
type Service struct {
	repo   Repository
	llm    LLMScorer
	logger *zap.Logger
	cfg    config.ScoringConfig
}

// NewService creates a new scoring service.
func NewService(repo Repository, llm LLMScorer, logger *zap.Logger, cfg config.ScoringConfig) *Service {
	return &Service{
		repo:   repo,
		llm:    llm,
		logger: logger.Named("scoring"),
		cfg:    cfg,
	}
}

// GetJob returns a job by ID (exposes repository method for handlers).
func (s *Service) GetJob(ctx context.Context, id uuid.UUID) (JobData, error) {
	return s.repo.GetJob(ctx, id)
}

// ScoreJob computes a match score for a job against the user's profile.
// Flow depends on ScoringConfig.Mode:
//   - "heuristic": fast keyword-based scoring only (no LLM cost)
//   - "llm": LLM-based semantic scoring only
//   - "hybrid": heuristic pre-filter (auto-reject obvious mismatches), LLM owns final score
func (s *Service) ScoreJob(ctx context.Context, jobID uuid.UUID) (*ScoreResult, error) {
	job, err := s.repo.GetJob(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("scoring: get job: %w", err)
	}

	profile, err := s.repo.GetProfile(ctx)
	if err != nil {
		return nil, fmt.Errorf("scoring: get profile: %w", err)
	}

	// Validate that profile has minimum data for scoring
	if err := validateProfile(profile); err != nil {
		return nil, fmt.Errorf("scoring: profile not configured: %w", err)
	}

	switch s.cfg.Mode {
	case "llm":
		// LLM-only: no heuristic computation, LLM owns the decision
		return s.scoreWithLLM(ctx, job, profile, jobID)

	case "hybrid":
		// Hybrid: heuristic pre-filter, then LLM for final decision
		// First, compute fast heuristics for pre-filtering
		heuristicDetails := computeFactors(job, profile)
		weights := Weights{
			Skill:       s.cfg.Weights.Skill,
			Experience:  s.cfg.Weights.Experience,
			Location:    s.cfg.Weights.Location,
			Salary:      s.cfg.Weights.Salary,
			Description: s.cfg.Weights.Description,
		}
		heuristicScore := ComputeScore(heuristicDetails, weights)
		heuristicTier := Tier(heuristicScore, s.cfg.AutoThreshold, s.cfg.ReviewThreshold)

		// Fast-path: auto-reject obvious mismatches without LLM cost.
		// The auto-reject threshold is reviewThreshold - margin.
		// If a job scores below this, it's so far from qualifying that
		// the LLM won't change the outcome — skip the API call.
		margin := s.cfg.HybridRejectMargin
		if margin == 0 {
			margin = 20
		}
		autoRejectThreshold := float64(s.cfg.ReviewThreshold) - float64(margin)
		if autoRejectThreshold < 0 {
			autoRejectThreshold = 0
		}
		if heuristicTier == TierReject && heuristicScore < autoRejectThreshold {
			s.logger.Debug("heuristic auto-reject, skipping LLM",
				zap.String("job_id", jobID.String()),
				zap.Float64("heuristic_score", heuristicScore),
				zap.Float64("auto_reject_threshold", autoRejectThreshold),
			)
			return s.persistAndReturn(ctx, jobID, heuristicScore, heuristicTier, "heuristic", "", &heuristicDetails)
		}

		// Not an obvious reject → LLM owns the final score (no blending)
		llmResult, err := s.llm.ScoreJob(ctx, job, profile)
		if err != nil {
			s.logger.Warn("LLM scoring failed, falling back to heuristics",
				zap.String("job_id", jobID.String()),
				zap.Error(err),
			)
			return s.persistAndReturn(ctx, jobID, heuristicScore, Tier(heuristicScore, s.cfg.AutoThreshold, s.cfg.ReviewThreshold), "heuristic", "", &heuristicDetails)
		}

		// LLM result is the final score (no blending)
		details := llmResult.Details
		if details == nil {
			details = &heuristicDetails // fallback for transparency
		}

		llmTier := Tier(llmResult.Score, s.cfg.AutoThreshold, s.cfg.ReviewThreshold)
		return s.persistAndReturn(ctx, jobID, llmResult.Score, llmTier, "hybrid", llmResult.Reasoning, details)

	default: // "heuristic"
		// Heuristic-only: fast keyword-based scoring
		heuristicDetails := computeFactors(job, profile)
		weights := Weights{
			Skill:       s.cfg.Weights.Skill,
			Experience:  s.cfg.Weights.Experience,
			Location:    s.cfg.Weights.Location,
			Salary:      s.cfg.Weights.Salary,
			Description: s.cfg.Weights.Description,
		}
		heuristicScore := ComputeScore(heuristicDetails, weights)
		tier := Tier(heuristicScore, s.cfg.AutoThreshold, s.cfg.ReviewThreshold)
		return s.persistAndReturn(ctx, jobID, heuristicScore, tier, "heuristic", "", &heuristicDetails)
	}
}

// scoreWithLLM uses only the LLM for scoring (mode = "llm").
func (s *Service) scoreWithLLM(ctx context.Context, job JobData, profile Profile, jobID uuid.UUID) (*ScoreResult, error) {
	llmResult, err := s.llm.ScoreJob(ctx, job, profile)
	if err != nil {
		return nil, fmt.Errorf("scoring: llm: %w", err)
	}

	// Use LLM details if available, otherwise compute heuristic details as fallback
	details := llmResult.Details
	if details == nil {
		d := computeFactors(job, profile)
		details = &d
	}
	tier := Tier(llmResult.Score, s.cfg.AutoThreshold, s.cfg.ReviewThreshold)

	return s.persistAndReturn(ctx, jobID, llmResult.Score, tier, "llm", llmResult.Reasoning, details)
}

// persistAndReturn saves the score and returns the result.
func (s *Service) persistAndReturn(ctx context.Context, jobID uuid.UUID, score float64, tier ApprovalTier, source, reasoning string, details *ScoreDetails) (*ScoreResult, error) {
	result := &ScoreResult{
		Score:     score,
		Tier:      tier,
		Reasoning: reasoning,
		Source:    source,
		Model:     s.getModelName(),
		Details:   details,
	}

	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return nil, fmt.Errorf("scoring: marshal details: %w", err)
	}
	if err := s.repo.PersistScore(ctx, jobID, score, string(tier), detailsJSON, reasoning, s.getModelName(), source); err != nil {
		return nil, fmt.Errorf("scoring: persist: %w", err)
	}

	s.logger.Debug("scored job",
		zap.String("job_id", jobID.String()),
		zap.Float64("score", score),
		zap.String("tier", string(tier)),
		zap.String("source", source),
		zap.String("mode", s.cfg.Mode),
	)

	return result, nil
}

// getModelName returns the model name from LLM scorer for metadata.
func (s *Service) getModelName() string {
	return s.llm.ModelName()
}

// computeFactors calculates individual factor scores.
// Pure function — no service state needed.
func computeFactors(job JobData, profile Profile) ScoreDetails {
	// Skill matching: use requirements only (title words like "senior" aren't skills)
	jobKeywords := extractKeywords(job.Requirements)
	profileSkills := NormalizeSkills(profile.Skills)
	skillMatch := SkillOverlap(jobKeywords, profileSkills)

	// Experience matching: use both title overlap and skills used in past roles
	experienceMatch := computeExperienceMatch(job.Title, jobKeywords, profile.Experience)

	// Location matching
	locationMatch := computeLocationMatch(job.Location, job.RemoteType, profile.Preferences)

	// Salary matching
	salaryMatch := computeSalaryMatch(job.SalaryMin, job.SalaryMax, profile.Preferences)

	// Description matching: profile skills mentioned in full job description
	descriptionMatch := KeywordOverlap(job.Description, profileSkills)

	return ScoreDetails{
		SkillMatch:       NormalizeScore(skillMatch),
		ExperienceMatch:  NormalizeScore(experienceMatch),
		LocationMatch:    NormalizeScore(locationMatch),
		SalaryMatch:      NormalizeScore(salaryMatch),
		DescriptionMatch: NormalizeScore(descriptionMatch),
	}
}

// computeExperienceMatch checks if profile experience is relevant to the job.
// Uses both title overlap and skills used in past roles.
// Averages the top 3 experience matches (not just the single best).
func computeExperienceMatch(jobTitle string, jobKeywords []string, experiences []ProfileExperience) float64 {
	if len(experiences) == 0 {
		return 50 // no experience = neutral
	}
	scores := make([]float64, 0, len(experiences))
	for _, exp := range experiences {
		// Title overlap signal
		titleScore := KeywordOverlap(strings.ToLower(exp.Title), extractKeywords(jobTitle))
		// Skills used in this role vs job requirements
		skillsScore := SkillOverlap(jobKeywords, exp.SkillsUsed)
		// Take the better of the two signals
		combined := math.Max(titleScore, skillsScore)
		scores = append(scores, combined)
	}
	// Sort descending, take top 3
	sort.Sort(sort.Reverse(sort.Float64Slice(scores)))
	limit := 3
	if len(scores) < limit {
		limit = len(scores)
	}
	sum := 0.0
	for i := 0; i < limit; i++ {
		sum += scores[i]
	}
	return sum / float64(limit)
}

// computeLocationMatch checks if job location matches preferences.
func computeLocationMatch(jobLocation, jobRemoteType string, prefs ProfilePreferences) float64 {
	if prefs.RemoteOnly && jobRemoteType == "remote" {
		return 100
	}
	if prefs.RemoteOnly && jobRemoteType == "hybrid" {
		return 50 // hybrid is a middle ground for remote-only users
	}
	if prefs.RemoteOnly {
		return 20 // on-site
	}
	if len(prefs.PreferredLocations) == 0 {
		return 75 // no preference = neutral-high
	}
	jobLocLower := strings.ToLower(jobLocation)
	for _, loc := range prefs.PreferredLocations {
		if strings.Contains(jobLocLower, strings.ToLower(loc)) {
			return 100
		}
	}
	return 40 // not in preferred locations
}

// computeSalaryMatch checks if job salary range aligns with preferences.
// Uses distance-based scoring: closer to preference = higher score.
func computeSalaryMatch(jobMin, jobMax int, prefs ProfilePreferences) float64 {
	if jobMin == 0 && jobMax == 0 {
		return 50 // no salary info = neutral
	}
	if prefs.SalaryMin == 0 && prefs.SalaryMax == 0 {
		return 75 // no preference = neutral-high
	}

	// Use midpoints for comparison
	jobMid := float64(jobMin+jobMax) / 2.0
	prefMid := float64(prefs.SalaryMin+prefs.SalaryMax) / 2.0

	// If ranges overlap, score based on how well they align
	if jobMax >= prefs.SalaryMin && (prefs.SalaryMax == 0 || jobMin <= prefs.SalaryMax) {
		// Ranges overlap — score based on midpoint proximity
		if prefMid == 0 {
			return 80
		}
		diff := math.Abs(jobMid-prefMid) / prefMid
		if diff < 0.1 {
			return 95 // very close match
		}
		if diff < 0.2 {
			return 85
		}
		return 75 // overlap but not aligned
	}

	// No overlap — distance-based penalty
	if prefMid == 0 {
		return 50
	}
	diff := math.Abs(jobMid-prefMid) / prefMid
	if diff < 0.1 {
		return 60 // barely outside range
	}
	if diff < 0.25 {
		return 40
	}
	if diff < 0.5 {
		return 25
	}
	return 10 // very far from preference
}

// validateProfile checks that the profile has minimum data for scoring.
func validateProfile(p Profile) error {
	if len(p.Skills) == 0 && len(p.Experience) == 0 {
		return errors.New("profile must have at least skills or experience entries")
	}
	return nil
}
