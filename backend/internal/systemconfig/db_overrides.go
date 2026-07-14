// Package systemconfig provides configuration resolution logic for the job search agent.
// This file implements database override resolution for configuration resolution.
//
// # Design Constraints
//
//   - DB overrides use dot-notation keys (e.g., "scoring.auto_threshold").
//   - Each key is routed to the appropriate section setter.
//   - Invalid DB overrides are logged and skipped (never block config resolution).
//   - The Sources map is updated to track that values came from the database.
package systemconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
)

// applyDBOverrides applies database overrides to the config using dot-notation keys.
// Each key is routed to the appropriate section setter. The Sources map is updated
// to track that the value came from the database.
func (r *Resolver) applyDBOverrides(cfg *EffectiveConfig, overrides map[string]json.RawMessage) {
	for key, rawValue := range overrides {
		if err := r.setNestedValue(cfg, key, rawValue); err != nil {
			// Log at warn level and skip — invalid DB overrides should not block config resolution
			r.logger.Warn("systemconfig: failed to apply db override",
				zap.String("key", key),
				zap.Error(err),
			)
			continue
		}
		cfg.Sources[key] = SourceDB
	}
}

// setNestedValue routes a dot-notation key to the appropriate section setter.
// Returns an error if the key prefix is unrecognized.
func (r *Resolver) setNestedValue(cfg *EffectiveConfig, key string, raw json.RawMessage) error {
	parts := strings.SplitN(key, ".", 2)
	if len(parts) < 2 {
		return fmt.Errorf("key must have at least 2 segments: %q", key)
	}

	prefix := parts[0]
	suffix := parts[1]

	switch prefix {
	case "scoring":
		return r.setScoringValue(cfg, suffix, raw)
	case "llm":
		return r.setLLMValue(cfg, suffix, raw)
	case "voice":
		return r.setVoiceValue(cfg, suffix, raw)
	case "interview":
		return r.setInterviewValue(cfg, suffix, raw)
	case "email":
		return r.setEmailValue(cfg, suffix, raw)
	case "rate_limits":
		return r.setRateLimitsValue(cfg, suffix, raw)
	case "automation", "queue":
		return r.setAutomationValue(cfg, suffix, raw)
	case "approval_tiers":
		return r.setApprovalTiersValue(cfg, suffix, raw)
	case "resume":
		return r.setResumeValue(cfg, suffix, raw)
	case "cover_letter":
		return r.setCoverLetterValue(cfg, suffix, raw)
	default:
		return fmt.Errorf("unknown config prefix: %q", prefix)
	}
}

// setScoringValue applies a scoring-related override.
func (r *Resolver) setScoringValue(cfg *EffectiveConfig, key string, raw json.RawMessage) error {
	switch key {
	case "auto_threshold":
		v, err := toInt(raw)
		if err != nil {
			return fmt.Errorf("scoring.auto_threshold: %w", err)
		}
		cfg.Scoring.AutoThreshold = v
	case "review_threshold":
		v, err := toInt(raw)
		if err != nil {
			return fmt.Errorf("scoring.review_threshold: %w", err)
		}
		cfg.Scoring.ReviewThreshold = v
	case "mode":
		v, err := toString(raw)
		if err != nil {
			return fmt.Errorf("scoring.mode: %w", err)
		}
		cfg.Scoring.Mode = ScoringMode(v)
	case "hybrid_reject_margin":
		v, err := toInt(raw)
		if err != nil {
			return fmt.Errorf("scoring.hybrid_reject_margin: %w", err)
		}
		cfg.Scoring.HybridRejectMargin = v
	case "weights.skill":
		v, err := toFloat(raw)
		if err != nil {
			return fmt.Errorf("scoring.weights.skill: %w", err)
		}
		cfg.Scoring.Weights.Skill = v
	case "weights.experience":
		v, err := toFloat(raw)
		if err != nil {
			return fmt.Errorf("scoring.weights.experience: %w", err)
		}
		cfg.Scoring.Weights.Experience = v
	case "weights.location":
		v, err := toFloat(raw)
		if err != nil {
			return fmt.Errorf("scoring.weights.location: %w", err)
		}
		cfg.Scoring.Weights.Location = v
	case "weights.salary":
		v, err := toFloat(raw)
		if err != nil {
			return fmt.Errorf("scoring.weights.salary: %w", err)
		}
		cfg.Scoring.Weights.Salary = v
	case "weights.description":
		v, err := toFloat(raw)
		if err != nil {
			return fmt.Errorf("scoring.weights.description: %w", err)
		}
		cfg.Scoring.Weights.Description = v
	default:
		return fmt.Errorf("unknown scoring key: %q", key)
	}
	return nil
}

