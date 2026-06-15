package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"backend/internal/config"
	"backend/internal/database"
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

	// Initialize Asynq server
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: fmt.Sprintf("localhost:%s", "6379")},
		asynq.Config{
			Concurrency: cfg.Queue.Concurrency,
		},
	)

	// Register task handlers
	mux := asynq.NewServeMux()
	mux.HandleFunc("scrape_source", handleScrapeSource)
	mux.HandleFunc("generate_resume", handleGenerateResume)
	mux.HandleFunc("generate_coverletter", handleGenerateCoverLetter)
	mux.HandleFunc("fill_form", handleFillForm)
	mux.HandleFunc("submit_application", handleSubmitApplication)
	mux.HandleFunc("sync_emails", handleSyncEmails)
	mux.HandleFunc("generate_interview_prep", handleGenerateInterviewPrep)
	mux.HandleFunc("create_embeddings", handleCreateEmbeddings)
	mux.HandleFunc("voice_session", handleVoiceSession)

	// Graceful shutdown
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		cancel()
	}()

	logger.Info("Worker started")

	// Start processing tasks
	if err := srv.Run(mux); err != nil {
		logger.Fatal("Worker failed", zap.Error(err))
	}

	logger.Info("Worker stopped")
}

// Task handler functions
func handleScrapeSource(ctx context.Context, t *asynq.Task) error {
	fmt.Println("Handling scrape_source task")
	// TODO: Implement scraping logic
	time.Sleep(5 * time.Second)
	return nil
}

func handleGenerateResume(ctx context.Context, t *asynq.Task) error {
	fmt.Println("Handling generate_resume task")
	// TODO: Implement resume generation
	time.Sleep(10 * time.Second)
	return nil
}

func handleGenerateCoverLetter(ctx context.Context, t *asynq.Task) error {
	fmt.Println("Handling generate_coverletter task")
	// TODO: Implement cover letter generation
	time.Sleep(5 * time.Second)
	return nil
}

func handleFillForm(ctx context.Context, t *asynq.Task) error {
	fmt.Println("Handling fill_form task")
	// TODO: Dispatch to browser agent
	time.Sleep(15 * time.Second)
	return nil
}

func handleSubmitApplication(ctx context.Context, t *asynq.Task) error {
	fmt.Println("Handling submit_application task")
	// TODO: Dispatch to browser agent
	time.Sleep(10 * time.Second)
	return nil
}

func handleSyncEmails(ctx context.Context, t *asynq.Task) error {
	fmt.Println("Handling sync_emails task")
	// TODO: Implement email sync
	time.Sleep(30 * time.Second)
	return nil
}

func handleGenerateInterviewPrep(ctx context.Context, t *asynq.Task) error {
	fmt.Println("Handling generate_interview_prep task")
	// TODO: Implement interview prep generation
	time.Sleep(20 * time.Second)
	return nil
}

func handleCreateEmbeddings(ctx context.Context, t *asynq.Task) error {
	fmt.Println("Handling create_embeddings task")
	// TODO: Implement embedding creation
	time.Sleep(5 * time.Second)
	return nil
}

func handleVoiceSession(ctx context.Context, t *asynq.Task) error {
	fmt.Println("Handling voice_session task")
	// TODO: Implement voice session
	time.Sleep(60 * time.Second)
	return nil
}
