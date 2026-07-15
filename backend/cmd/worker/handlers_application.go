package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"backend/internal/applications"
	"backend/internal/config"
	"backend/internal/embeddings"
	"backend/internal/jobs"
	"backend/internal/pgvector"
	"backend/internal/tasks"
)

// parseUUID is a helper that parses a string into a UUID.
// Returns uuid.Nil on parse failure so callers can still proceed gracefully.
func parseUUID(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil
	}
	return id
}

// newHandleFillForm processes fill_form tasks.
func newHandleFillForm(browserClient BrowserAgentClient, logger *zap.Logger, taskSvc *tasks.Service) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		log := logger.Named("task.fill_form")
		taskID := taskIDFromTask(t)

		// Mark task as running
		if taskID != "" {
			if _, err := taskSvc.Start(ctx, parseUUID(taskID)); err != nil {
				log.Warn("failed to mark task as running", zap.String("task_id", taskID), zap.Error(err))
			}
		}

		var payload tasks.FillFormPayload
		if err := json.Unmarshal(t.Payload(), &payload); err != nil {
			log.Error("unmarshal payload", zap.String("task_type", t.Type()), zap.Error(err))
			if taskID != "" {
				if _, err := taskSvc.Fail(ctx, parseUUID(taskID), fmt.Sprintf("unmarshal payload: %v", err)); err != nil {
					log.Warn("failed to mark task as failed", zap.Error(err))
				}
			}
			return fmt.Errorf("fill_form: unmarshal payload: %w", err)
		}

		ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()

		log.Info("filling form", zap.String("portal_url", payload.PortalURL))

		resp, err := browserClient.FillForm(ctx, FillFormRequest{
			PortalURL: payload.PortalURL, PortalType: payload.PortalType,
			FormData: payload.FormData, ResumePath: payload.ResumePath,
		})
		if err != nil {
			log.Error("fill form", zap.String("portal_url", payload.PortalURL), zap.Error(err))
			if taskID != "" {
				if _, failErr := taskSvc.Fail(ctx, parseUUID(taskID), fmt.Sprintf("%s: %v", payload.PortalURL, err)); failErr != nil {
					log.Warn("failed to mark task as failed", zap.Error(failErr))
				}
			}
			return fmt.Errorf("fill_form: %s: %w", payload.PortalURL, err)
		}

		log.Info("form filled", zap.String("portal_url", payload.PortalURL), zap.Bool("success", resp.Success))

		// Mark task as completed
		if taskID != "" {
			resultJSON, _ := json.Marshal(map[string]interface{}{
				"success": resp.Success,
			})
			if _, err := taskSvc.Complete(ctx, parseUUID(taskID), resultJSON); err != nil {
				log.Warn("failed to mark task as completed", zap.String("task_id", taskID), zap.Error(err))
			}
		}

		return nil
	}
}