// setLLMValue applies an LLM-related override.
func (r *Resolver) setLLMValue(cfg *EffectiveConfig, key string, raw json.RawMessage) error {
	parts := strings.SplitN(key, ".", 2)
	if len(parts) < 2 {
		return fmt.Errorf("llm key must have format 'provider.field': %q", key)
	}

	provider := parts[0]
	field := parts[1]

	var target *LLMProviderSection
	switch provider {
	case "primary":
		target = &cfg.LLM.Primary
	case "local":
		target = &cfg.LLM.Local
	case "embeddings":
		target = &cfg.LLM.Embeddings
	default:
		return fmt.Errorf("unknown llm provider: %q", provider)
	}

	switch field {
	case "provider":
		v, err := toString(raw)
		if err != nil {
			return fmt.Errorf("llm.%s.provider: %w", provider, err)
		}
		target.Provider = v
	case "model":
		v, err := toString(raw)
		if err != nil {
			return fmt.Errorf("llm.%s.model: %w", provider, err)
		}
		target.Model = v
	default:
		return fmt.Errorf("unknown llm.%s field: %q", provider, field)
	}
	return nil
}

// setVoiceValue applies a voice-related override.
func (r *Resolver) setVoiceValue(cfg *EffectiveConfig, key string, raw json.RawMessage) error {
	parts := strings.SplitN(key, ".", 2)
	if len(parts) < 2 {
		// Top-level voice fields
		switch key {
		case "provider":
			v, err := toString(raw)
			if err != nil {
				return fmt.Errorf("voice.provider: %w", err)
			}
			cfg.Voice.Provider = v
		case "model":
			v, err := toString(raw)
			if err != nil {
				return fmt.Errorf("voice.model: %w", err)
			}
			cfg.Voice.Model = v
		default:
			return fmt.Errorf("unknown voice key: %q", key)
		}
		return nil
	}

	section := parts[0]
	field := parts[1]

	switch section {
	case "livekit":
		switch field {
		case "url":
			v, err := toString(raw)
			if err != nil {
				return fmt.Errorf("voice.livekit.url: %w", err)
			}
			cfg.Voice.LiveKit.URL = v
		case "api_key":
			v, err := toString(raw)
			if err != nil {
				return fmt.Errorf("voice.livekit.api_key: %w", err)
			}
			cfg.Voice.LiveKit.APIKey = v
		default:
			return fmt.Errorf("unknown voice.livekit field: %q", field)
		}
	default:
		return fmt.Errorf("unknown voice section: %q", section)
	}
	return nil
}

// setInterviewValue applies an interview-related override.
func (r *Resolver) setInterviewValue(cfg *EffectiveConfig, key string, raw json.RawMessage) error {
	parts := strings.SplitN(key, ".", 2)
	if len(parts) < 2 {
		return fmt.Errorf("interview key must have format 'section.field': %q", key)
	}

	section := parts[0]
	field := parts[1]

	switch section {
	case "memory":
		switch field {
		case "max_recent_segments":
			v, err := toInt(raw)
			if err != nil {
				return fmt.Errorf("interview.memory.max_recent_segments: %w", err)
			}
			cfg.Interview.Memory.MaxRecentSegments = v
		case "keep_after_summarize":
			v, err := toInt(raw)
			if err != nil {
				return fmt.Errorf("interview.memory.keep_after_summarize: %w", err)
			}
			cfg.Interview.Memory.KeepAfterSummarize = v
		default:
			return fmt.Errorf("unknown interview.memory field: %q", field)
		}
	case "responder":
		switch field {
		case "llm.timeout_ms":
			v, err := toInt(raw)
			if err != nil {
				return fmt.Errorf("interview.responder.llm.timeout_ms: %w", err)
			}
			cfg.Interview.Responder.LLM.TimeoutMs = v
		case "llm.retries":
			v, err := toInt(raw)
			if err != nil {
				return fmt.Errorf("interview.responder.llm.retries: %w", err)
			}
			cfg.Interview.Responder.LLM.Retries = v
		default:
			return fmt.Errorf("unknown interview.responder field: %q", field)
		}
	case "planner":
		switch field {
		case "duplicate_threshold":
			v, err := toFloat(raw)
			if err != nil {
				return fmt.Errorf("interview.planner.duplicate_threshold: %w", err)
			}
			cfg.Interview.Planner.DuplicateThreshold = v
		case "min_substantive_length":
			v, err := toInt(raw)
			if err != nil {
				return fmt.Errorf("interview.planner.min_substantive_length: %w", err)
			}
			cfg.Interview.Planner.MinSubstantiveLength = v
		default:
			return fmt.Errorf("unknown interview.planner field: %q", field)
		}
	default:
		return fmt.Errorf("unknown interview section: %q", section)
	}
	return nil
}

