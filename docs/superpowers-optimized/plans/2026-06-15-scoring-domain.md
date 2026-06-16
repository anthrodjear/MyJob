# Scoring Domain Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers-optimized:subagent-driven-development (recommended) or superpowers-optimized:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a scoring domain that computes job-profile match scores (0-100) using weighted factors, determines approval tiers (auto/review/reject), and exposes API endpoints to view/trigger scoring.

**Architecture:** The scoring service reads job data from the jobs repository and profile data from the profiles table (JSONB). It computes weighted factor scores (skill, experience, location, salary, description) and returns a ScoreResult with tier classification. Scores are stored in the existing `jobs` table columns (`match_score`, `match_details`). No new database tables needed. The worker calls the scoring service when processing discovered jobs; the API handler exposes endpoints to read stored scores and trigger re-scoring.

**Tech Stack:** Go, Gin, sqlx, zap, config-driven weights via `config.ScoringConfig`

**Assumptions:**
- The `profiles` table stores user profile as JSONB in `data` column — scoring service reads and unmarshals it
- The `jobs` repository already provides `GetByID()` — scoring service calls it to read job data
- Factor weights are configurable via `config.ScoringConfig` (not hardcoded)
- The scoring domain does NOT write to the database — it computes and returns. The caller (worker or handler) decides what to do with the result
- Profile data structure (skills, experience, preferences) follows the design spec's `profile/master.yaml` format

---

## File Structure

| File | Responsibility |
|------|---------------|
| `internal/scoring/model.go` | Tier constants, ScoreResult/ScoreDetails types, ComputeScore(), Tier(), factor scoring helpers |
| `internal/scoring/dto.go` | API request/response types |
| `internal/scoring/service.go` | ScoreJob() — reads job + profile, computes weighted factors, returns ScoreResult |
| `internal/scoring/handler.go` | GET /jobs/:id/score (read stored), POST /jobs/:id/score (trigger re-scoring) |
| `internal/config/config.go` | Add `ScoringWeights` to `ScoringConfig` |
| `internal/api/router.go` | Register scoring routes |
| `internal/cmd/api/main.go` | Initialize scoring handler + service |

---

## Task 1: Add scoring weights to config

**Files:**
- Modify: `backend/internal/config/config.go`

**Does NOT cover:** Environment variable loading for individual weights (uses a single JSON env var or defaults)

- [ ] **Step 1: Add ScoringWeights struct and defaults to ScoringConfig**

```go
// In config.go, update ScoringConfig:

type ScoringWeights struct {
	Skill       float64 `json:"skill"`
	Experience  float64 `json:"experience"`
	Location    float64 `json:"location"`
	Salary      float64 `json:"salary"`
	Description float64 `json:"description"`
}

type ScoringConfig struct {
	AutoThreshold   int            // score >= this = auto apply
	ReviewThreshold int            // score >= this = human review
	Weights         ScoringWeights // factor weights (must sum to 1.0)
}
```

- [ ] **Step 2: Add default weights in Load() and validation in Validate()**

```go
// In Load(), update Scoring section:
Scoring: ScoringConfig{
	AutoThreshold:   getEnvInt("SCORING_AUTO_THRESHOLD", 95),
	ReviewThreshold: getEnvInt("SCORING_REVIEW_THRESHOLD", 80),
	Weights: ScoringWeights{
		Skill:       getEnvFloat("SCORING_WEIGHT_SKILL", 0.35),
		Experience:  getEnvFloat("SCORING_WEIGHT_EXPERIENCE", 0.25),
		Location:    getEnvFloat("SCORING_WEIGHT_LOCATION", 0.10),
		Salary:      getEnvFloat("SCORING_WEIGHT_SALARY", 0.15),
		Description: getEnvFloat("SCORING_WEIGHT_DESCRIPTION", 0.15),
	},
},

// Add getEnvFloat helper:
func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			return floatVal
		}
	}
	return defaultValue
}

// In Validate(), add weight sum check:
func (c *Config) validateScoringWeights() error {
	sum := c.Scoring.Weights.Skill + c.Scoring.Weights.Experience +
		c.Scoring.Weights.Location + c.Scoring.Weights.Salary +
		c.Scoring.Weights.Description
	if sum < 0.99 || sum > 1.01 { // float tolerance
		return fmt.Errorf("config: scoring weights must sum to 1.0, got %.2f", sum)
	}
	return nil
}
```