// newHandleSubmitApplication processes application_submit tasks.
func newHandleSubmitApplication(
	applicationsSvc *applications.Service,
	jobsSvc *jobs.Service,
	browserClient BrowserAgentClient,
	logger *zap.Logger,
	taskSvc *tasks.Service,
) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		log := logger.Named("task.application_submit")
		taskID := taskIDFromTask(t)

		// Mark task as running
		if taskID != "" {
			if _, err := taskSvc.Start(ctx, parseUUID(taskID)); err != nil {
				log.Warn("failed to mark task as running", zap.String("task_id", taskID), zap.Error(err))
			}
		}

		var payload tasks.ApplicationSubmitPayload
		if err := json.Unmarshal(t.Payload(), &payload); err != nil {
			log.Error("unmarshal payload", zap.String("task_type", t.Type()), zap.Error(err))
			if taskID != "" {
				if _, err := taskSvc.Fail(ctx, parseUUID(taskID), fmt.Sprintf("unmarshal payload: %v", err)); err != nil {
					log.Warn("failed to mark task as failed", zap.Error(err))
				}
			}
			return fmt.Errorf("application_submit: unmarshal payload: %w", err)
		}

		ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		defer cancel()

		log.Info("submitting application",
			zap.String("application_id", payload.ApplicationID.String()),
			zap.String("correlation_id", payload.CorrelationID.String()),
		)

		app, err := applicationsSvc.GetByID(ctx, payload.ApplicationID)
		if err != nil {
			if errors.Is(err, applications.ErrNotFound) {
				log.Warn("application not found, skipping")
				if taskID != "" {
					if _, cerr := taskSvc.Complete(ctx, parseUUID(taskID), nil); cerr != nil {
						log.Warn("failed to mark task as completed", zap.Error(cerr))
					}
				}
				return nil
			}
			log.Error("fetch application", zap.Error(err))
			if taskID != "" {
				if _, failErr := taskSvc.Fail(ctx, parseUUID(taskID), fmt.Sprintf("fetch %s: %v", payload.ApplicationID, err)); failErr != nil {
					log.Warn("failed to mark task as failed", zap.Error(failErr))
				}
			}
			return fmt.Errorf("application_submit: fetch %s: %w", payload.ApplicationID, err)
		}

		job, err := jobsSvc.GetByID(ctx, app.JobID)
		if err != nil {
			if errors.Is(err, jobs.ErrNotFound) {
				log.Warn("job not found, skipping", zap.String("job_id", app.JobID.String()))
				if taskID != "" {
					if _, cerr := taskSvc.Complete(ctx, parseUUID(taskID), nil); cerr != nil {
						log.Warn("failed to mark task as completed", zap.Error(cerr))
					}
				}
				return nil
			}
			log.Error("fetch job", zap.String("job_id", app.JobID.String()), zap.Error(err))
			if taskID != "" {
				if _, failErr := taskSvc.Fail(ctx, parseUUID(taskID), fmt.Sprintf("fetch job %s: %v", app.JobID, err)); failErr != nil {
					log.Warn("failed to mark task as failed", zap.Error(failErr))
				}
			}
			return fmt.Errorf("application_submit: fetch job %s: %w", app.JobID, err)
		}

		portalURL := job.URL
		if app.PortalURL != nil && *app.PortalURL != "" {
			portalURL = *app.PortalURL
		}
		portalType := "greenhouse"
		if app.PortalType != nil && *app.PortalType != "" {
			portalType = *app.PortalType
		}

		// Use application's stored form data (the task payload may have nil FormData
		// when dispatched via the approve→submit workflow adapter).
		var formData map[string]string
		if len(app.FormData) > 0 {
			if err := json.Unmarshal(app.FormData, &formData); err != nil {
				log.Warn("invalid form data on application", zap.Error(err))
				formData = nil
			}
		}

		resp, err := browserClient.FillForm(ctx, FillFormRequest{
			PortalURL: portalURL, PortalType: portalType, FormData: formData,
		})
		if err != nil {
			log.Error("fill form", zap.String("portal_url", portalURL), zap.Error(err))
			if taskID != "" {
				if _, failErr := taskSvc.Fail(ctx, parseUUID(taskID), fmt.Sprintf("fill form %s: %v", payload.ApplicationID, err)); failErr != nil {
					log.Warn("failed to mark task as failed", zap.Error(failErr))
				}
			}
			return fmt.Errorf("application_submit: fill form %s: %w", payload.ApplicationID, err)
		}
		if !resp.Success {
			log.Warn("form fill reported failure, skipping retry",
				zap.String("application_id", payload.ApplicationID.String()),
				zap.String("portal_url", portalURL),
				zap.String("message", resp.Message),
			)
			// Mark task as completed (business-level failure, not a task error)
			if taskID != "" {
				resultJSON, _ := json.Marshal(map[string]interface{}{
					"success": false,
					"message": resp.Message,
				})
				if _, err := taskSvc.Complete(ctx, parseUUID(taskID), resultJSON); err != nil {
					log.Warn("failed to mark task as completed", zap.String("task_id", taskID), zap.Error(err))
				}
			}
			return nil // business-level failure, not retryable
		}

		if err := applicationsSvc.UpdateStatus(ctx, payload.ApplicationID, applications.StatusApplied, "Submitted via browser agent"); err != nil {
			log.Error("update status", zap.Error(err))
			if taskID != "" {
				if _, failErr := taskSvc.Fail(ctx, parseUUID(taskID), fmt.Sprintf("update status %s: %v", payload.ApplicationID, err)); failErr != nil {
					log.Warn("failed to mark task as failed", zap.Error(failErr))
				}
			}
			return fmt.Errorf("application_submit: update status %s: %w", payload.ApplicationID, err)
		}

		log.Info("application submitted",
			zap.String("application_id", payload.ApplicationID.String()),
			zap.String("portal_url", portalURL),
		)

		// Mark task as completed
		if taskID != "" {
			resultJSON, _ := json.Marshal(map[string]interface{}{
				"application_id": payload.ApplicationID.String(),
				"portal_url":     portalURL,
			})
			if _, err := taskSvc.Complete(ctx, parseUUID(taskID), resultJSON); err != nil {
				log.Warn("failed to mark task as completed", zap.String("task_id", taskID), zap.Error(err))
			}
		}

		return nil
	}
}

