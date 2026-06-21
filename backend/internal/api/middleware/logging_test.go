package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// newLoggedRouter creates a gin.Engine with the Logging middleware and a
// capturing logger. Returns the engine and the observed log entries.
func newLoggedRouter() (*gin.Engine, *observer.ObservedLogs) {
	core, recorded := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)

	r := gin.New()
	r.Use(Logging(logger))
	return r, recorded
}

// TestLogging SuccessfulRequest verifies that a 200 request produces a log
// entry at Info level with the expected fields.
func TestLogging_SuccessfulRequest(t *testing.T) {
	r, recorded := newLoggedRouter()
	r.GET("/api/v1/jobs", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"jobs": []string{}})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/jobs?page=1", nil)
	req.RemoteAddr = "192.168.1.10:54321"
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	entries := recorded.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Level != zapcore.InfoLevel {
		t.Errorf("expected Info level, got %s", entry.Level)
	}
	if entry.Message != "request" {
		t.Errorf("expected message 'request', got %q", entry.Message)
	}

	// Verify key fields exist
	fields := entry.ContextMap()
	assertField(t, fields, "status", int64(200))
	assertField(t, fields, "method", "GET")
	assertField(t, fields, "path", "/api/v1/jobs")
	assertField(t, fields, "query", "page=1")
}

// TestLogging_SkipsHealthCheck verifies that the /health path produces
// no log output — it passes through without logging.
func TestLogging_SkipsHealthCheck(t *testing.T) {
	r, recorded := newLoggedRouter()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	if len(recorded.All()) != 0 {
		t.Errorf("expected no log entries for /health, got %d", len(recorded.All()))
	}
}

// TestLogging_FiveHundredLogsAtError verifies that 5xx responses produce
// log entries at Error level.
func TestLogging_FiveHundredLogsAtError(t *testing.T) {
	r, recorded := newLoggedRouter()
	r.GET("/crash", func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "boom"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/crash", nil)
	r.ServeHTTP(w, req)

	entries := recorded.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}
	if entries[0].Level != zapcore.ErrorLevel {
		t.Errorf("expected Error level for 5xx, got %s", entries[0].Level)
	}
}

// TestLogging_FourHundredLogsAtWarn verifies that 4xx responses produce
// log entries at Warn level.
func TestLogging_FourHundredLogsAtWarn(t *testing.T) {
	r, recorded := newLoggedRouter()
	r.GET("/notfound", func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/notfound", nil)
	r.ServeHTTP(w, req)

	entries := recorded.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}
	if entries[0].Level != zapcore.WarnLevel {
		t.Errorf("expected Warn level for 4xx, got %s", entries[0].Level)
	}
}

// TestLogging_ErrorFieldPresent verifies that when a gin.Error is attached
// to the context, the "error" field appears in the log entry.
func TestLogging_ErrorFieldPresent(t *testing.T) {
	r, recorded := newLoggedRouter()
	r.GET("/with-error", func(c *gin.Context) {
		c.Error(gin.Error{
			Err:  nil,
			Type: gin.ErrorTypePrivate,
			Meta: "something went wrong",
		})
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/with-error", nil)
	r.ServeHTTP(w, req)

	entries := recorded.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}

	// Check that error field exists in context
	fields := entries[0].ContextMap()
	if _, exists := fields["error"]; !exists {
		t.Error("expected 'error' field in log entry when gin.Error is attached")
	}
}

// TestLogging_LatencyNonNegative verifies the latency field is present
// and non-negative in the log entry.
func TestLogging_LatencyNonNegative(t *testing.T) {
	r, recorded := newLoggedRouter()
	r.GET("/fast", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/fast", nil)
	r.ServeHTTP(w, req)

	entries := recorded.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}

	fields := entries[0].ContextMap()
	latency, exists := fields["latency"]
	if !exists {
		t.Fatal("expected 'latency' field in log entry")
	}
	// zapcore stores duration as float64 nanoseconds
	if ns, ok := latency.(float64); ok {
		if ns < 0 {
			t.Errorf("latency must be non-negative, got %f", ns)
		}
	}
}

// assertField is a test helper that checks a field exists in the map
// with the expected value.
func assertField(t *testing.T, fields map[string]interface{}, key string, expected interface{}) {
	t.Helper()
	val, exists := fields[key]
	if !exists {
		t.Errorf("missing field %q", key)
		return
	}
	if val != expected {
		t.Errorf("field %q: expected %v (%T), got %v (%T)", key, expected, expected, val, val)
	}
}
