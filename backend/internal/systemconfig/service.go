// Package systemconfig provides business logic for runtime configuration overrides.
// This file implements the Service layer that orchestrates config resolution,
// key/value validation, and persistence via the repository.
//
// # Design Constraints
//
//   - The service enforces an allowlist of ~20 keys that can be overridden at runtime.
//     Keys not in the allowlist are rejected with a clear error message.
//   - Secrets (API keys, passwords, JWT secrets) are NEVER in the allowlist.
//   - Value validation is key-specific: range checks for ints, enum checks for strings,
//     weight sum validation for scoring weights.
//   - The service depends on the Resolver for config merging and the Repository for
//     persistence. Both are injected via constructor (dependency injection).
//
// # Usage
//
//	svc := systemconfig.NewService(repo, resolver)
//	effect, err := svc.GetEffectiveConfig(ctx)
//	err = svc.SetOverride(ctx, "scoring.auto_threshold", json.RawMessage(`90`))
package systemconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
)

// ---------------------------------------------------------------------------
// Service — business logic for config overrides
// ---------------------------------------------------------------------------

// Service implements the business logic for system configuration overrides.
// It validates keys against an allowlist, validates values for type/range correctness,
// and delegates persistence to the Repository and resolution to the Resolver.
type Service struct {
	repo     *Repository
	resolver *Resolver
}

// NewService creates a new systemconfig service.
// The repo provides DB access, resolver handles YAML→env→DB merge.
//
// Example:
//
//	svc := systemconfig.NewService(repo, resolver)
func NewService(repo *Repository, resolver *Resolver) *Service {
	return &Service{
		repo:     repo,
		resolver: resolver,
	}
}

// GetEffectiveConfig returns the fully resolved configuration tree.
// It calls the Resolver to merge YAML → env → DB layers and returns the result.
//
// Example:
//
//	effect, err := svc.GetEffectiveConfig(ctx)
//	if err != nil {
//	    return fmt.Errorf("systemconfig: get config: %w", err)
//	}
//	// effect.Scoring.AutoThreshold is the merged value
func (s *Service) GetEffectiveConfig(ctx context.Context) (*EffectiveConfig, error) {
	effect, err := s.resolver.Resolve(ctx, s.repo)
	if err != nil {
		return nil, fmt.Errorf("systemconfig: resolve config: %w", err)
	}
	return effect, nil
}

// SetOverride creates or updates a runtime configuration override.
// The key must be in the allowlist and the value must pass key-specific validation.
// The override is persisted to the database and takes effect on the next GetEffectiveConfig call.
//
// Returns ErrKeyNotAllowed if the key is not in the allowlist.
// Returns ErrInvalidValue if the value fails key-specific validation.
//
// Example:
//
//	err := svc.SetOverride(ctx, "scoring.auto_threshold", json.RawMessage(`90`))
//	if err != nil {
//	    return fmt.Errorf("set override: %w", err)
//	}
func (s *Service) SetOverride(ctx context.Context, key string, value json.RawMessage) error {
	// Validate key is in allowlist
	if err := s.validateKey(key); err != nil {
		return err
	}

	// Validate value is non-empty and well-formed
	if err := ValidateOverrideValue(value); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidValue, err)
	}

	// Key-specific value validation
	if err := s.validateValue(key, value); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidValue, err)
	}

	// Determine category from key prefix
	category := CategoryForKey(key)

	// Upsert to database
	if err := s.repo.UpsertOverride(ctx, key, value, category, "", nil); err != nil {
		return fmt.Errorf("systemconfig: upsert override: %w", err)
	}

	return nil
}