// setEmailValue applies an email-related override.
func (r *Resolver) setEmailValue(cfg *EffectiveConfig, key string, raw json.RawMessage) error {
	switch key {
	case "check_interval":
		v, err := toString(raw)
		if err != nil {
			return fmt.Errorf("email.check_interval: %w", err)
		}
		cfg.Email.CheckInterval = v
	case "folders":
		v, err := toStringSlice(raw)
		if err != nil {
			return fmt.Errorf("email.folders: %w", err)
		}
		cfg.Email.Folders = v
	case "provider":
		v, err := toString(raw)
		if err != nil {
			return fmt.Errorf("email.provider: %w", err)
		}
		cfg.Email.Provider = v
	default:
		return fmt.Errorf("unknown email key: %q", key)
	}
	return nil
}

// setRateLimitsValue applies a rate-limits-related override.
func (r *Resolver) setRateLimitsValue(cfg *EffectiveConfig, key string, raw json.RawMessage) error {
	switch key {
	case "rpm":
		v, err := toInt(raw)
		if err != nil {
			return fmt.Errorf("rate_limits.rpm: %w", err)
		}
		cfg.RateLimits.RPM = v
	case "burst":
		v, err := toInt(raw)
		if err != nil {
			return fmt.Errorf("rate_limits.burst: %w", err)
		}
		cfg.RateLimits.Burst = v
	default:
		return fmt.Errorf("unknown rate_limits key: %q", key)
	}
	return nil
}

// setAutomationValue applies an automation/queue-related override.
func (r *Resolver) setAutomationValue(cfg *EffectiveConfig, key string, raw json.RawMessage) error {
	// Handle queue.* keys
	if strings.HasPrefix(key, "queue.") {
		queueKey := strings.TrimPrefix(key, "queue.")
		switch queueKey {
		case "concurrency":
			v, err := toInt(raw)
			if err != nil {
				return fmt.Errorf("automation.queue.concurrency: %w", err)
			}
			cfg.Automation.Queue.Concurrency = v
		case "retry_attempts":
			v, err := toInt(raw)
			if err != nil {
				return fmt.Errorf("automation.queue.retry_attempts: %w", err)
			}
			cfg.Automation.Queue.RetryAttempts = v
		default:
			return fmt.Errorf("unknown automation.queue key: %q", queueKey)
		}
		return nil
	}

	// Handle auto_generate.* keys
	if strings.HasPrefix(key, "auto_generate.") {
		genKey := strings.TrimPrefix(key, "auto_generate.")
		switch genKey {
		case "resume":
			v, err := toBool(raw)
			if err != nil {
				return fmt.Errorf("automation.auto_generate.resume: %w", err)
			}
			cfg.Automation.AutoGenerate.Resume = v
		case "cover_letter":
			v, err := toBool(raw)
			if err != nil {
				return fmt.Errorf("automation.auto_generate.cover_letter: %w", err)
			}
			cfg.Automation.AutoGenerate.CoverLetter = v
		default:
			return fmt.Errorf("unknown automation.auto_generate key: %q", genKey)
		}
		return nil
	}

	return fmt.Errorf("unknown automation key: %q", key)
}

