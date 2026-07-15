package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"backend/internal/config"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestSetupRouter_HealthEndpoint(t *testing.T) {
	logger := zap.NewNop()

	// Minimal config — only health/version endpoints don't need handlers
	cfg := RouterConfig{
		Logger:      logger,
		CORSOrigins: []string{"http://localhost:3000"},
		RateLimitConfig: config.RateLimitConfig{
			RequestsPerMinute: 60,
			Burst:             10,
		},
		AuthRateLimitConfig: config.AuthRateLimitConfig{
			RequestsPerMinute: 10,
			Burst:             3,
		},
	}

	r := SetupRouter(cfg)

	// Test health endpoint
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/health", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "healthy")
}

func TestSetupRouter_VersionEndpoint(t *testing.T) {
	logger := zap.NewNop()
	cfg := RouterConfig{
		Logger:      logger,
		CORSOrigins: []string{"*"},
		RateLimitConfig: config.RateLimitConfig{
			RequestsPerMinute: 60,
			Burst:             10,
		},
		AuthRateLimitConfig: config.AuthRateLimitConfig{
			RequestsPerMinute: 10,
			Burst:             3,
		},
	}

	r := SetupRouter(cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/version", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "version")
}

func TestSetupRouter_RoutesExist(t *testing.T) {
	logger := zap.NewNop()
	cfg := RouterConfig{
		Logger:      logger,
		CORSOrigins: []string{"*"},
		RateLimitConfig: config.RateLimitConfig{
			RequestsPerMinute: 60,
			Burst:             10,
		},
		AuthRateLimitConfig: config.AuthRateLimitConfig{
			RequestsPerMinute: 10,
			Burst:             3,
		},
	}

	r := SetupRouter(cfg)

	// Collect registered routes
	routes := r.Routes()
	routeMap := make(map[string]bool)
	for _, route := range routes {
		key := route.Method + " " + route.Path
		routeMap[key] = true
	}

	// Verify public endpoints exist
	assert.True(t, routeMap["GET /health"], "health endpoint missing")
	assert.True(t, routeMap["GET /version"], "version endpoint missing")

	// Verify auth routes exist
	assert.True(t, routeMap["POST /api/v1/auth/login"], "login endpoint missing")
	assert.True(t, routeMap["POST /api/v1/auth/refresh"], "refresh endpoint missing")
	assert.True(t, routeMap["GET /api/v1/auth/setup/status"], "setup status endpoint missing")
	assert.True(t, routeMap["POST /api/v1/auth/setup"], "setup endpoint missing")
}

func TestSetupRouter_CORSConfig(t *testing.T) {
	logger := zap.NewNop()
	cfg := RouterConfig{
		Logger:      logger,
		CORSOrigins: []string{"http://localhost:3000"},
		RateLimitConfig: config.RateLimitConfig{
			RequestsPerMinute: 60,
			Burst:             10,
		},
		AuthRateLimitConfig: config.AuthRateLimitConfig{
			RequestsPerMinute: 10,
			Burst:             3,
		},
	}

	r := SetupRouter(cfg)

	// Test CORS preflight
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodOptions, "/api/v1/auth/login", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	r.ServeHTTP(w, req)

	// Should get 204 No Content for preflight
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestSetupRouter_CORSExposesETag(t *testing.T) {
	logger := zap.NewNop()
	cfg := RouterConfig{
		Logger:      logger,
		CORSOrigins: []string{"http://localhost:3000"},
		RateLimitConfig: config.RateLimitConfig{
			RequestsPerMinute: 60,
			Burst:             10,
		},
		AuthRateLimitConfig: config.AuthRateLimitConfig{
			RequestsPerMinute: 10,
			Burst:             3,
		},
	}

	r := SetupRouter(cfg)

	// A CORS response must expose ETag so the browser can read it cross-origin.
	// The profile hook reads the ETag header to build the If-Match header for
	// PUT/PATCH. If ETag is not exposed, res.headers.get("etag") returns null
	// cross-origin and the client throws a misleading ETAG_MISSING error
	// ("Could not load profile for patching") on every successful fetch.
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	r.ServeHTTP(w, req)

	expose := w.Header().Get("Access-Control-Expose-Headers")
	assert.Equal(t, http.StatusOK, w.Code)
	// gin-contrib/cors canonicalizes exposed header tokens (e.g. "ETag" -> "Etag")
	// and browsers match response headers case-insensitively, so assert on the
	// lowercased token list rather than a verbatim substring.
	entries := strings.Split(strings.ToLower(expose), ",")
	// ETag is read by the profile hook to build If-Match; the rate-limit headers
	// may be surfaced client-side to back off. All must be exposed cross-origin.
	expected := []string{"etag", "retry-after", "x-ratelimit-limit", "x-ratelimit-remaining"}
	for _, h := range expected {
		assert.Contains(t, entries, h,
			"Access-Control-Expose-Headers must include %q for cross-origin reads", h)
	}
}

func TestSetupRouter_NoSetupMiddleware(t *testing.T) {
	logger := zap.NewNop()
	cfg := RouterConfig{
		Logger:      logger,
		CORSOrigins: []string{"*"},
		RateLimitConfig: config.RateLimitConfig{
			RequestsPerMinute: 60,
			Burst:             10,
		},
		AuthRateLimitConfig: config.AuthRateLimitConfig{
			RequestsPerMinute: 10,
			Burst:             3,
		},
		IsSetupRequired:       nil, // no setup middleware
		IsOnboardingCompleted: nil, // no onboarding middleware
	}

	// Should not panic with nil setup functions
	require.NotPanics(t, func() {
		r := SetupRouter(cfg)
		assert.NotNil(t, r)
	})
}
