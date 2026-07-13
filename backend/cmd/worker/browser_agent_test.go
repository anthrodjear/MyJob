package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestLogger() *zap.Logger {
	return zap.NewNop()
}

// ---------------------------------------------------------------------------
// ScrapeJobs
// ---------------------------------------------------------------------------

func TestHTTPBrowserAgentClient_ScrapeJobs_Success(t *testing.T) {
	expected := ScrapeJobsResponse{
		Jobs: []ScrapedJob{
			{ExternalID: "ext-1", Title: "Go Developer", Company: "Acme"},
			{ExternalID: "ext-2", Title: "Rust Developer", Company: "Beta"},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/scrape/jobs", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var req ScrapeJobsRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "src-1", req.SourceID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expected)
	}))
	defer srv.Close()

	c := NewHTTPBrowserAgentClient(srv.URL, newTestLogger())
	resp, err := c.ScrapeJobs(context.Background(), ScrapeJobsRequest{SourceID: "src-1"})
	require.NoError(t, err)
	assert.Len(t, resp.Jobs, 2)
	assert.Equal(t, "Go Developer", resp.Jobs[0].Title)
}

func TestHTTPBrowserAgentClient_ScrapeJobs_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "internal error")
	}))
	defer srv.Close()

	c := NewHTTPBrowserAgentClient(srv.URL, newTestLogger())
	_, err := c.ScrapeJobs(context.Background(), ScrapeJobsRequest{SourceID: "src-1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestHTTPBrowserAgentClient_ScrapeJobs_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "not json")
	}))
	defer srv.Close()

	c := NewHTTPBrowserAgentClient(srv.URL, newTestLogger())
	_, err := c.ScrapeJobs(context.Background(), ScrapeJobsRequest{SourceID: "src-1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}

func TestHTTPBrowserAgentClient_ScrapeJobs_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c := NewHTTPBrowserAgentClient("http://localhost:1", newTestLogger())
	_, err := c.ScrapeJobs(ctx, ScrapeJobsRequest{SourceID: "src-1"})
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// FillForm
// ---------------------------------------------------------------------------

func TestHTTPBrowserAgentClient_FillForm_Success(t *testing.T) {
	expected := FillFormResponse{Success: true, Message: "Form submitted"}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/forms/fill", r.URL.Path)

		var req FillFormRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "https://example.com/apply", req.PortalURL)
		assert.Equal(t, "greenhouse", req.PortalType)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expected)
	}))
	defer srv.Close()

	c := NewHTTPBrowserAgentClient(srv.URL, newTestLogger())
	resp, err := c.FillForm(context.Background(), FillFormRequest{
		PortalURL:  "https://example.com/apply",
		PortalType: "greenhouse",
		FormData:   map[string]string{"name": "John"},
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, "Form submitted", resp.Message)
}

func TestHTTPBrowserAgentClient_FillForm_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "bad request")
	}))
	defer srv.Close()

	c := NewHTTPBrowserAgentClient(srv.URL, newTestLogger())
	_, err := c.FillForm(context.Background(), FillFormRequest{PortalURL: "https://example.com"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "400")
}

// ---------------------------------------------------------------------------
// CheckEmails
// ---------------------------------------------------------------------------

func TestHTTPBrowserAgentClient_CheckEmails_Success(t *testing.T) {
	expected := CheckEmailsResponse{
		Emails: []CheckedEmail{
			{From: "hr@acme.com", Subject: "Interview Invite", Classification: "interview"},
			{From: "spam@bad.com", Subject: "Buy now", Classification: "spam"},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/emails/check", r.URL.Path)

		var req CheckEmailsRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "tenant-1", req.TenantID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expected)
	}))
	defer srv.Close()

	c := NewHTTPBrowserAgentClient(srv.URL, newTestLogger())
	resp, err := c.CheckEmails(context.Background(), CheckEmailsRequest{
		TenantID: "tenant-1",
		ClientID: "client-1",
		Folders:  []string{"inbox"},
	})
	require.NoError(t, err)
	assert.Len(t, resp.Emails, 2)
	assert.Equal(t, "interview", resp.Emails[0].Classification)
}

func TestHTTPBrowserAgentClient_CheckEmails_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "unauthorized")
	}))
	defer srv.Close()

	c := NewHTTPBrowserAgentClient(srv.URL, newTestLogger())
	_, err := c.CheckEmails(context.Background(), CheckEmailsRequest{TenantID: "t1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}

// ---------------------------------------------------------------------------
// StartVoiceSession
// ---------------------------------------------------------------------------

func TestHTTPBrowserAgentClient_StartVoiceSession_Success(t *testing.T) {
	expected := VoiceSessionResponse{Success: true, Message: "Interview completed"}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/interviews/start", r.URL.Path)

		var req VoiceSessionRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "int-1", req.InterviewID)
		assert.Equal(t, "assist", req.Mode)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expected)
	}))
	defer srv.Close()

	c := NewHTTPBrowserAgentClient(srv.URL, newTestLogger())
	resp, err := c.StartVoiceSession(context.Background(), VoiceSessionRequest{
		InterviewID: "int-1",
		Mode:        "assist",
		Provider:    "openai_realtime",
		Model:       "gpt-4o",
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestHTTPBrowserAgentClient_StartVoiceSession_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGatewayTimeout)
		fmt.Fprint(w, "timeout")
	}))
	defer srv.Close()

	c := NewHTTPBrowserAgentClient(srv.URL, newTestLogger())
	_, err := c.StartVoiceSession(context.Background(), VoiceSessionRequest{InterviewID: "int-1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "504")
}

// ---------------------------------------------------------------------------
// Error body truncation
// ---------------------------------------------------------------------------

func TestHTTPBrowserAgentClient_ErrorBodyTruncation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		longMsg := make([]byte, 1000)
		for i := range longMsg {
			longMsg[i] = 'x'
		}
		w.Write(longMsg)
	}))
	defer srv.Close()

	c := NewHTTPBrowserAgentClient(srv.URL, newTestLogger())
	_, err := c.ScrapeJobs(context.Background(), ScrapeJobsRequest{SourceID: "s1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "truncated")
}
