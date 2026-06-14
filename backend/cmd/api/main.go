package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"backend/internal/config"
	"backend/internal/database"
)

func main() {
	// Load configuration
	cfg := config.Load()

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

	// Initialize Gin router
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"time":   time.Now().UTC(),
		})
	})

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Jobs
		v1.GET("/jobs", listJobs)
		v1.GET("/jobs/:id", getJob)
		v1.POST("/jobs/search", searchJobs)
		v1.GET("/jobs/:id/match", getJobMatch)

		// Applications
		v1.GET("/applications", listApplications)
		v1.GET("/applications/:id", getApplication)
		v1.POST("/applications", createApplication)
		v1.PUT("/applications/:id/status", updateApplicationStatus)
		v1.GET("/applications/stats", getApplicationStats)

		// Approvals
		v1.GET("/approvals", listApprovals)
		v1.GET("/approvals/:id", getApproval)
		v1.POST("/approvals/:id/approve", approveApplication)
		v1.POST("/approvals/:id/reject", rejectApplication)

		// Resumes
		v1.GET("/resumes", listResumes)
		v1.GET("/resumes/:id", getResume)
		v1.POST("/resumes/generate", generateResume)
		v1.GET("/resumes/:id/pdf", getResumePDF)

		// Cover Letters
		v1.GET("/cover-letters", listCoverLetters)
		v1.GET("/cover-letters/:id", getCoverLetter)
		v1.POST("/cover-letters/generate", generateCoverLetter)
		v1.GET("/cover-letters/:id/pdf", getCoverLetterPDF)

		// Emails
		v1.GET("/emails", listEmails)
		v1.POST("/emails/sync", syncEmails)
		v1.GET("/emails/:id", getEmail)
		v1.POST("/emails/:id/reply", draftEmailReply)

		// Interviews
		v1.GET("/interviews", listInterviews)
		v1.GET("/interviews/:id", getInterview)
		v1.POST("/interviews/:id/prep", generateInterviewPrep)
		v1.POST("/interviews/:id/mock", startMockInterview)

		// Tasks
		v1.GET("/tasks", listTasks)
		v1.GET("/tasks/:id", getTask)
		v1.POST("/tasks", createTask)
		v1.DELETE("/tasks/:id", cancelTask)

		// Profile
		v1.GET("/profile", getProfile)
		v1.PUT("/profile", updateProfile)

		// Config
		v1.GET("/config/jobsites", listJobSources)
		v1.POST("/config/jobsites", createJobSource)
		v1.PUT("/config/jobsites/:id", updateJobSource)
		v1.DELETE("/config/jobsites/:id", deleteJobSource)

		// RAG
		v1.POST("/rag/index", indexDocuments)
		v1.POST("/rag/search", searchRAG)
		v1.GET("/rag/stats", getRAGStats)

		// Dashboard
		v1.GET("/dashboard/summary", getDashboardSummary)
		v1.GET("/dashboard/timeline", getDashboardTimeline)
		v1.GET("/dashboard/activity", getDashboardActivity)
	}

	// Start server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}

// Placeholder handler functions
func listJobs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "List jobs"})
}

func getJob(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get job"})
}

func searchJobs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Search jobs"})
}

func getJobMatch(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get job match"})
}

func listApplications(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "List applications"})
}

func getApplication(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get application"})
}

func createApplication(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Create application"})
}

func updateApplicationStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Update application status"})
}

func getApplicationStats(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get application stats"})
}

func listApprovals(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "List approvals"})
}

func getApproval(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get approval"})
}

func approveApplication(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Approve application"})
}

func rejectApplication(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Reject application"})
}

func listResumes(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "List resumes"})
}

func getResume(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get resume"})
}

func generateResume(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Generate resume"})
}

func getResumePDF(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get resume PDF"})
}

func listCoverLetters(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "List cover letters"})
}

func getCoverLetter(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get cover letter"})
}

func generateCoverLetter(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Generate cover letter"})
}

func getCoverLetterPDF(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get cover letter PDF"})
}

func listEmails(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "List emails"})
}

func syncEmails(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Sync emails"})
}

func getEmail(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get email"})
}

func draftEmailReply(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Draft email reply"})
}

func listInterviews(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "List interviews"})
}

func getInterview(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get interview"})
}

func generateInterviewPrep(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Generate interview prep"})
}

func startMockInterview(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Start mock interview"})
}

func listTasks(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "List tasks"})
}

func getTask(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get task"})
}

func createTask(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Create task"})
}

func cancelTask(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Cancel task"})
}

func getProfile(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get profile"})
}

func updateProfile(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Update profile"})
}

func listJobSources(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "List job sources"})
}

func createJobSource(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Create job source"})
}

func updateJobSource(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Update job source"})
}

func deleteJobSource(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Delete job source"})
}

func indexDocuments(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Index documents"})
}

func searchRAG(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Search RAG"})
}

func getRAGStats(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get RAG stats"})
}

func getDashboardSummary(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get dashboard summary"})
}

func getDashboardTimeline(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get dashboard timeline"})
}

func getDashboardActivity(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get dashboard activity"})
}
