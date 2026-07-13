package systemconfig

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// setEnv sets an env var and registers a cleanup to restore the previous value.
func setEnv(t *testing.T, key, value string) {
	t.Helper()
	prev, existed := os.LookupEnv(key)
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("failed to set env %s: %v", key, err)
	}
	t.Cleanup(func() {
		if existed {
			os.Setenv(key, prev)
		} else {
			os.Unsetenv(key)
		}
	})
}

// unsetEnv unsets an env var and registers a cleanup to restore the previous value.
func unsetEnv(t *testing.T, key string) {
	t.Helper()
	prev, existed := os.LookupEnv(key)
	os.Unsetenv(key)
	t.Cleanup(func() {
		if existed {
			os.Setenv(key, prev)
		} else {
			os.Unsetenv(key)
		}
	})
}

// newTestConfig creates an EffectiveConfig with initialized Sources map.
func newTestConfig() *EffectiveConfig {
	return &EffectiveConfig{
		Sources: make(map[string]ConfigSource),
	}
}

// newTestResolver creates a Resolver with a no-op logger and a minimal yamlCfg.
func newTestResolver() *Resolver {
	return &Resolver{
		yamlCfg: testYAMLConfig(),
	}
}

// testYAMLConfig returns a fully populated YAMLConfig for use in tests.
func testYAMLConfig() *YAMLConfig {
	return &YAMLConfig{
		Application: YAMLApplication{
			ApprovalTiers: YAMLApprovalTiers{
				AutoApply: YAMLTierDef{MinScore: 95, MaxScore: 100, Action: "auto_apply", Notify: true},
				Review:    YAMLTierDef{MinScore: 80, MaxScore: 94, Action: "require_review"},
				Reject:    YAMLTierDef{MaxScore: 79, Action: "reject", Log: true},
			},
			AutoGenerate: YAMLAutoGenerate{Resume: true, CoverLetter: true},
			Resume:       YAMLResume{Engine: "latex", TemplateDir: "templates/resume"},
			CoverLetter:  YAMLCoverLetter{Engine: "latex", TemplateDir: "templates/cover_letter", MaxLength: 500},
		},
		Queue: YAMLQueue{Concurrency: 4, RetryAttempts: 3},
		LLM: YAMLLLM{
			Primary:    YAMLLLMProvider{Provider: "openai", Model: "gpt-4o"},
			Local:      YAMLLLMProvider{Provider: "ollama", Model: "qwen2.5"},
			Embeddings: YAMLLLMProvider{Provider: "openai", Model: "text-embedding-3-small"},
		},
		Voice: YAMLVoice{
			Provider: "deepgram",
			Model:    "deepgram-2",
			LiveKit:  YAMLLiveKit{URL: "wss://livekit.example.com", APIKey: "test-key", APISecret: "test-secret"},
		},
		Interview: YAMLInterview{
			Memory:    YAMLInterviewMemory{MaxRecentSegments: 50, KeepAfterSummarize: 10},
			Responder: YAMLInterviewResponder{LLM: YAMLLLMTimeout{TimeoutMs: 30000, Retries: 3}},
			Planner:   YAMLInterviewPlanner{DuplicateThreshold: 0.85, MinSubstantiveLength: 20},
		},
		Email: YAMLEmail{
			Provider:      "graphapi",
			CheckInterval: "5m",
			Folders:       []string{"INBOX", "JOBS"},
		},
	}
}

// ---------------------------------------------------------------------------
// applyEnvOverrides — integer env vars
// ---------------------------------------------------------------------------

func TestApplyEnvOverrides_IntValues(t *testing.T) {
	setEnv(t, "SCORING_AUTO_THRESHOLD", "90")
	setEnv(t, "SCORING_REVIEW_THRESHOLD", "75")
	setEnv(t, "SCORING_HYBRID_REJECT_MARGIN", "15")
	setEnv(t, "QUEUE_CONCURRENCY", "8")
	setEnv(t, "RATE_LIMIT_RPM", "100")
	setEnv(t, "RATE_LIMIT_BURST", "25")
	setEnv(t, "INTERVIEW_MEMORY_MAX_RECENT", "100")
	setEnv(t, "INTERVIEW_RESPONDER_TIMEOUT_MS", "60000")

	r := newTestResolver()
	cfg := newTestConfig()
	r.applyEnvOverrides(cfg)

	assert.Equal(t, 90, cfg.Scoring.AutoThreshold)
	assert.Equal(t, 75, cfg.Scoring.ReviewThreshold)
	assert.Equal(t, 15, cfg.Scoring.HybridRejectMargin)
	assert.Equal(t, 8, cfg.Automation.Queue.Concurrency)
	assert.Equal(t, 100, cfg.RateLimits.RPM)
	assert.Equal(t, 25, cfg.RateLimits.Burst)
	assert.Equal(t, 100, cfg.Interview.Memory.MaxRecentSegments)
	assert.Equal(t, 60000, cfg.Interview.Responder.LLM.TimeoutMs)
}

