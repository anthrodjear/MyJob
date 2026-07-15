package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"backend/internal/activity"
	"backend/internal/applications"
	"backend/internal/approvals"
	"backend/internal/jobs"
	"backend/internal/scoring"
	"backend/internal/tasks"
)

// taskIDFromTask extracts the DB task UUID from the asynq task.
// The dispatcher sets the asynq task ID to the DB task UUID via asynq.TaskID,
// so we can retrieve it from the ResultWriter.
func taskIDFromTask(t *asynq.Task) string {
	return t.ResultWriter().TaskID()
}

// newHandleScrapeSource processes job_discovery tasks.
// Flow: parse payload → call browser agent → store jobs.
func newHandleScrapeSource(
	jobsSvc *jobs.Service,
	_ *scoring.Service, // reserved for future per-job scoring dispatch
	browserClient BrowserAgentClient,
	logger *zap.Logger,
	taskSvc *tasks.Service,
) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		log := logger.Named("task.discovery")
		taskID := taskIDFromTask(t)

		// Mark task as running
		if taskID != "" {
			if _, err := taskSvc.Start(ctx, parseUUID(taskID)); err != nil {
				log.Warn("failed to mark task as running", zap.String("task_id", taskID), zap.Error(err))
			}
		}

		var payload tasks.JobDiscoveryPayload
		if err := json.Unmarshal(t.Payload(), &payload); err != nil {
			log.Error("unmarshal payload", zap.String("task_type", t.Type()), zap.Error(err))
			if taskID != "" {
				if _, err := taskSvc.Fail(ctx, parseUUID(taskID), fmt.Sprintf("unmarshal payload: %v", err)); err != nil {
					log.Warn("failed to mark task as failed", zap.Error(err))
				}
			}
			return fmt.Errorf("discovery: unmarshal payload: %w", err)
		}

		ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()

		// Resolve source UUID to source name for browser-agent.
		// The browser-agent matches against YAML config name/type fields
		// (e.g. "greenhouse"), not database UUIDs.
		sourceName, err := jobsSvc.GetSourceNameByID(ctx, payload.SourceID)
		if err != nil {
			log.Error("resolve source name",
				zap.String("source_id", payload.SourceID.String()),
				zap.Error(err),
			)
			if taskID != "" {
				if _, failErr := taskSvc.Fail(ctx, parseUUID(taskID), fmt.Sprintf("resolve source %s: %v", payload.SourceID, err)); failErr != nil {
					log.Warn("failed to mark task as failed", zap.Error(failErr))
				}
			}
			return fmt.Errorf("discovery: resolve source %s: %w", payload.SourceID, err)
		}

		log.Info("scraping source",
			zap.String("source_id", payload.SourceID.String()),
			zap.String("source_name", sourceName),
			zap.String("correlation_id", payload.CorrelationID.String()),
			zap.Strings("keywords", payload.Keywords),
		)

		resp, err := browserClient.ScrapeJobs(ctx, ScrapeJobsRequest{
			SourceID: sourceName,
			Location: payload.Location,
			Keywords: payload.Keywords,
		})
		if err != nil {
			log.Error("scrape jobs",
				zap.String("source_id", payload.SourceID.String()),
				zap.String("source_name", sourceName),
				zap.String("correlation_id", payload.CorrelationID.String()),
				zap.Error(err),
			)
			if taskID != "" {
				if _, failErr := taskSvc.Fail(ctx, parseUUID(taskID), fmt.Sprintf("scrape source %s: %v", payload.SourceID, err)); failErr != nil {
					log.Warn("failed to mark task as failed", zap.Error(failErr))
				}
			}
			return fmt.Errorf("discovery: scrape source %s: %w", payload.SourceID, err)
		}

		inputs := make([]jobs.CreateJobInput, 0, len(resp.Jobs))
		for _, s := range resp.Jobs {
			inputs = append(inputs, jobs.CreateJobInput{
				SourceID: payload.SourceID, ExternalID: s.ExternalID,
				Title: s.Title, Company: s.Company, Location: s.Location,
				RemoteType: s.RemoteType, SalaryMin: s.SalaryMin, SalaryMax: s.SalaryMax,
				SalaryCurrency: s.SalaryCurrency, Description: s.Description,
				Requirements: s.Requirements, URL: s.URL, ApplicationURL: s.ApplicationURL,
				CompanyURL: s.CompanyURL, Source: s.Source,
			})
		}

		importCtx, importCancel := context.WithTimeout(ctx, 30*time.Second)
		defer importCancel()

		result, err := jobsSvc.BulkImport(importCtx, inputs)
		if err != nil {
			log.Error("bulk import",
				zap.String("source_id", payload.SourceID.String()),
				zap.String("correlation_id", payload.CorrelationID.String()),
				zap.Error(err),
			)
			if taskID != "" {
				if _, failErr := taskSvc.Fail(ctx, parseUUID(taskID), fmt.Sprintf("bulk import source %s: %v", payload.SourceID, err)); failErr != nil {
					log.Warn("failed to mark task as failed", zap.Error(failErr))
				}
			}
			return fmt.Errorf("discovery: bulk import source %s: %w", payload.SourceID, err)
		}

		log.Info("jobs imported",
			zap.String("source_id", payload.SourceID.String()),
			zap.String("correlation_id", payload.CorrelationID.String()),
			zap.Int("imported", result.Imported),
			zap.Int("skipped", result.Skipped),
		)

		if len(resp.Errors) > 0 {
			log.Warn("scraping errors",
				zap.String("source_id", payload.SourceID.String()),
				zap.Strings("errors", resp.Errors),
			)
		}

		// Mark task as completed
		if taskID != "" {
			resultJSON, _ := json.Marshal(map[string]interface{}{
				"imported": result.Imported,
				"skipped":  result.Skipped,
			})
			if _, err := taskSvc.Complete(ctx, parseUUID(taskID), resultJSON); err != nil {
				log.Warn("failed to mark task as completed", zap.String("task_id", taskID), zap.Error(err))
			}
		}

		return nil
	}
}

