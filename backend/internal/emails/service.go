// Service contains business logic for the emails domain.
//
// Responsibilities:
//   - Store incoming emails (deduplication by message_id)
//   - Retrieve emails with filters and pagination
//   - Update email state (read status, reply draft)
//   - Re-classify emails via LLM
//
// This file contains NO HTTP handling, NO database queries, and NO LLM calls directly.
// It orchestrates between Repository (persistence) and Classifier (LLM).
package emails

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Repository Interface

// RepositoryInterface defines the persistence contract for emails.
// The service depends on this interface, not the concrete implementation.
type RepositoryInterface interface {
	GetByID(ctx context.Context, id uuid.UUID) (*Email, error)
	GetByMessageID(ctx context.Context, messageID string) (*Email, error)
	List(ctx context.Context, filter ListFilter) ([]Email, int64, error)
	Upsert(ctx context.Context, e *Email) (uuid.UUID, error)
	UpdateReadStatus(ctx context.Context, id uuid.UUID, isRead bool) error
	UpdateClassification(ctx context.Context, id uuid.UUID, classification string) error
	UpdateReplyDraft(ctx context.Context, id uuid.UUID, draft *string) error
}

// Classifier Interface

// ClassifierInterface defines the contract for email classification.
// The service depends on this interface, not the concrete classifier.
type ClassifierInterface interface {
	Classify(ctx context.Context, from, subject, body string) (*ClassifyResult, error)
}

// ListFilter is defined in repository.go (same package).

// Service

// Service contains business logic for the emails domain.
type Service struct {
	repo       RepositoryInterface
	classifier ClassifierInterface
}

// NewService creates a new emails service.
func NewService(repo RepositoryInterface, classifier ClassifierInterface) *Service {
	return &Service{
		repo:       repo,
		classifier: classifier,
	}
}

// Email Operations

// StoreEmailParams holds parameters for Store operation.
// Reduces parameter count and improves readability.
type StoreEmailParams struct {
	ApplicationID  *uuid.UUID
	MessageID      string
	FromAddress    string
	ToAddress      *string
	Subject        *string
	Body           *string
	ReceivedAt     time.Time
	Classification *string
}

// Store stores an incoming email, classifying it if classification is not provided.
// Returns the email ID and the classification used.
// If an email with the same message_id exists, updates it (upsert semantics).
func (s *Service) Store(ctx context.Context, params StoreEmailParams) (uuid.UUID, string, error) {
	// Use provided classification or classify via LLM
	var class string
	if params.Classification != nil && *params.Classification != "" {
		if !IsValidClassification(*params.Classification) {
			return uuid.Nil, "", ErrInvalidClassification
		}
		class = *params.Classification
	} else if s.classifier != nil {
		result, err := s.classifier.Classify(ctx, params.FromAddress, ptrOrEmpty(params.Subject), ptrOrEmpty(params.Body))
		if err != nil {
			return uuid.Nil, "", fmt.Errorf("classify email: %w", err)
		}
		class = result.Category
	} else {
		class = ClassificationOther
	}

	email := &Email{
		ID:             uuid.New(),
		ApplicationID:  params.ApplicationID,
		MessageID:      params.MessageID,
		FromAddress:    params.FromAddress,
		ToAddress:      params.ToAddress,
		Subject:        params.Subject,
		Body:           params.Body,
		ReceivedAt:     params.ReceivedAt,
		Classification: &class,
		IsRead:         false,
		ReplyDraft:     nil,
		CreatedAt:      time.Now(),
	}

	id, err := s.repo.Upsert(ctx, email)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("store email: %w", err)
	}
	return id, class, nil
}

// GetByID retrieves an email by ID.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Email, error) {
	return s.getEmail(ctx, id)
}

// List returns emails matching the filter with total count.
func (s *Service) List(ctx context.Context, filter ListFilter) ([]Email, int64, error) {
	// Cap limit at 100
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	return s.repo.List(ctx, filter)
}

// MarkRead marks an email as read or unread.
func (s *Service) MarkRead(ctx context.Context, id uuid.UUID, isRead bool) error {
	if err := s.repo.UpdateReadStatus(ctx, id, isRead); err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("mark read: %w", err)
	}
	return nil
}

// UpdateDraft updates the reply draft for an email.
func (s *Service) UpdateDraft(ctx context.Context, id uuid.UUID, draft *string) error {
	if err := s.repo.UpdateReplyDraft(ctx, id, draft); err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("update draft: %w", err)
	}
	return nil
}

// Reclassify re-classifies an email using the LLM and updates its classification.
func (s *Service) Reclassify(ctx context.Context, id uuid.UUID) (*ClassifyResult, error) {
	if s.classifier == nil {
		return nil, fmt.Errorf("classifier not configured")
	}

	email, err := s.getEmail(ctx, id)
	if err != nil {
		return nil, err
	}

	result, err := s.classifier.Classify(ctx, email.FromAddress, ptrOrEmpty(email.Subject), ptrOrEmpty(email.Body))
	if err != nil {
		return nil, fmt.Errorf("reclassify email: %w", err)
	}

	if err := s.repo.UpdateClassification(ctx, id, result.Category); err != nil {
		return nil, fmt.Errorf("update classification: %w", err)
	}

	return result, nil
}

// Helpers

// getEmail retrieves an email by ID with consistent error handling.
// Returns ErrNotFound if not found, wraps other errors.
func (s *Service) getEmail(ctx context.Context, id uuid.UUID) (*Email, error) {
	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get email: %w", err)
	}
	return e, nil
}

// ptrOrEmpty returns the string value or empty string if nil.
func ptrOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