- [ ] **Step 3: Verify build**

Run: `go build ./...`
Expected: PASS

---

## Task 2: Create scoring model

**Files:**
- Create: `backend/internal/scoring/model.go`

**Does NOT cover:** API types (dto.go), service logic, handler logic

- [ ] **Step 1: Create model.go with tier constants, types, and factor helpers**

```go
package scoring

import (
	"math"
	"strings"
)

// Tier constants for score-based approval.
const (
	TierReject = "reject" // score < ReviewThreshold
	TierReview = "review" // score >= ReviewThreshold, < AutoThreshold
	TierAuto   = "auto"   // score >= AutoThreshold
)

// Tier returns the approval tier for a given score and thresholds.
func Tier(score float64, autoThreshold, reviewThreshold int) string {
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
	Score   float64        `json:"score"`   // 0-100
	Tier    string         `json:"tier"`    // auto, review, reject
	Details *ScoreDetails  `json:"details"` // breakdown of scoring factors
}

// ScoreDetails provides per-factor scoring breakdown.
type ScoreDetails struct {
	SkillMatch       float64 `json:"skill_match"`       // 0-100
	ExperienceMatch  float64 `json:"experience_match"`  // 0-100
	LocationMatch    float64 `json:"location_match"`    // 0-100
	SalaryMatch      float64 `json:"salary_match"`      // 0-100
	DescriptionMatch float64 `json:"description_match"` // 0-100
}

// ComputeScore calculates a final score from individual factor scores using provided weights.
// Returns a score in [0, 100] range.
func ComputeScore(details ScoreDetails, weights Weights) float64 {
	total := 0.0
	total += details.SkillMatch * weights.Skill
	total += details.ExperienceMatch * weights.Experience
	total += details.LocationMatch * weights.Location
	total += details.SalaryMatch * weights.Salary
	total += details.DescriptionMatch * weights.Description
	return math.Round(total*100) / 100
}

// Weights holds factor weights for score computation.
type Weights struct {
	Skill       float64
	Experience  float64
	Location    float64
	Salary      float64
	Description float64
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
// Returns 0-100.
func SkillOverlap(required, candidate []string) float64 {
	if len(required) == 0 {
		return 100
	}
	candidateSet := make(map[string]struct{}, len(candidate))
	for _, s := range candidate {
		candidateSet[s] = struct{}{}
	}
	matched := 0
	for _, r := range required {
		if _, ok := candidateSet[r]; ok {
			matched++
		}
	}
	return float64(matched) / float64(len(required)) * 100
}

// KeywordOverlap computes what fraction of keywords appear in the text.
// Returns 0-100.
func KeywordOverlap(text string, keywords []string) float64 {
	if len(keywords) == 0 {
		return 50
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
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: PASS

---

## Task 3: Create scoring DTOs

**Files:**
- Create: `backend/internal/scoring/dto.go`

**Does NOT cover:** Service logic, handler logic

- [ ] **Step 1: Create dto.go with API types**

```go
package scoring

import "encoding/json"

// ScoreResponse is the API response for a scored job.
type ScoreResponse struct {
	JobID   string         `json:"job_id"`
	Score   float64        `json:"score"`
	Tier    string         `json:"tier"`
	Details *ScoreDetails  `json:"details,omitempty"`
}

// ScoreBreakdownResponse provides detailed scoring breakdown.
type ScoreBreakdownResponse struct {
	JobID   string         `json:"job_id"`
	Score   float64        `json:"score"`
	Tier    string         `json:"tier"`
	Details ScoreDetails   `json:"details"`
	Weight  WeightResponse `json:"weight"`
}

// WeightResponse shows the weights used for scoring.
type WeightResponse struct {
	Skill       float64 `json:"skill"`
	Experience  float64 `json:"experience"`
	Location    float64 `json:"location"`
	Salary      float64 `json:"salary"`
	Description float64 `json:"description"`
}

