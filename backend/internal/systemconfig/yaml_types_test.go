package systemconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// YAMLConfig — marshal/unmarshal round trip
// ---------------------------------------------------------------------------

func TestYAMLConfig_UnmarshalFull(t *testing.T) {
	yamlContent := `
application:
  approval_tiers:
    auto_apply:
      min_score: 95
      max_score: 100
      action: auto_apply
      notify: true
      log: false
    review:
      min_score: 80
      max_score: 94
      action: require_review
      notify: false
      log: false
    reject:
      min_score: 0
      max_score: 79
      action: reject
      notify: false
      log: true
  auto_generate:
    resume: true
    cover_letter: false
  resume:
    engine: latex
    template_dir: templates/resume
  cover_letter:
    engine: latex
    template_dir: templates/cover
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
    api_key: lk-test-key
    api_secret: super-secret
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

	var cfg YAMLConfig
	err := yaml.Unmarshal([]byte(yamlContent), &cfg)
	require.NoError(t, err, "should parse valid YAML without error")

	// Application / ApprovalTiers
	assert.Equal(t, 95, cfg.Application.ApprovalTiers.AutoApply.MinScore)
	assert.Equal(t, 100, cfg.Application.ApprovalTiers.AutoApply.MaxScore)
	assert.Equal(t, "auto_apply", cfg.Application.ApprovalTiers.AutoApply.Action)
	assert.True(t, cfg.Application.ApprovalTiers.AutoApply.Notify)
	assert.False(t, cfg.Application.ApprovalTiers.AutoApply.Log)

	assert.Equal(t, 80, cfg.Application.ApprovalTiers.Review.MinScore)
	assert.Equal(t, 94, cfg.Application.ApprovalTiers.Review.MaxScore)
	assert.Equal(t, "require_review", cfg.Application.ApprovalTiers.Review.Action)

	assert.Equal(t, 0, cfg.Application.ApprovalTiers.Reject.MinScore)
	assert.Equal(t, 79, cfg.Application.ApprovalTiers.Reject.MaxScore)
	assert.Equal(t, "reject", cfg.Application.ApprovalTiers.Reject.Action)
	assert.True(t, cfg.Application.ApprovalTiers.Reject.Log)

	// Application / AutoGenerate
	assert.True(t, cfg.Application.AutoGenerate.Resume)
	assert.False(t, cfg.Application.AutoGenerate.CoverLetter)

	// Application / Resume & CoverLetter
	assert.Equal(t, "latex", cfg.Application.Resume.Engine)
	assert.Equal(t, "templates/resume", cfg.Application.Resume.TemplateDir)
	assert.Equal(t, "latex", cfg.Application.CoverLetter.Engine)
	assert.Equal(t, "templates/cover", cfg.Application.CoverLetter.TemplateDir)
	assert.Equal(t, 500, cfg.Application.CoverLetter.MaxLength)

	// Queue
	assert.Equal(t, 4, cfg.Queue.Concurrency)
	assert.Equal(t, 3, cfg.Queue.RetryAttempts)

	// LLM
	assert.Equal(t, "openai", cfg.LLM.Primary.Provider)
	assert.Equal(t, "gpt-4o", cfg.LLM.Primary.Model)
	assert.Equal(t, "ollama", cfg.LLM.Local.Provider)
	assert.Equal(t, "qwen2.5", cfg.LLM.Local.Model)
	assert.Equal(t, "openai", cfg.LLM.Embeddings.Provider)
	assert.Equal(t, "text-embedding-3-small", cfg.LLM.Embeddings.Model)

	// Voice
	assert.Equal(t, "deepgram", cfg.Voice.Provider)
	assert.Equal(t, "deepgram-2", cfg.Voice.Model)
	assert.Equal(t, "wss://livekit.example.com", cfg.Voice.LiveKit.URL)
	assert.Equal(t, "lk-test-key", cfg.Voice.LiveKit.APIKey)
	assert.Equal(t, "super-secret", cfg.Voice.LiveKit.APISecret)

	// Interview
	assert.Equal(t, 50, cfg.Interview.Memory.MaxRecentSegments)
	assert.Equal(t, 10, cfg.Interview.Memory.KeepAfterSummarize)
	assert.Equal(t, 30000, cfg.Interview.Responder.LLM.TimeoutMs)
	assert.Equal(t, 3, cfg.Interview.Responder.LLM.Retries)
	assert.Equal(t, 0.85, cfg.Interview.Planner.DuplicateThreshold)
	assert.Equal(t, 20, cfg.Interview.Planner.MinSubstantiveLength)

	// Email
	assert.Equal(t, "graphapi", cfg.Email.Provider)
	assert.Equal(t, "5m", cfg.Email.CheckInterval)
	assert.Equal(t, []string{"INBOX", "JOBS"}, cfg.Email.Folders)
}

func TestYAMLConfig_EmptyUnmarshal(t *testing.T) {
	// Empty YAML should produce zero-valued struct
	yamlContent := ``

	var cfg YAMLConfig
	err := yaml.Unmarshal([]byte(yamlContent), &cfg)
	require.NoError(t, err)

	assert.Equal(t, 0, cfg.Application.ApprovalTiers.AutoApply.MinScore)
	assert.Equal(t, "", cfg.LLM.Primary.Provider)
	assert.Equal(t, 0, cfg.Queue.Concurrency)
	assert.Nil(t, cfg.Email.Folders)
}

func TestYAMLConfig_PartialUnmarshal(t *testing.T) {
	yamlContent := `
application:
  resume:
    engine: markdown
queue:
  concurrency: 8
`

	var cfg YAMLConfig
	err := yaml.Unmarshal([]byte(yamlContent), &cfg)
	require.NoError(t, err)

	// Set fields
	assert.Equal(t, "markdown", cfg.Application.Resume.Engine)
	assert.Equal(t, 8, cfg.Queue.Concurrency)

	// Unset fields should be zero-valued
	assert.Equal(t, 0, cfg.Queue.RetryAttempts)
	assert.Equal(t, "", cfg.LLM.Primary.Provider)
	assert.Equal(t, "", cfg.Voice.Provider)
}

func TestYAMLConfig_InvalidYAML(t *testing.T) {
	yamlContent := `invalid: [{unclosed`

	var cfg YAMLConfig
	err := yaml.Unmarshal([]byte(yamlContent), &cfg)
	assert.Error(t, err, "invalid YAML should produce an error")
}

// ---------------------------------------------------------------------------
// YAMLConfig field tags
// ---------------------------------------------------------------------------

func TestYAMLConfig_FieldTags(t *testing.T) {
	// Verify YAML struct tags match expected YAML keys by marshaling a populated
	// struct and checking the output contains the right field names.
	cfg := YAMLConfig{
		Application: YAMLApplication{
			ApprovalTiers: YAMLApprovalTiers{
				AutoApply: YAMLTierDef{MinScore: 95, Action: "auto_apply"},
			},
			AutoGenerate: YAMLAutoGenerate{Resume: true},
			Resume:       YAMLResume{Engine: "latex"},
			CoverLetter:  YAMLCoverLetter{Engine: "latex", MaxLength: 500},
		},
		Queue: YAMLQueue{Concurrency: 4, RetryAttempts: 4},
		LLM: YAMLLLM{
			Primary: YAMLLLMProvider{Provider: "openai"},
		},
		Voice: YAMLVoice{
			Provider: "deepgram",
			LiveKit:  YAMLLiveKit{URL: "wss://example.com"},
		},
		Interview: YAMLInterview{
			Memory: YAMLInterviewMemory{MaxRecentSegments: 50},
		},
		Email: YAMLEmail{
			Provider: "graphapi",
			Folders:  []string{"INBOX"},
		},
	}

	data, err := yaml.Marshal(&cfg)
	require.NoError(t, err)
	output := string(data)

	assert.Contains(t, output, "min_score: 95")
	assert.Contains(t, output, "action: auto_apply")
	assert.Contains(t, output, "resume: true")
	assert.Contains(t, output, "engine: latex")
	assert.Contains(t, output, "max_length: 500")
	assert.Contains(t, output, "concurrency: 4")
	assert.Contains(t, output, "provider: openai")
	assert.Contains(t, output, "provider: deepgram")
	assert.Contains(t, output, "url: wss://example.com")
	assert.Contains(t, output, "max_recent_segments: 50")
	assert.Contains(t, output, "retryAttempts: 4")
	assert.Contains(t, output, "check_interval")
}

// ---------------------------------------------------------------------------
// YAMLTierDef zero values
// ---------------------------------------------------------------------------

func TestYAMLTierDef_ZeroValues(t *testing.T) {
	var def YAMLTierDef
	assert.Equal(t, 0, def.MinScore)
	assert.Equal(t, 0, def.MaxScore)
	assert.Equal(t, "", def.Action)
	assert.False(t, def.Notify)
	assert.False(t, def.Log)
}

// ---------------------------------------------------------------------------
// YAMLInterview types zero values
// ---------------------------------------------------------------------------

func TestYAMLInterviewTypes_ZeroValues(t *testing.T) {
	var mem YAMLInterviewMemory
	assert.Equal(t, 0, mem.MaxRecentSegments)
	assert.Equal(t, 0, mem.KeepAfterSummarize)

	var responder YAMLInterviewResponder
	assert.Equal(t, 0, responder.LLM.TimeoutMs)
	assert.Equal(t, 0, responder.LLM.Retries)

	var planner YAMLInterviewPlanner
	assert.Equal(t, 0.0, planner.DuplicateThreshold)
	assert.Equal(t, 0, planner.MinSubstantiveLength)
}
