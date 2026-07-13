package systemconfig

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newTestDBResolver creates a Resolver with a no-op logger for DB override tests.
func newTestDBResolver() *Resolver {
	return &Resolver{
		logger:  zap.NewNop(),
		yamlCfg: testYAMLConfig(),
	}
}

// newScoringConfig creates an EffectiveConfig pre-populated with known defaults.
func newScoringConfig() *EffectiveConfig {
	return &EffectiveConfig{
		Scoring: ScoringSection{
			AutoThreshold:      95,
			ReviewThreshold:    80,
			Mode:               ModeHybrid,
			HybridRejectMargin: 20,
			Weights: ScoringWeights{
				Skill:       0.35,
				Experience:  0.25,
				Location:    0.10,
				Salary:      0.15,
				Description: 0.15,
			},
		},
		LLM: LLMSection{
			Primary:    LLMProviderSection{Provider: "openai", Model: "gpt-4o"},
			Local:      LLMProviderSection{Provider: "ollama", Model: "qwen2.5"},
			Embeddings: LLMProviderSection{Provider: "openai", Model: "text-embedding-3-small"},
		},
		Voice: VoiceSection{
			Provider: "deepgram",
			Model:    "deepgram-2",
			LiveKit:  LiveKitSection{URL: "wss://livekit.example.com", APIKey: "test-key"},
		},
		ApprovalTiers: ApprovalTiersSection{
			AutoApply: ApprovalTierDef{MinScore: 95, Action: "auto_apply", Notify: true},
			Review:    ApprovalTierDef{MinScore: 80, MaxScore: 94, Action: "require_review"},
			Reject:    ApprovalTierDef{MaxScore: 79, Action: "reject", Log: true},
		},
		ResumeConfig:      ResumeConfigSection{Engine: "latex", TemplateDir: "templates/resume"},
		CoverLetterConfig: CoverLetterConfigSection{Engine: "latex", TemplateDir: "templates/cl", MaxLength: 500},
		Automation: AutomationSection{
			Queue:        QueueSection{Concurrency: 4, RetryAttempts: 3},
			AutoGenerate: AutoGenerateSection{Resume: true, CoverLetter: true},
		},
		Interview: InterviewSection{
			Memory:    InterviewMemory{MaxRecentSegments: 50, KeepAfterSummarize: 10},
			Responder: InterviewResponder{LLM: LLMTimeout{TimeoutMs: 30000, Retries: 3}},
			Planner:   InterviewPlanner{DuplicateThreshold: 0.85, MinSubstantiveLength: 20},
		},
		Email: EmailSection{
			Provider:      "graphapi",
			CheckInterval: "5m",
			Folders:       []string{"INBOX"},
		},
		RateLimits: RateLimitsSection{
			RPM:   60,
			Burst: 10,
		},
		Sources: make(map[string]ConfigSource),
	}
}

// ScoringWeights is a local alias to avoid import cycle — mirrors scoring.Weights.
type ScoringWeights = struct {
	Skill       float64
	Experience  float64
	Location    float64
	Salary      float64
	Description float64
}

// ---------------------------------------------------------------------------
// setScoringValue
// ---------------------------------------------------------------------------

