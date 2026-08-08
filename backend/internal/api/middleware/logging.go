package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// RequestIDHeader is the canonical header used both for inbound propagation
// and for echoing the id back to the client.
const RequestIDHeader = "X-Request-Id"

// requestIDContextKey is the gin.Context key under which the request id is stored.
const requestIDContextKey = "request_id"

// RequestID returns a Gin middleware that assigns each request a stable id,
// stores it on the context, and echoes it via the X-Request-Id response header.
//
// If the inbound request already carries an X-Request-Id header, that value is
// reused so a trace can be correlated across services. Otherwise a fresh
// UUIDv4 is generated. Downstream handlers and logging middleware can read
// the id from c.GetString("request_id").
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}
		c.Set(requestIDContextKey, id)
		c.Writer.Header().Set(RequestIDHeader, id)
		c.Next()
	}
}

// RequestIDFromContext returns the request id stored on the context, or "" if
// the RequestID middleware did not run.
func RequestIDFromContext(c *gin.Context) string {
	return c.GetString(requestIDContextKey)
}

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
		requestID := RequestIDFromContext(c)

		// Build log fields
		fields := []zap.Field{
			zap.Int("status", status),
			zap.String("method", method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", clientIP),
			zap.Duration("latency", latency),
			zap.Int("body_size", c.Writer.Size()),
			zap.String("request_id", requestID),
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
