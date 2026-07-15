package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

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

		log.Info("scraping source",
			zap.String("source_id", payload.SourceID.String()),
			zap.String("correlation_id", payload.CorrelationID.String()),
			zap.Strings("keywords", payload.Keywords),
		)

		resp, err := browserClient.ScrapeJobs(ctx, ScrapeJobsRequest{
			SourceID: payload.SourceID.String(),
			Location: payload.Location,
			Keywords: payload.Keywords,
		})
		if err != nil {
			log.Error("scrape jobs",
				zap.String("source_id", payload.SourceID.String()),
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
func newHandleScoring(svc *scoring.Service, logger *zap.Logger, taskSvc *tasks.Service) asynq.HandlerFunc {
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
