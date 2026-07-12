package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SetupChecker is a function that returns true if setup is required.
type SetupChecker func() bool

// OnboardingChecker is a function that returns true if onboarding is completed.
type OnboardingChecker func() bool

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
			strings.HasPrefix(path, "/api/v1/auth/setup") {
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

// OnboardingCompleteMiddleware blocks onboarding routes after onboarding is completed.
// Prevents unauthenticated access to config endpoints (API key overwrite, etc.).
// NOTE: This uses IsOnboardingCompleted, NOT IsSetupRequired.
// IsSetupRequired returns false after step 1 (user created), but onboarding continues.
func OnboardingCompleteMiddleware(isOnboardingCompleted OnboardingChecker, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// If onboarding is not completed yet, pass through (onboarding in progress)
		if !isOnboardingCompleted() {
			c.Next()
			return
		}

		path := c.Request.URL.Path

		// Block onboarding sub-routes (except status and base setup which are safe)
		if strings.HasPrefix(path, "/api/v1/auth/setup/") &&
			path != "/api/v1/auth/setup/status" &&
			path != "/api/v1/auth/setup" {
			logger.Warn("onboarding complete — blocking setup route",
				zap.String("path", path),
				zap.String("method", c.Request.Method),
			)

			c.AbortWithStatusJSON(http.StatusGone, gin.H{
				"error": gin.H{
					"code":    "ONBOARDING_COMPLETE",
					"message": "onboarding has already been completed",
				},
			})
			return
		}

		c.Next()
	}
}