// ScoreFromJobResult converts a stored job score to API response.
func ScoreFromJobResult(jobID string, score float64, details json.RawMessage) ScoreResponse {
	resp := ScoreResponse{
		JobID: jobID,
		Score: score,
		Tier:  Tier(score, 95, 80), // defaults, overridden by service
	}
	if details != nil {
		var d ScoreDetails
		if err := json.Unmarshal(details, &d); err == nil {
			resp.Details = &d
		}
	}
	return resp
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: PASS

---

## Task 4: Create scoring service

**Files:**
- Create: `backend/internal/scoring/service.go`

**Does NOT cover:** Handler logic, API routing

- [ ] **Step 1: Create service.go with ScoreJob logic**

```go
package scoring

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"backend/internal/config"
)

var (
	ErrNotFound = errors.New("scoring: not found")
)

// Profile represents the user's master profile data (from profiles table JSONB).
type Profile struct {
	Skills     []string          `json:"skills"`
	Experience []ProfileExperience `json:"experience"`
	Preferences ProfilePreferences `json:"preferences"`
}

// ProfileExperience represents a work experience entry.
type ProfileExperience struct {
	Title       string   `json:"title"`
	Company     string   `json:"company"`
	SkillsUsed  []string `json:"skills_used"`
}

// ProfilePreferences holds location and salary preferences.
type ProfilePreferences struct {
	PreferredLocations []string `json:"preferred_locations"`
	RemoteOnly         bool     `json:"remote_only"`
	SalaryMin          int      `json:"salary_min"`
	SalaryMax          int      `json:"salary_max"`
}

// JobReader provides read-only access to job data.
type JobReader interface {
	GetByID(ctx context.Context, id interface{}) (JobData, error)
}

// JobData holds the job fields needed for scoring.
type JobData struct {
	ID           string
	Title        string
	Description  string
	Requirements string
	Location     string
	RemoteType   string
	SalaryMin    int
	SalaryMax    int
}

// Service handles scoring business logic.
type Service struct {
	db     *sqlx.DB
	logger *zap.Logger
	cfg    config.ScoringConfig
}

// NewService creates a new scoring service.
func NewService(db *sqlx.DB, logger *zap.Logger, cfg config.ScoringConfig) *Service {
	return &Service{
		db:     db,
		logger: logger.Named("scoring"),
		cfg:    cfg,
	}
}

// ScoreJob computes a match score for a job against the user's profile.
// Reads job data and profile from DB, computes weighted factor scores.
func (s *Service) ScoreJob(ctx context.Context, jobID string) (*ScoreResult, error) {
	// Fetch job
	job, err := s.getJob(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("scoring: get job: %w", err)
	}

	// Fetch profile
	profile, err := s.getProfile(ctx)
	if err != nil {
		return nil, fmt.Errorf("scoring: get profile: %w", err)
	}

	// Compute factor scores
	details := s.computeFactors(job, profile)

	// Compute final weighted score
	weights := Weights{
		Skill:       s.cfg.Weights.Skill,
		Experience:  s.cfg.Weights.Experience,
		Location:    s.cfg.Weights.Location,
		Salary:      s.cfg.Weights.Salary,
		Description: s.cfg.Weights.Description,
	}
	score := ComputeScore(details, weights)

	return &ScoreResult{
		Score:   score,
		Tier:    Tier(score, s.cfg.AutoThreshold, s.cfg.ReviewThreshold),
		Details: &details,
	}, nil
}

// getJob fetches job data by ID.
func (s *Service) getJob(ctx context.Context, jobID string) (JobData, error) {
	var job JobData
	err := s.db.QueryRowxContext(ctx,
		"SELECT id, title, description, requirements, location, remote_type, salary_min, salary_max FROM jobs WHERE id = $1",
		jobID,
	).StructScan(&job)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return job, ErrNotFound
		}
		return job, err
	}
	return job, nil
}

// getProfile fetches the user's master profile.
func (s *Service) getProfile(ctx context.Context) (Profile, error) {
	var profile Profile
	var data json.RawMessage
	err := s.db.QueryRowxContext(ctx,
		"SELECT data FROM profiles LIMIT 1",
	).Scan(&data)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return profile, nil // no profile = neutral scores
		}
		return profile, err
	}
	if err := json.Unmarshal(data, &profile); err != nil {
		return profile, fmt.Errorf("unmarshal profile: %w", err)
	}
	return profile, nil
}

// computeFactors calculates individual factor scores.
func (s *Service) computeFactors(job JobData, profile Profile) ScoreDetails {
	// Skill match: extract keywords from job requirements, compare to profile skills
	jobKeywords := extractKeywords(job.Requirements + " " + job.Title)
	profileSkills := NormalizeSkills(profile.Skills)

	skillMatch := SkillOverlap(jobKeywords, profileSkills)

	// Experience match: check if profile has relevant job titles
	experienceMatch := computeExperienceMatch(job.Title, profile.Experience)

	// Location match
	locationMatch := computeLocationMatch(job.Location, job.RemoteType, profile.Preferences)

	// Salary match
	salaryMatch := computeSalaryMatch(job.SalaryMin, job.SalaryMax, profile.Preferences)

	// Description match: keyword overlap in job description
	descriptionMatch := KeywordOverlap(job.Description, jobKeywords)

	return ScoreDetails{
		SkillMatch:       NormalizeScore(skillMatch),
		ExperienceMatch:  NormalizeScore(experienceMatch),
		LocationMatch:    NormalizeScore(locationMatch),
		SalaryMatch:      NormalizeScore(salaryMatch),
		DescriptionMatch: NormalizeScore(descriptionMatch),
	}
}

// extractKeywords splits text into meaningful keywords (lowercase, trimmed).
func extractKeywords(text string) []string {
	// Simple implementation: split on whitespace and common delimiters
	// In production, this would use NLP or LLM for better extraction
	words := strings.Fields(strings.ToLower(text))
	seen := make(map[string]struct{})
	var keywords []string
	for _, w := range words {
		w = strings.Trim(w, ".,;:!?()[]{}\"'")
		if len(w) < 3 {
			continue
		}
		if _, exists := seen[w]; exists {
			continue
		}
		seen[w] = struct{}{}
		keywords = append(keywords, w)
	}
	return keywords
}

// computeExperienceMatch checks if profile experience is relevant to the job.
func computeExperienceMatch(jobTitle string, experiences []ProfileExperience) float64 {
	if len(experiences) == 0 {
		return 50 // no experience = neutral
	}
	jobTitleLower := strings.ToLower(jobTitle)
	bestMatch := 0.0
	for _, exp := range experiences {
		titleLower := strings.ToLower(exp.Title)
		// Simple title similarity: check if words overlap
		overlap := KeywordOverlap(titleLower, strings.Fields(jobTitleLower))
		if overlap > bestMatch {
			bestMatch = overlap
		}
	}
	return bestMatch
}

// computeLocationMatch checks if job location matches preferences.
func computeLocationMatch(jobLocation, jobRemoteType string, prefs ProfilePreferences) float64 {
	if prefs.RemoteOnly && jobRemoteType == "remote" {
		return 100
	}
	if prefs.RemoteOnly && jobRemoteType != "remote" {
		return 20
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
func computeSalaryMatch(jobMin, jobMax int, prefs ProfilePreferences) float64 {
	if jobMin == 0 && jobMax == 0 {
		return 50 // no salary info = neutral
	}
	if prefs.SalaryMin == 0 && prefs.SalaryMax == 0 {
		return 75 // no preference = neutral-high
	}
	// Check overlap between ranges
	if jobMax > 0 && jobMax < prefs.SalaryMin {
		return 20 // job pays less than minimum
	}
	if prefs.SalaryMax > 0 && jobMin > prefs.SalaryMax {
		return 30 // job pays more than maximum (overqualified?)
	}
	return 80 // ranges overlap
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: PASS

---

## Task 5: Create scoring handler

**Files:**
- Create: `backend/internal/scoring/handler.go`

**Does NOT cover:** Router registration, main.go initialization

- [ ] **Step 1: Create handler.go**

```go
package scoring

import (
	"database/sql"
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"backend/internal/httpresp"
)

// Handler holds the scoring HTTP handlers.
type Handler struct {
	svc    *Service
	logger *zap.Logger
}

// NewHandler creates a new scoring handler.
func NewHandler(svc *Service, logger *zap.Logger) *Handler {
	return &Handler{
		svc:    svc,
		logger: logger.Named("scoring"),
	}
}

// RegisterRoutes registers scoring routes on the router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	scores := rg.Group("/jobs/:id/score")
	{
		scores.GET("", h.GetScore)
		scores.POST("", h.TriggerScore)
	}
}

