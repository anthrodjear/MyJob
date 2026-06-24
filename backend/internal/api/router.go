package api

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"backend/internal/activity"
	"backend/internal/api/middleware"
	"backend/internal/applications"
	"backend/internal/approvals"
	"backend/internal/auth"
	"backend/internal/config"
	"backend/internal/emails"
	"backend/internal/interviews"
	"backend/internal/jobs"
	"backend/internal/profile"
	"backend/internal/rag"
	"backend/internal/resumes"
	"backend/internal/scoring"
	"backend/internal/systemconfig"
)

// RouterConfig holds dependencies for router setup.
type RouterConfig struct {
	AuthHandler         *auth.Handler
	AuthService         *auth.Service
	RateLimitConfig     config.RateLimitConfig
	JobsHandler         *jobs.Handler
	ApplicationsHandler *applications.Handler
	ResumesHandler      *resumes.Handler
	ScoringHandler      *scoring.Handler
	InterviewsHandler   *interviews.Handler
	ProfileHandler      *profile.Handler
	ApprovalsHandler    *approvals.Handler
	RAGHandler          *rag.Handler
	EmailsHandler       *emails.Handler
	ActivityHandler     *activity.Handler
	SystemConfigHandler *systemconfig.Handler
	Logger              *zap.Logger
}

// SetupRouter creates and configures the Gin router.
// Middleware stack: Recovery → Logging → Rate Limit → Auth (on protected routes).
func SetupRouter(cfg RouterConfig) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logging(cfg.Logger))
	r.Use(middleware.RateLimit(cfg.RateLimitConfig, cfg.Logger))

	// Health check (public, no rate limit logging)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "healthy",
			"time":   "now",
		})
	})

	// API v1 group
	v1 := r.Group("/api/v1")
	{
		// Public auth routes (no JWT)
		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/login", cfg.AuthHandler.Login)
			authGroup.POST("/change-password", cfg.AuthHandler.ChangePassword)
		}

		// Protected routes (require JWT)
		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware(cfg.AuthService))
		{
			cfg.JobsHandler.RegisterRoutes(protected)
			cfg.ApplicationsHandler.RegisterRoutes(protected)
			cfg.ResumesHandler.RegisterRoutes(protected)
			cfg.ScoringHandler.RegisterRoutes(protected)
			cfg.InterviewsHandler.RegisterRoutes(protected)
			cfg.ProfileHandler.RegisterRoutes(protected)
			cfg.ApprovalsHandler.RegisterRoutes(protected)
			cfg.RAGHandler.RegisterRoutes(protected)
			cfg.EmailsHandler.RegisterRoutes(protected)
			cfg.ActivityHandler.RegisterRoutes(protected)
			cfg.SystemConfigHandler.RegisterRoutes(protected)
		}
	}

	return r
}
