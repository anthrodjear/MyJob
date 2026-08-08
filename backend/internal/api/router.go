package api

import (
	"crypto/subtle"
	"net/http"
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
	"backend/internal/tasks"
)

// RouterConfig holds dependencies for router setup.
type RouterConfig struct {
	AuthHandler               *auth.Handler
	AuthService               *auth.Service
	IsSetupRequired           func() bool // setup check function
	IsOnboardingCompleted     func() bool // onboarding completion check function
	CORSOrigins               []string
	RateLimitConfig           config.RateLimitConfig
	AuthRateLimitConfig       config.AuthRateLimitConfig
	OnboardingRateLimitConfig config.AuthRateLimitConfig
	ServerConfig              config.ServerConfig
	JobsHandler               *jobs.Handler
	ApplicationsHandler       *applications.Handler
	ResumesHandler            *resumes.Handler
	ScoringHandler            *scoring.Handler
	InterviewsHandler         *interviews.Handler
	ProfileHandler            *profile.Handler
	ApprovalsHandler          *approvals.Handler
	RAGHandler                *rag.Handler
	EmailsHandler             *emails.Handler
	ActivityHandler           *activity.Handler
	TasksHandler              *tasks.Handler
	SystemConfigHandler       *systemconfig.Handler
	Logger                    *zap.Logger
}

// internalSecretMiddleware returns a Gin handler that enforces an
// X-Internal-Secret header matching the configured shared secret. The
// browser-agent's Express server sends this header on every callback;
// calls without the correct header are rejected with 401.
//
// If expectedSecret is empty the middleware logs a one-time warning and
// fails open (allows the request) — this matches the dev-only behaviour
// the parent task spec called for.
func internalSecretMiddleware(expectedSecret string, logger *zap.Logger) gin.HandlerFunc {
	if expectedSecret == "" {
		logger.Warn("INTERNAL_API_SHARED_SECRET not configured; /internal routes are unauthenticated (dev only)")
		return func(c *gin.Context) {
			c.Next()
		}
	}
	expected := []byte(expectedSecret)
	return func(c *gin.Context) {
		provided := c.GetHeader("X-Internal-Secret")
		if provided == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "MISSING_INTERNAL_SECRET", "message": "X-Internal-Secret header required"},
			})
			return
		}
		// Constant-time compare to avoid timing leaks.
		if subtle.ConstantTimeCompare([]byte(provided), expected) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "INVALID_INTERNAL_SECRET", "message": "X-Internal-Secret is invalid"},
			})
			return
		}
		c.Next()
	}
}

