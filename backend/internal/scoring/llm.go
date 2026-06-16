// Package scoring provides job-candidate matching and scoring functionality.
// It supports three scoring modes: heuristic (keyword-based), LLM (semantic), and hybrid (pre-filter + LLM).
// The service computes factor scores (skills, experience, location, salary, description) and combines them
// into a final 0-100 score with approval tier (auto/review/reject).
package scoring

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/template"
	"time"

	"go.uber.org/zap"
	"backend/internal/config"
)

// LLMScorer defines the interface for LLM-based job scoring.
// Implementations call an LLM to understand job-profile fit semantically.
type LLMScorer interface {
	// ScoreJob evaluates a job against the user's profile using an LLM.
	// Returns a score (0-100), tier, and human-readable reasoning.
	ScoreJob(ctx context.Context, job JobData, profile Profile) (*LLMScoreResult, error)
	// ModelName returns the identifier of the LLM model (e.g., "gpt-4o", "qwen2.5:latest").
	ModelName() string
}

// LLMScoreResult holds the LLM's scoring output.
type LLMScoreResult struct {
	Score      float64      `json:"score"`
	Reasoning  string       `json:"reasoning"`
	Strengths  []string     `json:"strengths,omitempty"`
	Gaps       []string     `json:"gaps,omitempty"`
	Details    *ScoreDetails `json:"details,omitempty"`
	Confidence float64      `json:"confidence,omitempty"`
}

// NoopLLMScorer is a fallback when LLM is disabled or unavailable.
// Returns neutral scores with a placeholder reasoning.
type NoopLLMScorer struct {
	logger *zap.Logger
}

// NewNoopLLMScorer creates a no-op LLM scorer.
func NewNoopLLMScorer(logger *zap.Logger) *NoopLLMScorer {
	return &NoopLLMScorer{logger: logger.Named("llm.noop")}
}

// ScoreJob returns neutral scores — used when LLM scoring is disabled.
func (n *NoopLLMScorer) ScoreJob(_ context.Context, _ JobData, _ Profile) (*LLMScoreResult, error) {
	return &LLMScoreResult{
		Score:     50,
		Reasoning: "LLM scoring disabled, using heuristic pre-filter only",
		Strengths: []string{},
		Gaps:      []string{},
		Confidence: 0.1,
	}, nil
}

// ModelName returns the model identifier.
func (n *NoopLLMScorer) ModelName() string {
	return "noop"
}

// OllamaLLMScorer calls a local Ollama model for scoring.
type OllamaLLMScorer struct {
	logger    *zap.Logger
	baseURL   string
	model     string
	prompt    config.PromptPair
	client    *http.Client
}

// NewOllamaLLMScorer creates a new Ollama-based LLM scorer with prompt from config.
func NewOllamaLLMScorer(logger *zap.Logger, baseURL, model string, prompt config.PromptPair) *OllamaLLMScorer {
	return &OllamaLLMScorer{
		logger:  logger.Named("llm.ollama"),
		baseURL: baseURL,
		model:   model,
		prompt:  prompt,
		client:  &http.Client{Timeout: 2 * time.Minute},
	}
}

// ModelName returns the Ollama model identifier.
func (o *OllamaLLMScorer) ModelName() string {
	return o.model
}

// ScoreJob sends job + profile to Ollama for semantic scoring.
func (o *OllamaLLMScorer) ScoreJob(ctx context.Context, job JobData, profile Profile) (*LLMScoreResult, error) {
	prompt := o.buildPrompt(job, profile)

	// Call Ollama API
	result, err := o.callOllama(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("ollama scoring: %w", err)
	}

	return result, nil
}

// ollamaRequest is the payload for POST /api/generate.
type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// ollamaResponse is the response from POST /api/generate.
type ollamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// callOllama calls the Ollama /api/generate endpoint and parses the JSON response.
func (o *OllamaLLMScorer) callOllama(ctx context.Context, prompt string) (*LLMScoreResult, error) {
	reqBody, err := json.Marshal(ollamaRequest{
		Model:  o.model,
		Prompt: prompt,
		Stream: false,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := strings.TrimRight(o.baseURL, "/") + "/api/generate"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call ollama: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		msg := string(body)
		if len(msg) > 500 {
			msg = msg[:500] + "... (truncated)"
		}
		return nil, fmt.Errorf("ollama returned %d: %s", resp.StatusCode, msg)
	}

	var ollamaResp ollamaResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return nil, fmt.Errorf("unmarshal ollama response: %w", err)
	}

	// Parse the LLM's JSON response as the scoring result
	result, err := ParseLLMScoreResult([]byte(ollamaResp.Response))
	if err != nil {
		return nil, fmt.Errorf("parse LLM score result: %w", err)
	}

	return result, nil
}

