package emails

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Helpers
// ============================================================================

// newMockRepo creates a sqlmock DB wrapped in sqlx and an emails Repository.
// Returns the mock, the repo, and a cleanup func that asserts all expectations were met.
func newMockRepo(t *testing.T) (sqlmock.Sqlmock, *Repository, func()) {
	t.Helper()

	db, mock, err := sqlmock.New()
	require.NoError(t, err, "failed to open sqlmock database")

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewRepository(sqlxDB)

	cleanup := func() {
		err := mock.ExpectationsWereMet()
		assert.NoError(t, err, "unmet sqlmock expectations")
		db.Close()
	}

	return mock, repo, cleanup
}

// emailColNames is the column name slice used when building sqlmock.NewRows.
// Duplicates the columns from the emailColumns const (model.go) as a []string.
var emailColNames = []string{
	"id", "application_id", "message_id", "from_address", "to_address",
	"subject", "body", "received_at", "classification", "is_read", "reply_draft", "created_at",
}

// buildEmailQuery builds the expected SELECT query prefix for the emails table.
// The emailColumns constant includes leading/trailing whitespace, so we normalise
// it by constructing the actual query the same way the repo does.
var selectPrefix = "SELECT \n\tid, application_id, message_id, from_address, to_address,\n\tsubject, body, received_at, classification, is_read, reply_draft, created_at\n FROM emails"

// newTestEmail returns a fully populated Email for use in test expectations.
func newTestEmail(id uuid.UUID) *Email {
	appID := uuid.New()
	toAddr := "recipient@example.com"
	subject := "Interview Invitation"
	body := "We would like to invite you..."
	class := ClassificationInterviewInvite
	draft := "Thank you for your interest"
	now := time.Now().Truncate(time.Microsecond)

	return &Email{
		ID:             id,
		ApplicationID:  &appID,
		MessageID:      "msg-001",
		FromAddress:    "hr@company.com",
		ToAddress:      &toAddr,
		Subject:        &subject,
		Body:           &body,
		ReceivedAt:     now,
		Classification: &class,
		IsRead:         false,
		ReplyDraft:     &draft,
		CreatedAt:      now,
	}
}

// addEmailRow adds a single row to the given sqlmock rows from an Email struct.
func addEmailRow(rows *sqlmock.Rows, e *Email) {
	rows.AddRow(
		e.ID, e.ApplicationID, e.MessageID, e.FromAddress, e.ToAddress,
		e.Subject, e.Body, e.ReceivedAt, e.Classification, e.IsRead, e.ReplyDraft, e.CreatedAt,
	)
}

// ============================================================================
// GetByID
// ============================================================================

func TestRepository_GetByID_Success(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	email := newTestEmail(uuid.New())

	rows := sqlmock.NewRows(emailColNames)
	addEmailRow(rows, email)

	mock.ExpectQuery(regexp.QuoteMeta(selectPrefix + " WHERE id = $1")).
		WithArgs(email.ID).
		WillReturnRows(rows)

	got, err := repo.GetByID(context.Background(), email.ID)
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, email.ID, got.ID)
	assert.Equal(t, email.ApplicationID, got.ApplicationID)
	assert.Equal(t, email.MessageID, got.MessageID)
	assert.Equal(t, email.FromAddress, got.FromAddress)
	assert.Equal(t, email.ToAddress, got.ToAddress)
	assert.Equal(t, email.Subject, got.Subject)
	assert.Equal(t, email.Body, got.Body)
	assert.Equal(t, email.ReceivedAt.Unix(), got.ReceivedAt.Unix())
	assert.Equal(t, email.Classification, got.Classification)
	assert.Equal(t, email.IsRead, got.IsRead)
	assert.Equal(t, email.ReplyDraft, got.ReplyDraft)
}

func TestRepository_GetByID_ErrNotFound(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	id := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(selectPrefix + " WHERE id = $1")).
		WithArgs(id).
		WillReturnError(sql.ErrNoRows)

	got, err := repo.GetByID(context.Background(), id)
	assert.ErrorIs(t, err, ErrNotFound)
	assert.Nil(t, got)
}