// newHandleSyncEmails processes email_check tasks.
func newHandleSyncEmails(
	applicationsSvc *applications.Service,
	browserClient BrowserAgentClient,
	emailCfg config.EmailConfig,
	logger *zap.Logger,
	taskSvc *tasks.Service,
) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		log := logger.Named("task.email_check")
		taskID := taskIDFromTask(t)

		// Mark task as running
		if taskID != "" {
			if _, err := taskSvc.Start(ctx, parseUUID(taskID)); err != nil {
				log.Warn("failed to mark task as running", zap.String("task_id", taskID), zap.Error(err))
			}
		}

		var payload tasks.EmailCheckPayload
		if err := json.Unmarshal(t.Payload(), &payload); err != nil {
			log.Error("unmarshal payload", zap.String("task_type", t.Type()), zap.Error(err))
			if taskID != "" {
				if _, err := taskSvc.Fail(ctx, parseUUID(taskID), fmt.Sprintf("unmarshal payload: %v", err)); err != nil {
					log.Warn("failed to mark task as failed", zap.Error(err))
				}
			}
			return fmt.Errorf("email_check: unmarshal payload: %w", err)
		}

		ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()

		log.Info("checking emails",
			zap.String("application_id", payload.ApplicationID.String()),
			zap.String("correlation_id", payload.CorrelationID.String()),
		)

		_, err := applicationsSvc.GetByID(ctx, payload.ApplicationID)
		if err != nil {
			if errors.Is(err, applications.ErrNotFound) {
				log.Warn("application not found, skipping")
				if taskID != "" {
					if _, cerr := taskSvc.Complete(ctx, parseUUID(taskID), nil); cerr != nil {
						log.Warn("failed to mark task as completed", zap.Error(cerr))
					}
				}
				return nil
			}
			log.Error("fetch application", zap.Error(err))
			if taskID != "" {
				if _, failErr := taskSvc.Fail(ctx, parseUUID(taskID), fmt.Sprintf("fetch %s: %v", payload.ApplicationID, err)); failErr != nil {
					log.Warn("failed to mark task as failed", zap.Error(failErr))
				}
			}
			return fmt.Errorf("email_check: fetch %s: %w", payload.ApplicationID, err)
		}

		if emailCfg.Provider == "" || emailCfg.TenantID == "" {
			log.Warn("email provider not configured, skipping")
			if taskID != "" {
				if _, cerr := taskSvc.Complete(ctx, parseUUID(taskID), nil); cerr != nil {
					log.Warn("failed to mark task as completed", zap.Error(cerr))
				}
			}
			return nil
		}

		resp, err := browserClient.CheckEmails(ctx, CheckEmailsRequest{
			TenantID: emailCfg.TenantID, ClientID: emailCfg.ClientID,
			ClientSecret: emailCfg.ClientSecret, Folders: emailCfg.Folders,
			ApplicationID: payload.ApplicationID.String(),
		})
		if err != nil {
			log.Error("check emails", zap.Error(err))
			if taskID != "" {
				if _, failErr := taskSvc.Fail(ctx, parseUUID(taskID), fmt.Sprintf("%s: %v", payload.ApplicationID, err)); failErr != nil {
					log.Warn("failed to mark task as failed", zap.Error(failErr))
				}
			}
			return fmt.Errorf("email_check: %s: %w", payload.ApplicationID, err)
		}

		for _, email := range resp.Emails {
			log.Info("email received",
				zap.String("application_id", payload.ApplicationID.String()),
				zap.String("from", email.From),
				zap.String("subject", email.Subject),
				zap.String("classification", email.Classification),
				zap.String("correlation_id", payload.CorrelationID.String()),
			)

			var statusErr error
			switch email.Classification {
			case "rejection":
				statusErr = applicationsSvc.UpdateStatus(ctx, payload.ApplicationID, applications.StatusRejected, "Rejected: "+email.Subject)
			case "interview":
				statusErr = applicationsSvc.UpdateStatus(ctx, payload.ApplicationID, applications.StatusPhoneScreen, "Interview: "+email.Subject)
			case "offer":
				statusErr = applicationsSvc.UpdateStatus(ctx, payload.ApplicationID, applications.StatusOffer, "Offer: "+email.Subject)
			}
			if statusErr != nil {
				log.Error("update application status",
					zap.String("application_id", payload.ApplicationID.String()),
					zap.String("classification", email.Classification),
					zap.String("correlation_id", payload.CorrelationID.String()),
					zap.Error(statusErr),
				)
				if taskID != "" {
					if _, failErr := taskSvc.Fail(ctx, parseUUID(taskID), fmt.Sprintf("update status %s: %v", payload.ApplicationID, statusErr)); failErr != nil {
						log.Warn("failed to mark task as failed", zap.Error(failErr))
					}
				}
				return fmt.Errorf("email_check: update status %s: %w", payload.ApplicationID, statusErr)
			}
		}

		log.Info("email check complete",
			zap.String("application_id", payload.ApplicationID.String()),
			zap.Int("emails_found", len(resp.Emails)),
		)

		// Mark task as completed
		if taskID != "" {
			resultJSON, _ := json.Marshal(map[string]interface{}{
				"emails_found": len(resp.Emails),
			})
			if _, err := taskSvc.Complete(ctx, parseUUID(taskID), resultJSON); err != nil {
				log.Warn("failed to mark task as completed", zap.String("task_id", taskID), zap.Error(err))
			}
		}

		return nil
	}
}

