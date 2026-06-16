package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"backend/internal/api"
	"backend/internal/applications"
	"backend/internal/auth"
	"backend/internal/config"
	"backend/internal/database"
	"backend/internal/jobs"
	"backend/internal/resumes"
	"backend/internal/scoring"
	"backend/internal/tasks"
)

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
	resumesService := resumes.NewService(resumesRepo, resumesLLM, logger)
	resumesHandler := resumes.NewHandler(resumesService, logger)

	// Initialize scoring domain
	scoringRepo := scoring.NewRepository(postgres.DB)
	scoringLLM := scoring.NewLLMScorerFromConfig(logger, cfg.LLM, cfg.Prompts)
	scoringService := scoring.NewService(scoringRepo, scoringLLM, logger, cfg.Scoring)
	scoringHandler := scoring.NewHandler(scoringService, dispatcher, logger)

	// Setup router with all routes
	router := api.SetupRouter(api.RouterConfig{
		AuthHandler:         authHandler,
		AuthService:         authService,
		JobsHandler:         jobsHandler,
		ApplicationsHandler: appsHandler,
		ResumesHandler:      resumesHandler,
		ScoringHandler:      scoringHandler,
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
