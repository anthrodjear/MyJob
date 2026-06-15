package api

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"backend/internal/auth"
	"backend/internal/api/middleware"
)

// RouterConfig holds dependencies for router setup.
type RouterConfig struct {
	AuthHandler *auth.Handler
	AuthService *auth.Service
	Logger      *zap.Logger
}

// SetupRouter creates and configures the Gin router.
func SetupRouter(cfg RouterConfig) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	// Health check (public)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "healthy",
			"time":   "now", // placeholder
		})
	})

	// API v1 group
	v1 := r.Group("/api/v1")
	{
		// Public auth routes (no middleware)
		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/login", cfg.AuthHandler.Login)
			authGroup.POST("/change-password", cfg.AuthHandler.ChangePassword)
		}

		// Protected routes (require JWT)
		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware(cfg.AuthService))
		{
			// TODO: Add protected routes here as domains are implemented
			// protected.GET("/jobs", jobsHandler.List)
			// protected.POST("/jobs/scan", jobsHandler.Scan)
			// protected.GET("/applications", appsHandler.List)
			// protected.POST("/applications", appsHandler.Create)
			// etc.
		}
	}

	return r
}