// newHandleScoring processes job_scoring tasks.
// After scoring, it chains: apply match score → log activity → create approval (if review tier).
func newHandleScoring(
	svc *scoring.Service,
	jobsSvc *jobs.Service,
	activitySvc *activity.Service,
	approvalsSvc *approvals.Service,
	appsSvc *applications.Service,
	logger *zap.Logger,
	taskSvc *tasks.Service,
) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		log := logger.Named("task.scoring")
		taskID := taskIDFromTask(t)

		// Mark task as running
		if taskID != "" {
			if _, err := taskSvc.Start(ctx, parseUUID(taskID)); err != nil {
				log.Warn("failed to mark task as running", zap.String("task_id", taskID), zap.Error(err))
			}
		}

		var payload tasks.JobScoringPayload
		if err := json.Unmarshal(t.Payload(), &payload); err != nil {
			log.Error("unmarshal payload", zap.Error(err))
			if taskID != "" {
				if _, err := taskSvc.Fail(ctx, parseUUID(taskID), fmt.Sprintf("unmarshal payload: %v", err)); err != nil {
					log.Warn("failed to mark task as failed", zap.Error(err))
				}
			}
			return fmt.Errorf("scoring: unmarshal payload: %w", err)
		}

		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		log.Info("scoring job",
			zap.String("job_id", payload.JobID.String()),
			zap.String("correlation_id", payload.CorrelationID.String()),
		)

		result, err := svc.ScoreJob(ctx, payload.JobID)
		if err != nil {
			log.Error("score job",
				zap.String("job_id", payload.JobID.String()),
				zap.String("correlation_id", payload.CorrelationID.String()),
				zap.Error(err),
			)
			if taskID != "" {
				if _, failErr := taskSvc.Fail(ctx, parseUUID(taskID), fmt.Sprintf("score job %s: %v", payload.JobID, err)); failErr != nil {
					log.Warn("failed to mark task as failed", zap.Error(failErr))
				}
			}
			return fmt.Errorf("scoring: score job %s: %w", payload.JobID, err)
		}

		log.Info("job scored",
			zap.String("job_id", payload.JobID.String()),
			zap.String("correlation_id", payload.CorrelationID.String()),
			zap.Float64("score", result.Score),
			zap.String("tier", string(result.Tier)),
			zap.String("source", result.Source),
		)

		// --- Chain: score → job status → activity → approval ---

		// 1. Apply match score to job (transitions discovered → matched if auto tier)
		var detailsJSON json.RawMessage
		if result.Details != nil {
			detailsJSON, _ = json.Marshal(result.Details)
		}
		if err := jobsSvc.ApplyMatchScore(ctx, payload.JobID, result.Score, detailsJSON); err != nil {
			log.Warn("failed to apply match score", zap.Error(err))
		}

		// 2. Log scoring activity event
		if err := activitySvc.LogEvent(ctx, activity.EventJobScored, "job", payload.JobID, activity.Details{
			"score":  result.Score,
			"tier":   string(result.Tier),
			"source": result.Source,
		}); err != nil {
			log.Warn("failed to log scoring activity", zap.Error(err))
		}

		// 3. If review tier → create application + approval request
		if result.Tier == scoring.TierReview {
			job, err := jobsSvc.GetByID(ctx, payload.JobID)
			if err != nil {
				log.Warn("failed to get job for approval", zap.Error(err))
			} else {
				// Create a draft application to anchor the approval request
				app, err := appsSvc.Create(ctx, applications.CreateApplicationRequest{
					JobID: payload.JobID,
				})
				if err != nil {
					log.Warn("failed to create application for approval", zap.Error(err))
				} else {
					var requirements []string
				if trimmed := strings.TrimSpace(job.Requirements); trimmed != "" {
					requirements = strings.Split(trimmed, "\n")
				}
					approval := &approvals.ApprovalRequest{
						ApplicationID: app.ID,
						Status:        approvals.ApprovalStatusPending,
						JobSnapshot: approvals.JobSnapshot{
							Title:        job.Title,
							Company:      job.Company,
							Location:     job.Location,
							URL:          job.URL,
							Description:  job.Description,
							Requirements: requirements,
							Score:        result.Score,
							Tier:         string(result.Tier),
							ScoredAt:     time.Now().Format(time.RFC3339),
						},
					}
					if err := approvalsSvc.Create(ctx, approval); err != nil {
						log.Warn("failed to create approval request", zap.Error(err))
					} else {
						// Log approval requested activity
						if err := activitySvc.LogEvent(ctx, activity.EventApprovalRequested, "application", app.ID, activity.Details{
							"job_id": payload.JobID.String(),
							"score":  result.Score,
							"tier":   string(result.Tier),
						}); err != nil {
							log.Warn("failed to log approval activity", zap.Error(err))
						}
					}
				}
			}
		}

		// Mark task as completed
		if taskID != "" {
			resultJSON, _ := json.Marshal(map[string]interface{}{
				"score":  result.Score,
				"tier":   result.Tier,
				"source": result.Source,
			})
			if _, err := taskSvc.Complete(ctx, parseUUID(taskID), resultJSON); err != nil {
				log.Warn("failed to mark task as completed", zap.String("task_id", taskID), zap.Error(err))
			}
		}

		return nil
	}
}
