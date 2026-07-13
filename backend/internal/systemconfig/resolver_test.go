package systemconfig

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// writeTempYAML creates a temporary YAML config file for testing.
func writeTempYAML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "application.yaml")
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err, "failed to write temp YAML")
	return path
}

// minimalYAML is the smallest valid YAML config that the resolver can load.
const minimalYAML = `
application:
  approval_tiers:
    auto_apply:
      min_score: 95
      action: auto_apply
    review:
      min_score: 80
      action: require_review
    reject:
      max_score: 79
      action: reject
  auto_generate:
    resume: true
    cover_letter: true
  resume:
    engine: latex
    template_dir: templates/resume
  cover_letter:
    engine: latex
    template_dir: templates/cover_letter
    max_length: 500
queue:
  concurrency: 4
  retryAttempts: 3
llm:
  primary:
    provider: openai
    model: gpt-4o
  local:
    provider: ollama
    model: qwen2.5
  embeddings:
    provider: openai
    model: text-embedding-3-small
voice:
  provider: deepgram
  model: deepgram-2
  livekit:
    url: wss://livekit.example.com
    api_key: test-key
    api_secret: test-secret
interview:
  memory:
    max_recent_segments: 50
    keep_after_summarize: 10
  responder:
    llm:
      timeout_ms: 30000
      retries: 3
  planner:
    duplicate_threshold: 0.85
    min_substantive_length: 20
email:
  provider: graphapi
  check_interval: 5m
  folders:
    - INBOX
    - JOBS
`

// ---------------------------------------------------------------------------
// NewResolver
// ---------------------------------------------------------------------------

func TestNewResolver_Success(t *testing.T) {
	logger := zap.NewNop()
	yamlPath := writeTempYAML(t, minimalYAML)

	resolver, err := NewResolver(logger, yamlPath)
	require.NoError(t, err)
	require.NotNil(t, resolver)
	assert.NotNil(t, resolver.yamlCfg)
	assert.Equal(t, yamlPath, resolver.yamlPath)
}

func TestNewResolver_FileNotFound(t *testing.T) {
	logger := zap.NewNop()

	resolver, err := NewResolver(logger, "/nonexistent/path/application.yaml")
	require.Error(t, err)
	assert.Nil(t, resolver)
	assert.Contains(t, err.Error(), "systemconfig: load yaml")
	assert.Contains(t, err.Error(), "read yaml file")
}

func TestNewResolver_InvalidYAML(t *testing.T) {
	logger := zap.NewNop()
	yamlPath := writeTempYAML(t, `invalid: [{unclosed bracket`)

	resolver, err := NewResolver(logger, yamlPath)
	require.Error(t, err)
	assert.Nil(t, resolver)
	assert.Contains(t, err.Error(), "systemconfig: parse yaml")
}

func TestNewResolver_EmptyYAML(t *testing.T) {
	logger := zap.NewNop()
	yamlPath := writeTempYAML(t, ``)

	resolver, err := NewResolver(logger, yamlPath)
	require.NoError(t, err, "empty YAML is valid, should not error")
	require.NotNil(t, resolver)
	// All fields should be zero-valued
	assert.Equal(t, 0, resolver.yamlCfg.Queue.Concurrency)
	assert.Equal(t, "", resolver.yamlCfg.LLM.Primary.Provider)
}

// ---------------------------------------------------------------------------
// buildFromYAML
// ---------------------------------------------------------------------------

