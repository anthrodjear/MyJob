package resumes

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"go.uber.org/zap"

	"backend/internal/config"
)

// ResumeGenerator defines the interface for LLM-based resume content generation.
type ResumeGenerator interface {
	// GenerateContent creates structured resume content based on profile and optional job context.
	GenerateContent(ctx context.Context, profile map[string]any, jobTitle, jobRequirements string) (*ResumeContent, error)
	// ModelName returns the identifier of the LLM model used.
	ModelName() string
}

// OllamaResumeGenerator calls a local Ollama model for resume generation.
type OllamaResumeGenerator struct {
	logger  *zap.Logger
	baseURL string
	model   string
	prompt  config.PromptPair
}

// NewOllamaResumeGenerator creates a new Ollama-based resume generator.
func NewOllamaResumeGenerator(logger *zap.Logger, baseURL, model string, prompt config.PromptPair) *OllamaResumeGenerator {
	return &OllamaResumeGenerator{
		logger:  logger.Named("llm.resume"),
		baseURL: baseURL,
		model:   model,
		prompt:  prompt,
	}
}

// ModelName returns the Ollama model identifier.
func (o *OllamaResumeGenerator) ModelName() string {
	return o.model
}

// GenerateContent sends profile + job context to Ollama for structured resume generation.
func (o *OllamaResumeGenerator) GenerateContent(ctx context.Context, profile map[string]any, jobTitle, jobRequirements string) (*ResumeContent, error) {
	prompt := o.buildPrompt(profile, jobTitle, jobRequirements)
	_ = prompt // TODO: use prompt in Ollama API call

	// TODO: Implement Ollama API call (POST /api/generate)
	// For now, return a placeholder
	o.logger.Debug("LLM resume generation not yet implemented, returning placeholder",
		zap.String("model", o.model),
	)

	return &ResumeContent{
		Summary: fmt.Sprintf("Experienced %s with expertise in the requested domain.", profile["Specialization"]),
		Skills:  []string{"placeholder"},
	}, nil
}

// buildPrompt constructs the generation prompt from config template using text/template.
func (o *OllamaResumeGenerator) buildPrompt(data map[string]any, jobTitle, jobRequirements string) string {
	system := o.prompt.System
	user := o.prompt.User

	if system == "" {
		system = `You are a resume content generator. Create a structured, ATS-friendly resume.`
	}
	if user == "" {
		user = `Generate structured resume content based on this profile.

Name: {{.Name}}
Specialization: {{.Specialization}}
Skills: {{.Skills}}
Experience: {{.Experience}}

Return ONLY valid JSON.`
	}

	// Add job context to template data (safe: map is only used once per call)
	data["JobTitle"] = jobTitle
	data["JobRequirements"] = jobRequirements

	systemBuf := new(strings.Builder)
	systemTmpl, err := template.New("system").Parse(system)
	if err != nil {
		o.logger.Warn("system template parse error, using fallback", zap.Error(err))
		systemBuf.WriteString(`You are a resume content generator. Create a structured, ATS-friendly resume.`)
	} else if err := systemTmpl.Execute(systemBuf, data); err != nil {
		o.logger.Warn("system template execute error", zap.Error(err))
	}

	userBuf := new(strings.Builder)
	userTmpl, err := template.New("user").Parse(user)
	if err != nil {
		o.logger.Warn("user template parse error, using fallback", zap.Error(err))
		userBuf.WriteString(`Generate structured resume content. Return ONLY valid JSON.`)
	} else if err := userTmpl.Execute(userBuf, data); err != nil {
		o.logger.Warn("user template execute error", zap.Error(err))
	}

	return systemBuf.String() + "\n\n" + userBuf.String()
}

// NoopResumeGenerator is a fallback when LLM is disabled or unavailable.
type NoopResumeGenerator struct {
	logger *zap.Logger
}

// NewNoopResumeGenerator creates a no-op resume generator.
func NewNoopResumeGenerator(logger *zap.Logger) *NoopResumeGenerator {
	return &NoopResumeGenerator{logger: logger.Named("llm.resume.noop")}
}

// GenerateContent returns empty content — used when LLM generation is disabled.
func (n *NoopResumeGenerator) GenerateContent(_ context.Context, _ map[string]any, _, _ string) (*ResumeContent, error) {
	return &ResumeContent{
		Summary: "LLM generation disabled — please add content manually.",
		Skills:  []string{},
	}, nil
}

// ModelName returns the model identifier.
func (n *NoopResumeGenerator) ModelName() string {
	return "noop"
}

// NewResumeGeneratorFromConfig creates a ResumeGenerator based on configuration.
func NewResumeGeneratorFromConfig(logger *zap.Logger, cfg config.LLMConfig, prompts config.PromptsConfig) ResumeGenerator {
	if cfg.Local.Provider == "ollama" && cfg.Local.BaseURL != "" {
		return NewOllamaResumeGenerator(logger, cfg.Local.BaseURL, cfg.Local.Model, prompts.ResumeGeneration)
	}
	return NewNoopResumeGenerator(logger)
}

// ParseResumeContent parses raw JSON response from an LLM into ResumeContent.
func ParseResumeContent(data []byte) (*ResumeContent, error) {
	var content ResumeContent
	if err := json.Unmarshal(data, &content); err != nil {
		return nil, fmt.Errorf("unmarshal resume content: %w", err)
	}
	if err := validateContent(&content); err != nil {
		return nil, err
	}
	return &content, nil
}

// --- Cover Letter Generation ---

