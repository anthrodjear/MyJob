// Package systemconfig provides configuration resolution logic for the job search agent.
// This file implements environment variable overrides for configuration resolution.
//
// # Design Constraints
//
//   - Env vars use UPPER_SNAKE_CASE naming (e.g., SCORING_AUTO_THRESHOLD).
//   - Only non-empty env vars are applied — defaults remain unchanged otherwise.
//   - Invalid env var values are silently ignored (defaults preserved).
package systemconfig

import (
	"os"
	"strconv"
	"strings"
)

// applyEnvOverrides applies environment variable overrides to the config.
// Env vars use UPPER_SNAKE_CASE naming (e.g., SCORING_AUTO_THRESHOLD).
// Only non-empty env vars are applied — defaults remain unchanged otherwise.
func (r *Resolver) applyEnvOverrides(cfg *EffectiveConfig) {
	// Scoring
	if v := os.Getenv("SCORING_AUTO_THRESHOLD"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Scoring.AutoThreshold = n
			cfg.Sources["scoring.auto_threshold"] = SourceEnv
		}
	}
	if v := os.Getenv("SCORING_REVIEW_THRESHOLD"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Scoring.ReviewThreshold = n
			cfg.Sources["scoring.review_threshold"] = SourceEnv
		}
	}
	if v := os.Getenv("SCORING_MODE"); v != "" {
		cfg.Scoring.Mode = ScoringMode(v)
		cfg.Sources["scoring.mode"] = SourceEnv
	}
	if v := os.Getenv("SCORING_HYBRID_REJECT_MARGIN"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Scoring.HybridRejectMargin = n
			cfg.Sources["scoring.hybrid_reject_margin"] = SourceEnv
		}
	}
	if v := os.Getenv("SCORING_WEIGHT_SKILL"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.Scoring.Weights.Skill = f
			cfg.Sources["scoring.weights.skill"] = SourceEnv
		}
	}
	if v := os.Getenv("SCORING_WEIGHT_EXPERIENCE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.Scoring.Weights.Experience = f
			cfg.Sources["scoring.weights.experience"] = SourceEnv
		}
	}
	if v := os.Getenv("SCORING_WEIGHT_LOCATION"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.Scoring.Weights.Location = f
			cfg.Sources["scoring.weights.location"] = SourceEnv
		}
	}
	if v := os.Getenv("SCORING_WEIGHT_SALARY"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.Scoring.Weights.Salary = f
			cfg.Sources["scoring.weights.salary"] = SourceEnv
		}
	}
	if v := os.Getenv("SCORING_WEIGHT_DESCRIPTION"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.Scoring.Weights.Description = f
			cfg.Sources["scoring.weights.description"] = SourceEnv
		}
	}

	// LLM
	if v := os.Getenv("LLM_PRIMARY_PROVIDER"); v != "" {
		cfg.LLM.Primary.Provider = v
		cfg.Sources["llm.primary.provider"] = SourceEnv
	}
	if v := os.Getenv("OPENAI_MODEL"); v != "" {
		cfg.LLM.Primary.Model = v
		cfg.Sources["llm.primary.model"] = SourceEnv
	}
	if v := os.Getenv("OLLAMA_MODEL"); v != "" {
		cfg.LLM.Local.Model = v
		cfg.Sources["llm.local.model"] = SourceEnv
	}
	if v := os.Getenv("OLLAMA_EMBED_MODEL"); v != "" {
		cfg.LLM.Embeddings.Model = v
		cfg.Sources["llm.embeddings.model"] = SourceEnv
	}

	// Queue
	if v := os.Getenv("QUEUE_CONCURRENCY"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Automation.Queue.Concurrency = n
			cfg.Sources["automation.queue.concurrency"] = SourceEnv
		}
	}

	// Rate limits
	if v := os.Getenv("RATE_LIMIT_RPM"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.RateLimits.RPM = n
			cfg.Sources["rate_limits.rpm"] = SourceEnv
		}
	}
	if v := os.Getenv("RATE_LIMIT_BURST"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.RateLimits.Burst = n
			cfg.Sources["rate_limits.burst"] = SourceEnv
		}
	}

	// Email
	if v := os.Getenv("EMAIL_CHECK_INTERVAL"); v != "" {
		cfg.Email.CheckInterval = v
		cfg.Sources["email.check_interval"] = SourceEnv
	}
	if v := os.Getenv("EMAIL_FOLDERS"); v != "" {
		cfg.Email.Folders = strings.Split(v, ",")
		cfg.Sources["email.folders"] = SourceEnv
	}

	// Interview
	if v := os.Getenv("INTERVIEW_MEMORY_MAX_RECENT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Interview.Memory.MaxRecentSegments = n
			cfg.Sources["interview.memory.max_recent_segments"] = SourceEnv
		}
	}
	if v := os.Getenv("INTERVIEW_RESPONDER_TIMEOUT_MS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Interview.Responder.LLM.TimeoutMs = n
			cfg.Sources["interview.responder.llm.timeout_ms"] = SourceEnv
		}
	}
	if v := os.Getenv("INTERVIEW_PLANNER_DUPLICATE_THRESHOLD"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.Interview.Planner.DuplicateThreshold = f
			cfg.Sources["interview.planner.duplicate_threshold"] = SourceEnv
		}
	}
}