// setApprovalTiersValue applies an approval-tier-related override.
func (r *Resolver) setApprovalTiersValue(cfg *EffectiveConfig, key string, raw json.RawMessage) error {
	parts := strings.SplitN(key, ".", 2)
	if len(parts) < 2 {
		return fmt.Errorf("approval_tiers key must have format 'tier.field': %q", key)
	}

	tier := parts[0]
	field := parts[1]

	var target *ApprovalTierDef
	switch tier {
	case "auto_apply":
		target = &cfg.ApprovalTiers.AutoApply
	case "review":
		target = &cfg.ApprovalTiers.Review
	case "reject":
		target = &cfg.ApprovalTiers.Reject
	default:
		return fmt.Errorf("unknown approval_tiers tier: %q", tier)
	}

	switch field {
	case "min_score":
		v, err := toInt(raw)
		if err != nil {
			return fmt.Errorf("approval_tiers.%s.min_score: %w", tier, err)
		}
		target.MinScore = v
	case "max_score":
		v, err := toInt(raw)
		if err != nil {
			return fmt.Errorf("approval_tiers.%s.max_score: %w", tier, err)
		}
		target.MaxScore = v
	case "action":
		v, err := toString(raw)
		if err != nil {
			return fmt.Errorf("approval_tiers.%s.action: %w", tier, err)
		}
		target.Action = v
	case "notify":
		v, err := toBool(raw)
		if err != nil {
			return fmt.Errorf("approval_tiers.%s.notify: %w", tier, err)
		}
		target.Notify = v
	case "log":
		v, err := toBool(raw)
		if err != nil {
			return fmt.Errorf("approval_tiers.%s.log: %w", tier, err)
		}
		target.Log = v
	default:
		return fmt.Errorf("unknown approval_tiers.%s field: %q", tier, field)
	}
	return nil
}

// setResumeValue applies a resume-related override.
func (r *Resolver) setResumeValue(cfg *EffectiveConfig, key string, raw json.RawMessage) error {
	switch key {
	case "engine":
		v, err := toString(raw)
		if err != nil {
			return fmt.Errorf("resume.engine: %w", err)
		}
		cfg.ResumeConfig.Engine = v
	case "template_dir":
		v, err := toString(raw)
		if err != nil {
			return fmt.Errorf("resume.template_dir: %w", err)
		}
		cfg.ResumeConfig.TemplateDir = v
	default:
		return fmt.Errorf("unknown resume key: %q", key)
	}
	return nil
}

// setCoverLetterValue applies a cover-letter-related override.
func (r *Resolver) setCoverLetterValue(cfg *EffectiveConfig, key string, raw json.RawMessage) error {
	switch key {
	case "engine":
		v, err := toString(raw)
		if err != nil {
			return fmt.Errorf("cover_letter.engine: %w", err)
		}
		cfg.CoverLetterConfig.Engine = v
	case "template_dir":
		v, err := toString(raw)
		if err != nil {
			return fmt.Errorf("cover_letter.template_dir: %w", err)
		}
		cfg.CoverLetterConfig.TemplateDir = v
	case "max_length":
		v, err := toInt(raw)
		if err != nil {
			return fmt.Errorf("cover_letter.max_length: %w", err)
		}
		cfg.CoverLetterConfig.MaxLength = v
	default:
		return fmt.Errorf("unknown cover_letter key: %q", key)
	}
	return nil
}

// getIntegrations reads integration connection status from environment variables.
// This is read-only — integrations cannot be overridden via DB.
func (r *Resolver) getIntegrations() IntegrationsSection {
	// LiveKit status
	liveKitStatus := StatusDisconnected
	if os.Getenv("LIVEKIT_API_KEY") != "" {
		liveKitStatus = StatusConnected
	}
	liveKitURL := os.Getenv("LIVEKIT_WS_URL")

	// Email status
	emailStatus := StatusDisconnected
	if os.Getenv("MS_CLIENT_ID") != "" {
		emailStatus = StatusConnected
	}

	// AI providers
	aiProviders := make(map[string]AIProviderInfo)
	if v := os.Getenv("OPENAI_API_KEY"); v != "" {
		aiProviders["openai"] = AIProviderInfo{
			Model:  getEnvOrDefault("OPENAI_MODEL", "gpt-4o"),
			Status: StatusConnected,
		}
	}
	if v := os.Getenv("OLLAMA_BASE_URL"); v != "" {
		aiProviders["ollama"] = AIProviderInfo{
			Model:  getEnvOrDefault("OLLAMA_MODEL", "qwen2.5:latest"),
			Status: StatusConnected,
		}
	}
	if v := os.Getenv("ANTHROPIC_API_KEY"); v != "" {
		aiProviders["anthropic"] = AIProviderInfo{
			Model:  getEnvOrDefault("ANTHROPIC_MODEL", "claude-sonnet-4"),
			Status: StatusConnected,
		}
	}

	return IntegrationsSection{
		LiveKit: IntegrationStatus{
			Status: liveKitStatus,
			URL:    &liveKitURL,
		},
		Email: IntegrationStatus{
			Status: emailStatus,
		},
		AIProviders: aiProviders,
	}
}

// getEnvOrDefault returns the environment variable value or the default.
func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
