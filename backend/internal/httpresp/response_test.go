package httpresp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// executeHandler is a test helper that runs a gin.HandlerFunc through
// a httptest recorder and returns the status code and parsed body.
func executeHandler(t *testing.T, fn gin.HandlerFunc) (int, ErrorResponse) {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	fn(c)
	var resp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}
	return w.Code, resp
}

// executeDataHandler is like executeHandler but for success helpers
// that return arbitrary JSON (not ErrorResponse).
func executeDataHandler(t *testing.T, fn gin.HandlerFunc) (int, map[string]interface{}) {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	fn(c)
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}
	return w.Code, resp
}

// --- Success helpers (return arbitrary JSON) ---

func TestOK_Returns200(t *testing.T) {
	code, body := executeDataHandler(t, func(c *gin.Context) {
		OK(c, gin.H{"jobs": []string{"a", "b"}})
	})
	if code != http.StatusOK {
		t.Errorf("expected 200, got %d", code)
	}
	jobs, ok := body["jobs"].([]interface{})
	if !ok || len(jobs) != 2 {
		t.Errorf("expected jobs array with 2 elements, got %v", body["jobs"])
	}
}

func TestCreated_Returns201(t *testing.T) {
	code, body := executeDataHandler(t, func(c *gin.Context) {
		Created(c, gin.H{"id": "abc-123"})
	})
	if code != http.StatusCreated {
		t.Errorf("expected 201, got %d", code)
	}
	if body["id"] != "abc-123" {
		t.Errorf("expected id=abc-123, got %v", body["id"])
	}
}

func TestAccepted_Returns202(t *testing.T) {
	code, body := executeDataHandler(t, func(c *gin.Context) {
		Accepted(c, gin.H{"taskId": "task-456"})
	})
	if code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", code)
	}
	if body["taskId"] != "task-456" {
		t.Errorf("expected taskId=task-456, got %v", body["taskId"])
	}
}

func TestMultiStatus_Returns207(t *testing.T) {
	code, _ := executeDataHandler(t, func(c *gin.Context) {
		MultiStatus(c, gin.H{"succeeded": 3, "failed": 1})
	})
	if code != http.StatusMultiStatus {
		t.Errorf("expected 207, got %d", code)
	}
}

// --- Error helpers (return ErrorResponse with code + message) ---

func TestBadRequest_Returns400(t *testing.T) {
	code, resp := executeHandler(t, func(c *gin.Context) {
		BadRequest(c, "INVALID_INPUT", "email is required")
	})
	if code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", code)
	}
	if resp.Error.Code != "INVALID_INPUT" {
		t.Errorf("expected code=INVALID_INPUT, got %s", resp.Error.Code)
	}
	if resp.Error.Message != "email is required" {
		t.Errorf("expected message='email is required', got %s", resp.Error.Message)
	}
}

func TestNotFound_Returns404(t *testing.T) {
	code, resp := executeHandler(t, func(c *gin.Context) {
		NotFound(c, "NOT_FOUND", "job not found")
	})
	if code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", code)
	}
	if resp.Error.Code != "NOT_FOUND" {
		t.Errorf("expected code=NOT_FOUND, got %s", resp.Error.Code)
	}
}

func TestUnauthorized_Returns401(t *testing.T) {
	code, resp := executeHandler(t, func(c *gin.Context) {
		Unauthorized(c, "UNAUTHORIZED", "invalid token")
	})
	if code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", code)
	}
	if resp.Error.Code != "UNAUTHORIZED" {
		t.Errorf("expected code=UNAUTHORIZED, got %s", resp.Error.Code)
	}
}

func TestInternalError_Returns500(t *testing.T) {
	code, resp := executeHandler(t, func(c *gin.Context) {
		InternalError(c)
	})
	if code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", code)
	}
	if resp.Error.Code != "INTERNAL_ERROR" {
		t.Errorf("expected code=INTERNAL_ERROR, got %s", resp.Error.Code)
	}
	if resp.Error.Message != "internal error" {
		t.Errorf("expected message='internal error', got %s", resp.Error.Message)
	}
}

func TestTooManyRequests_Returns429(t *testing.T) {
	code, resp := executeHandler(t, func(c *gin.Context) {
		TooManyRequests(c, "RATE_LIMITED", "slow down")
	})
	if code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", code)
	}
	if resp.Error.Code != "RATE_LIMITED" {
		t.Errorf("expected code=RATE_LIMITED, got %s", resp.Error.Code)
	}
	if resp.Error.Message != "slow down" {
		t.Errorf("expected message='slow down', got %s", resp.Error.Message)
	}
}

func TestConflict_Returns409(t *testing.T) {
	code, resp := executeHandler(t, func(c *gin.Context) {
		Conflict(c, "VERSION_CONFLICT", "resource was modified")
	})
	if code != http.StatusConflict {
		t.Errorf("expected 409, got %d", code)
	}
	if resp.Error.Code != "VERSION_CONFLICT" {
		t.Errorf("expected code=VERSION_CONFLICT, got %s", resp.Error.Code)
	}
}

// --- Body structure test ---

func TestErrorResponse_JSONStructure(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	BadRequest(c, "TEST_CODE", "test message")

	// Verify the raw JSON has the correct nested structure: {"error":{"code":"...","message":"..."}}
	var raw map[string]map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("failed to parse as nested JSON: %v", err)
	}
	if raw["error"]["code"] != "TEST_CODE" {
		t.Errorf("nested JSON: expected error.code=TEST_CODE, got %s", raw["error"]["code"])
	}
	if raw["error"]["message"] != "test message" {
		t.Errorf("nested JSON: expected error.message='test message', got %s", raw["error"]["message"])
	}
}
