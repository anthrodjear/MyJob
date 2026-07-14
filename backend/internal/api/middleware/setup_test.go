package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestSetupMiddleware_WhenSetupRequired(t *testing.T) {
	logger := zap.NewNop()
	checker := func() bool { return true }

	r := gin.New()
	r.Use(SetupMiddleware(checker, logger))
	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	r.GET("/version", func(c *gin.Context) { c.JSON(200, gin.H{"version": "1.0"}) })
	r.GET("/api/v1/auth/setup/status", func(c *gin.Context) { c.JSON(200, gin.H{"setup_required": true}) })
	r.POST("/api/v1/auth/setup", func(c *gin.Context) { c.JSON(200, gin.H{"message": "ok"}) })
	r.GET("/api/v1/jobs", func(c *gin.Context) { c.JSON(200, gin.H{"jobs": []string{}}) })

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{"health allowed", "GET", "/health", 200},
		{"version allowed", "GET", "/version", 200},
		{"setup status allowed", "GET", "/api/v1/auth/setup/status", 200},
		{"setup post allowed", "POST", "/api/v1/auth/setup", 200},
		{"jobs blocked", "GET", "/api/v1/jobs", 403},
		{"login blocked", "POST", "/api/v1/auth/login", 403},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequestWithContext(context.Background(), tt.method, tt.path, nil)
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestSetupMiddleware_WhenSetupNotRequired(t *testing.T) {
	logger := zap.NewNop()
	checker := func() bool { return false }

	r := gin.New()
	r.Use(SetupMiddleware(checker, logger))
	r.GET("/api/v1/jobs", func(c *gin.Context) { c.JSON(200, gin.H{"jobs": []string{}}) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/api/v1/jobs", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200 — all routes should pass through when setup not required", w.Code)
	}
}