// CoverLetterGenResult holds the LLM output for cover letter generation.
type CoverLetterGenResult struct {
	Content   string   `json:"content"`
	Strengths []string `json:"strengths"`
	Gaps      []string `json:"gaps"`
}

// CoverLetterGenerator defines the interface for LLM-based cover letter generation.
type CoverLetterGenerator interface {
	// GenerateContent creates cover letter content with traceability metadata.
	GenerateContent(ctx context.Context, jobTitle, jobRequirements, jobDescription string, resumeContent *ResumeContent) (*CoverLetterGenResult, error)
	// ModelName returns the identifier of the LLM model used.
	ModelName() string
}

// OllamaCoverLetterGenerator calls a local Ollama model for cover letter generation.
type OllamaCoverLetterGenerator struct {
	logger  *zap.Logger
	baseURL string
	model   string
	prompt  config.PromptPair
}

// NewOllamaCoverLetterGenerator creates a new Ollama-based cover letter generator.
func NewOllamaCoverLetterGenerator(logger *zap.Logger, baseURL, model string, prompt config.PromptPair) *OllamaCoverLetterGenerator {
	return &OllamaCoverLetterGenerator{
		logger:  logger.Named("llm.cover_letter"),
		baseURL: baseURL,
		model:   model,
		prompt:  prompt,
	}
}

// ModelName returns the Ollama model identifier.
func (o *OllamaCoverLetterGenerator) ModelName() string {
	return o.model
}

// GenerateContent sends job + resume context to Ollama for cover letter generation.
func (o *OllamaCoverLetterGenerator) GenerateContent(ctx context.Context, jobTitle, jobRequirements, jobDescription string, resumeContent *ResumeContent) (*CoverLetterGenResult, error) {
	data := map[string]any{
		"JobTitle":       jobTitle,
		"JobRequirements": jobRequirements,
		"JobDescription": jobDescription,
	}
	if resumeContent != nil {
		data["Skills"] = resumeContent.Skills
		data["Experience"] = resumeContent.Experience
		data["Summary"] = resumeContent.Summary
	}

	prompt := o.buildPrompt(data)
	_ = prompt // TODO: use prompt in Ollama API call

	// TODO: Implement Ollama API call (POST /api/generate)
	// For now, return a placeholder
	o.logger.Debug("LLM cover letter generation not yet implemented, returning placeholder",
		zap.String("model", o.model),
	)

	return &CoverLetterGenResult{
		Content:   fmt.Sprintf("Dear Hiring Manager,\n\nI am writing to express my interest in the %s position.", jobTitle),
		Strengths: []string{"placeholder"},
		Gaps:      []string{},
	}, nil
}

// buildPrompt constructs the generation prompt from config template using text/template.
func (o *OllamaCoverLetterGenerator) buildPrompt(data map[string]any) string {
	system := o.prompt.System
	user := o.prompt.User

	if system == "" {
		system = `You are a professional cover letter writer. Write compelling, personalized cover letters.`
	}
	if user == "" {
		user = `Write a cover letter for this job application.

Job Title: {{.JobTitle}}
Requirements: {{.JobRequirements}}

Return the cover letter text only, no JSON.`
	}

	systemBuf := new(strings.Builder)
	systemTmpl, err := template.New("system").Parse(system)
	if err != nil {
		o.logger.Warn("system template parse error, using fallback", zap.Error(err))
		systemBuf.WriteString(`You are a professional cover letter writer. Write compelling, personalized cover letters.`)
	} else if err := systemTmpl.Execute(systemBuf, data); err != nil {
		o.logger.Warn("system template execute error", zap.Error(err))
	}

	userBuf := new(strings.Builder)
	userTmpl, err := template.New("user").Parse(user)
	if err != nil {
		o.logger.Warn("user template parse error, using fallback", zap.Error(err))
		userBuf.WriteString(`Write a cover letter. Return the cover letter text only, no JSON.`)
	} else if err := userTmpl.Execute(userBuf, data); err != nil {
		o.logger.Warn("user template execute error", zap.Error(err))
	}

	return systemBuf.String() + "\n\n" + userBuf.String()
}

// NoopCoverLetterGenerator is a fallback when LLM is disabled or unavailable.
type NoopCoverLetterGenerator struct {
	logger *zap.Logger
}

// NewNoopCoverLetterGenerator creates a no-op cover letter generator.
func NewNoopCoverLetterGenerator(logger *zap.Logger) *NoopCoverLetterGenerator {
	return &NoopCoverLetterGenerator{logger: logger.Named("llm.cover_letter.noop")}
}

// GenerateContent returns empty content — used when LLM generation is disabled.
func (n *NoopCoverLetterGenerator) GenerateContent(_ context.Context, _, _, _ string, _ *ResumeContent) (*CoverLetterGenResult, error) {
	return &CoverLetterGenResult{
		Content:   "LLM generation disabled — please add content manually.",
		Strengths: []string{},
		Gaps:      []string{},
	}, nil
}

// ModelName returns the model identifier.
func (n *NoopCoverLetterGenerator) ModelName() string {
	return "noop"
}

// NewCoverLetterGeneratorFromConfig creates a CoverLetterGenerator based on configuration.
func NewCoverLetterGeneratorFromConfig(logger *zap.Logger, cfg config.LLMConfig, prompts config.PromptsConfig) CoverLetterGenerator {
	if cfg.Local.Provider == "ollama" && cfg.Local.BaseURL != "" {
		return NewOllamaCoverLetterGenerator(logger, cfg.Local.BaseURL, cfg.Local.Model, prompts.CoverLetter)
	}
	return NewNoopCoverLetterGenerator(logger)
}
