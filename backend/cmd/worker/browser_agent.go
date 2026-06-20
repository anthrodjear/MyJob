package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// BrowserAgentClient defines the interface for communicating with the browser agent service.
// The browser agent handles Playwright-based scraping and form filling.
type BrowserAgentClient interface {
	// ScrapeJobs calls the browser agent to scrape jobs from a source.
	ScrapeJobs(ctx context.Context, req ScrapeJobsRequest) (*ScrapeJobsResponse, error)

	// FillForm calls the browser agent to fill and submit a form on a portal.
	FillForm(ctx context.Context, req FillFormRequest) (*FillFormResponse, error)

	// CheckEmails calls the browser agent to check emails via Microsoft Graph.
	CheckEmails(ctx context.Context, req CheckEmailsRequest) (*CheckEmailsResponse, error)

	// StartVoiceSession calls the browser agent to start a voice interview session.
	// This is a long-running call (up to 30 minutes) that blocks until the interview completes.
	StartVoiceSession(ctx context.Context, req VoiceSessionRequest) (*VoiceSessionResponse, error)
}

// VoiceSessionRequest is the payload sent to the browser agent for starting a voice interview.
type VoiceSessionRequest struct {
	InterviewID     string `json:"interview_id"`
	ApplicationID   string `json:"application_id"`
	Mode            string `json:"mode"`
	ExternalSession string `json:"external_session"`
	Provider        string `json:"provider"`
	Model           string `json:"model"`
}

// VoiceSessionResponse is the response from the browser agent after the voice interview completes.
type VoiceSessionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// ScrapeJobsRequest is the payload sent to the browser agent for job scraping.
type ScrapeJobsRequest struct {
	SourceID string   `json:"source_id"`
	BaseURL  string   `json:"base_url"`
	Keywords []string `json:"keywords"`
	Location string   `json:"location"`
	Config   any      `json:"config,omitempty"`
}

// ScrapeJobsResponse is the response from the browser agent after scraping.
type ScrapeJobsResponse struct {
	Jobs   []ScrapedJob `json:"jobs"`
	Errors []string     `json:"errors,omitempty"`
}

// ScrapedJob represents a job scraped by the browser agent.
type ScrapedJob struct {
	ExternalID     string `json:"external_id"`
	Title          string `json:"title"`
	Company        string `json:"company"`
	Location       string `json:"location"`
	RemoteType     string `json:"remote_type"`
	SalaryMin      int    `json:"salary_min"`
	SalaryMax      int    `json:"salary_max"`
	SalaryCurrency string `json:"salary_currency"`
	Description    string `json:"description"`
	Requirements   string `json:"requirements"`
	URL            string `json:"url"`
	ApplicationURL string `json:"application_url"`
	CompanyURL     string `json:"company_url"`
	Source         string `json:"source"`
}

// FillFormRequest is the payload sent to the browser agent for form filling.
type FillFormRequest struct {
	PortalURL  string            `json:"portal_url"`
	PortalType string            `json:"portal_type"`
	FormData   map[string]string `json:"form_data"`
	ResumePath string            `json:"resume_path,omitempty"`
}

// FillFormResponse is the response from the browser agent after form filling.
type FillFormResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message,omitempty"`
	ScreenshotURL string `json:"screenshot_url,omitempty"`
}

