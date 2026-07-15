package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"backend/internal/activity"
	"backend/internal/applications"
	"backend/internal/approvals"
	"backend/internal/config"
	"backend/internal/database"
	"backend/internal/embeddings"
	"backend/internal/jobs"
	"backend/internal/resumes"
	"backend/internal/scoring"
	"backend/internal/tasks"
)

func main() {
	cfg := config.Load()

	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	if err := cfg.Validate(); err != nil {
		logger.Fatal("config validation failed", zap.Error(err))
	}

	postgres, err := database.NewPostgresDB(cfg.Database, logger)
	if err != nil {
		logger.Fatal("Failed to connect to PostgreSQL", zap.Error(err))
	}
	defer postgres.Close()

	// Run database migrations
	if err := database.RunMigrations(cfg.Database.URL); err != nil {
		logger.Fatal("Failed to run database migrations", zap.Error(err))
	}
	logger.Info("Database migrations applied")

	redis, err := database.NewRedisClient(cfg.Redis.URL, logger)
	if err != nil {
		logger.Fatal("Failed to create Redis client", zap.Error(err))
	}
	defer redis.Close()

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redis.Connect(ctx); err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}

	// --- Domain initialization ---
	scoringRepo := scoring.NewRepository(postgres.DB)
	scoringLLM := scoring.NewLLMScorerFromConfig(logger, cfg.LLM, cfg.Prompts)
	scoringService := scoring.NewService(scoringRepo, scoringLLM, logger, cfg.Scoring)

	jobsRepo := jobs.NewRepository(postgres.DB)
	jobsService := jobs.NewService(jobsRepo, nil, cfg.Scoring)

	resumesRepo := resumes.NewRepository(postgres.DB)
	resumesLLM := resumes.NewResumeGeneratorFromConfig(logger, cfg.LLM, cfg.Prompts)
	coverLetterLLM := resumes.NewCoverLetterGeneratorFromConfig(logger, cfg.LLM, cfg.Prompts)
	resumeTailor := resumes.NewResumeTailorFromConfig(logger, cfg.LLM, cfg.Prompts)
	resumesService := resumes.NewService(resumesRepo, resumesLLM, coverLetterLLM, resumeTailor, logger)

	applicationsRepo := applications.NewRepository(postgres.DB)
	applicationsService := applications.NewService(applicationsRepo, logger)

	activityRepo := activity.NewRepository(postgres.DB)
	activityService := activity.NewService(activityRepo, logger)

	approvalsRepo := approvals.NewRepository(postgres.DB)
	approvalsService := approvals.NewService(approvalsRepo, logger)

	browserAgentURL := getEnv("BROWSER_AGENT_URL", "http://localhost:3000")
	browserClient := NewHTTPBrowserAgentClient(browserAgentURL, logger)

	embeddingClient := embeddings.NewEmbeddingClientFromConfig(logger, cfg.LLM)

	// --- Tasks service (for lifecycle tracking) ---
	tasksRepo := tasks.NewRepository(postgres.DB)
	tasksService := tasks.NewService(tasksRepo)

	// --- Asynq client + dispatcher (for enqueuing tasks from handlers) ---
	redisAddr := parseRedisAddr(cfg.Redis.URL, logger)

	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})
	defer asynqClient.Close()

	taskDispatcher := tasks.NewDispatcher(asynqClient, tasksService, logger)

	// --- Approval workflow (approve → dispatch submission) ---
	_ = approvals.NewWorkflow(approvalsService, approvalsDispatcherAdapter{dispatcher: taskDispatcher}, activityService, logger)

	// --- Asynq server ---
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{
			Concurrency: cfg.Queue.Concurrency,
		},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(tasks.TypeJobDiscovery, newHandleScrapeSource(jobsService, scoringService, browserClient, logger, tasksService))
	mux.HandleFunc(tasks.TypeJobScoring, newHandleScoring(scoringService, jobsService, activityService, approvalsService, applicationsService, logger, tasksService))
	mux.HandleFunc(tasks.TypeResumeGenerate, newHandleGenerateResume(resumesService, jobsService, logger, tasksService))
	mux.HandleFunc(tasks.TypeCoverLetterGen, newHandleGenerateCoverLetter(resumesService, jobsService, logger, tasksService))
	mux.HandleFunc(tasks.TypeResumeTailor, newHandleTailorResume(resumesService, jobsService, logger, tasksService))
	mux.HandleFunc(tasks.TypeFillForm, newHandleFillForm(browserClient, logger, tasksService))
	mux.HandleFunc(tasks.TypeApplicationSubmit, newHandleSubmitApplication(applicationsService, jobsService, browserClient, logger, tasksService))
	mux.HandleFunc(tasks.TypeEmailCheck, newHandleSyncEmails(applicationsService, browserClient, cfg.Email, logger, tasksService))
	mux.HandleFunc(tasks.TypeInterviewPrep, newHandleGenerateInterviewPrep(applicationsService, jobsService, logger, tasksService))
	mux.HandleFunc(tasks.TypeEmbeddingGenerate, newHandleCreateEmbeddings(embeddingClient, postgres.DB, logger, tasksService))
	mux.HandleFunc(tasks.TypeVoiceSession, newHandleVoiceSession(browserClient, logger, tasksService))

	logger.Info("Worker started")

	if err := srv.Run(mux); err != nil {
		logger.Error("Worker failed", zap.Error(err))
		return // return instead of os.Exit — allows defers to close postgres/redis connections
	}

	logger.Info("Worker stopped")
}

// parseRedisAddr extracts host:port from a redis:// URL for asynq.
func parseRedisAddr(redisURL string, logger *zap.Logger) string {
	if !strings.Contains(redisURL, "://") {
		return redisURL
	}
	u, err := url.Parse(redisURL)
	if err != nil {
		logger.Fatal("invalid Redis URL", zap.String("url", redisURL), zap.Error(err))
	}
	addr := u.Host
	if !strings.Contains(addr, ":") {
		addr += ":6379"
	}
	return addr
}

// getEnv returns the environment variable value or a default.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
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
