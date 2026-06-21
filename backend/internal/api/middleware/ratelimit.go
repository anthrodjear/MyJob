// Package middleware provides Gin middleware handlers for the API layer.
//
// Rate limiting uses a per-IP token bucket algorithm via golang.org/x/time/rate.
// Each client IP gets an independent limiter. Stale limiters are cleaned up
// periodically to prevent memory growth.
package middleware

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"backend/internal/config"
	"backend/internal/httpresp"
)

// RateLimit returns a Gin middleware that enforces per-IP rate limiting.
// Uses a token bucket algorithm — each client IP gets an independent limiter
// with the configured requests-per-minute and burst size.
//
// Exceeded requests receive a 429 Too Many Requests response with
// Retry-After and X-RateLimit-* headers.
//
// The middleware starts a background goroutine to clean up stale limiters
// every minute, preventing unbounded memory growth from idle clients.
func RateLimit(cfg config.RateLimitConfig, logger *zap.Logger) gin.HandlerFunc {
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)

	// Validate config — fail loud if misconfigured.
	// RPM of 0 would create a limiter that denies every request.
	if cfg.RequestsPerMinute <= 0 {
		logger.Error("invalid rate limit config, defaulting to 60 RPM",
			zap.Int("rpm", cfg.RequestsPerMinute),
		)
		cfg.RequestsPerMinute = 60
	}

	// Convert RPM to requests-per-second for the rate limiter.
	// Burst allows short spikes above the steady rate.
	rps := rate.Limit(float64(cfg.RequestsPerMinute) / 60.0)
	burst := cfg.Burst
	if burst <= 0 {
		burst = 1
	}

	// Background cleanup: remove clients idle for more than 3 minutes.
	// Prevents memory leak from ephemeral IPs (NAT, VPN reconnects).
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			mu.Lock()
			for ip, c := range clients {
				if time.Since(c.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()

		mu.Lock()
		cl, exists := clients[ip]
		if !exists {
			cl = &client{limiter: rate.NewLimiter(rps, burst)}
			clients[ip] = cl
		}
		cl.lastSeen = time.Now()
		mu.Unlock()

		// Check if request is allowed
		if !cl.limiter.Allow() {
			// Compute dynamic Retry-After based on actual token refill rate
			retryAfter := int(math.Ceil(1.0 / float64(rps)))

			logger.Warn("rate limit exceeded",
				zap.String("ip", ip),
				zap.String("path", c.Request.URL.Path),
				zap.Int("rpm", cfg.RequestsPerMinute),
			)
			c.Header("Retry-After", fmt.Sprintf("%d", retryAfter))
			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", cfg.RequestsPerMinute))
			c.Header("X-RateLimit-Remaining", "0")
			httpresp.TooManyRequests(c, "RATE_LIMITED", "rate limit exceeded, try again later")
			c.Abort()
			return
		}

		c.Next()
	}
}
