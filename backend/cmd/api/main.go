// @title           AI Job Search Agent API
// @version         1.0
// @description     Local-first AI-powered job search automation platform. Discovers, scores, and applies to jobs with tailored resumes and cover letters.
// @termsOfService  https://github.com/your-org/ai-job-search-agent

// @contact.name   API Support
// @contact.url    https://github.com/your-org/ai-job-search-agent/issues
// @contact.email  support@example.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT Authorization header using the Bearer scheme. Example: "Authorization: Bearer {token}"

// @tag.name Auth
// @tag.description Authentication and onboarding endpoints

// @tag.name Jobs
// @tag.description Job discovery, listing, and management

// @tag.name Applications
// @tag.description Application lifecycle management with audit trail

// @tag.name Resumes
// @tag.description Resume CRUD, versioning, and LLM generation

// @tag.name CoverLetters
// @tag.description Cover letter management and LLM generation

// @tag.name Scoring
// @tag.description Job-candidate matching and scoring pipeline

// @tag.name Interviews
// @tag.description Interview session management and voice agent

// @tag.name Profile
// @tag.description User profile and preferences (JSONB with optimistic locking)

// @tag.name Approvals
// @tag.description Human-in-the-loop approval gate for auto-apply

// @tag.name Emails
// @tag.description Recruiter email monitoring and classification

// @tag.name RAG
// @tag.description Semantic search and embeddings for resume/job matching

// @tag.name Tasks
// @tag.description Async task queue status and polling

// @tag.name SystemConfig
// @tag.description System configuration management (YAML/env/DB merge)

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
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	_ "backend/docs" // swagger docs

	"backend/internal/activity"
	"backend/internal/api"
	"backend/internal/applications"
	"backend/internal/approvals"
	"backend/internal/auth"
	"backend/internal/config"
	"backend/internal/database"
	"backend/internal/emails"
	"backend/internal/embeddings"
	"backend/internal/interviews"
	"backend/internal/jobs"
	"backend/internal/profile"
	"backend/internal/rag"
	"backend/internal/resumes"
	"backend/internal/scoring"
	"backend/internal/systemconfig"
	"backend/internal/tasks"
)

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
	logger, err := zap.NewProduction()
	if err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}
	defer logger.Sync()

	// Connect to PostgreSQL
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

	// Connect to Redis
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

	// Initialize system config (needed by auth service for onboarding)
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config/application.yaml"
	}
	systemConfigResolver, err := systemconfig.NewResolver(logger, configPath)
	if err != nil {
		logger.Fatal("Failed to create system config resolver", zap.Error(err))
	}
	systemConfigRepo := systemconfig.NewRepository(postgres.DB)
	systemConfigService := systemconfig.NewService(systemConfigRepo, systemConfigResolver)

	// Initialize auth domain
	authRepo, err := auth.NewRepository(postgres.DB, cfg.Auth)
	if err != nil {
		logger.Fatal("Failed to create auth repository", zap.Error(err))
	}
	authService := auth.NewService(authRepo, cfg.Auth, systemConfigService, logger)
	authHandler := auth.NewHandler(authService, logger)

	// Setup check function — closure over authRepo
	isSetupRequired := func() bool {
		required, err := authRepo.IsSetupRequired(context.Background())
		if err != nil {
			logger.Error("failed to check setup status", zap.Error(err))
			return true // fail closed — require setup on error
		}
		return required
	}

	// Onboarding completion check function — closure over authRepo
	isOnboardingCompleted := func() bool {
		completed, err := authRepo.IsOnboardingCompleted(context.Background())
		if err != nil {
			logger.Error("failed to check onboarding status", zap.Error(err))
			return true // fail closed — block onboarding routes on error
		}
		return completed
	}

	// Initialize tasks domain (repo + service first, needed by dispatcher)
	tasksRepo := tasks.NewRepository(postgres.DB)
	tasksService := tasks.NewService(tasksRepo)

	// Initialize asynq client for task dispatch
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.Redis.URL})
	defer asynqClient.Close()
	dispatcher := tasks.NewDispatcher(asynqClient, tasksService, logger)

	tasksHandler := tasks.NewHandler(tasksService, logger)

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

	// Initialize activity domain
	activityRepo := activity.NewRepository(postgres.DB)
	activityService := activity.NewService(activityRepo, logger)
	activityHandler := activity.NewHandler(activityService, logger)

	// Initialize approvals domain
	approvalsRepo := approvals.NewRepository(postgres.DB)
	approvalsService := approvals.NewService(approvalsRepo, logger)
	// Adapter: approvals.SubmitDispatcher interface → *tasks.Dispatcher
	approvalsDispatcher := approvalsDispatcherAdapter{dispatcher: dispatcher}
	approvalsWorkflow := approvals.NewWorkflow(approvalsService, approvalsDispatcher, activityService, logger)
	approvalsHandler := approvals.NewHandler(approvalsWorkflow, approvalsService, logger)

	// Initialize RAG domain
	ragRepo := rag.NewRepository(postgres.DB)
	embeddingClient := embeddings.NewEmbeddingClientFromConfig(logger, cfg.LLM)
	ragService := rag.NewService(ragRepo, embeddingClient, logger)
	ragHandler := rag.NewHandler(ragService, logger)

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

	// Initialize system config handler (service already initialized above)
	systemConfigHandler := systemconfig.NewHandler(systemConfigService, logger)

	// Setup router with all routes
	router := api.SetupRouter(api.RouterConfig{
		AuthHandler:           authHandler,
		AuthService:           authService,
		IsSetupRequired:       isSetupRequired,
		IsOnboardingCompleted: isOnboardingCompleted,
		CORSOrigins:           cfg.Server.CORSOrigins,
		RateLimitConfig:       cfg.RateLimit,
		AuthRateLimitConfig:   cfg.AuthRateLimit,
		JobsHandler:           jobsHandler,
		ApplicationsHandler:   appsHandler,
		ResumesHandler:        resumesHandler,
		ScoringHandler:        scoringHandler,
		InterviewsHandler:     interviewsHandler,
		ProfileHandler:        profileHandler,
		ApprovalsHandler:      approvalsHandler,
		RAGHandler:            ragHandler,
		EmailsHandler:         emailsHandler,
		ActivityHandler:       activityHandler,
		TasksHandler:          tasksHandler,
		SystemConfigHandler:   systemConfigHandler,
		Logger:                logger,
	})

	// Swagger UI - only enabled in non-production environments
	if !cfg.IsProduction() {
		router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
		logger.Info("Swagger UI enabled at /swagger/index.html")
	}

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

	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
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