func TestBuildFromYAML(t *testing.T) {
	r := newTestResolver()
	cfg := r.buildFromYAML()

	require.NotNil(t, cfg)
	assert.NotNil(t, cfg.Sources, "Sources map should be initialized")

	// Verify scoring defaults
	assert.Equal(t, 95, cfg.Scoring.AutoThreshold)
	assert.Equal(t, 80, cfg.Scoring.ReviewThreshold)
	assert.Equal(t, ModeHybrid, cfg.Scoring.Mode, "mode should default to hybrid")
	assert.Equal(t, 20, cfg.Scoring.HybridRejectMargin, "hybrid reject margin should default to 20")

	// Verify scoring weights defaults
	assert.InDelta(t, 0.35, cfg.Scoring.Weights.Skill, 0.0001)
	assert.InDelta(t, 0.25, cfg.Scoring.Weights.Experience, 0.0001)
	assert.InDelta(t, 0.10, cfg.Scoring.Weights.Location, 0.0001)
	assert.InDelta(t, 0.15, cfg.Scoring.Weights.Salary, 0.0001)
	assert.InDelta(t, 0.15, cfg.Scoring.Weights.Description, 0.0001)

	// Verify LLM section
	assert.Equal(t, "openai", cfg.LLM.Primary.Provider)
	assert.Equal(t, "gpt-4o", cfg.LLM.Primary.Model)
	assert.Equal(t, "ollama", cfg.LLM.Local.Provider)
	assert.Equal(t, "qwen2.5", cfg.LLM.Local.Model)
	assert.Equal(t, "openai", cfg.LLM.Embeddings.Provider)
	assert.Equal(t, "text-embedding-3-small", cfg.LLM.Embeddings.Model)

	// Verify Voice section
	assert.Equal(t, "deepgram", cfg.Voice.Provider)
	assert.Equal(t, "deepgram-2", cfg.Voice.Model)
	assert.Equal(t, "wss://livekit.example.com", cfg.Voice.LiveKit.URL)
	assert.Equal(t, "test-key", cfg.Voice.LiveKit.APIKey)

	// Verify ApprovalTiers
	assert.Equal(t, 95, cfg.ApprovalTiers.AutoApply.MinScore)
	assert.Equal(t, "auto_apply", cfg.ApprovalTiers.AutoApply.Action)
	assert.True(t, cfg.ApprovalTiers.AutoApply.Notify)
	assert.Equal(t, 80, cfg.ApprovalTiers.Review.MinScore)
	assert.Equal(t, 94, cfg.ApprovalTiers.Review.MaxScore)
	assert.Equal(t, "require_review", cfg.ApprovalTiers.Review.Action)
	assert.Equal(t, 79, cfg.ApprovalTiers.Reject.MaxScore)
	assert.Equal(t, "reject", cfg.ApprovalTiers.Reject.Action)
	assert.True(t, cfg.ApprovalTiers.Reject.Log)

	// Verify Resume & CoverLetter
	assert.Equal(t, "latex", cfg.ResumeConfig.Engine)
	assert.Equal(t, "templates/resume", cfg.ResumeConfig.TemplateDir)
	assert.Equal(t, "latex", cfg.CoverLetterConfig.Engine)
	assert.Equal(t, "templates/cover_letter", cfg.CoverLetterConfig.TemplateDir)
	assert.Equal(t, 500, cfg.CoverLetterConfig.MaxLength)

	// Verify Automation
	assert.Equal(t, 4, cfg.Automation.Queue.Concurrency)
	assert.Equal(t, 3, cfg.Automation.Queue.RetryAttempts)
	assert.True(t, cfg.Automation.AutoGenerate.Resume)
	assert.True(t, cfg.Automation.AutoGenerate.CoverLetter)

	// Verify Interview
	assert.Equal(t, 50, cfg.Interview.Memory.MaxRecentSegments)
	assert.Equal(t, 10, cfg.Interview.Memory.KeepAfterSummarize)
	assert.Equal(t, 30000, cfg.Interview.Responder.LLM.TimeoutMs)
	assert.Equal(t, 3, cfg.Interview.Responder.LLM.Retries)
	assert.InDelta(t, 0.85, cfg.Interview.Planner.DuplicateThreshold, 0.0001)
	assert.Equal(t, 20, cfg.Interview.Planner.MinSubstantiveLength)

	// Verify Email
	assert.Equal(t, "graphapi", cfg.Email.Provider)
	assert.Equal(t, "5m", cfg.Email.CheckInterval)
	assert.Equal(t, []string{"INBOX", "JOBS"}, cfg.Email.Folders)

	// Verify RateLimits defaults
	assert.Equal(t, 60, cfg.RateLimits.RPM)
	assert.Equal(t, 10, cfg.RateLimits.Burst)

	// Verify Sources map is empty after build (Sources set during env/DB apply)
	assert.Empty(t, cfg.Sources)
}

// ---------------------------------------------------------------------------
// Resolve — without repo (YAML + env only)
// ---------------------------------------------------------------------------