// DeleteOverride removes a runtime configuration override by key.
// Returns nil if the key didn't exist (idempotent). The override takes effect
// on the next GetEffectiveConfig call — the config reverts to YAML/env defaults.
//
// Example:
//
//	err := svc.DeleteOverride(ctx, "scoring.auto_threshold")
//	if err != nil {
//	    return fmt.Errorf("delete override: %w", err)
//	}
func (s *Service) DeleteOverride(ctx context.Context, key string) error {
	// Validate key format (even for deletes, we enforce format)
	if err := ValidateOverrideKey(key); err != nil {
		return err
	}

	if err := s.repo.DeleteOverride(ctx, key); err != nil {
		return fmt.Errorf("systemconfig: delete override: %w", err)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Allowlist — keys that can be overridden at runtime
// ---------------------------------------------------------------------------

// allowedKeys is the set of configuration keys that can be overridden at runtime.
// Secrets (API keys, passwords, JWT secrets) are intentionally excluded.
// This list covers runtime-tunable values across all config sections.
var allowedKeys = map[string]bool{
	// Scoring
	"scoring.auto_threshold":       true,
	"scoring.review_threshold":     true,
	"scoring.mode":                 true,
	"scoring.hybrid_reject_margin": true,
	"scoring.weights.skill":        true,
	"scoring.weights.experience":   true,
	"scoring.weights.location":     true,
	"scoring.weights.salary":       true,
	"scoring.weights.description":  true,

	// LLM (models only, not API keys)
	"llm.primary.provider":    true,
	"llm.primary.model":       true,
	"llm.local.provider":      true,
	"llm.local.model":         true,
	"llm.embeddings.provider": true,
	"llm.embeddings.model":    true,

	// Voice
	"voice.provider":    true,
	"voice.model":       true,
	"voice.livekit.url": true,

	// Interview
	"interview.memory.max_recent_segments":     true,
	"interview.memory.keep_after_summarize":    true,
	"interview.responder.llm.timeout_ms":       true,
	"interview.responder.llm.retries":          true,
	"interview.planner.duplicate_threshold":    true,
	"interview.planner.min_substantive_length": true,

	// Email
	"email.check_interval": true,
	"email.folders":        true,
	"email.provider":       true,

	// Rate limits
	"rate_limits.rpm":   true,
	"rate_limits.burst": true,

	// Automation
	"automation.queue.concurrency":          true,
	"automation.queue.retry_attempts":       true,
	"automation.auto_generate.resume":       true,
	"automation.auto_generate.cover_letter": true,

	// Approval tiers
	"approval_tiers.auto_apply.min_score": true,
	"approval_tiers.auto_apply.action":    true,
	"approval_tiers.auto_apply.notify":    true,
	"approval_tiers.review.min_score":     true,
	"approval_tiers.review.max_score":     true,
	"approval_tiers.review.action":        true,
	"approval_tiers.reject.max_score":     true,
	"approval_tiers.reject.action":        true,
	"approval_tiers.reject.log":           true,

	// Resume & cover letter
	"resume.engine":             true,
	"resume.template_dir":       true,
	"cover_letter.engine":       true,
	"cover_letter.template_dir": true,
	"cover_letter.max_length":   true,
}

// validateKey checks that the key is in the allowlist.
// Returns ErrInvalidKeyFormat if the key format is malformed.
// Returns ErrKeyNotAllowed if the key is not in the allowlist.
func (s *Service) validateKey(key string) error {
	// First check format
	if err := ValidateOverrideKey(key); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidKeyFormat, err)
	}

	// Then check allowlist
	if !allowedKeys[key] {
		return fmt.Errorf("%w: %q", ErrKeyNotAllowed, key)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Value validation — key-specific range/enum checks
// ---------------------------------------------------------------------------

// validateValue performs key-specific value validation beyond the basic
// non-empty/null checks in ValidateOverrideValue. This includes range checks
// for integers, enum checks for strings, and weight sum validation.
func (s *Service) validateValue(key string, raw json.RawMessage) error {
	switch key {
	case "scoring.auto_threshold":
		return validateIntRange(raw, 0, 100, "scoring.auto_threshold")
	case "scoring.review_threshold":
		return validateIntRange(raw, 0, 100, "scoring.review_threshold")
	case "scoring.mode":
		return validateEnum(raw, []string{"heuristic", "llm", "hybrid"}, "scoring.mode")
	case "scoring.hybrid_reject_margin":
		return validateIntRange(raw, 0, 50, "scoring.hybrid_reject_margin")

	case "scoring.weights.skill", "scoring.weights.experience",
		"scoring.weights.location", "scoring.weights.salary",
		"scoring.weights.description":
		return validateFloatRange(raw, 0, 1, key)

	case "interview.memory.max_recent_segments":
		return validateIntRange(raw, 1, 500, key)
	case "interview.memory.keep_after_summarize":
		return validateIntRange(raw, 0, 100, key)
	case "interview.responder.llm.timeout_ms":
		return validateIntRange(raw, 1000, 120000, key)
	case "interview.responder.llm.retries":
		return validateIntRange(raw, 0, 10, key)
	case "interview.planner.duplicate_threshold":
		return validateFloatRange(raw, 0, 1, key)
	case "interview.planner.min_substantive_length":
		return validateIntRange(raw, 1, 50, key)

	case "rate_limits.rpm":
		return validateIntRange(raw, 1, 1000, key)
	case "rate_limits.burst":
		return validateIntRange(raw, 1, 100, key)

	case "automation.queue.concurrency":
		return validateIntRange(raw, 1, 50, key)
	case "automation.queue.retry_attempts":
		return validateIntRange(raw, 0, 20, key)

	case "approval_tiers.auto_apply.min_score":
		return validateIntRange(raw, 0, 100, key)
	case "approval_tiers.review.min_score":
		return validateIntRange(raw, 0, 100, key)
	case "approval_tiers.review.max_score":
		return validateIntRange(raw, 0, 100, key)
	case "approval_tiers.reject.max_score":
		return validateIntRange(raw, 0, 100, key)

	case "cover_letter.max_length":
		return validateIntRange(raw, 50, 2000, key)

	case "resume.engine", "resume.template_dir",
		"cover_letter.engine", "cover_letter.template_dir",
		"email.provider", "email.check_interval":
		// String fields — just validate non-empty (already done by ValidateOverrideValue)
		return nil

	case "email.folders":
		// []string — validate non-empty
		var folders []string
		if err := json.Unmarshal(raw, &folders); err != nil {
			return fmt.Errorf("%s: not a valid string array: %w", key, err)
		}
		if len(folders) == 0 {
			return fmt.Errorf("%s: must have at least one folder", key)
		}
		return nil

	case "automation.auto_generate.resume", "automation.auto_generate.cover_letter",
		"approval_tiers.auto_apply.notify", "approval_tiers.reject.log":
		// Booleans — just validate it parses (already done by ValidateOverrideValue)
		return nil

	case "voice.provider", "voice.model", "voice.livekit.url",
		"llm.primary.provider", "llm.primary.model",
		"llm.local.provider", "llm.local.model",
		"llm.embeddings.provider", "llm.embeddings.model":
		// String fields — validate non-empty
		return nil

	default:
		// Unknown key — should not reach here if allowlist is correct
		return fmt.Errorf("systemconfig: no validation defined for key %q", key)
	}
}

// validateIntRange checks that a JSON value is an integer within [min, max].
func validateIntRange(raw json.RawMessage, min, max int, key string) error {
	var v int
	if err := json.Unmarshal(raw, &v); err != nil {
		return fmt.Errorf("%s: not a valid integer: %w", key, err)
	}
	if v < min || v > max {
		return fmt.Errorf("%s: must be between %d and %d, got %d", key, min, max, v)
	}
	return nil
}

// validateFloatRange checks that a JSON value is a float within [min, max].
func validateFloatRange(raw json.RawMessage, min, max float64, key string) error {
	var v float64
	if err := json.Unmarshal(raw, &v); err != nil {
		return fmt.Errorf("%s: not a valid float: %w", key, err)
	}
	if v < min || v > max+0.001 {
		return fmt.Errorf("%s: must be between %.2f and %.2f, got %.2f", key, min, max, v)
	}
	return nil
}

// validateEnum checks that a JSON string value is one of the allowed values.
func validateEnum(raw json.RawMessage, allowed []string, key string) error {
	var v string
	if err := json.Unmarshal(raw, &v); err != nil {
		return fmt.Errorf("%s: not a valid string: %w", key, err)
	}
	for _, a := range allowed {
		if v == a {
			return nil
		}
	}
	return fmt.Errorf("%s: must be one of %v, got %q", key, allowed, v)
}

// ---------------------------------------------------------------------------
// Weight sum validation — special case
// ---------------------------------------------------------------------------

// ValidateWeightSum checks that scoring weights sum to approximately 1.0.
// This is called after setting any individual weight to ensure consistency.
// Returns the current sum and whether it's valid.
func (s *Service) ValidateWeightSum(ctx context.Context) (float64, bool, error) {
	effect, err := s.GetEffectiveConfig(ctx)
	if err != nil {
		return 0, false, err
	}

	sum := effect.Scoring.Weights.Skill +
		effect.Scoring.Weights.Experience +
		effect.Scoring.Weights.Location +
		effect.Scoring.Weights.Salary +
		effect.Scoring.Weights.Description

	valid := math.Abs(sum-1.0) < 0.01
	return sum, valid, nil
}
