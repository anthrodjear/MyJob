package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Logging returns a Gin middleware that logs every request with structured fields.
// Captures method, path, status code, latency, client IP, and any error message.
// Uses zap for structured logging consistent with the rest of the codebase.
//
// Skips logging for health check endpoints to reduce noise.
func Logging(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip health checks to reduce log noise
		if c.Request.URL.Path == "/health" {
			c.Next()
			return
		}

		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Process request
		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		errorMsg := c.Errors.ByType(gin.ErrorTypePrivate).String()

		// Build log fields
		fields := []zap.Field{
			zap.Int("status", status),
			zap.String("method", method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", clientIP),
			zap.Duration("latency", latency),
			zap.Int("body_size", c.Writer.Size()),
		}

		if errorMsg != "" {
			fields = append(fields, zap.String("error", errorMsg))
		}

		// Log at appropriate level based on status code
		switch {
		case status >= 500:
			logger.Error("request", fields...)
		case status >= 400:
			logger.Warn("request", fields...)
		default:
			logger.Info("request", fields...)
		}
	}
}