func TestRepository_GetByID_DatabaseError(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	id := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(selectPrefix + " WHERE id = $1")).
		WithArgs(id).
		WillReturnError(sql.ErrConnDone)

	got, err := repo.GetByID(context.Background(), id)
	assert.ErrorIs(t, err, sql.ErrConnDone)
	assert.Contains(t, err.Error(), "get email by id")
	assert.Nil(t, got)
}

func TestRepository_GetByID_NullableFields(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	id := uuid.New()
	now := time.Now().Truncate(time.Microsecond)

	// All nullable fields set to nil
	rows := sqlmock.NewRows(emailColNames).AddRow(
		id, nil, "msg-nullable", "sender@example.com",
		nil, nil, nil, now, nil, false, nil, now,
	)

	mock.ExpectQuery(regexp.QuoteMeta(selectPrefix + " WHERE id = $1")).
		WithArgs(id).
		WillReturnRows(rows)

	got, err := repo.GetByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, id, got.ID)
	assert.Nil(t, got.ApplicationID)
	assert.Nil(t, got.ToAddress)
	assert.Nil(t, got.Subject)
	assert.Nil(t, got.Body)
	assert.Nil(t, got.Classification)
	assert.Nil(t, got.ReplyDraft)
	assert.False(t, got.IsRead)
}

// ============================================================================
// GetByMessageID
// ============================================================================

func TestRepository_GetByMessageID_Success(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	email := newTestEmail(uuid.New())

	rows := sqlmock.NewRows(emailColNames)
	addEmailRow(rows, email)

	mock.ExpectQuery(regexp.QuoteMeta(selectPrefix + " WHERE message_id = $1")).
		WithArgs(email.MessageID).
		WillReturnRows(rows)

	got, err := repo.GetByMessageID(context.Background(), email.MessageID)
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, email.ID, got.ID)
	assert.Equal(t, email.MessageID, got.MessageID)
}

func TestRepository_GetByMessageID_ErrNotFound(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	mock.ExpectQuery(regexp.QuoteMeta(selectPrefix + " WHERE message_id = $1")).
		WithArgs("nonexistent").
		WillReturnError(sql.ErrNoRows)

	got, err := repo.GetByMessageID(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, ErrNotFound)
	assert.Nil(t, got)
}

func TestRepository_GetByMessageID_DatabaseError(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	mock.ExpectQuery(regexp.QuoteMeta(selectPrefix + " WHERE message_id = $1")).
		WithArgs("msg-001").
		WillReturnError(sql.ErrConnDone)

	got, err := repo.GetByMessageID(context.Background(), "msg-001")
	assert.ErrorIs(t, err, sql.ErrConnDone)
	assert.Contains(t, err.Error(), "get email by message_id")
	assert.Nil(t, got)
}

// ============================================================================
// List
// ============================================================================

func TestRepository_List_Success(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	email1 := newTestEmail(uuid.New())
	email2 := newTestEmail(uuid.New())
	email2.MessageID = "msg-002"
	email2.FromAddress = "recruiter@other.com"

	appID := email1.ApplicationID

	// COUNT
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM emails WHERE application_id = $1 AND classification = $2")).
		WithArgs(*appID, ClassificationInterviewInvite).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	// SELECT with filters
	filterQuery := selectPrefix + " WHERE application_id = $1 AND classification = $2 ORDER BY received_at DESC"
	rows := sqlmock.NewRows(emailColNames)
	addEmailRow(rows, email1)
	addEmailRow(rows, email2)

	mock.ExpectQuery(regexp.QuoteMeta(filterQuery)).
		WithArgs(*appID, ClassificationInterviewInvite).
		WillReturnRows(rows)

	got, total, err := repo.List(context.Background(), ListFilter{
		ApplicationID:  *appID,
		Classification: ClassificationInterviewInvite,
	})
	require.NoError(t, err)

	assert.Equal(t, int64(2), total)
	assert.Len(t, got, 2)
	assert.Equal(t, email1.ID, got[0].ID)
	assert.Equal(t, email2.ID, got[1].ID)
}

