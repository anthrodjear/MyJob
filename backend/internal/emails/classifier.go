// Classifier uses an LLM to classify incoming emails into job-search categories.
//
// The classifier calls Ollama's /api/generate endpoint with the
// email_classifier prompt from config/application.yaml. The prompt
// expects structured JSON output: {"category", "confidence", "reasoning"}.
//
// Categories (from config):
//   - interview_invite — scheduling or confirming an interview
//   - rejection — employer declined your application
//   - offer — job offer or salary discussion
//   - follow_up — employer following up on your application
//   - spam — unrelated recruitment or marketing
//   - phishing — suspicious links or credential requests
//   - other — doesn't fit other categories
//
// Usage:
//
//	classifier := emails.NewClassifier(logger, llmClient, prompts)
//	result, err := classifier.Classify(ctx, from, subject, body)
package emails

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"text/template"
	"time"

	"go.uber.org/zap"
)

// LLM Client Interface

// LLMClient defines the contract for LLM text generation.
// The classifier uses this to call the email_classifier prompt.
type LLMClient interface {
	Generate(ctx context.Context, system, prompt string) (string, error)
}

// Ollama HTTP Client (implements LLMClient)

// ollamaRequest is the payload for POST /api/generate.
type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	System string `json:"system"`
	Stream bool   `json:"stream"`
}

// ollamaResponse is the response from POST /api/generate.
type ollamaResponse struct {
	Response string `json:"response"`
}

// OllamaClient implements LLMClient using Ollama's HTTP API.
type OllamaClient struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewOllamaClient creates an LLMClient for Ollama.
func NewOllamaClient(baseURL, model string, timeout time.Duration) *OllamaClient {
	return &OllamaClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Generate implements LLMClient.
func (o *OllamaClient) Generate(ctx context.Context, system, prompt string) (string, error) {
	reqBody, err := json.Marshal(ollamaRequest{
		Model:  o.model,
		Prompt: prompt,
		System: system,
		Stream: false,
	})
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := o.baseURL + "/api/generate"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("call ollama: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		msg := string(bodyBytes)
		if len(msg) > 500 {
			msg = msg[:500] + "... (truncated)"
		}
		return "", fmt.Errorf("ollama returned %d: %s", resp.StatusCode, msg)
	}

	var ollamaResp ollamaResponse
	if err := json.Unmarshal(bodyBytes, &ollamaResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}
	return ollamaResp.Response, nil
}

// Prompt Templates

// PromptPair holds the system and user prompt templates for email classification.
type PromptPair struct {
	System string
	User   string
}

// Classification Result

// ClassifyResult holds the LLM classification output for a single email.
type ClassifyResult struct {
	Category   string  `json:"category"`
	Confidence float64 `json:"confidence"`
	Reasoning  string  `json:"reasoning"`
}

// Classifier

// Classifier uses an LLM to classify emails into job-search categories.
type Classifier struct {
	logger         *zap.Logger
	llm            LLMClient
	prompts        PromptPair
	userPromptTmpl *template.Template
}

// NewClassifier creates a new email classifier.
// The LLMClient is used for generation; the prompts pair holds the templates.
// Pre-compiles the user prompt template for performance.
func NewClassifier(logger *zap.Logger, llm LLMClient, prompts PromptPair) (*Classifier, error) {
	// #nosec G708 -- Templates come from application config (YAML), not user input. Local-first app.
	userPromptTmpl, err := template.New("user").Parse(prompts.User)
	if err != nil {
		return nil, fmt.Errorf("parse user prompt template: %w", err)
	}
	return &Classifier{
		logger:         logger.Named("emails.classifier"),
		llm:            llm,
		prompts:        prompts,
		userPromptTmpl: userPromptTmpl,
	}, nil
}

// NewClassifierFromConfig creates a classifier using Ollama's HTTP API.
// It builds an OllamaClient (implementing LLMClient) and passes it to NewClassifier.
// The timeout is configurable via the timeout parameter.
func NewClassifierFromConfig(logger *zap.Logger, baseURL, model string, timeout time.Duration, prompts PromptPair) (*Classifier, error) {
	ollamaClient := NewOllamaClient(baseURL, model, timeout)
	return NewClassifier(logger, ollamaClient, prompts)
}

// Classification

// Classify determines the email category using the LLM.
//
// Parameters:
//   - from: sender email address
//   - subject: email subject line
//   - body: email body text (may be truncated for large emails)
//
// Returns a ClassifyResult with category, confidence, and reasoning.
// The category is validated against known values; unknown categories
// are mapped to "other".
func (c *Classifier) Classify(ctx context.Context, from, subject, body string) (*ClassifyResult, error) {
	return c.classify(ctx, from, subject, body)
}

// classify is the shared implementation for both LLMClient and HTTP paths.
func (c *Classifier) classify(ctx context.Context, from, subject, body string) (*ClassifyResult, error) {
	// Build prompt variables
	truncatedBody := truncate(body, 2000)
	if len(truncatedBody) < len(body) {
		c.logger.Debug(
			"truncated email body for classification",
			zap.Int("original_len", len(body)),
			zap.Int("truncated_len", len(truncatedBody)),
		)
	}
	vars := struct {
		From    string
		Subject string
		Body    string
	}{
		From:    from,
		Subject: subject,
		Body:    truncatedBody,
	}

	var userPrompt bytes.Buffer
	if err := c.userPromptTmpl.Execute(&userPrompt, vars); err != nil {
		return nil, fmt.Errorf("execute user prompt: %w", err)
	}

	// Call LLM (via LLMClient interface)
	output, err := c.llm.Generate(ctx, c.prompts.System, userPrompt.String())
	if err != nil {
		return nil, fmt.Errorf("classify email: %w", err)
	}

	// Parse structured output
	result, err := parseClassifyOutput(output)
	if err != nil {
		c.logger.Warn(
			"failed to parse LLM output",
			zap.String("output", output),
			zap.Error(err),
		)
		return nil, fmt.Errorf("parse LLM output: %w", err)
	}

	// Validate category
	if !IsValidClassification(result.Category) {
		c.logger.Warn(
			"unknown classification from LLM",
			zap.String("category", result.Category),
		)
		result.Category = ClassificationOther
	}

	return result, nil
}

// Helpers

// parseClassifyOutput extracts the JSON classification from LLM output.
// Handles cases where the LLM wraps JSON in markdown code blocks.
// Falls back to direct parsing if code fence extraction fails.
func parseClassifyOutput(output string) (*ClassifyResult, error) {
	output = strings.TrimSpace(output)

	// First attempt: try direct JSON parsing (most common case)
	var result ClassifyResult
	if err := json.Unmarshal([]byte(output), &result); err == nil {
		return &result, nil
	}

	// Second attempt: extract from markdown code fences
	codeFenceRegex := regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)\\s*```")
	if matches := codeFenceRegex.FindStringSubmatch(output); len(matches) > 1 {
		extracted := strings.TrimSpace(matches[1])
		if err := json.Unmarshal([]byte(extracted), &result); err == nil {
			return &result, nil
		}
	}

	// Fallback failed - return error
	return nil, fmt.Errorf("unmarshal classify output: invalid JSON format")
}

// truncate shortens a string to maxLen characters, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "... (truncated)"
}