// newHandleGenerateInterviewPrep processes interview_prep tasks.
func newHandleGenerateInterviewPrep(
	applicationsSvc *applications.Service,
	jobsSvc *jobs.Service,
	logger *zap.Logger,
	taskSvc *tasks.Service,
) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		log := logger.Named("task.interview_prep")
		taskID := taskIDFromTask(t)

		// Mark task as running
		if taskID != "" {
			if _, err := taskSvc.Start(ctx, parseUUID(taskID)); err != nil {
				log.Warn("failed to mark task as running", zap.String("task_id", taskID), zap.Error(err))
			}
		}

		var payload tasks.InterviewPrepPayload
		if err := json.Unmarshal(t.Payload(), &payload); err != nil {
			log.Error("unmarshal payload", zap.String("task_type", t.Type()), zap.Error(err))
			if taskID != "" {
				if _, err := taskSvc.Fail(ctx, parseUUID(taskID), fmt.Sprintf("unmarshal payload: %v", err)); err != nil {
					log.Warn("failed to mark task as failed", zap.Error(err))
				}
			}
			return fmt.Errorf("interview_prep: unmarshal payload: %w", err)
		}

		ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()

		log.Info("generating interview prep",
			zap.String("application_id", payload.ApplicationID.String()),
			zap.String("correlation_id", payload.CorrelationID.String()),
		)

		app, err := applicationsSvc.GetByID(ctx, payload.ApplicationID)
		if err != nil {
			if errors.Is(err, applications.ErrNotFound) {
				log.Warn("application not found, skipping")
				if taskID != "" {
					if _, cerr := taskSvc.Complete(ctx, parseUUID(taskID), nil); cerr != nil {
						log.Warn("failed to mark task as completed", zap.Error(cerr))
					}
				}
				return nil
			}
			log.Error("fetch application", zap.Error(err))
			if taskID != "" {
				if _, failErr := taskSvc.Fail(ctx, parseUUID(taskID), fmt.Sprintf("fetch %s: %v", payload.ApplicationID, err)); failErr != nil {
					log.Warn("failed to mark task as failed", zap.Error(failErr))
				}
			}
			return fmt.Errorf("interview_prep: fetch %s: %w", payload.ApplicationID, err)
		}

		job, err := jobsSvc.GetByID(ctx, app.JobID)
		if err != nil {
			if errors.Is(err, jobs.ErrNotFound) {
				log.Warn("job not found, skipping", zap.String("job_id", app.JobID.String()))
				if taskID != "" {
					if _, cerr := taskSvc.Complete(ctx, parseUUID(taskID), nil); cerr != nil {
						log.Warn("failed to mark task as completed", zap.Error(cerr))
					}
				}
				return nil
			}
			log.Error("fetch job", zap.Error(err))
			if taskID != "" {
				if _, failErr := taskSvc.Fail(ctx, parseUUID(taskID), fmt.Sprintf("fetch job %s: %v", app.JobID, err)); failErr != nil {
					log.Warn("failed to mark task as failed", zap.Error(failErr))
				}
			}
			return fmt.Errorf("interview_prep: fetch job %s: %w", app.JobID, err)
		}

		log.Info("interview prep generated",
			zap.String("application_id", payload.ApplicationID.String()),
			zap.String("job_title", job.Title),
			zap.String("company", job.Company),
			zap.String("status", "placeholder — LLM generation pending"),
		)

		// Mark task as completed
		if taskID != "" {
			resultJSON, _ := json.Marshal(map[string]interface{}{
				"job_title": job.Title,
				"company":   job.Company,
			})
			if _, err := taskSvc.Complete(ctx, parseUUID(taskID), resultJSON); err != nil {
				log.Warn("failed to mark task as completed", zap.String("task_id", taskID), zap.Error(err))
			}
		}

		return nil
	}
}

