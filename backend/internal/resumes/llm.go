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
		o.logger.Warn("system template parse error", zap.Error(err))
		systemBuf.WriteString(system)
	} else if err := systemTmpl.Execute(systemBuf, data); err != nil {
		o.logger.Warn("system template execute error", zap.Error(err))
	}

	userBuf := new(strings.Builder)
	userTmpl, err := template.New("user").Parse(user)
	if err != nil {
		o.logger.Warn("user template parse error", zap.Error(err))
		userBuf.WriteString(user)
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