// ---------------------------------------------------------------------------
// applyEnvOverrides — string env vars
// ---------------------------------------------------------------------------

func TestApplyEnvOverrides_StringValues(t *testing.T) {
	setEnv(t, "SCORING_MODE", "heuristic")
	setEnv(t, "LLM_PRIMARY_PROVIDER", "anthropic")
	setEnv(t, "OPENAI_MODEL", "gpt-4o-mini")
	setEnv(t, "OLLAMA_MODEL", "llama3")
	setEnv(t, "OLLAMA_EMBED_MODEL", "nomic-embed-text")
	setEnv(t, "EMAIL_CHECK_INTERVAL", "10m")
	setEnv(t, "EMAIL_FOLDERS", "INBOX,PROJECTS,ARCHIVE")

	r := newTestResolver()
	cfg := newTestConfig()
	r.applyEnvOverrides(cfg)

	assert.Equal(t, ScoringMode("heuristic"), cfg.Scoring.Mode)
	assert.Equal(t, "anthropic", cfg.LLM.Primary.Provider)
	assert.Equal(t, "gpt-4o-mini", cfg.LLM.Primary.Model)
	assert.Equal(t, "llama3", cfg.LLM.Local.Model)
	assert.Equal(t, "nomic-embed-text", cfg.LLM.Embeddings.Model)
	assert.Equal(t, "10m", cfg.Email.CheckInterval)
	assert.Equal(t, []string{"INBOX", "PROJECTS", "ARCHIVE"}, cfg.Email.Folders)
}

// ---------------------------------------------------------------------------
// applyEnvOverrides — float env vars
// ---------------------------------------------------------------------------

func TestApplyEnvOverrides_FloatValues(t *testing.T) {
	setEnv(t, "SCORING_WEIGHT_SKILL", "0.40")
	setEnv(t, "SCORING_WEIGHT_EXPERIENCE", "0.30")
	setEnv(t, "SCORING_WEIGHT_LOCATION", "0.10")
	setEnv(t, "SCORING_WEIGHT_SALARY", "0.10")
	setEnv(t, "SCORING_WEIGHT_DESCRIPTION", "0.10")
	setEnv(t, "INTERVIEW_PLANNER_DUPLICATE_THRESHOLD", "0.95")

	r := newTestResolver()
	cfg := newTestConfig()
	r.applyEnvOverrides(cfg)

	assert.InDelta(t, 0.40, cfg.Scoring.Weights.Skill, 0.0001)
	assert.InDelta(t, 0.30, cfg.Scoring.Weights.Experience, 0.0001)
	assert.InDelta(t, 0.10, cfg.Scoring.Weights.Location, 0.0001)
	assert.InDelta(t, 0.10, cfg.Scoring.Weights.Salary, 0.0001)
	assert.InDelta(t, 0.10, cfg.Scoring.Weights.Description, 0.0001)
	assert.InDelta(t, 0.95, cfg.Interview.Planner.DuplicateThreshold, 0.0001)
}

// ---------------------------------------------------------------------------
// applyEnvOverrides — empty env vars are no-ops
// ---------------------------------------------------------------------------

func TestApplyEnvOverrides_EmptyEnvVars(t *testing.T) {
	// Ensure all relevant env vars are empty
	envKeys := []string{
		"SCORING_AUTO_THRESHOLD", "SCORING_REVIEW_THRESHOLD", "SCORING_MODE",
		"SCORING_HYBRID_REJECT_MARGIN", "SCORING_WEIGHT_SKILL",
		"LLM_PRIMARY_PROVIDER", "OPENAI_MODEL",
		"QUEUE_CONCURRENCY",
		"RATE_LIMIT_RPM", "RATE_LIMIT_BURST",
		"EMAIL_CHECK_INTERVAL", "EMAIL_FOLDERS",
		"INTERVIEW_MEMORY_MAX_RECENT", "INTERVIEW_RESPONDER_TIMEOUT_MS",
		"INTERVIEW_PLANNER_DUPLICATE_THRESHOLD",
	}
	for _, k := range envKeys {
		unsetEnv(t, k)
	}

	r := newTestResolver()
	cfg := newTestConfig()
	r.applyEnvOverrides(cfg)

	// All values should remain at their zero values (not set by env)
	assert.Equal(t, 0, cfg.Scoring.AutoThreshold)
	assert.Equal(t, 0, cfg.Scoring.ReviewThreshold)
	assert.Equal(t, ScoringMode(""), cfg.Scoring.Mode)
	assert.Equal(t, 0, cfg.Scoring.HybridRejectMargin)
	assert.InDelta(t, 0.0, cfg.Scoring.Weights.Skill, 0.0001)
	assert.Equal(t, "", cfg.LLM.Primary.Provider)
	assert.Equal(t, "", cfg.LLM.Primary.Model)
	assert.Equal(t, 0, cfg.Automation.Queue.Concurrency)
	assert.Equal(t, 0, cfg.RateLimits.RPM)
	assert.Equal(t, 0, cfg.RateLimits.Burst)
	assert.Equal(t, "", cfg.Email.CheckInterval)
	assert.Nil(t, cfg.Email.Folders)
	assert.Equal(t, 0, cfg.Interview.Memory.MaxRecentSegments)
	assert.Equal(t, 0, cfg.Interview.Responder.LLM.TimeoutMs)
	assert.InDelta(t, 0.0, cfg.Interview.Planner.DuplicateThreshold, 0.0001)

	// Sources should be empty — no env vars applied
	assert.Empty(t, cfg.Sources)
}

