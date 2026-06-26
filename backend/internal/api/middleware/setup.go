package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SetupChecker is a function that returns true if setup is required.
type SetupChecker func() bool

// SetupMiddleware blocks all routes except setup endpoints, health, and version
// when no users exist in the database (first boot).
func SetupMiddleware(isSetupRequired SetupChecker, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// If setup is not required, pass through
		if !isSetupRequired() {
			c.Next()
			return
		}

		path := c.Request.URL.Path

		// Allow setup endpoints, health, and version
		if path == "/health" ||
			path == "/version" ||
			path == "/api/v1/auth/setup/status" ||
			path == "/api/v1/auth/setup" {
			c.Next()
			return
		}

		// Block everything else
		logger.Warn("setup required — blocking request",
			zap.String("path", path),
			zap.String("method", c.Request.Method),
		)

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": gin.H{
				"code":    "SETUP_REQUIRED",
				"message": "setup required — no users exist",
			},
		})
	}
}
