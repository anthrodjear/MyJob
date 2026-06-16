package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"backend/internal/jobs"
	"backend/internal/resumes"
	"backend/internal/tasks"
)

// newHandleGenerateResume processes resume_generate tasks.
func newHandleGenerateResume(
	resumesSvc *resumes.Service,
	jobsSvc *jobs.Service,
	logger *zap.Logger,
) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		log := logger.Named("task.resume_generate")

		var payload tasks.ResumeGeneratePayload
		if err := json.Unmarshal(t.Payload(), &payload); err != nil {
			log.Error("unmarshal payload", zap.String("task_type", t.Type()), zap.Error(err))
			return fmt.Errorf("resume_generate: unmarshal payload: %w", err)
		}

		ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()

		log.Info("generating resume",
			zap.String("job_id", payload.JobID.String()),
			zap.String("correlation_id", payload.CorrelationID.String()),
		)

		job, err := jobsSvc.GetByID(ctx, payload.JobID)
		if err != nil {
			if errors.Is(err, jobs.ErrNotFound) {
				log.Warn("job not found, skipping", zap.String("job_id", payload.JobID.String()))
				return nil
			}
			log.Error("fetch job", zap.String("job_id", payload.JobID.String()), zap.Error(err))
			return fmt.Errorf("resume_generate: fetch job %s: %w", payload.JobID, err)
		}

		resumeList, _, err := resumesSvc.List(ctx, 1, 0)
		if err != nil {
			log.Error("list resumes", zap.Error(err))
			return fmt.Errorf("resume_generate: list resumes: %w", err)
		}
		if len(resumeList) == 0 {
			log.Warn("no resumes found, skipping")
			return nil
		}
		resumeID := resumeList[0].ID

		_, version, err := resumesSvc.GenerateContent(ctx, resumeID, resumes.GenerateResumeContentRequest{
			JobID: &payload.JobID, JobTitle: job.Title, JobRequirements: job.Requirements,
		})
		if err != nil {
			log.Error("generate content",
				zap.String("job_id", payload.JobID.String()),
				zap.String("resume_id", resumeID.String()),
				zap.Error(err),
			)
			return fmt.Errorf("resume_generate: generate for job %s: %w", payload.JobID, err)
		}

		log.Info("resume generated",
			zap.String("job_id", payload.JobID.String()),
			zap.String("resume_id", resumeID.String()),
			zap.String("correlation_id", payload.CorrelationID.String()),
			zap.Int32("version", version),
		)

		return nil
	}
}

// newHandleGenerateCoverLetter processes cover_letter_gen tasks.
func newHandleGenerateCoverLetter(
	resumesSvc *resumes.Service,
	jobsSvc *jobs.Service,
	logger *zap.Logger,
) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		log := logger.Named("task.cover_letter_gen")

		var payload tasks.CoverLetterGenPayload
		if err := json.Unmarshal(t.Payload(), &payload); err != nil {
			log.Error("unmarshal payload", zap.String("task_type", t.Type()), zap.Error(err))
			return fmt.Errorf("cover_letter_gen: unmarshal payload: %w", err)
		}

		ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()

		log.Info("generating cover letter",
			zap.String("cover_letter_id", payload.CoverLetterID.String()),
			zap.String("correlation_id", payload.CorrelationID.String()),
		)

		cl, err := resumesSvc.GetCoverLetterByID(ctx, payload.CoverLetterID)
		if err != nil {
			if errors.Is(err, resumes.ErrNotFound) {
				log.Warn("cover letter not found, skipping")
				return nil
			}
			log.Error("fetch cover letter", zap.Error(err))
			return fmt.Errorf("cover_letter_gen: fetch %s: %w", payload.CoverLetterID, err)
		}

		if cl.JobID == nil {
			log.Warn("cover letter has no job ID, skipping")
			return nil
		}

		job, err := jobsSvc.GetByID(ctx, *cl.JobID)
		if err != nil {
			if errors.Is(err, jobs.ErrNotFound) {
				log.Warn("job not found, skipping", zap.String("job_id", cl.JobID.String()))
				return nil
			}
			log.Error("fetch job", zap.String("job_id", cl.JobID.String()), zap.Error(err))
			return fmt.Errorf("cover_letter_gen: fetch job %s: %w", cl.JobID, err)
		}

		_, err = resumesSvc.GenerateCoverLetter(ctx, payload.CoverLetterID, resumes.GenerateCoverLetterRequest{
			JobTitle: job.Title, JobRequirements: job.Requirements,
			JobDescription: job.Description, ResumeID: cl.ResumeID,
		})
		if err != nil {
			log.Error("generate cover letter", zap.Error(err))
			return fmt.Errorf("cover_letter_gen: generate %s: %w", payload.CoverLetterID, err)
		}

		log.Info("cover letter generated",
			zap.String("cover_letter_id", payload.CoverLetterID.String()),
			zap.String("correlation_id", payload.CorrelationID.String()),
			zap.Int32("version", cl.Version),
		)

		return nil
	}
}