func TestResolve_WithoutRepo(t *testing.T) {
	logger := zap.NewNop()
	yamlPath := writeTempYAML(t, minimalYAML)

	resolver, err := NewResolver(logger, yamlPath)
	require.NoError(t, err)

	ctx := context.Background()
	cfg, err := resolver.Resolve(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify YAML values are present
	assert.Equal(t, "openai", cfg.LLM.Primary.Provider)
	assert.Equal(t, "latex", cfg.ResumeConfig.Engine)
	assert.Equal(t, 95, cfg.Scoring.AutoThreshold)

	// Verify defaults for values not in YAML
	assert.Equal(t, ModeHybrid, cfg.Scoring.Mode)
	assert.Equal(t, 60, cfg.RateLimits.RPM)
	assert.Equal(t, 10, cfg.RateLimits.Burst)

	// Sources should be empty (not set by buildFromYAML, only by env/DB)
	assert.Empty(t, cfg.Sources)

	// Integrations should be present (always resolved)
	assert.NotNil(t, cfg.Integrations.AIProviders)
	assert.NotEmpty(t, cfg.Integrations.LiveKit.Status)
	assert.NotEmpty(t, cfg.Integrations.Email.Status)
}

// ---------------------------------------------------------------------------
// Resolve — with env overrides, without repo
// ---------------------------------------------------------------------------

func TestResolve_WithEnvOverrides(t *testing.T) {
	// Set env vars for override testing
	setEnv(t, "SCORING_AUTO_THRESHOLD", "85")
	setEnv(t, "SCORING_MODE", "heuristic")
	setEnv(t, "QUEUE_CONCURRENCY", "16")

	logger := zap.NewNop()
	yamlPath := writeTempYAML(t, minimalYAML)

	resolver, err := NewResolver(logger, yamlPath)
	require.NoError(t, err)

	ctx := context.Background()
	cfg, err := resolver.Resolve(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// YAML-origin values
	assert.Equal(t, "latex", cfg.ResumeConfig.Engine)

	// Env-overridden values
	assert.Equal(t, 85, cfg.Scoring.AutoThreshold)
	assert.Equal(t, ScoringMode("heuristic"), cfg.Scoring.Mode)
	assert.Equal(t, 16, cfg.Automation.Queue.Concurrency)

	// Source tracking
	assert.Equal(t, SourceEnv, cfg.Sources["scoring.auto_threshold"])
	assert.Equal(t, SourceEnv, cfg.Sources["scoring.mode"])
	assert.Equal(t, SourceEnv, cfg.Sources["automation.queue.concurrency"])
}

// ---------------------------------------------------------------------------
// Resolve — with invalid YAML path (error propagation)
// ---------------------------------------------------------------------------

func TestResolve_ErrorOnLoad(t *testing.T) {
	logger := zap.NewNop()

	_, err := NewResolver(logger, "/tmp/nonexistent-12345.yaml")
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// buildFromYAML — empty YAML config (zero values)
// ---------------------------------------------------------------------------

func TestBuildFromYAML_EmptyYAML(t *testing.T) {
	r := &Resolver{
		yamlCfg: &YAMLConfig{},
	}
	cfg := r.buildFromYAML()

	require.NotNil(t, cfg)
	// Zero-valued YAML produces zero-valued effective config
	assert.Equal(t, 0, cfg.Scoring.AutoThreshold)
	assert.Equal(t, 0, cfg.Scoring.ReviewThreshold)
	assert.Equal(t, "", cfg.LLM.Primary.Provider)
	assert.Equal(t, 0, cfg.Automation.Queue.Concurrency)
	assert.Equal(t, 0, cfg.Automation.Queue.RetryAttempts)
	assert.Equal(t, "", cfg.Email.Provider)
	assert.Nil(t, cfg.Email.Folders)
	// Defaults that are NOT from YAML should still be set
	assert.Equal(t, ModeHybrid, cfg.Scoring.Mode)
	assert.Equal(t, 20, cfg.Scoring.HybridRejectMargin)
	assert.Equal(t, 60, cfg.RateLimits.RPM)
	assert.Equal(t, 10, cfg.RateLimits.Burst)
}