// newHandleCreateEmbeddings processes embedding_generate tasks.
// It calls the configured embeddings provider, then stores the resulting
// vector in the embeddings table for semantic search (RAG).
func newHandleCreateEmbeddings(
	embeddingClient embeddings.EmbeddingClient,
	db *sqlx.DB,
	logger *zap.Logger,
	taskSvc *tasks.Service,
) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		log := logger.Named("task.embedding_generate")
		taskID := taskIDFromTask(t)

		// Mark task as running
		if taskID != "" {
			if _, err := taskSvc.Start(ctx, parseUUID(taskID)); err != nil {
				log.Warn("failed to mark task as running", zap.String("task_id", taskID), zap.Error(err))
			}
		}

		var payload tasks.EmbeddingPayload
		if err := json.Unmarshal(t.Payload(), &payload); err != nil {
			log.Error("unmarshal payload", zap.String("task_type", t.Type()), zap.Error(err))
			if taskID != "" {
				if _, err := taskSvc.Fail(ctx, parseUUID(taskID), fmt.Sprintf("unmarshal payload: %v", err)); err != nil {
					log.Warn("failed to mark task as failed", zap.Error(err))
				}
			}
			return fmt.Errorf("embedding_generate: unmarshal payload: %w", err)
		}

		ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()

		log.Info("generating embedding",
			zap.String("source_type", payload.SourceType),
			zap.String("source_id", payload.SourceID.String()),
			zap.Int("content_length", len(payload.Content)),
		)

		vec, err := embeddingClient.Embed(ctx, payload.Content)
		if err != nil {
			log.Error("generate embedding",
				zap.String("source_type", payload.SourceType),
				zap.String("source_id", payload.SourceID.String()),
				zap.Error(err),
			)
			if taskID != "" {
				if _, failErr := taskSvc.Fail(ctx, parseUUID(taskID), fmt.Sprintf("embed %s/%s: %v", payload.SourceType, payload.SourceID, err)); failErr != nil {
					log.Warn("failed to mark task as failed", zap.Error(failErr))
				}
			}
			return fmt.Errorf("embedding_generate: embed %s/%s: %w", payload.SourceType, payload.SourceID, err)
		}

		// Convert []float32 → pgvector string format: "[1.0,2.0,3.0,...]"
		vecStr := pgvector.FormatVector(vec)

		query := `
			INSERT INTO embeddings (id, source_type, source_id, content, embedding, created_at)
			VALUES (uuid_generate_v4(), $1, $2, $3, $4::vector, NOW())
			ON CONFLICT (source_type, source_id) DO UPDATE
				SET content   = EXCLUDED.content,
					embedding = EXCLUDED.embedding,
					created_at = NOW()
		`
		if _, err := db.ExecContext(ctx, query,
			payload.SourceType,
			payload.SourceID,
			payload.Content,
			vecStr,
		); err != nil {
			log.Error("upsert embedding",
				zap.String("source_type", payload.SourceType),
				zap.String("source_id", payload.SourceID.String()),
				zap.Error(err),
			)
			if taskID != "" {
				if _, failErr := taskSvc.Fail(ctx, parseUUID(taskID), fmt.Sprintf("upsert %s/%s: %v", payload.SourceType, payload.SourceID, err)); failErr != nil {
					log.Warn("failed to mark task as failed", zap.Error(failErr))
				}
			}
			return fmt.Errorf("embedding_generate: upsert %s/%s: %w", payload.SourceType, payload.SourceID, err)
		}

		log.Info("embedding stored",
			zap.String("source_type", payload.SourceType),
			zap.String("source_id", payload.SourceID.String()),
			zap.Int("dimensions", len(vec)),
		)

		// Mark task as completed
		if taskID != "" {
			resultJSON, _ := json.Marshal(map[string]interface{}{
				"source_type": payload.SourceType,
				"source_id":   payload.SourceID.String(),
				"dimensions":  len(vec),
			})
			if _, err := taskSvc.Complete(ctx, parseUUID(taskID), resultJSON); err != nil {
				log.Warn("failed to mark task as completed", zap.String("task_id", taskID), zap.Error(err))
			}
		}

		return nil
	}
}