// SetupRouter creates and configures the Gin router.
// Middleware stack: Recovery → CORS → RequestID → Logging → Rate Limit → Setup Check → Auth (on protected routes).
func SetupRouter(cfg RouterConfig) *gin.Engine {
	r := gin.New()
	// Disable trusted-proxy parsing so c.ClientIP() returns the TCP peer.
	// Without this, Gin's default trusts all proxies and will honor a
	// client-controlled X-Forwarded-For header — trivial spoof for any
	// request that reaches this server directly (e.g. via a misconfigured
	// reverse proxy). Set explicit proxies when one is actually in front
	// of the API.
	if err := r.SetTrustedProxies(nil); err != nil {
		cfg.Logger.Warn("failed to set trusted proxies", zap.Error(err))
	}

	// CORS middleware - must be before other middleware to handle preflight requests
	corsConfig := cors.Config{
		AllowOrigins:     cfg.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Content-Length", "Accept", "Accept-Encoding", "Authorization", "X-Request-Id", "X-CSRF-Token"},
		ExposeHeaders:    []string{"Content-Length", "X-Request-Id", "ETag", "Retry-After", "X-RateLimit-Limit", "X-RateLimit-Remaining"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	r.Use(cors.New(corsConfig))

	// Request ID — assigns a stable id per request, echoes it via the
	// X-Request-Id response header so downstream log lines and clients can
	// correlate a request end-to-end. Must run before Logging so the id
	// is available in every log line.
	r.Use(middleware.RequestID())

	r.Use(middleware.Logging(cfg.Logger))
	r.Use(middleware.RateLimit(cfg.RateLimitConfig, cfg.Logger))

	// Setup middleware — blocks non-setup routes when no users exist
	if cfg.IsSetupRequired != nil {
		r.Use(middleware.SetupMiddleware(cfg.IsSetupRequired, cfg.Logger))
	}

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
			"version":    os.Getenv("APP_VERSION"),
			"git_commit": os.Getenv("GIT_COMMIT"),
			"build_time": os.Getenv("BUILD_TIME"),
			"env":        os.Getenv("APP_ENV"),
		})
	})

	// API v1 group
	v1 := r.Group("/api/v1")
	{
		// Public auth routes (no JWT) - with stricter rate limiting
		authGroup := v1.Group("/auth")
		authGroup.Use(middleware.RateLimit(config.RateLimitConfig{
			RequestsPerMinute: cfg.AuthRateLimitConfig.RequestsPerMinute,
			Burst:             cfg.AuthRateLimitConfig.Burst,
		}, cfg.Logger))
		{
			authGroup.POST("/login", cfg.AuthHandler.Login)
			authGroup.POST("/refresh", cfg.AuthHandler.Refresh)

			// Password reset routes (public, no JWT needed)
			authGroup.POST("/password/reset", cfg.AuthHandler.RequestPasswordReset)
			authGroup.POST("/password/reset/confirm", cfg.AuthHandler.ResetPassword)

			// Setup routes — public, no JWT needed
			authGroup.GET("/setup/status", cfg.AuthHandler.SetupStatus)
			authGroup.POST("/setup", cfg.AuthHandler.CompleteSetup)

			// Onboarding routes — guarded: blocked after onboarding completes
			// Prevents unauthenticated access to API key config, etc.
			// NOTE: Uses IsOnboardingCompleted, NOT IsSetupRequired.
			// IsSetupRequired returns false after step 1, but onboarding continues.
			// These routes are intentionally unauthenticated — they must be accessible
			// before the user has a JWT (post-setup, pre-onboarding).
			// Safe ONLY because this server is designed for localhost access.
			if cfg.IsOnboardingCompleted != nil {
				onboarding := authGroup.Group("")
				// Tighter rate limit for the onboarding sub-group — these
				// routes accept user-supplied provider credentials and should
				// not be reachable in tight loops (cheaper than bcrypt at
				// login, but still abusive in volume).
				onboarding.Use(middleware.RateLimit(config.RateLimitConfig{
					RequestsPerMinute: cfg.OnboardingRateLimitConfig.RequestsPerMinute,
					Burst:             cfg.OnboardingRateLimitConfig.Burst,
				}, cfg.Logger))
				onboarding.Use(middleware.OnboardingCompleteMiddleware(cfg.IsOnboardingCompleted, cfg.Logger))
				{
					onboarding.POST("/setup/test-llm", cfg.AuthHandler.TestLLMKey)
					onboarding.POST("/setup/test-voice", cfg.AuthHandler.TestVoiceConfig)
					onboarding.POST("/setup/test-email", cfg.AuthHandler.TestEmailConfig)
					onboarding.POST("/setup/config", cfg.AuthHandler.SaveOnboardingConfig)
					onboarding.POST("/setup/onboarding-step", cfg.AuthHandler.UpdateOnboardingStep)
					onboarding.POST("/setup/complete-onboarding", cfg.AuthHandler.CompleteOnboarding)
				}
			}
		}

		// Protected routes (require JWT)
		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware(cfg.AuthService))
		{
			protected.POST("/auth/change-password", cfg.AuthHandler.ChangePassword)
			protected.POST("/auth/logout", cfg.AuthHandler.Logout)
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
			cfg.TasksHandler.RegisterRoutes(protected)
			cfg.SystemConfigHandler.RegisterRoutes(protected)
		}

		// Internal routes (require X-Internal-Secret shared secret).
		// These are callbacks from the browser-agent and must NOT sit behind
		// the JWT middleware. The browser-agent is wired to send the secret
		// header on every internal request; if INTERNAL_API_SHARED_SECRET is
		// unset the middleware fails open (dev mode) and logs a warning.
		internalGroup := v1.Group("/internal", internalSecretMiddleware(cfg.ServerConfig.InternalSecret, cfg.Logger))
		cfg.InterviewsHandler.RegisterInternalRoutes(internalGroup)
	}

	return r
}