// GetScore handles GET /jobs/:id/score
// Returns the stored score for a job from the jobs table.
func (h *Handler) GetScore(c *gin.Context) {
	jobID := c.Param("id")
	if _, err := uuid.Parse(jobID); err != nil {
		httpresp.BadRequest(c, "INVALID_ID", "invalid job id")
		return
	}

	// Read stored score from jobs table
	var score float64
	var details sql.NullString
	err := h.svc.db.QueryRowContext(c.Request.Context(),
		"SELECT match_score, match_details FROM jobs WHERE id = $1",
		jobID,
	).Scan(&score, &details)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpresp.NotFound(c, "JOB_NOT_FOUND", "job not found")
			return
		}
		h.logger.Error("get score", zap.String("job_id", jobID), zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	var rawDetails []byte
	if details.Valid {
		rawDetails = []byte(details.String)
	}

	resp := ScoreFromJobResult(jobID, score, rawDetails)
	httpresp.OK(c, resp)
}

// TriggerScore handles POST /jobs/:id/score
// Runs scoring computation and returns the result (sync for now).
func (h *Handler) TriggerScore(c *gin.Context) {
	jobID := c.Param("id")
	if _, err := uuid.Parse(jobID); err != nil {
		httpresp.BadRequest(c, "INVALID_ID", "invalid job id")
		return
	}

	result, err := h.svc.ScoreJob(c.Request.Context(), jobID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "JOB_NOT_FOUND", "job not found")
			return
		}
		h.logger.Error("trigger score", zap.String("job_id", jobID), zap.Error(err))
		httpresp.InternalError(c)
		return
	}

	httpresp.OK(c, ScoreResponse{
		JobID:   jobID,
		Score:   result.Score,
		Tier:    result.Tier,
		Details: result.Details,
	})
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: PASS