func TestRepository_List_NoFilters(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	email := newTestEmail(uuid.New())

	// COUNT (no WHERE clause)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM emails")).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// SELECT (no WHERE, no LIMIT/OFFSET)
	query := selectPrefix + " ORDER BY received_at DESC"
	rows := sqlmock.NewRows(emailColNames)
	addEmailRow(rows, email)

	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WillReturnRows(rows)

	got, total, err := repo.List(context.Background(), ListFilter{})
	require.NoError(t, err)

	assert.Equal(t, int64(1), total)
	assert.Len(t, got, 1)
	assert.Equal(t, email.ID, got[0].ID)
}

func TestRepository_List_WithPagination(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	email := newTestEmail(uuid.New())

	// COUNT
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM emails")).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(50))

	// SELECT with LIMIT $1 OFFSET $2
	query := selectPrefix + " ORDER BY received_at DESC LIMIT $1 OFFSET $2"
	rows := sqlmock.NewRows(emailColNames)
	addEmailRow(rows, email)

	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(10, 20).
		WillReturnRows(rows)

	got, total, err := repo.List(context.Background(), ListFilter{
		Limit:  10,
		Offset: 20,
	})
	require.NoError(t, err)

	assert.Equal(t, int64(50), total)
	assert.Len(t, got, 1)
}

func TestRepository_List_NoResults(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	appID := uuid.New()

	// COUNT = 0
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM emails WHERE application_id = $1")).
		WithArgs(appID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	// SELECT returns empty
	query := selectPrefix + " WHERE application_id = $1 ORDER BY received_at DESC"
	rows := sqlmock.NewRows(emailColNames)

	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(appID).
		WillReturnRows(rows)

	got, total, err := repo.List(context.Background(), ListFilter{
		ApplicationID: appID,
	})
	require.NoError(t, err)

	assert.Equal(t, int64(0), total)
	assert.Empty(t, got)
}

func TestRepository_List_CountError(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM emails")).
		WillReturnError(sql.ErrConnDone)

	got, total, err := repo.List(context.Background(), ListFilter{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "count emails")
	assert.ErrorIs(t, err, sql.ErrConnDone)
	assert.Nil(t, got)
	assert.Equal(t, int64(0), total)
}

func TestRepository_List_SelectError(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	// COUNT succeeds
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM emails")).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

	// SELECT fails
	query := selectPrefix + " ORDER BY received_at DESC"
	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WillReturnError(sql.ErrConnDone)

	got, total, err := repo.List(context.Background(), ListFilter{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "list emails")
	assert.ErrorIs(t, err, sql.ErrConnDone)
	assert.Nil(t, got)
	assert.Equal(t, int64(0), total)
}

func TestRepository_List_OnlyLimit(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	email := newTestEmail(uuid.New())

	// COUNT
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM emails")).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	// SELECT with LIMIT $1 only
	query := selectPrefix + " ORDER BY received_at DESC LIMIT $1"
	rows := sqlmock.NewRows(emailColNames)
	addEmailRow(rows, email)

	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(5).
		WillReturnRows(rows)

	got, total, err := repo.List(context.Background(), ListFilter{
		Limit: 5,
	})
	require.NoError(t, err)

	assert.Equal(t, int64(3), total)
	assert.Len(t, got, 1)
}

func TestRepository_List_OnlyOffset(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	email := newTestEmail(uuid.New())

	// COUNT
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM emails")).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))

	// SELECT with OFFSET $1 only
	query := selectPrefix + " ORDER BY received_at DESC OFFSET $1"
	rows := sqlmock.NewRows(emailColNames)
	addEmailRow(rows, email)

	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(100).
		WillReturnRows(rows)

	got, total, err := repo.List(context.Background(), ListFilter{
		Offset: 100,
	})
	require.NoError(t, err)

	assert.Equal(t, int64(10), total)
	assert.Len(t, got, 1)
}

// ============================================================================
// Upsert
// ============================================================================

const upsertQuery = `INSERT INTO emails (id, application_id, message_id, from_address, to_address,
		                     subject, body, received_at, classification, is_read, reply_draft)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 ON CONFLICT (message_id) DO UPDATE
			SET application_id  = COALESCE(EXCLUDED.application_id, emails.application_id),
				from_address    = EXCLUDED.from_address,
				to_address      = COALESCE(EXCLUDED.to_address, emails.to_address),
				subject         = COALESCE(EXCLUDED.subject, emails.subject),
				body            = COALESCE(EXCLUDED.body, emails.body),
				classification  = COALESCE(EXCLUDED.classification, emails.classification)
		 RETURNING id`

func TestRepository_Upsert_Success(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	email := newTestEmail(uuid.New())

	mock.ExpectQuery(regexp.QuoteMeta(upsertQuery)).
		WithArgs(
			email.ID, email.ApplicationID, email.MessageID, email.FromAddress, email.ToAddress,
			email.Subject, email.Body, email.ReceivedAt, email.Classification, email.IsRead, email.ReplyDraft,
		).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(email.ID))

	id, err := repo.Upsert(context.Background(), email)
	require.NoError(t, err)

	assert.Equal(t, email.ID, id)
}

func TestRepository_Upsert_ReturnsExistingID(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	email := newTestEmail(uuid.New())
	existingID := uuid.New()

	// Simulate ON CONFLICT returning a different existing ID
	mock.ExpectQuery(regexp.QuoteMeta(upsertQuery)).
		WithArgs(
			email.ID, email.ApplicationID, email.MessageID, email.FromAddress, email.ToAddress,
			email.Subject, email.Body, email.ReceivedAt, email.Classification, email.IsRead, email.ReplyDraft,
		).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(existingID))

	id, err := repo.Upsert(context.Background(), email)
	require.NoError(t, err)

	assert.Equal(t, existingID, id)
}

func TestRepository_Upsert_DatabaseError(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	email := newTestEmail(uuid.New())

	mock.ExpectQuery(regexp.QuoteMeta(upsertQuery)).
		WithArgs(
			email.ID, email.ApplicationID, email.MessageID, email.FromAddress, email.ToAddress,
			email.Subject, email.Body, email.ReceivedAt, email.Classification, email.IsRead, email.ReplyDraft,
		).
		WillReturnError(sql.ErrConnDone)

	id, err := repo.Upsert(context.Background(), email)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "upsert email")
	assert.ErrorIs(t, err, sql.ErrConnDone)
	assert.Equal(t, uuid.Nil, id)
}