// buildPrompt constructs the scoring prompt for the LLM from config template using text/template.
func (o *OllamaLLMScorer) buildPrompt(job JobData, profile Profile) string {
	// Use config prompt templates, with fallbacks
	system := o.prompt.System
	user := o.prompt.User

	if system == "" {
		system = `You are a job match scoring agent. Evaluate how well a job posting matches a candidate's profile. Be precise, consider semantic meaning (not just keywords), and provide honest assessments.`
	}
	if user == "" {
		user = `Evaluate how well this job matches the candidate's profile.

## Job
Title: {{.Title}}
Company: {{.Company}}
Location: {{.Location}} (remote_type: {{.RemoteType}})
Salary: {{.SalaryMin}}-{{.SalaryMax}}
Requirements: {{.Requirements}}
Description: {{.Description}}

## Candidate Profile
Skills: {{.Skills}}
Experience: {{.Experience}}
Preferences: {{.Preferences}}
Specializations: {{.Specializations}}
Industries: {{.Industries}}
CareerGoals: {{.CareerGoals}}

## Scoring Rules
- Score 0-100 based on: skill match, experience relevance, location fit, salary alignment
- Job titles often differ but represent similar work (e.g., "Software Engineer", "Backend Engineer", "Platform Engineer", "Application Developer", "Programmer" are strongly related)
- Do not rely on keyword overlap — evaluate actual responsibilities and skills
- Consider semantic meaning, not just keyword matching

## Output Format
Return ONLY valid JSON. Do not wrap in markdown. Do not explain your answer.
{
  "score": <number 0-100>,
  "reasoning": "<2-3 sentence explanation>",
  "strengths": ["<strength1>", "<strength2>", "..."],
  "gaps": ["<gap1>", "<gap2>", "..."],
  "details": {
    "skill_match": <0-100>,
    "experience_match": <0-100>,
    "location_match": <0-100>,
    "salary_match": <0-100>,
    "description_match": <0-100>
  },
  "confidence": <0.0-1.0>
}`
	}

	data := map[string]any{
		"Title":             job.Title,
		"Company":           job.Company,
		"Location":          job.Location,
		"RemoteType":        job.RemoteType,
		"SalaryMin":         job.SalaryMin,
		"SalaryMax":         job.SalaryMax,
		"Requirements":      job.Requirements,
		"Description":       job.Description,
		"Skills":            strings.Join(profile.Skills, ", "),
		"Experience":        formatExperience(profile.Experience),
		"Preferences":       formatPreferences(profile.Preferences),
		"Specializations":   strings.Join(profile.Specializations, ", "),
		"Industries":        strings.Join(profile.Industries, ", "),
		"CareerGoals":       strings.Join(profile.CareerGoals, ", "),
	}

	// Execute system template
	systemBuf := new(strings.Builder)
	systemTmpl, err := template.New("system").Parse(system)
	if err != nil {
		o.logger.Warn("system template parse error, using fallback", zap.Error(err))
		systemBuf.WriteString("You are a job match scoring agent. Evaluate how well a job posting matches a candidate's profile.")
	} else if err := systemTmpl.Execute(systemBuf, data); err != nil {
		o.logger.Warn("system template execute error", zap.Error(err))
	}

	// Execute user template
	userBuf := new(strings.Builder)
	userTmpl, err := template.New("user").Parse(user)
	if err != nil {
		o.logger.Warn("user template parse error, using fallback", zap.Error(err))
		userBuf.WriteString("Evaluate how well this job matches the candidate's profile. Return ONLY valid JSON.")
	} else if err := userTmpl.Execute(userBuf, data); err != nil {
		o.logger.Warn("user template execute error", zap.Error(err))
	}

	return systemBuf.String() + "\n\n" + userBuf.String()
}

// formatExperience formats experience entries for the prompt.
func formatExperience(exp []ProfileExperience) string {
	var parts []string
	for _, e := range exp {
		parts = append(parts, fmt.Sprintf("%s at %s (%s)", e.Title, e.Company, strings.Join(e.SkillsUsed, ", ")))
	}
	return strings.Join(parts, "; ")
}

// formatPreferences formats preferences for the prompt.
func formatPreferences(p ProfilePreferences) string {
	return fmt.Sprintf("locations: %v, remote_only: %v, salary: %d-%d",
		p.PreferredLocations, p.RemoteOnly, p.SalaryMin, p.SalaryMax)
}

// NewLLMScorerFromConfig creates an LLMScorer based on configuration.
// Returns NoopLLMScorer if LLM is not configured or disabled.
func NewLLMScorerFromConfig(logger *zap.Logger, cfg config.LLMConfig, prompts config.PromptsConfig) LLMScorer {
	// Use local (Ollama) by default for local-first
	if cfg.Local.Provider == "ollama" && cfg.Local.BaseURL != "" {
		return NewOllamaLLMScorer(logger, cfg.Local.BaseURL, cfg.Local.Model, prompts.Scoring)
	}
	// Could add OpenAI/Anthropic implementations here
	return NewNoopLLMScorer(logger)
}

// ValidateLLMScoreResult validates and normalizes the LLM score result.
// Returns an error if the result is invalid.
func ValidateLLMScoreResult(r *LLMScoreResult) error {
	if r == nil {
		return fmt.Errorf("nil result")
	}
	if r.Score < 0 || r.Score > 100 {
		return fmt.Errorf("score out of range [0,100]: %.2f", r.Score)
	}
	if r.Reasoning == "" {
		return fmt.Errorf("reasoning required")
	}
	if r.Confidence < 0 || r.Confidence > 1 {
		r.Confidence = 0.5 // default
	}
	if r.Details != nil {
		if err := r.Details.Validate(); err != nil {
			return fmt.Errorf("invalid details: %w", err)
		}
	}
	// Ensure lists are not nil (use empty slices for JSON)
	if r.Strengths == nil {
		r.Strengths = []string{}
	}
	if r.Gaps == nil {
		r.Gaps = []string{}
	}
	return nil
}

// ParseLLMScoreResult parses and validates the raw JSON response from an LLM.
func ParseLLMScoreResult(data []byte) (*LLMScoreResult, error) {
	var result LLMScoreResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal LLM response: %w", err)
	}
	if err := ValidateLLMScoreResult(&result); err != nil {
		return nil, err
	}
	return &result, nil
}