---

## Task 6: Wire scoring into router and main.go

**Files:**
- Modify: `backend/internal/api/router.go`
- Modify: `backend/cmd/api/main.go`

**Does NOT cover:** Worker task handlers (Wave 2)

- [ ] **Step 1: Update router.go to include scoring**

```go
// Add import:
import "backend/internal/scoring"

// Add to RouterConfig:
ScoringHandler *scoring.Handler

// Add to protected routes group:
cfg.ScoringHandler.RegisterRoutes(protected)
```

- [ ] **Step 2: Update main.go to initialize scoring**

```go
// Add import:
import "backend/internal/scoring"

// After jobs initialization:
scoringService := scoring.NewService(postgres.DB, logger, cfg.Scoring)
scoringHandler := scoring.NewHandler(scoringService, logger)

// Add to RouterConfig:
ScoringHandler: scoringHandler,
```

- [ ] **Step 3: Full build verification**

Run: `go build ./...`
Expected: PASS

---

## Self-Review

**1. Spec coverage:**
- ✅ Score computation (model.go)
- ✅ Tier classification (model.go)
- ✅ Weighted factors (config-driven via ScoringConfig)
- ✅ API endpoints (handler.go: GET + POST)
- ✅ Profile reading (service.go: getProfile)
- ✅ Job reading (service.go: getJob)
- ✅ No new DB tables (uses existing jobs + profiles tables)
- ✅ Config-driven weights (config.go update)

**2. Placeholder scan:** No TBD/TODO found. All code blocks are complete.

**3. Type consistency:**
- `ScoreResult` used consistently in service and handler
- `ScoreResponse` used in handler responses
- `ScoreDetails` used in both model and DTO
- `config.ScoringConfig.Weights` matches `Weights` struct in model

**Potential issue:** The handler reads scores directly from the DB (GetScore), but the service also reads jobs. This is intentional — GetScore is a simple read (no scoring computation), while TriggerScore runs the full algorithm.

---

**Plan complete and saved to `docs/superpowers-optimized/plans/2026-06-15-scoring-domain.md`.**

Two execution options:

1. **Subagent-Driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration

2. **Inline Execution** — Execute tasks in this session using executing-plans, with checkpoints

Which approach?
