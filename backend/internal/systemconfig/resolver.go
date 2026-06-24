// Package systemconfig provides configuration resolution logic for the job search agent.
// This file implements the YAML → env vars → DB overrides merge strategy that produces
// the EffectiveConfig returned by the GET API.
//
// # Merge Strategy
//
// Configuration is resolved in layers, each overriding the previous:
//
//  1. YAML defaults (config/application.yaml) — baseline values
//  2. Environment variables — infrastructure secrets (DATABASE_URL, API keys, etc.)
//  3. Database overrides — user-defined runtime tuning via the admin UI
//
// The Sources map on EffectiveConfig tracks which layer produced each leaf value,
// enabling the frontend to display "system default" vs "user overridden" indicators.
//
// # Design Constraints
//
//   - Prompts are NOT resolved here — they stay in code (prompts.go) and are never
//     exposed via the system config API.
//   - Secrets (API keys, passwords, JWT secrets) are NOT exposed in the API response.
//     The resolver skips sensitive fields entirely.
//   - Validation of override keys and values happens in the service layer, not here.
//     The resolver trusts that inputs are pre-validated.
//
// # Usage
//
//	resolver, err := systemconfig.NewResolver(logger, "config/application.yaml")
//	if err != nil {
//	    return fmt.Errorf("init resolver: %w", err)
//	}
//	effect, err := resolver.Resolve(ctx, repo)
//	if err != nil {
//	    return fmt.Errorf("resolve config: %w", err)
//	}
package systemconfig

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"backend/internal/scoring"
)

// ---------------------------------------------------------------------------
// Resolver — configuration merge engine
// ---------------------------------------------------------------------------

// Resolver implements the YAML → env → DB merge strategy for configuration resolution.
// It loads YAML defaults once at construction time, then applies env and DB overrides
// on each Resolve call. The resolver does NOT validate inputs — that is the service
// layer's responsibility.
type Resolver struct {
	logger   *zap.Logger
	yamlPath string
	yamlCfg  *YAMLConfig
}

// NewResolver creates a new configuration resolver.
// It loads and parses the YAML config file immediately, failing fast on parse errors.
// The yamlPath should point to config/application.yaml (or the path from CONFIG_PATH).
//
// Example:
//
//	resolver, err := systemconfig.NewResolver(logger, "config/application.yaml")
//	if err != nil {
//	    return fmt.Errorf("init resolver: %w", err)
//	}
func NewResolver(logger *zap.Logger, yamlPath string) (*Resolver, error) {
	r := &Resolver{
		logger:   logger,
		yamlPath: yamlPath,
	}

	if err := r.loadYAML(); err != nil {
		return nil, fmt.Errorf("systemconfig: load yaml: %w", err)
	}

	return r, nil
}

// loadYAML parses the YAML config file into YAMLConfig.
// Called once during construction. Fails fast on parse errors.
func (r *Resolver) loadYAML() error {
	data, err := os.ReadFile(r.yamlPath)
	if err != nil {
		return fmt.Errorf("systemconfig: read yaml file %s: %w", r.yamlPath, err)
	}

	var cfg YAMLConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("systemconfig: parse yaml: %w", err)
	}

	r.yamlCfg = &cfg
	return nil
}

// Resolve produces the fully merged EffectiveConfig by applying layers in order:
// YAML defaults → env vars → DB overrides. Each leaf value's origin is tracked
// in the Sources map. The repo parameter provides DB override access; pass nil
// to skip DB layer (useful for testing).
//
// Example:
//
//	effect, err := resolver.Resolve(ctx, repo)
//	if err != nil {
//	    return fmt.Errorf("resolve config: %w", err)
//	}
//	// effect.Scoring.AutoThreshold is the merged value
//	// effect.Sources["scoring.auto_threshold"] tells you where it came from
func (r *Resolver) Resolve(ctx context.Context, repo *Repository) (*EffectiveConfig, error) {
	// Start with YAML defaults
	cfg := r.buildFromYAML()

	// Layer 2: env var overrides
	r.applyEnvOverrides(cfg)

	// Layer 3: DB overrides (if repo provided)
	if repo != nil {
		overrides, err := repo.GetAllOverrides(ctx)
		if err != nil {
			return nil, fmt.Errorf("systemconfig: get db overrides: %w", err)
		}
		r.applyDBOverrides(cfg, overrides)
	}

	// Add integrations (read-only, always from env)
	cfg.Integrations = r.getIntegrations()

	return cfg, nil
}