// CheckEmailsRequest is the payload sent to the browser agent for email checking.
type CheckEmailsRequest struct {
	TenantID     string `json:"tenant_id"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Folders      []string `json:"folders"`
	ApplicationID string `json:"application_id,omitempty"`
}

// CheckedEmail represents a single email checked by the browser agent.
type CheckedEmail struct {
	From        string `json:"from"`
	Subject     string `json:"subject"`
	Body        string `json:"body"`
	ReceivedAt  string `json:"received_at"`
	Classification string `json:"classification"` // "rejection", "interview", "offer", "spam", "other"
}

// CheckEmailsResponse is the response from the browser agent after email checking.
type CheckEmailsResponse struct {
	Emails []CheckedEmail `json:"emails"`
	Errors []string       `json:"errors,omitempty"`
}

// HTTPBrowserAgentClient implements BrowserAgentClient using HTTP calls.
type HTTPBrowserAgentClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewHTTPBrowserAgentClient creates a new HTTP client for the browser agent.
func NewHTTPBrowserAgentClient(baseURL string, logger *zap.Logger) *HTTPBrowserAgentClient {
	return &HTTPBrowserAgentClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 35 * time.Minute, // must exceed longest task (voice sessions: 30min)
		},
		logger: logger.Named("browser-agent-client"),
	}
}

// ScrapeJobs calls the browser agent's scrape endpoint.
func (c *HTTPBrowserAgentClient) ScrapeJobs(ctx context.Context, req ScrapeJobsRequest) (*ScrapeJobsResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("browser-agent: marshal request: %w", err)
	}

	url := c.baseURL + "/api/scrape/jobs"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("browser-agent: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	c.logger.Debug("scraping jobs",
		zap.String("source_id", req.SourceID),
		zap.String("url", url),
	)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("browser-agent: scrape jobs: %w", err)
	}
	defer resp.Body.Close()

	// Limit response body to 10MB to prevent memory exhaustion
	const maxResponseBody = 10 << 20
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return nil, fmt.Errorf("browser-agent: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Truncate large error bodies for logging
		msg := string(body)
		if len(msg) > 500 {
			msg = msg[:500] + "... (truncated)"
		}
		return nil, fmt.Errorf("browser-agent: scrape jobs returned %d: %s", resp.StatusCode, msg)
	}

	var result ScrapeJobsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("browser-agent: unmarshal response: %w", err)
	}

	return &result, nil
}

// FillForm calls the browser agent's form-filling endpoint.
func (c *HTTPBrowserAgentClient) FillForm(ctx context.Context, req FillFormRequest) (*FillFormResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("browser-agent: marshal fill form request: %w", err)
	}

	url := c.baseURL + "/api/forms/fill"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("browser-agent: create fill form request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	c.logger.Debug("filling form",
		zap.String("portal_url", req.PortalURL),
		zap.String("portal_type", req.PortalType),
	)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("browser-agent: fill form: %w", err)
	}
	defer resp.Body.Close()

	const maxResponseBody = 10 << 20
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return nil, fmt.Errorf("browser-agent: read fill form response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		msg := string(body)
		if len(msg) > 500 {
			msg = msg[:500] + "... (truncated)"
		}
		return nil, fmt.Errorf("browser-agent: fill form returned %d: %s", resp.StatusCode, msg)
	}

	var result FillFormResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("browser-agent: unmarshal fill form response: %w", err)
	}

	return &result, nil
}

// CheckEmails calls the browser agent's email checking endpoint.
func (c *HTTPBrowserAgentClient) CheckEmails(ctx context.Context, req CheckEmailsRequest) (*CheckEmailsResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("browser-agent: marshal check emails request: %w", err)
	}

	url := c.baseURL + "/api/emails/check"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("browser-agent: create check emails request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	c.logger.Debug("checking emails")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("browser-agent: check emails: %w", err)
	}
	defer resp.Body.Close()

	const maxResponseBody = 10 << 20
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return nil, fmt.Errorf("browser-agent: read check emails response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		msg := string(body)
		if len(msg) > 500 {
			msg = msg[:500] + "... (truncated)"
		}
		return nil, fmt.Errorf("browser-agent: check emails returned %d: %s", resp.StatusCode, msg)
	}

	var result CheckEmailsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("browser-agent: unmarshal check emails response: %w", err)
	}

	return &result, nil
}

// StartVoiceSession calls the browser agent's voice interview endpoint.
// This is a long-running call that blocks until the interview completes
// (up to 30 minutes). The browser-agent joins a LiveKit room, runs the
// interview, and returns when the session ends.
func (c *HTTPBrowserAgentClient) StartVoiceSession(ctx context.Context, req VoiceSessionRequest) (*VoiceSessionResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("browser-agent: marshal voice session request: %w", err)
	}

	url := c.baseURL + "/api/interviews/start"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("browser-agent: create voice session request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	c.logger.Info("starting voice session",
		zap.String("interview_id", req.InterviewID),
		zap.String("mode", req.Mode),
		zap.String("provider", req.Provider),
	)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("browser-agent: start voice session: %w", err)
	}
	defer resp.Body.Close()

	const maxResponseBody = 10 << 20
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return nil, fmt.Errorf("browser-agent: read voice session response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		msg := string(body)
		if len(msg) > 500 {
			msg = msg[:500] + "... (truncated)"
		}
		return nil, fmt.Errorf("browser-agent: voice session returned %d: %s", resp.StatusCode, msg)
	}

	var result VoiceSessionResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("browser-agent: unmarshal voice session response: %w", err)
	}

	c.logger.Info("voice session completed",
		zap.String("interview_id", req.InterviewID),
		zap.Bool("success", result.Success),
	)

	return &result, nil
}