// ============================================================================
// UpdateReadStatus
// ============================================================================

func TestRepository_UpdateReadStatus_Success(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	id := uuid.New()

	mock.ExpectExec(regexp.QuoteMeta("UPDATE emails SET is_read = $1 WHERE id = $2")).
		WithArgs(true, id).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateReadStatus(context.Background(), id, true)
	assert.NoError(t, err)
}

func TestRepository_UpdateReadStatus_Success_False(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	id := uuid.New()

	mock.ExpectExec(regexp.QuoteMeta("UPDATE emails SET is_read = $1 WHERE id = $2")).
		WithArgs(false, id).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateReadStatus(context.Background(), id, false)
	assert.NoError(t, err)
}

func TestRepository_UpdateReadStatus_ErrNotFound(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	id := uuid.New()

	mock.ExpectExec(regexp.QuoteMeta("UPDATE emails SET is_read = $1 WHERE id = $2")).
		WithArgs(true, id).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.UpdateReadStatus(context.Background(), id, true)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestRepository_UpdateReadStatus_DatabaseError(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	id := uuid.New()

	mock.ExpectExec(regexp.QuoteMeta("UPDATE emails SET is_read = $1 WHERE id = $2")).
		WithArgs(true, id).
		WillReturnError(sql.ErrConnDone)

	err := repo.UpdateReadStatus(context.Background(), id, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update read status")
	assert.ErrorIs(t, err, sql.ErrConnDone)
}

func TestRepository_UpdateReadStatus_RowsAffectedError(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	id := uuid.New()

	mock.ExpectExec(regexp.QuoteMeta("UPDATE emails SET is_read = $1 WHERE id = $2")).
		WithArgs(true, id).
		WillReturnResult(sqlmock.NewErrorResult(sql.ErrConnDone))

	err := repo.UpdateReadStatus(context.Background(), id, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rows affected")
}

// ============================================================================
// UpdateClassification
// ============================================================================

func TestRepository_UpdateClassification_Success(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	id := uuid.New()

	mock.ExpectExec(regexp.QuoteMeta("UPDATE emails SET classification = $1 WHERE id = $2")).
		WithArgs(ClassificationInterviewInvite, id).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateClassification(context.Background(), id, ClassificationInterviewInvite)
	assert.NoError(t, err)
}

func TestRepository_UpdateClassification_ErrNotFound(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	id := uuid.New()

	mock.ExpectExec(regexp.QuoteMeta("UPDATE emails SET classification = $1 WHERE id = $2")).
		WithArgs(ClassificationRejection, id).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.UpdateClassification(context.Background(), id, ClassificationRejection)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestRepository_UpdateClassification_DatabaseError(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	id := uuid.New()

	mock.ExpectExec(regexp.QuoteMeta("UPDATE emails SET classification = $1 WHERE id = $2")).
		WithArgs(ClassificationSpam, id).
		WillReturnError(sql.ErrConnDone)

	err := repo.UpdateClassification(context.Background(), id, ClassificationSpam)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update classification")
	assert.ErrorIs(t, err, sql.ErrConnDone)
}

func TestRepository_UpdateClassification_RowsAffectedError(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	id := uuid.New()

	mock.ExpectExec(regexp.QuoteMeta("UPDATE emails SET classification = $1 WHERE id = $2")).
		WithArgs(ClassificationOffer, id).
		WillReturnResult(sqlmock.NewErrorResult(sql.ErrConnDone))

	err := repo.UpdateClassification(context.Background(), id, ClassificationOffer)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rows affected")
}

// ============================================================================
// UpdateReplyDraft
// ============================================================================

func TestRepository_UpdateReplyDraft_Success(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	id := uuid.New()
	draft := "Thank you for the interview opportunity"

	mock.ExpectExec(regexp.QuoteMeta("UPDATE emails SET reply_draft = $1 WHERE id = $2")).
		WithArgs(&draft, id).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateReplyDraft(context.Background(), id, &draft)
	assert.NoError(t, err)
}

func TestRepository_UpdateReplyDraft_NilDraft(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	id := uuid.New()

	mock.ExpectExec(regexp.QuoteMeta("UPDATE emails SET reply_draft = $1 WHERE id = $2")).
		WithArgs(nil, id).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateReplyDraft(context.Background(), id, nil)
	assert.NoError(t, err)
}

func TestRepository_UpdateReplyDraft_ErrNotFound(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	id := uuid.New()
	draft := "Draft text"

	mock.ExpectExec(regexp.QuoteMeta("UPDATE emails SET reply_draft = $1 WHERE id = $2")).
		WithArgs(&draft, id).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.UpdateReplyDraft(context.Background(), id, &draft)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestRepository_UpdateReplyDraft_DatabaseError(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	id := uuid.New()
	draft := "Draft text"

	mock.ExpectExec(regexp.QuoteMeta("UPDATE emails SET reply_draft = $1 WHERE id = $2")).
		WithArgs(&draft, id).
		WillReturnError(sql.ErrConnDone)

	err := repo.UpdateReplyDraft(context.Background(), id, &draft)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update reply draft")
	assert.ErrorIs(t, err, sql.ErrConnDone)
}

func TestRepository_UpdateReplyDraft_RowsAffectedError(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	id := uuid.New()
	draft := "Draft text"

	mock.ExpectExec(regexp.QuoteMeta("UPDATE emails SET reply_draft = $1 WHERE id = $2")).
		WithArgs(&draft, id).
		WillReturnResult(sqlmock.NewErrorResult(sql.ErrConnDone))

	err := repo.UpdateReplyDraft(context.Background(), id, &draft)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rows affected")
}

// ============================================================================
// NewRepository
// ============================================================================

func TestNewRepository(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	defer sqlxDB.Close()

	repo := NewRepository(sqlxDB)
	require.NotNil(t, repo)
	assert.Equal(t, sqlxDB, repo.db)
}
