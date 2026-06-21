package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"backend/internal/activity"
	"backend/internal/api"
	"backend/internal/applications"
	"backend/internal/approvals"
	"backend/internal/auth"
	"backend/internal/config"
	"backend/internal/database"
	"backend/internal/embeddings"
	"backend/internal/emails"
	"backend/internal/interviews"
	"backend/internal/jobs"
	"backend/internal/profile"
	"backend/internal/rag"
	"backend/internal/resumes"
	"backend/internal/scoring"
	"backend/internal/tasks"
)

// getEnvDuration parses a duration from environment variable with a fallback.
func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return fallback
}

// toEmailPromptPair converts config.PromptPair to emails.PromptPair.
func toEmailPromptPair(cfg config.PromptPair) emails.PromptPair {
	return emails.PromptPair{
		System: cfg.System,
		User:   cfg.User,
	}
}

func main() {
	// Load configuration
	cfg := config.Load()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		panic(fmt.Sprintf("config validation failed: %v", err))
	}

	// Initialize logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Connect to PostgreSQL
	postgres, err := database.NewPostgresDB(cfg.Database.URL, logger)
	if err != nil {
		logger.Fatal("Failed to connect to PostgreSQL", zap.Error(err))
	}
	defer postgres.Close()

	// Connect to Redis
	redis, err := database.NewRedisClient(cfg.Redis.URL, logger)
	if err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	defer redis.Close()

	// Initialize auth domain
	authRepo, err := auth.NewRepository(postgres.DB, cfg.Auth)
	if err != nil {
		logger.Fatal("Failed to create auth repository", zap.Error(err))
	}
	authService := auth.NewService(authRepo, cfg.Auth)
	authHandler := auth.NewHandler(authService, logger)

	// Initialize asynq client for task dispatch
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.Redis.URL})
	defer asynqClient.Close()
	dispatcher := tasks.NewDispatcher(asynqClient, logger)

	// Initialize jobs domain
	jobsRepo := jobs.NewRepository(postgres.DB)
	jobsService := jobs.NewService(jobsRepo, dispatcher, cfg.Scoring)
	jobsHandler := jobs.NewHandler(jobsService, logger)

	// Initialize applications domain
	appsRepo := applications.NewRepository(postgres.DB)
	appsService := applications.NewService(appsRepo, logger)
	appsHandler := applications.NewHandler(appsService, logger)

	// Initialize resumes domain
	resumesRepo := resumes.NewRepository(postgres.DB)
	resumesLLM := resumes.NewResumeGeneratorFromConfig(logger, cfg.LLM, cfg.Prompts)
	coverLetterLLM := resumes.NewCoverLetterGeneratorFromConfig(logger, cfg.LLM, cfg.Prompts)
	resumeTailor := resumes.NewResumeTailorFromConfig(logger, cfg.LLM, cfg.Prompts)
	resumesService := resumes.NewService(resumesRepo, resumesLLM, coverLetterLLM, resumeTailor, logger)
	resumesHandler := resumes.NewHandler(resumesService, logger)

	// Initialize scoring domain
	scoringRepo := scoring.NewRepository(postgres.DB)
	scoringLLM := scoring.NewLLMScorerFromConfig(logger, cfg.LLM, cfg.Prompts)
	scoringService := scoring.NewService(scoringRepo, scoringLLM, logger, cfg.Scoring)
	scoringHandler := scoring.NewHandler(scoringService, dispatcher, logger)

	// Initialize interviews domain
	interviewsRepo := interviews.NewRepository(postgres.DB)
	interviewsService := interviews.NewService(interviewsRepo, dispatcher, logger)
	interviewsHandler := interviews.NewHandler(interviewsService, logger)

	// Initialize profile domain
	profileRepo := profile.NewRepository(postgres.DB)
	profileService := profile.NewService(profileRepo, logger)
	profileHandler := profile.NewHandler(profileService, logger)

	// Initialize approvals domain
	approvalsRepo := approvals.NewRepository(postgres.DB)
	approvalsService := approvals.NewService(approvalsRepo, logger)
	// Adapter: approvals.SubmitDispatcher interface → *tasks.Dispatcher
	approvalsDispatcher := approvalsDispatcherAdapter{dispatcher: dispatcher}
	approvalsWorkflow := approvals.NewWorkflow(approvalsService, approvalsDispatcher, logger)
	approvalsHandler := approvals.NewHandler(approvalsWorkflow, approvalsService, logger)

	// Initialize RAG domain
	ragRepo := rag.NewRepository(postgres.DB)
	embeddingClient := embeddings.NewEmbeddingClientFromConfig(logger, cfg.LLM)
	ragService := rag.NewService(ragRepo, embeddingClient, logger)
	ragHandler := rag.NewHandler(ragService, logger)

	// Initialize activity domain
	activityRepo := activity.NewRepository(postgres.DB)
	activityService := activity.NewService(activityRepo, logger)
	activityHandler := activity.NewHandler(activityService, logger)

	// Initialize emails domain
	emailsRepo := emails.NewRepository(postgres.DB)
	// Classifier uses dedicated email classifier LLM config
	emailClassifier, err := emails.NewClassifierFromConfig(
		logger,
		cfg.LLM.EmailClassifier.BaseURL,
		cfg.LLM.EmailClassifier.Model,
		cfg.LLM.EmailClassifier.Timeout,
		toEmailPromptPair(cfg.Prompts.EmailClassifier),
	)
	if err != nil {
		logger.Fatal("Failed to create email classifier", zap.Error(err))
	}
	emailsService := emails.NewService(emailsRepo, emailClassifier)
	emailsHandler := emails.NewHandler(emailsService, logger)

	// Setup router with all routes
	router := api.SetupRouter(api.RouterConfig{
		AuthHandler:     authHandler,
		AuthService:     authService,
		RateLimitConfig: cfg.RateLimit,
		JobsHandler:     jobsHandler,
		ApplicationsHandler: appsHandler,
		ResumesHandler:      resumesHandler,
		ScoringHandler:      scoringHandler,
		InterviewsHandler:   interviewsHandler,
		ProfileHandler:      profileHandler,
		ApprovalsHandler:    approvalsHandler,
		RAGHandler:          ragHandler,
		EmailsHandler:       emailsHandler,
		ActivityHandler:     activityHandler,
		Logger:              logger,
	})

	// Start HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	logger.Info("API server started", zap.Int("port", cfg.Server.Port))

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}

// ---------------------------------------------------------------------------
// Adapters
// ---------------------------------------------------------------------------

// approvalsDispatcherAdapter adapts *tasks.Dispatcher to approvals.SubmitDispatcher.
// The workflow interface uses (ctx, applicationID, correlationID) signature.
// The concrete dispatcher uses (ctx, tasks.ApplicationSubmitPayload) signature.
type approvalsDispatcherAdapter struct {
	dispatcher *tasks.Dispatcher
}

func (a approvalsDispatcherAdapter) DispatchApplicationSubmit(ctx context.Context, applicationID uuid.UUID, correlationID uuid.UUID) error {
	_, err := a.dispatcher.DispatchApplicationSubmit(ctx, tasks.ApplicationSubmitPayload{
		ApplicationID: applicationID,
		CorrelationID: correlationID,
	})
	return err
}