// ---------------------------------------------------------------------------
// applyEnvOverrides — invalid int values silently ignored
// ---------------------------------------------------------------------------

func TestApplyEnvOverrides_InvalidIntValues(t *testing.T) {
	setEnv(t, "SCORING_AUTO_THRESHOLD", "not-a-number")
	setEnv(t, "SCORING_HYBRID_REJECT_MARGIN", "12.5") // float, not int
	setEnv(t, "RATE_LIMIT_RPM", "1e2")                // scientific, not int

	r := newTestResolver()
	cfg := newTestConfig()
	cfg.Scoring.AutoThreshold = 50 // set a baseline
	cfg.Scoring.HybridRejectMargin = 10
	cfg.RateLimits.RPM = 30

	r.applyEnvOverrides(cfg)

	// Values should remain at baseline because env var values are invalid
	assert.Equal(t, 50, cfg.Scoring.AutoThreshold, "should keep baseline when env var is invalid")
	assert.Equal(t, 10, cfg.Scoring.HybridRejectMargin, "float in int env var should be ignored")
	assert.Equal(t, 30, cfg.RateLimits.RPM, "scientific notation should be ignored")

	// Sources should NOT have entries for failed parses
	assert.NotContains(t, cfg.Sources, "scoring.auto_threshold")
	assert.NotContains(t, cfg.Sources, "scoring.hybrid_reject_margin")
	assert.NotContains(t, cfg.Sources, "rate_limits.rpm")
}

// ---------------------------------------------------------------------------
// applyEnvOverrides — invalid float values silently ignored
// ---------------------------------------------------------------------------

func TestApplyEnvOverrides_InvalidFloatValues(t *testing.T) {
	setEnv(t, "SCORING_WEIGHT_SKILL", "abc")
	setEnv(t, "INTERVIEW_PLANNER_DUPLICATE_THRESHOLD", "")

	r := newTestResolver()
	cfg := newTestConfig()
	cfg.Scoring.Weights.Skill = 0.5
	cfg.Interview.Planner.DuplicateThreshold = 0.5

	r.applyEnvOverrides(cfg)

	assert.InDelta(t, 0.5, cfg.Scoring.Weights.Skill, 0.0001, "invalid float should be ignored")
	assert.InDelta(t, 0.5, cfg.Interview.Planner.DuplicateThreshold, 0.0001, "empty env var should be ignored")
}

// ---------------------------------------------------------------------------
// applyEnvOverrides — Source tracking
// ---------------------------------------------------------------------------

func TestApplyEnvOverrides_SourceTracking(t *testing.T) {
	setEnv(t, "SCORING_AUTO_THRESHOLD", "95")
	setEnv(t, "LLM_PRIMARY_PROVIDER", "openai")
	setEnv(t, "EMAIL_CHECK_INTERVAL", "10m")

	r := newTestResolver()
	cfg := newTestConfig()
	r.applyEnvOverrides(cfg)

	assert.Equal(t, SourceEnv, cfg.Sources["scoring.auto_threshold"])
	assert.Equal(t, SourceEnv, cfg.Sources["llm.primary.provider"])
	assert.Equal(t, SourceEnv, cfg.Sources["email.check_interval"])
	assert.Len(t, cfg.Sources, 3, "should have exactly 3 source entries")
}

// ---------------------------------------------------------------------------
// applyEnvOverrides — non-overridden values not in sources
// ---------------------------------------------------------------------------

func TestApplyEnvOverrides_NonOverriddenNotInSources(t *testing.T) {
	setEnv(t, "SCORING_AUTO_THRESHOLD", "95")

	r := newTestResolver()
	cfg := newTestConfig()
	r.applyEnvOverrides(cfg)

	assert.Equal(t, SourceEnv, cfg.Sources["scoring.auto_threshold"])
	assert.NotContains(t, cfg.Sources, "scoring.review_threshold", "unset env vars should not appear in sources")
	assert.NotContains(t, cfg.Sources, "llm.primary.provider")
}
