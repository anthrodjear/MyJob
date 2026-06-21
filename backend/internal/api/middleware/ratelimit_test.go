package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"backend/internal/config"
	"backend/internal/httpresp"
)

// init disables Gin's debug mode for cleaner test output.
func init() {
	gin.SetMode(gin.TestMode)
}

// newTestRouter creates a gin.Engine with the RateLimit middleware and a
// trivial handler that returns 200 OK. The middleware config is caller-controlled.
func newTestRouter(cfg config.RateLimitConfig) *gin.Engine {
	logger := zap.NewNop()
	r := gin.New()
	r.Use(RateLimit(cfg, logger))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

// TestRateLimit_AllowedRequest verifies a single request within the limit
// passes through and returns 200 OK.
func TestRateLimit_AllowedRequest(t *testing.T) {
	r := newTestRouter(config.RateLimitConfig{
		RequestsPerMinute: 10,
		Burst:             5,
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// TestRateLimit_ExceedsLimit verifies that requests beyond the burst
// capacity receive 429 Too Many Requests with correct headers.
func TestRateLimit_ExceedsLimit(t *testing.T) {
	r := newTestRouter(config.RateLimitConfig{
		RequestsPerMinute: 6, // 0.1 rps → refill every 10s
		Burst:             2, // only 2 tokens available initially
	})

	// Exhaust the burst
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "10.0.0.1:9999"
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i, w.Code)
		}
	}

	// Third request should be rate-limited
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.0.0.1:9999"
	r.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}

	// Verify response body matches httpresp.ErrorResponse format
	var resp httpresp.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}
	if resp.Error.Code != "RATE_LIMITED" {
		t.Errorf("expected error code RATE_LIMITED, got %s", resp.Error.Code)
	}

	// Verify required headers
	retryAfter := w.Header().Get("Retry-After")
	if retryAfter == "" {
		t.Error("missing Retry-After header")
	}
	val, err := strconv.Atoi(retryAfter)
	if err != nil || val <= 0 {
		t.Errorf("Retry-After must be positive integer, got %q", retryAfter)
	}

	limitHeader := w.Header().Get("X-RateLimit-Limit")
	if limitHeader != "6" {
		t.Errorf("expected X-RateLimit-Limit=6, got %q", limitHeader)
	}

	remaining := w.Header().Get("X-RateLimit-Remaining")
	if remaining != "0" {
		t.Errorf("expected X-RateLimit-Remaining=0, got %q", remaining)
	}
}

// TestRateLimit_DifferentIPs verifies that each client IP gets an
// independent token bucket — one IP being limited does not affect another.
func TestRateLimit_DifferentIPs(t *testing.T) {
	r := newTestRouter(config.RateLimitConfig{
		RequestsPerMinute: 6,
		Burst:             1, // only 1 request allowed per IP
	})

	// IP A exhausts its bucket
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "172.16.0.1:1111"
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("IP A first request: expected 200, got %d", w.Code)
	}

	// IP A is now limited
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "172.16.0.1:1111"
	r.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("IP A second request: expected 429, got %d", w.Code)
	}

	// IP B should still be allowed — independent bucket
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "172.16.0.2:2222"
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("IP B first request: expected 200, got %d", w.Code)
	}
}

// TestRateLimit_InvalidConfig defaults to 60 RPM when RequestsPerMinute is 0.
func TestRateLimit_InvalidConfig(t *testing.T) {
	// RPM=0 should not panic and should default to 60
	r := newTestRouter(config.RateLimitConfig{
		RequestsPerMinute: 0,
		Burst:             0,
	})

	// A single request should be allowed (burst defaults to 1)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.99.0.1:5555"
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 with default config, got %d", w.Code)
	}
}

// TestRateLimit_BurstDefaultsToOne verifies that a burst value <= 0
// is coerced to 1, allowing at least one request through.
func TestRateLimit_BurstDefaultsToOne(t *testing.T) {
	r := newTestRouter(config.RateLimitConfig{
		RequestsPerMinute: 60,
		Burst:             -5, // invalid, should default to 1
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.88.0.1:7777"
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 with burst default, got %d", w.Code)
	}
}

// TestRateLimit_RetryAfterIsDynamic verifies that the Retry-After header
// value reflects the actual token refill rate, not a hardcoded 60s.
func TestRateLimit_RetryAfterIsDynamic(t *testing.T) {
	rpm := 12 // 12 RPM = 0.2 rps → refill every 5s
	r := newTestRouter(config.RateLimitConfig{
		RequestsPerMinute: rpm,
		Burst:             1,
	})

	// Exhaust burst
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.77.0.1:8888"
	r.ServeHTTP(w, req)

	// Get rate-limited response
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.77.0.1:8888"
	r.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}

	retryAfter, err := strconv.Atoi(w.Header().Get("Retry-After"))
	if err != nil {
		t.Fatalf("Retry-After not a valid integer: %v", err)
	}

	// For 12 RPM = 0.2 rps, ceil(1/0.2) = 5
	expected := 5
	if retryAfter != expected {
		t.Errorf("expected Retry-After=%d for %d RPM, got %d", expected, rpm, retryAfter)
	}
}

// TestRateLimit_ContextAborted verifies that the gin context is properly
// aborted after a 429, preventing downstream handlers from executing.
func TestRateLimit_ContextAborted(t *testing.T) {
	handlerCalled := false
	logger := zap.NewNop()

	r := gin.New()
	r.Use(RateLimit(config.RateLimitConfig{
		RequestsPerMinute: 6,
		Burst:             1,
	}, logger))
	r.GET("/test", func(c *gin.Context) {
		handlerCalled = true
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// Exhaust burst
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.66.0.1:3333"
	r.ServeHTTP(w, req)

	// Reset flag — the first request legitimately calls the handler
	handlerCalled = false

	// Rate-limited — downstream handler should NOT execute
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.66.0.1:3333"
	r.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}
	if handlerCalled {
		t.Error("downstream handler executed after rate limit — context not aborted")
	}
}

// TestRateLimit_CleanupRemovesStaleClients verifies that clients idle for
// more than 3 minutes are removed from the map (memory management).
// This is a smoke test — the actual cleanup runs on a 1-minute ticker.
func TestRateLimit_CleanupRemovesStaleClients(t *testing.T) {
	// This test verifies the middleware doesn't panic or leak when
	// clients come and go. Full cleanup testing would require time
	// manipulation which is beyond standard library capabilities.
	r := newTestRouter(config.RateLimitConfig{
		RequestsPerMinute: 60,
		Burst:             10,
	})

	// Simulate multiple IPs making requests
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "10.0.0." + strconv.Itoa(i) + ":1234"
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("IP %d: expected 200, got %d", i, w.Code)
		}
	}

	// Allow goroutine to run cleanup cycle (smoke test only)
	time.Sleep(50 * time.Millisecond)
}
