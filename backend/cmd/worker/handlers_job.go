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

// newHandleScrapeSource processes job_discovery tasks.
// Flow: parse payload → call browser agent → store jobs.
func newHandleScrapeSource(
	jobsSvc *jobs.Service,
	_ *scoring.Service, // reserved for future per-job scoring dispatch
	browserClient BrowserAgentClient,
	logger *zap.Logger,
) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		log := logger.Named("task.discovery")

		var payload tasks.JobDiscoveryPayload
		if err := json.Unmarshal(t.Payload(), &payload); err != nil {
			log.Error("unmarshal payload", zap.String("task_type", t.Type()), zap.Error(err))
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

		return nil
	}
}

// newHandleScoring processes job_scoring tasks.
func newHandleScoring(svc *scoring.Service, logger *zap.Logger) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		log := logger.Named("task.scoring")

		var payload tasks.JobScoringPayload
		if err := json.Unmarshal(t.Payload(), &payload); err != nil {
			log.Error("unmarshal payload", zap.Error(err))
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
			return fmt.Errorf("scoring: score job %s: %w", payload.JobID, err)
		}

		log.Info("job scored",
			zap.String("job_id", payload.JobID.String()),
			zap.String("correlation_id", payload.CorrelationID.String()),
			zap.Float64("score", result.Score),
			zap.String("tier", string(result.Tier)),
			zap.String("source", result.Source),
		)

		return nil
	}
}
