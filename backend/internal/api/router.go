package api

import (
	"os"
	"time"

	"github.com/gin-contrib/cors"
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
// Middleware stack: Recovery → CORS → Logging → Rate Limit → Auth (on protected routes).
func SetupRouter(cfg RouterConfig) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	// CORS middleware - must be before other middleware to handle preflight requests
	corsConfig := cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Content-Length", "Accept", "Accept-Encoding", "Authorization", "X-Request-Id", "X-CSRF-Token"},
		ExposeHeaders:    []string{"Content-Length", "X-Request-Id"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	r.Use(cors.New(corsConfig))

	r.Use(middleware.Logging(cfg.Logger))
	r.Use(middleware.RateLimit(cfg.RateLimitConfig, cfg.Logger))

	// Health check (public, no rate limit logging)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "healthy",
			"time":   "now",
		})
	})

	// Version endpoint — returns deployed version for monitoring
	r.GET("/version", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"version":   os.Getenv("APP_VERSION"),
			"git_commit": os.Getenv("GIT_COMMIT"),
			"build_time": os.Getenv("BUILD_TIME"),
			"env":        os.Getenv("APP_ENV"),
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
