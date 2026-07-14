// Package scoring provides job-candidate matching and scoring functionality.
// It supports three scoring modes: heuristic (keyword-based), LLM (semantic), and hybrid (pre-filter + LLM).
// The service computes factor scores (skills, experience, location, salary, description) and combines them
// into a final 0-100 score with approval tier (auto/review/reject).
package scoring

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"backend/internal/httpresp"
	"backend/internal/tasks"
)

// Handler holds the scoring HTTP handlers.
type Handler struct {
	svc        *Service
	dispatcher *tasks.Dispatcher
	logger     *zap.Logger
}

// NewHandler creates a new scoring handler.
func NewHandler(svc *Service, dispatcher *tasks.Dispatcher, logger *zap.Logger) *Handler {
	return &Handler{
		svc:        svc,
		dispatcher: dispatcher,
		logger:     logger.Named("scoring"),
	}
}

// RegisterRoutes registers scoring routes on the router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	scoring := rg.Group("/scoring")
	{
		scoring.POST("/score", h.ScoreJobAsync)
		scoring.GET("/score/:jobId", h.GetScore)
		scoring.POST("/batch", h.BatchScoreAsync)
	}
}

// scoreRequest is the payload for POST /scoring/score.
type scoreRequest struct {
	JobID uuid.UUID `json:"job_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// ScoreJobAsync handles POST /scoring/score.
// @Summary Score a job asynchronously
// @Description Enqueue a job scoring task using LLM semantic matching
// @Tags Scoring
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body scoreRequest true "Job ID to score"
// @Success 202 {object} map[string]interface{} "Task ID for polling"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid request body"
// @Failure 401 {object} httpresp.ErrorResponse "Unauthorized"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /scoring/score [post]
func (h *Handler) ScoreJobAsync(c *gin.Context) {
	var req scoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, "INVALID_INPUT", "invalid request body: job_id required")
		return
	}

	taskID, err := h.dispatcher.DispatchJobScoring(c.Request.Context(), tasks.JobScoringPayload{
		JobID: req.JobID,
	})
	if err != nil {
		h.logger.Error("dispatch scoring task",
			zap.String("job_id", req.JobID.String()),
			zap.Error(err),
		)
		httpresp.InternalError(c)
		return
	}

	httpresp.Accepted(c, gin.H{
		"task_id": taskID,
		"status":  "queued",
		"message": "scoring task enqueued, poll GET /api/v1/tasks/" + taskID,
	})
}

// GetScore handles GET /scoring/score/:jobId.
// @Summary Get job score
// @Description Get the persisted scoring result for a job with details
// @Tags Scoring
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param jobId path string true "Job UUID" format(uuid)
// @Success 200 {object} ScoreResponse "Job score with details"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid job ID"
// @Failure 401 {object} httpresp.ErrorResponse "Unauthorized"
// @Failure 404 {object} httpresp.ErrorResponse "Job not found"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /scoring/score/{jobId} [get]
func (h *Handler) GetScore(c *gin.Context) {
	jobID, err := uuid.Parse(c.Param("jobId"))
	if err != nil {
		httpresp.BadRequest(c, "INVALID_JOB_ID", "invalid job_id format")
		return
	}

	job, err := h.svc.GetJob(c.Request.Context(), jobID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpresp.NotFound(c, "JOB_NOT_FOUND", err.Error())
			return
		}
		h.logger.Error("get job for score",
			zap.String("job_id", jobID.String()),
			zap.Error(err),
		)
		httpresp.InternalError(c)
		return
	}

	// Parse match_details JSONB if present
	var details *ScoreDetails
	if len(job.MatchDetails) > 0 {
		var d ScoreDetails
		if err := json.Unmarshal(job.MatchDetails, &d); err == nil {
			details = &d
		}
	}

	// Parse strengths/gaps from reasoning if available
	strengths, gaps := extractStrengthsAndGaps(job.ScoringReasoning)

	resp := ScoreResponse{
		JobID:      jobID,
		Score:      job.MatchScore,
		Tier:       ApprovalTier(job.ScoreTier),
		Reasoning:  job.ScoringReasoning,
		Source:     job.ScoringSource,
		Model:      job.ScoringModel,
		Confidence: calculateConfidence(job.MatchScore, job.ScoringSource),
		Strengths:  strengths,
		Gaps:       gaps,
		Details:    details,
	}

	httpresp.OK(c, resp)
}

// batchScoreRequest is the payload for POST /scoring/batch.
type batchScoreRequest struct {
	JobIDs []uuid.UUID `json:"job_ids" binding:"required,min=1,max=100" example:"[\"550e8400-e29b-41d4-a716-446655440000\",\"550e8400-e29b-41d4-a716-446655440001\"]"`
}

// BatchScoreAsync handles POST /scoring/batch.
// @Summary Batch score jobs
// @Description Enqueue scoring tasks for multiple jobs
// @Tags Scoring
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body batchScoreRequest true "List of job IDs to score"
// @Success 202 {object} map[string]interface{} "Task IDs for polling"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid request body"
// @Failure 401 {object} httpresp.ErrorResponse "Unauthorized"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /scoring/batch [post]
func (h *Handler) BatchScoreAsync(c *gin.Context) {
	var req batchScoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, "INVALID_INPUT", "invalid request body: job_ids required")
		return
	}

	// Deduplicate
	seen := make(map[uuid.UUID]struct{}, len(req.JobIDs))
	jobIDs := make([]uuid.UUID, 0, len(req.JobIDs))
	for _, id := range req.JobIDs {
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		jobIDs = append(jobIDs, id)
	}

	taskIDs := make([]string, 0, len(jobIDs))
	failed := make([]uuid.UUID, 0)
	for _, jobID := range jobIDs {
		taskID, err := h.dispatcher.DispatchJobScoring(c.Request.Context(), tasks.JobScoringPayload{
			JobID: jobID,
		})
		if err != nil {
			h.logger.Error("dispatch batch scoring task",
				zap.String("job_id", jobID.String()),
				zap.Error(err),
			)
			failed = append(failed, jobID)
			continue
		}
		taskIDs = append(taskIDs, taskID)
	}

	httpresp.Accepted(c, gin.H{
		"total":    len(jobIDs),
		"queued":   len(taskIDs),
		"failed":   len(failed),
		"task_ids": taskIDs,
	})
}

// extractStrengthsAndGaps parses reasoning for strengths/gaps using keyword heuristics.
// In production, this could be enhanced with a dedicated LLM call.
func extractStrengthsAndGaps(reasoning string) ([]string, []string) {
	if reasoning == "" {
		return nil, nil
	}

	reasoningLower := strings.ToLower(reasoning)

	// Strength keywords (positive signals)
	strengthKeywords := map[string]string{
		"strong":      "Strong match",
		"excellent":   "Excellent fit",
		"great":       "Great alignment",
		"relevant":    "Relevant experience",
		"matches":     "Good skill match",
		"experienced": "Experienced candidate",
		"proficient":  "Proficient in required skills",
		"skilled":     "Skilled match",
		"qualified":   "Well qualified",
		"ideal":       "Ideal candidate",
		"perfect":     "Perfect fit",
		"highly":      "Highly relevant",
		"extensive":   "Extensive experience",
		"deep":        "Deep expertise",
		"expert":      "Expert level",
		"senior":      "Senior experience",
		"lead":        "Leadership experience",
		"architect":   "Architecture skills",
		"scale":       "Scalability experience",
		"production":  "Production experience",
	}

	// Gap keywords (negative signals)
	gapKeywords := map[string]string{
		"lack":         "Missing skill",
		"missing":      "Missing requirement",
		"no":           "Absent skill",
		"without":      "Without required skill",
		"limited":      "Limited experience",
		"weak":         "Weak area",
		"gap":          "Skill gap",
		"insufficient": "Insufficient experience",
		"not":          "Missing qualification",
		"unfamiliar":   "Unfamiliar technology",
		"beginner":     "Junior level",
		"junior":       "Junior experience",
		"basic":        "Basic knowledge only",
		"minimal":      "Minimal exposure",
		"none":         "No experience",
		"never":        "Never worked with",
	}

	var strengths, gaps []string
	seenStrengths := make(map[string]struct{})
	seenGaps := make(map[string]struct{})

	for kw, label := range strengthKeywords {
		if strings.Contains(reasoningLower, kw) {
			if _, exists := seenStrengths[label]; !exists {
				seenStrengths[label] = struct{}{}
				strengths = append(strengths, label)
			}
		}
	}

	for kw, label := range gapKeywords {
		if strings.Contains(reasoningLower, kw) {
			if _, exists := seenGaps[label]; !exists {
				seenGaps[label] = struct{}{}
				gaps = append(gaps, label)
			}
		}
	}

	// Limit to top 5 each
	if len(strengths) > 5 {
		strengths = strengths[:5]
	}
	if len(gaps) > 5 {
		gaps = gaps[:5]
	}

	return strengths, gaps
}

// calculateConfidence returns a confidence score based on score and source.
func calculateConfidence(score float64, source string) float64 {
	// Higher confidence for LLM scores, lower for heuristic
	base := 0.7
	switch source {
	case "llm":
		base = 0.85
	case "hybrid":
		base = 0.9
	}
	// Adjust by score distance from thresholds
	if score >= 90 || score <= 30 {
		base += 0.1 // very confident at extremes
	}
	if base > 1.0 {
		base = 1.0
	}
	return base
}