// handleVoiceSession processes voice_session tasks.
// It calls the browser-agent to start a voice interview session, which
// joins a LiveKit room and runs the interview (assist or autonomous mode).
// This is a long-running task (up to 30 minutes).
func newHandleVoiceSession(browserClient BrowserAgentClient, logger *zap.Logger, taskSvc *tasks.Service) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		log := logger.Named("task.voice_session")
		taskID := taskIDFromTask(t)

		// Mark task as running
		if taskID != "" {
			if _, err := taskSvc.Start(ctx, parseUUID(taskID)); err != nil {
				log.Warn("failed to mark task as running", zap.String("task_id", taskID), zap.Error(err))
			}
		}

		var payload tasks.VoiceSessionPayload
		if err := json.Unmarshal(t.Payload(), &payload); err != nil {
			log.Error("unmarshal payload", zap.String("task_type", t.Type()), zap.Error(err))
			if taskID != "" {
				if _, err := taskSvc.Fail(ctx, parseUUID(taskID), fmt.Sprintf("unmarshal payload: %v", err)); err != nil {
					log.Warn("failed to mark task as failed", zap.Error(err))
				}
			}
			return fmt.Errorf("voice_session: unmarshal payload: %w", err)
		}

		ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
		defer cancel()

		log.Info("starting voice session",
			zap.String("interview_id", payload.InterviewID.String()),
			zap.String("application_id", payload.ApplicationID.String()),
			zap.String("mode", payload.Mode),
			zap.String("external_session", payload.ExternalSession),
		)

		resp, err := browserClient.StartVoiceSession(ctx, VoiceSessionRequest{
			InterviewID:     payload.InterviewID.String(),
			ApplicationID:   payload.ApplicationID.String(),
			Mode:            payload.Mode,
			ExternalSession: payload.ExternalSession,
			Provider:        payload.Provider,
			Model:           payload.Model,
		})
		if err != nil {
			log.Error("voice session failed",
				zap.String("interview_id", payload.InterviewID.String()),
				zap.Error(err),
			)
			if taskID != "" {
				if _, failErr := taskSvc.Fail(ctx, parseUUID(taskID), fmt.Sprintf("%s: %v", payload.InterviewID, err)); failErr != nil {
					log.Warn("failed to mark task as failed", zap.Error(failErr))
				}
			}
			return fmt.Errorf("voice_session: %s: %w", payload.InterviewID, err)
		}

		log.Info("voice session completed",
			zap.String("interview_id", payload.InterviewID.String()),
			zap.Bool("success", resp.Success),
			zap.String("message", resp.Message),
		)

		// Mark task as completed
		if taskID != "" {
			resultJSON, _ := json.Marshal(map[string]interface{}{
				"success": resp.Success,
				"message": resp.Message,
			})
			if _, err := taskSvc.Complete(ctx, parseUUID(taskID), resultJSON); err != nil {
				log.Warn("failed to mark task as completed", zap.String("task_id", taskID), zap.Error(err))
			}
		}

		return nil
	}
}
