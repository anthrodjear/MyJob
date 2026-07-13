package api

import (
	"net/http"
	"net/http/httptest"
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
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
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
	req, _ := http.NewRequest(http.MethodGet, "/version", nil)
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
	req, _ := http.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	r.ServeHTTP(w, req)

	// Should get 204 No Content for preflight
	assert.Equal(t, http.StatusNoContent, w.Code)
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