// buildFromYAML constructs an EffectiveConfig from YAML defaults.
// All Sources are initialized to "yaml" since we're starting from YAML values.
func (r *Resolver) buildFromYAML() *EffectiveConfig {
	yaml := r.yamlCfg

	return &EffectiveConfig{
		Scoring: ScoringSection{
			AutoThreshold:      yaml.Application.ApprovalTiers.AutoApply.MinScore,
			ReviewThreshold:    yaml.Application.ApprovalTiers.Review.MinScore,
			Mode:               ModeHybrid, // default
			HybridRejectMargin: 20,         // default
			Weights: scoring.Weights{
				Skill:       0.35,
				Experience:  0.25,
				Location:    0.10,
				Salary:      0.15,
				Description: 0.15,
			},
		},
		LLM: LLMSection{
			Primary: LLMProviderSection{
				Provider: yaml.LLM.Primary.Provider,
				Model:    yaml.LLM.Primary.Model,
			},
			Local: LLMProviderSection{
				Provider: yaml.LLM.Local.Provider,
				Model:    yaml.LLM.Local.Model,
			},
			Embeddings: LLMProviderSection{
				Provider: yaml.LLM.Embeddings.Provider,
				Model:    yaml.LLM.Embeddings.Model,
			},
		},
		Voice: VoiceSection{
			Provider: yaml.Voice.Provider,
			Model:    yaml.Voice.Model,
			LiveKit: LiveKitSection{
				URL:    yaml.Voice.LiveKit.URL,
				APIKey: yaml.Voice.LiveKit.APIKey,
				// APISecret intentionally omitted — secrets are not exposed via API
			},
		},
		ApprovalTiers: ApprovalTiersSection{
			AutoApply: ApprovalTierDef{
				MinScore: yaml.Application.ApprovalTiers.AutoApply.MinScore,
				Action:   yaml.Application.ApprovalTiers.AutoApply.Action,
				Notify:   yaml.Application.ApprovalTiers.AutoApply.Notify,
			},
			Review: ApprovalTierDef{
				MinScore: yaml.Application.ApprovalTiers.Review.MinScore,
				MaxScore: yaml.Application.ApprovalTiers.Review.MaxScore,
				Action:   yaml.Application.ApprovalTiers.Review.Action,
			},
			Reject: ApprovalTierDef{
				MaxScore: yaml.Application.ApprovalTiers.Reject.MaxScore,
				Action:   yaml.Application.ApprovalTiers.Reject.Action,
				Log:      yaml.Application.ApprovalTiers.Reject.Log,
			},
		},
		ResumeConfig: ResumeConfigSection{
			Engine:      yaml.Application.Resume.Engine,
			TemplateDir: yaml.Application.Resume.TemplateDir,
		},
		CoverLetterConfig: CoverLetterConfigSection{
			Engine:      yaml.Application.CoverLetter.Engine,
			TemplateDir: yaml.Application.CoverLetter.TemplateDir,
			MaxLength:   yaml.Application.CoverLetter.MaxLength,
		},
		Automation: AutomationSection{
			Queue: QueueSection{
				Concurrency:   yaml.Queue.Concurrency,
				RetryAttempts: yaml.Queue.RetryAttempts,
			},
			AutoGenerate: AutoGenerateSection{
				Resume:      yaml.Application.AutoGenerate.Resume,
				CoverLetter: yaml.Application.AutoGenerate.CoverLetter,
			},
		},
		Interview: InterviewSection{
			Memory: InterviewMemory{
				MaxRecentSegments:  yaml.Interview.Memory.MaxRecentSegments,
				KeepAfterSummarize: yaml.Interview.Memory.KeepAfterSummarize,
			},
			Responder: InterviewResponder{
				LLM: LLMTimeout{
					TimeoutMs: yaml.Interview.Responder.LLM.TimeoutMs,
					Retries:   yaml.Interview.Responder.LLM.Retries,
				},
			},
			Planner: InterviewPlanner{
				DuplicateThreshold:   yaml.Interview.Planner.DuplicateThreshold,
				MinSubstantiveLength: yaml.Interview.Planner.MinSubstantiveLength,
			},
		},
		Email: EmailSection{
			Provider:      yaml.Email.Provider,
			CheckInterval: yaml.Email.CheckInterval,
			Folders:       yaml.Email.Folders,
		},
		RateLimits: RateLimitsSection{
			RPM:   60, // default
			Burst: 10, // default
		},
		Sources: make(map[string]ConfigSource),
	}
}