func TestSetScoringValue(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		raw     json.RawMessage
		wantFn  func(t *testing.T, cfg *EffectiveConfig)
		wantErr string
	}{
		{
			name: "auto_threshold valid",
			key:  "auto_threshold",
			raw:  json.RawMessage(`90`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, 90, cfg.Scoring.AutoThreshold)
			},
		},
		{
			name:    "auto_threshold invalid type",
			key:     "auto_threshold",
			raw:     json.RawMessage(`"abc"`),
			wantErr: "scoring.auto_threshold: not a valid integer",
		},
		{
			name: "review_threshold valid",
			key:  "review_threshold",
			raw:  json.RawMessage(`70`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, 70, cfg.Scoring.ReviewThreshold)
			},
		},
		{
			name: "mode valid string",
			key:  "mode",
			raw:  json.RawMessage(`"llm"`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, ScoringMode("llm"), cfg.Scoring.Mode)
			},
		},
		{
			name: "mode valid hybrid",
			key:  "mode",
			raw:  json.RawMessage(`"hybrid"`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, ModeHybrid, cfg.Scoring.Mode)
			},
		},
		{
			name:    "mode invalid type (int)",
			key:     "mode",
			raw:     json.RawMessage(`42`),
			wantErr: "scoring.mode: not a valid string",
		},
		{
			name: "hybrid_reject_margin valid",
			key:  "hybrid_reject_margin",
			raw:  json.RawMessage(`15`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, 15, cfg.Scoring.HybridRejectMargin)
			},
		},
		{
			name: "weights.skill valid",
			key:  "weights.skill",
			raw:  json.RawMessage(`0.40`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.InDelta(t, 0.40, cfg.Scoring.Weights.Skill, 0.0001)
			},
		},
		{
			name: "weights.experience valid",
			key:  "weights.experience",
			raw:  json.RawMessage(`0.30`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.InDelta(t, 0.30, cfg.Scoring.Weights.Experience, 0.0001)
			},
		},
		{
			name: "weights.location valid",
			key:  "weights.location",
			raw:  json.RawMessage(`0.10`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.InDelta(t, 0.10, cfg.Scoring.Weights.Location, 0.0001)
			},
		},
		{
			name: "weights.salary valid",
			key:  "weights.salary",
			raw:  json.RawMessage(`0.20`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.InDelta(t, 0.20, cfg.Scoring.Weights.Salary, 0.0001)
			},
		},
		{
			name: "weights.description valid",
			key:  "weights.description",
			raw:  json.RawMessage(`0.05`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.InDelta(t, 0.05, cfg.Scoring.Weights.Description, 0.0001)
			},
		},
		{
			name:    "weights.skill invalid type",
			key:     "weights.skill",
			raw:     json.RawMessage(`"abc"`),
			wantErr: "scoring.weights.skill: not a valid float",
		},
		{
			name:    "unknown scoring key",
			key:     "unknown_field",
			raw:     json.RawMessage(`1`),
			wantErr: "unknown scoring key",
		},
	}

	r := newTestDBResolver()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newScoringConfig()
			err := r.setScoringValue(cfg, tt.key, tt.raw)

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			assert.NoError(t, err)
			if tt.wantFn != nil {
				tt.wantFn(t, cfg)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// setLLMValue
// ---------------------------------------------------------------------------

func TestSetLLMValue(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		raw     json.RawMessage
		wantFn  func(t *testing.T, cfg *EffectiveConfig)
		wantErr string
	}{
		{
			name: "primary.provider",
			key:  "primary.provider",
			raw:  json.RawMessage(`"anthropic"`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, "anthropic", cfg.LLM.Primary.Provider)
			},
		},
		{
			name: "primary.model",
			key:  "primary.model",
			raw:  json.RawMessage(`"claude-sonnet-4"`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, "claude-sonnet-4", cfg.LLM.Primary.Model)
			},
		},
		{
			name: "local.provider",
			key:  "local.provider",
			raw:  json.RawMessage(`"ollama"`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, "ollama", cfg.LLM.Local.Provider)
			},
		},
		{
			name: "local.model",
			key:  "local.model",
			raw:  json.RawMessage(`"llama3"`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, "llama3", cfg.LLM.Local.Model)
			},
		},
		{
			name: "embeddings.provider",
			key:  "embeddings.provider",
			raw:  json.RawMessage(`"cohere"`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, "cohere", cfg.LLM.Embeddings.Provider)
			},
		},
		{
			name: "embeddings.model",
			key:  "embeddings.model",
			raw:  json.RawMessage(`"embed-english-v3.0"`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, "embed-english-v3.0", cfg.LLM.Embeddings.Model)
			},
		},
		{
			name:    "unknown provider",
			key:     "nonexistent.model",
			raw:     json.RawMessage(`"x"`),
			wantErr: "unknown llm provider",
		},
		{
			name:    "unknown field",
			key:     "primary.api_key",
			raw:     json.RawMessage(`"sk-xxx"`),
			wantErr: "unknown llm.primary field",
		},
		{
			name:    "single segment key",
			key:     "primary",
			raw:     json.RawMessage(`"x"`),
			wantErr: "llm key must have format 'provider.field'",
		},
		{
			name:    "primary.model invalid type",
			key:     "primary.model",
			raw:     json.RawMessage(`42`),
			wantErr: "llm.primary.model: not a valid string",
		},
	}

	r := newTestDBResolver()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newScoringConfig()
			err := r.setLLMValue(cfg, tt.key, tt.raw)

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			assert.NoError(t, err)
			if tt.wantFn != nil {
				tt.wantFn(t, cfg)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// setVoiceValue
// ---------------------------------------------------------------------------

func TestSetVoiceValue(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		raw     json.RawMessage
		wantFn  func(t *testing.T, cfg *EffectiveConfig)
		wantErr string
	}{
		{
			name: "provider top-level",
			key:  "provider",
			raw:  json.RawMessage(`"elevenlabs"`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, "elevenlabs", cfg.Voice.Provider)
			},
		},
		{
			name: "model top-level",
			key:  "model",
			raw:  json.RawMessage(`"eleven-turbo-v2"`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, "eleven-turbo-v2", cfg.Voice.Model)
			},
		},
		{
			name: "livekit.url",
			key:  "livekit.url",
			raw:  json.RawMessage(`"wss://new.livekit.com"`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, "wss://new.livekit.com", cfg.Voice.LiveKit.URL)
			},
		},
		{
			name: "livekit.api_key",
			key:  "livekit.api_key",
			raw:  json.RawMessage(`"new-api-key"`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, "new-api-key", cfg.Voice.LiveKit.APIKey)
			},
		},
		{
			name:    "unknown top-level key",
			key:     "unknown",
			raw:     json.RawMessage(`"x"`),
			wantErr: "unknown voice key",
		},
		{
			name:    "unknown livekit field",
			key:     "livekit.api_secret",
			raw:     json.RawMessage(`"s"`),
			wantErr: "unknown voice.livekit field",
		},
		{
			name:    "unknown voice section",
			key:     "other.field",
			raw:     json.RawMessage(`"x"`),
			wantErr: "unknown voice section",
		},
		{
			name:    "provider invalid type",
			key:     "provider",
			raw:     json.RawMessage(`42`),
			wantErr: "voice.provider: not a valid string",
		},
	}

	r := newTestDBResolver()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newScoringConfig()
			err := r.setVoiceValue(cfg, tt.key, tt.raw)

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			assert.NoError(t, err)
			if tt.wantFn != nil {
				tt.wantFn(t, cfg)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// setInterviewValue
// ---------------------------------------------------------------------------

func TestSetInterviewValue(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		raw     json.RawMessage
		wantFn  func(t *testing.T, cfg *EffectiveConfig)
		wantErr string
	}{
		{
			name: "memory.max_recent_segments",
			key:  "memory.max_recent_segments",
			raw:  json.RawMessage(`100`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, 100, cfg.Interview.Memory.MaxRecentSegments)
			},
		},
		{
			name: "memory.keep_after_summarize",
			key:  "memory.keep_after_summarize",
			raw:  json.RawMessage(`20`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, 20, cfg.Interview.Memory.KeepAfterSummarize)
			},
		},
		{
			name:    "memory unknown field",
			key:     "memory.unknown",
			raw:     json.RawMessage(`1`),
			wantErr: "unknown interview.memory field",
		},
		{
			name: "responder.llm.timeout_ms",
			key:  "responder.llm.timeout_ms",
			raw:  json.RawMessage(`45000`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, 45000, cfg.Interview.Responder.LLM.TimeoutMs)
			},
		},
		{
			name: "responder.llm.retries",
			key:  "responder.llm.retries",
			raw:  json.RawMessage(`5`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, 5, cfg.Interview.Responder.LLM.Retries)
			},
		},
		{
			name:    "responder unknown field",
			key:     "responder.unknown",
			raw:     json.RawMessage(`1`),
			wantErr: "unknown interview.responder field",
		},
		{
			name: "planner.duplicate_threshold",
			key:  "planner.duplicate_threshold",
			raw:  json.RawMessage(`0.95`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.InDelta(t, 0.95, cfg.Interview.Planner.DuplicateThreshold, 0.0001)
			},
		},
		{
			name: "planner.min_substantive_length",
			key:  "planner.min_substantive_length",
			raw:  json.RawMessage(`30`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, 30, cfg.Interview.Planner.MinSubstantiveLength)
			},
		},
		{
			name:    "planner unknown field",
			key:     "planner.unknown",
			raw:     json.RawMessage(`1`),
			wantErr: "unknown interview.planner field",
		},
		{
			name:    "unknown interview section",
			key:     "invalid.max_recent_segments",
			raw:     json.RawMessage(`1`),
			wantErr: "unknown interview section",
		},
		{
			name:    "single segment key",
			key:     "memory",
			raw:     json.RawMessage(`1`),
			wantErr: "interview key must have format 'section.field'",
		},
		{
			name:    "memory.max_recent_segments invalid type",
			key:     "memory.max_recent_segments",
			raw:     json.RawMessage(`"abc"`),
			wantErr: "interview.memory.max_recent_segments: not a valid integer",
		},
	}

	r := newTestDBResolver()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newScoringConfig()
			err := r.setInterviewValue(cfg, tt.key, tt.raw)

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			assert.NoError(t, err)
			if tt.wantFn != nil {
				tt.wantFn(t, cfg)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// setEmailValue
// ---------------------------------------------------------------------------

func TestSetEmailValue(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		raw     json.RawMessage
		wantFn  func(t *testing.T, cfg *EffectiveConfig)
		wantErr string
	}{
		{
			name: "check_interval",
			key:  "check_interval",
			raw:  json.RawMessage(`"10m"`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, "10m", cfg.Email.CheckInterval)
			},
		},
		{
			name: "folders",
			key:  "folders",
			raw:  json.RawMessage(`["INBOX","PROJECTS"]`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, []string{"INBOX", "PROJECTS"}, cfg.Email.Folders)
			},
		},
		{
			name: "provider",
			key:  "provider",
			raw:  json.RawMessage(`"smtp"`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, "smtp", cfg.Email.Provider)
			},
		},
		{
			name:    "unknown email key",
			key:     "unknown",
			raw:     json.RawMessage(`"x"`),
			wantErr: "unknown email key",
		},
		{
			name:    "check_interval invalid type",
			key:     "check_interval",
			raw:     json.RawMessage(`42`),
			wantErr: "email.check_interval: not a valid string",
		},
		{
			name:    "folders invalid type",
			key:     "folders",
			raw:     json.RawMessage(`"not-array"`),
			wantErr: "email.folders: not a valid string array",
		},
	}

	r := newTestDBResolver()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newScoringConfig()
			err := r.setEmailValue(cfg, tt.key, tt.raw)

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			assert.NoError(t, err)
			if tt.wantFn != nil {
				tt.wantFn(t, cfg)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// setRateLimitsValue
// ---------------------------------------------------------------------------

func TestSetRateLimitsValue(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		raw     json.RawMessage
		wantFn  func(t *testing.T, cfg *EffectiveConfig)
		wantErr string
	}{
		{
			name: "rpm",
			key:  "rpm",
			raw:  json.RawMessage(`120`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, 120, cfg.RateLimits.RPM)
			},
		},
		{
			name: "burst",
			key:  "burst",
			raw:  json.RawMessage(`50`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, 50, cfg.RateLimits.Burst)
			},
		},
		{
			name:    "unknown rate_limits key",
			key:     "unknown",
			raw:     json.RawMessage(`1`),
			wantErr: "unknown rate_limits key",
		},
		{
			name:    "rpm invalid type",
			key:     "rpm",
			raw:     json.RawMessage(`"fast"`),
			wantErr: "rate_limits.rpm: not a valid integer",
		},
	}

	r := newTestDBResolver()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newScoringConfig()
			err := r.setRateLimitsValue(cfg, tt.key, tt.raw)

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			assert.NoError(t, err)
			if tt.wantFn != nil {
				tt.wantFn(t, cfg)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// setAutomationValue
// ---------------------------------------------------------------------------

func TestSetAutomationValue(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		raw     json.RawMessage
		wantFn  func(t *testing.T, cfg *EffectiveConfig)
		wantErr string
	}{
		{
			name: "queue.concurrency",
			key:  "queue.concurrency",
			raw:  json.RawMessage(`8`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, 8, cfg.Automation.Queue.Concurrency)
			},
		},
		{
			name: "queue.retry_attempts",
			key:  "queue.retry_attempts",
			raw:  json.RawMessage(`5`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, 5, cfg.Automation.Queue.RetryAttempts)
			},
		},
		{
			name:    "queue unknown key",
			key:     "queue.unknown",
			raw:     json.RawMessage(`1`),
			wantErr: "unknown automation.queue key",
		},
		{
			name: "auto_generate.resume",
			key:  "auto_generate.resume",
			raw:  json.RawMessage(`false`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.False(t, cfg.Automation.AutoGenerate.Resume)
			},
		},
		{
			name: "auto_generate.cover_letter",
			key:  "auto_generate.cover_letter",
			raw:  json.RawMessage(`false`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.False(t, cfg.Automation.AutoGenerate.CoverLetter)
			},
		},
		{
			name:    "auto_generate unknown key",
			key:     "auto_generate.unknown",
			raw:     json.RawMessage(`true`),
			wantErr: "unknown automation.auto_generate key",
		},
		{
			name:    "unknown automation key (no prefix match)",
			key:     "no_prefix_match",
			raw:     json.RawMessage(`1`),
			wantErr: "unknown automation key",
		},
		{
			name:    "queue.concurrency invalid type",
			key:     "queue.concurrency",
			raw:     json.RawMessage(`"high"`),
			wantErr: "automation.queue.concurrency: not a valid integer",
		},
		{
			name:    "auto_generate.resume invalid type",
			key:     "auto_generate.resume",
			raw:     json.RawMessage(`"yes"`),
			wantErr: "automation.auto_generate.resume: not a valid boolean",
		},
	}

	r := newTestDBResolver()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newScoringConfig()
			err := r.setAutomationValue(cfg, tt.key, tt.raw)

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			assert.NoError(t, err)
			if tt.wantFn != nil {
				tt.wantFn(t, cfg)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// setApprovalTiersValue
// ---------------------------------------------------------------------------

func TestSetApprovalTiersValue(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		raw     json.RawMessage
		wantFn  func(t *testing.T, cfg *EffectiveConfig)
		wantErr string
	}{
		{
			name: "auto_apply.min_score",
			key:  "auto_apply.min_score",
			raw:  json.RawMessage(`90`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, 90, cfg.ApprovalTiers.AutoApply.MinScore)
			},
		},
		{
			name: "auto_apply.action",
			key:  "auto_apply.action",
			raw:  json.RawMessage(`"auto"`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, "auto", cfg.ApprovalTiers.AutoApply.Action)
			},
		},
		{
			name: "auto_apply.notify",
			key:  "auto_apply.notify",
			raw:  json.RawMessage(`false`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.False(t, cfg.ApprovalTiers.AutoApply.Notify)
			},
		},
		{
			name: "review.min_score",
			key:  "review.min_score",
			raw:  json.RawMessage(`75`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, 75, cfg.ApprovalTiers.Review.MinScore)
			},
		},
		{
			name: "review.max_score",
			key:  "review.max_score",
			raw:  json.RawMessage(`90`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, 90, cfg.ApprovalTiers.Review.MaxScore)
			},
		},
		{
			name: "review.action",
			key:  "review.action",
			raw:  json.RawMessage(`"manual"`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, "manual", cfg.ApprovalTiers.Review.Action)
			},
		},
		{
			name: "reject.max_score",
			key:  "reject.max_score",
			raw:  json.RawMessage(`69`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, 69, cfg.ApprovalTiers.Reject.MaxScore)
			},
		},
		{
			name: "reject.action",
			key:  "reject.action",
			raw:  json.RawMessage(`"discard"`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, "discard", cfg.ApprovalTiers.Reject.Action)
			},
		},
		{
			name: "reject.log",
			key:  "reject.log",
			raw:  json.RawMessage(`false`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.False(t, cfg.ApprovalTiers.Reject.Log)
			},
		},
		{
			name:    "unknown tier",
			key:     "unknown_tier.min_score",
			raw:     json.RawMessage(`50`),
			wantErr: "unknown approval_tiers tier",
		},
		{
			name:    "unknown auto_apply field",
			key:     "auto_apply.unknown",
			raw:     json.RawMessage(`1`),
			wantErr: "unknown approval_tiers.auto_apply field",
		},
		{
			name:    "single segment key",
			key:     "auto_apply",
			raw:     json.RawMessage(`1`),
			wantErr: "approval_tiers key must have format 'tier.field'",
		},
		{
			name:    "auto_apply.min_score invalid type",
			key:     "auto_apply.min_score",
			raw:     json.RawMessage(`"high"`),
			wantErr: "approval_tiers.auto_apply.min_score: not a valid integer",
		},
		{
			name:    "auto_apply.action invalid type",
			key:     "auto_apply.action",
			raw:     json.RawMessage(`42`),
			wantErr: "approval_tiers.auto_apply.action: not a valid string",
		},
		{
			name:    "auto_apply.notify invalid type",
			key:     "auto_apply.notify",
			raw:     json.RawMessage(`"yes"`),
			wantErr: "approval_tiers.auto_apply.notify: not a valid boolean",
		},
	}

	r := newTestDBResolver()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newScoringConfig()
			err := r.setApprovalTiersValue(cfg, tt.key, tt.raw)

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			assert.NoError(t, err)
			if tt.wantFn != nil {
				tt.wantFn(t, cfg)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// setResumeValue
// ---------------------------------------------------------------------------

func TestSetResumeValue(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		raw     json.RawMessage
		wantFn  func(t *testing.T, cfg *EffectiveConfig)
		wantErr string
	}{
		{
			name: "engine",
			key:  "engine",
			raw:  json.RawMessage(`"markdown"`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, "markdown", cfg.ResumeConfig.Engine)
			},
		},
		{
			name: "template_dir",
			key:  "template_dir",
			raw:  json.RawMessage(`"templates/new"`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, "templates/new", cfg.ResumeConfig.TemplateDir)
			},
		},
		{
			name:    "unknown resume key",
			key:     "unknown",
			raw:     json.RawMessage(`"x"`),
			wantErr: "unknown resume key",
		},
		{
			name:    "engine invalid type",
			key:     "engine",
			raw:     json.RawMessage(`42`),
			wantErr: "resume.engine: not a valid string",
		},
	}

	r := newTestDBResolver()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newScoringConfig()
			err := r.setResumeValue(cfg, tt.key, tt.raw)

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			assert.NoError(t, err)
			if tt.wantFn != nil {
				tt.wantFn(t, cfg)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// setCoverLetterValue
// ---------------------------------------------------------------------------

func TestSetCoverLetterValue(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		raw     json.RawMessage
		wantFn  func(t *testing.T, cfg *EffectiveConfig)
		wantErr string
	}{
		{
			name: "engine",
			key:  "engine",
			raw:  json.RawMessage(`"markdown"`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, "markdown", cfg.CoverLetterConfig.Engine)
			},
		},
		{
			name: "template_dir",
			key:  "template_dir",
			raw:  json.RawMessage(`"templates/new"`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, "templates/new", cfg.CoverLetterConfig.TemplateDir)
			},
		},
		{
			name: "max_length",
			key:  "max_length",
			raw:  json.RawMessage(`1000`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, 1000, cfg.CoverLetterConfig.MaxLength)
			},
		},
		{
			name:    "unknown cover_letter key",
			key:     "unknown",
			raw:     json.RawMessage(`"x"`),
			wantErr: "unknown cover_letter key",
		},
		{
			name:    "engine invalid type",
			key:     "engine",
			raw:     json.RawMessage(`42`),
			wantErr: "cover_letter.engine: not a valid string",
		},
		{
			name:    "max_length invalid type",
			key:     "max_length",
			raw:     json.RawMessage(`"long"`),
			wantErr: "cover_letter.max_length: not a valid integer",
		},
	}

	r := newTestDBResolver()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newScoringConfig()
			err := r.setCoverLetterValue(cfg, tt.key, tt.raw)

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			assert.NoError(t, err)
			if tt.wantFn != nil {
				tt.wantFn(t, cfg)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// setNestedValue
// ---------------------------------------------------------------------------

func TestSetNestedValue(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		raw     json.RawMessage
		wantFn  func(t *testing.T, cfg *EffectiveConfig)
		wantErr string
	}{
		{
			name: "scoring prefix",
			key:  "scoring.auto_threshold",
			raw:  json.RawMessage(`85`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, 85, cfg.Scoring.AutoThreshold)
			},
		},
		{
			name: "llm prefix",
			key:  "llm.primary.model",
			raw:  json.RawMessage(`"gpt-4"`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, "gpt-4", cfg.LLM.Primary.Model)
			},
		},
		{
			name: "voice prefix",
			key:  "voice.provider",
			raw:  json.RawMessage(`"google"`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, "google", cfg.Voice.Provider)
			},
		},
		{
			name: "interview prefix",
			key:  "interview.memory.max_recent_segments",
			raw:  json.RawMessage(`200`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, 200, cfg.Interview.Memory.MaxRecentSegments)
			},
		},
		{
			name: "email prefix",
			key:  "email.provider",
			raw:  json.RawMessage(`"smtp"`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, "smtp", cfg.Email.Provider)
			},
		},
		{
			name: "rate_limits prefix",
			key:  "rate_limits.rpm",
			raw:  json.RawMessage(`200`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, 200, cfg.RateLimits.RPM)
			},
		},
		{
			name: "automation prefix",
			key:  "automation.queue.concurrency",
			raw:  json.RawMessage(`12`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, 12, cfg.Automation.Queue.Concurrency)
			},
		},
		{
			name:    "queue prefix strips to 'concurrency' which automation rejects",
			key:     "queue.concurrency",
			raw:     json.RawMessage(`16`),
			wantErr: "unknown automation key",
		},
		{
			name: "approval_tiers prefix",
			key:  "approval_tiers.auto_apply.min_score",
			raw:  json.RawMessage(`85`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, 85, cfg.ApprovalTiers.AutoApply.MinScore)
			},
		},
		{
			name: "resume prefix",
			key:  "resume.engine",
			raw:  json.RawMessage(`"markdown"`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, "markdown", cfg.ResumeConfig.Engine)
			},
		},
		{
			name: "cover_letter prefix",
			key:  "cover_letter.max_length",
			raw:  json.RawMessage(`750`),
			wantFn: func(t *testing.T, cfg *EffectiveConfig) {
				assert.Equal(t, 750, cfg.CoverLetterConfig.MaxLength)
			},
		},
		{
			name:    "unknown prefix",
			key:     "unknown.key",
			raw:     json.RawMessage(`1`),
			wantErr: "unknown config prefix",
		},
		{
			name:    "single segment key",
			key:     "scoring",
			raw:     json.RawMessage(`1`),
			wantErr: "key must have at least 2 segments",
		},
	}

	r := newTestDBResolver()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newScoringConfig()
			err := r.setNestedValue(cfg, tt.key, tt.raw)

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			assert.NoError(t, err)
			if tt.wantFn != nil {
				tt.wantFn(t, cfg)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// applyDBOverrides
// ---------------------------------------------------------------------------

func TestApplyDBOverrides_AllValid(t *testing.T) {
	r := newTestDBResolver()
	cfg := newScoringConfig()

	overrides := map[string]json.RawMessage{
		"scoring.auto_threshold":           json.RawMessage(`85`),
		"scoring.mode":                     json.RawMessage(`"heuristic"`),
		"llm.primary.model":               json.RawMessage(`"gpt-4o-mini"`),
		"email.check_interval":             json.RawMessage(`"15m"`),
		"automation.queue.concurrency":     json.RawMessage(`12`),
		"approval_tiers.auto_apply.notify": json.RawMessage(`false`),
	}

	r.applyDBOverrides(cfg, overrides)

	assert.Equal(t, 85, cfg.Scoring.AutoThreshold)
	assert.Equal(t, ScoringMode("heuristic"), cfg.Scoring.Mode)
	assert.Equal(t, "gpt-4o-mini", cfg.LLM.Primary.Model)
	assert.Equal(t, "15m", cfg.Email.CheckInterval)
	assert.Equal(t, 12, cfg.Automation.Queue.Concurrency)
	assert.False(t, cfg.ApprovalTiers.AutoApply.Notify)

	// Source tracking
	assert.Equal(t, SourceDB, cfg.Sources["scoring.auto_threshold"])
	assert.Equal(t, SourceDB, cfg.Sources["scoring.mode"])
	assert.Equal(t, SourceDB, cfg.Sources["llm.primary.model"])
	assert.Equal(t, SourceDB, cfg.Sources["email.check_interval"])
	assert.Equal(t, SourceDB, cfg.Sources["automation.queue.concurrency"])
	assert.Equal(t, SourceDB, cfg.Sources["approval_tiers.auto_apply.notify"])
	assert.Len(t, cfg.Sources, 6)
}

func TestApplyDBOverrides_AllInvalid(t *testing.T) {
	r := newTestDBResolver()
	cfg := newScoringConfig()

	overrides := map[string]json.RawMessage{
		"scoring.auto_threshold": json.RawMessage(`"not-a-number"`),
		"scoring.mode":           json.RawMessage(`42`),
		"email.folders":          json.RawMessage(`"not-an-array"`),
		"unknown.key":            json.RawMessage(`1`),
	}

	r.applyDBOverrides(cfg, overrides)

	// Values should remain at their defaults
	assert.Equal(t, 95, cfg.Scoring.AutoThreshold)
	assert.Equal(t, ModeHybrid, cfg.Scoring.Mode)
	assert.Equal(t, []string{"INBOX"}, cfg.Email.Folders)

	// Sources should be empty — no DB overrides took effect
	assert.Empty(t, cfg.Sources)
}

func TestApplyDBOverrides_MixedValidAndInvalid(t *testing.T) {
	r := newTestDBResolver()
	cfg := newScoringConfig()

	overrides := map[string]json.RawMessage{
		"scoring.auto_threshold":       json.RawMessage(`75`),           // valid
		"scoring.mode":                 json.RawMessage(`42`),            // invalid
		"llm.primary.model":            json.RawMessage(`"claude-3.5"`), // valid
		"nonexistent.prefix.key":       json.RawMessage(`1`),            // unknown prefix
		"approval_tiers.auto_apply.log": json.RawMessage(`true`),        // valid
	}

	r.applyDBOverrides(cfg, overrides)

	// Valid overrides applied
	assert.Equal(t, 75, cfg.Scoring.AutoThreshold)
	assert.Equal(t, "claude-3.5", cfg.LLM.Primary.Model)
	assert.True(t, cfg.ApprovalTiers.AutoApply.Log)

	// Invalid overrides did not apply
	assert.Equal(t, ModeHybrid, cfg.Scoring.Mode) // unchanged

	// Source tracking — only valid entries
	assert.Equal(t, SourceDB, cfg.Sources["scoring.auto_threshold"])
	assert.Equal(t, SourceDB, cfg.Sources["llm.primary.model"])
	assert.Equal(t, SourceDB, cfg.Sources["approval_tiers.auto_apply.log"])
	assert.Len(t, cfg.Sources, 3)
}

func TestApplyDBOverrides_EmptyOverrides(t *testing.T) {
	r := newTestDBResolver()
	cfg := newScoringConfig()

	overrides := map[string]json.RawMessage{}
	r.applyDBOverrides(cfg, overrides)

	// No changes
	assert.Equal(t, 95, cfg.Scoring.AutoThreshold)
	assert.Empty(t, cfg.Sources)
}

func TestApplyDBOverrides_NilOverridesMap(t *testing.T) {
	r := newTestDBResolver()
	cfg := newScoringConfig()

	// A nil map should be handled gracefully (zero-length iteration)
	r.applyDBOverrides(cfg, nil)

	assert.Equal(t, 95, cfg.Scoring.AutoThreshold)
	assert.Empty(t, cfg.Sources)
}

// ---------------------------------------------------------------------------
// getIntegrations
// ---------------------------------------------------------------------------

func TestGetIntegrations_NoEnvVars(t *testing.T) {
	// Unset all integration-related env vars
	envKeys := []string{
		"LIVEKIT_API_KEY", "LIVEKIT_WS_URL", "MS_CLIENT_ID",
		"OPENAI_API_KEY", "OLLAMA_BASE_URL", "ANTHROPIC_API_KEY",
	}
	for _, k := range envKeys {
		unsetEnv(t, k)
	}

	r := newTestDBResolver()
	integrations := r.getIntegrations()

	assert.Equal(t, StatusDisconnected, integrations.LiveKit.Status)
	assert.Equal(t, StatusDisconnected, integrations.Email.Status)
	assert.Empty(t, integrations.AIProviders, "no provider connections without env vars")

	// URL should be empty string but non-nil pointer
	require.NotNil(t, integrations.LiveKit.URL)
	assert.Equal(t, "", *integrations.LiveKit.URL)
}

func TestGetIntegrations_LiveKitConnected(t *testing.T) {
	setEnv(t, "LIVEKIT_API_KEY", "some-key")
	setEnv(t, "LIVEKIT_WS_URL", "wss://livekit.prod.com")
	unsetEnv(t, "MS_CLIENT_ID")
	unsetEnv(t, "OPENAI_API_KEY")
	unsetEnv(t, "OLLAMA_BASE_URL")
	unsetEnv(t, "ANTHROPIC_API_KEY")

	r := newTestDBResolver()
	integrations := r.getIntegrations()

	assert.Equal(t, StatusConnected, integrations.LiveKit.Status)
	require.NotNil(t, integrations.LiveKit.URL)
	assert.Equal(t, "wss://livekit.prod.com", *integrations.LiveKit.URL)
	assert.Equal(t, StatusDisconnected, integrations.Email.Status)
	assert.Empty(t, integrations.AIProviders)
}

func TestGetIntegrations_EmailConnected(t *testing.T) {
	unsetEnv(t, "LIVEKIT_API_KEY")
	setEnv(t, "MS_CLIENT_ID", "azure-client-id")
	unsetEnv(t, "OPENAI_API_KEY")
	unsetEnv(t, "OLLAMA_BASE_URL")
	unsetEnv(t, "ANTHROPIC_API_KEY")

	r := newTestDBResolver()
	integrations := r.getIntegrations()

	assert.Equal(t, StatusDisconnected, integrations.LiveKit.Status)
	assert.Equal(t, StatusConnected, integrations.Email.Status)
}

func TestGetIntegrations_AIProviders(t *testing.T) {
	setEnv(t, "OPENAI_API_KEY", "sk-test")
	setEnv(t, "OPENAI_MODEL", "gpt-4o-mini")
	setEnv(t, "OLLAMA_BASE_URL", "http://localhost:11434")
	setEnv(t, "OLLAMA_MODEL", "llama3.1")
	setEnv(t, "ANTHROPIC_API_KEY", "sk-ant-test")
	setEnv(t, "ANTHROPIC_MODEL", "claude-opus-4")
	unsetEnv(t, "LIVEKIT_API_KEY")
	unsetEnv(t, "MS_CLIENT_ID")

	r := newTestDBResolver()
	integrations := r.getIntegrations()

	require.Len(t, integrations.AIProviders, 3)

	openai, ok := integrations.AIProviders["openai"]
	require.True(t, ok, "openai provider should be present")
	assert.Equal(t, "gpt-4o-mini", openai.Model)
	assert.Equal(t, StatusConnected, openai.Status)

	ollama, ok := integrations.AIProviders["ollama"]
	require.True(t, ok, "ollama provider should be present")
	assert.Equal(t, "llama3.1", ollama.Model)
	assert.Equal(t, StatusConnected, ollama.Status)

	anthropic, ok := integrations.AIProviders["anthropic"]
	require.True(t, ok, "anthropic provider should be present")
	assert.Equal(t, "claude-opus-4", anthropic.Model)
	assert.Equal(t, StatusConnected, anthropic.Status)
}

func TestGetIntegrations_AIProvidersDefaultModels(t *testing.T) {
	setEnv(t, "OPENAI_API_KEY", "sk-test")
	setEnv(t, "OLLAMA_BASE_URL", "http://localhost:11434")
	setEnv(t, "ANTHROPIC_API_KEY", "sk-ant-test")
	// Do NOT set the model-specific env vars — defaults should apply
	unsetEnv(t, "LIVEKIT_API_KEY")
	unsetEnv(t, "MS_CLIENT_ID")

	// Clear model overrides
	unsetEnv(t, "OPENAI_MODEL")
	unsetEnv(t, "OLLAMA_MODEL")
	unsetEnv(t, "ANTHROPIC_MODEL")

	r := newTestDBResolver()
	integrations := r.getIntegrations()

	require.Len(t, integrations.AIProviders, 3)
	assert.Equal(t, "gpt-4o", integrations.AIProviders["openai"].Model)
	assert.Equal(t, "qwen2.5:latest", integrations.AIProviders["ollama"].Model)
	assert.Equal(t, "claude-sonnet-4", integrations.AIProviders["anthropic"].Model)
}

// ---------------------------------------------------------------------------
// getEnvOrDefault
// ---------------------------------------------------------------------------

func TestGetEnvOrDefault(t *testing.T) {
	setEnv(t, "TEST_EXISTING_KEY", "actual-value")
	unsetEnv(t, "TEST_MISSING_KEY")

	// Existing key returns the actual value
	got := getEnvOrDefault("TEST_EXISTING_KEY", "default")
	assert.Equal(t, "actual-value", got)

	// Missing key returns the default
	got = getEnvOrDefault("TEST_MISSING_KEY", "fallback")
	assert.Equal(t, "fallback", got)

	// Empty value is treated as missing
	setEnv(t, "TEST_EMPTY_KEY", "")
	got = getEnvOrDefault("TEST_EMPTY_KEY", "default-for-empty")
	assert.Equal(t, "default-for-empty", got)
}
