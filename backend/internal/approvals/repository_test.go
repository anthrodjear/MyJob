package approvals

import (
	"context"
	"database/sql"
	"encoding/json"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newMockDB creates a Repository backed by sqlmock for unit testing.
func newMockDB(t *testing.T) (*Repository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(db, "postgres")
	return NewRepository(sqlxDB), mock
}

// testApprovalRequest builds a fully-populated ApprovalRequest for use in tests.
func testApprovalRequest(id, appID uuid.UUID, status string, now time.Time) *ApprovalRequest {
	resumePath := "resumes/test.pdf"
	coverPreview := "Cover letter preview text"
	return &ApprovalRequest{
		ID:            id,
		ApplicationID: appID,
		JobSnapshot: JobSnapshot{
			Title:        "Software Engineer",
			Company:      "Acme Corp",
			Location:     "Remote",
			URL:          "https://example.com/job/123",
			Description:  "Job description for a software engineer position",
			Requirements: []string{"Go", "React", "PostgreSQL"},
			Score:        85.5,
			Tier:         "review",
			ScoredAt:     now.Format(time.RFC3339),
		},
		ResumePreviewPath:  &resumePath,
		CoverLetterPreview: &coverPreview,
		Status:             status,
		CreatedAt:          now,
	}
}

// mustMarshalJobSnapshot marshals a JobSnapshot to JSON bytes, failing the test on error.
func mustMarshalJobSnapshot(t *testing.T, js JobSnapshot) []byte {
	t.Helper()
	data, err := json.Marshal(js)
	require.NoError(t, err)
	return data
}

// approvalRowColumns is the canonical column list for approval_requests queries.
var approvalRowColumns = []string{
	"id", "application_id", "job_snapshot", "resume_preview_path",
	"cover_letter_preview", "status", "rejection_reason", "created_at", "reviewed_at",
}

// getByIDSQL returns the exact SQL used by GetByID (and internally by UpdateStatus).
func getByIDSQL() string {
	return "SELECT " + approvalRequestColumns + " FROM approval_requests WHERE id = $1"
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestRepository_GetByID(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		repo, mock := newMockDB(t)
		defer repo.db.Close()

		id := uuid.New()
		appID := uuid.New()
		now := time.Now().Truncate(time.Microsecond)
		a := testApprovalRequest(id, appID, ApprovalStatusPending, now)
		jsBytes := mustMarshalJobSnapshot(t, a.JobSnapshot)

		rows := sqlmock.NewRows(approvalRowColumns).
			AddRow(id.String(), appID.String(), jsBytes,
				*a.ResumePreviewPath, *a.CoverLetterPreview,
				a.Status, nil, a.CreatedAt, nil)

		mock.ExpectQuery(regexp.QuoteMeta(getByIDSQL())).
			WithArgs(id).
			WillReturnRows(rows)

		got, err := repo.GetByID(context.Background(), id)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, id, got.ID)
		assert.Equal(t, appID, got.ApplicationID)
		assert.Equal(t, a.JobSnapshot.Title, got.JobSnapshot.Title)
		assert.Equal(t, a.JobSnapshot.Company, got.JobSnapshot.Company)
		assert.Equal(t, a.JobSnapshot.Score, got.JobSnapshot.Score)
		assert.Equal(t, a.JobSnapshot.Tier, got.JobSnapshot.Tier)
		assert.Equal(t, a.Status, got.Status)
		require.NotNil(t, got.ResumePreviewPath)
		assert.Equal(t, *a.ResumePreviewPath, *got.ResumePreviewPath)
		require.NotNil(t, got.CoverLetterPreview)
		assert.Equal(t, *a.CoverLetterPreview, *got.CoverLetterPreview)
		assert.Nil(t, got.RejectionReason)
		assert.Nil(t, got.ReviewedAt)
		assert.Equal(t, a.CreatedAt.Unix(), got.CreatedAt.Unix())

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		repo, mock := newMockDB(t)
		defer repo.db.Close()

		id := uuid.New()

		mock.ExpectQuery(regexp.QuoteMeta(getByIDSQL())).
			WithArgs(id).
			WillReturnError(sql.ErrNoRows)

		got, err := repo.GetByID(context.Background(), id)
		assert.ErrorIs(t, err, ErrNotFound)
		assert.Nil(t, got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		t.Parallel()
		repo, mock := newMockDB(t)
		defer repo.db.Close()

		id := uuid.New()

		mock.ExpectQuery(regexp.QuoteMeta(getByIDSQL())).
			WithArgs(id).
			WillReturnError(sql.ErrConnDone)

		got, err := repo.GetByID(context.Background(), id)
		assert.Error(t, err)
		assert.Nil(t, got)
		assert.Contains(t, err.Error(), "get approval by id")
		assert.ErrorIs(t, err, sql.ErrConnDone)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestRepository_List(t *testing.T) {
	t.Parallel()

	// shared test data
	appID := uuid.New()
	now := time.Now().Truncate(time.Microsecond)

	t.Run("no filter", func(t *testing.T) {
		t.Parallel()
		repo, mock := newMockDB(t)
		defer repo.db.Close()

		id1 := uuid.New()
		id2 := uuid.New()
		a1 := testApprovalRequest(id1, appID, ApprovalStatusPending, now)
		a2 := testApprovalRequest(id2, appID, ApprovalStatusApproved, now.Add(-time.Hour))
		jsBytes1 := mustMarshalJobSnapshot(t, a1.JobSnapshot)
		jsBytes2 := mustMarshalJobSnapshot(t, a2.JobSnapshot)

		// count
		mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM approval_requests")).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

		// select
		rows := sqlmock.NewRows(approvalRowColumns).
			AddRow(id1.String(), appID.String(), jsBytes1,
				*a1.ResumePreviewPath, *a1.CoverLetterPreview,
				a1.Status, nil, a1.CreatedAt, nil).
			AddRow(id2.String(), appID.String(), jsBytes2,
				*a2.ResumePreviewPath, *a2.CoverLetterPreview,
				a2.Status, nil, a2.CreatedAt, nil)

		mock.ExpectQuery(`SELECT\s+id,\s+application_id,\s+job_snapshot,.*FROM approval_requests ORDER BY created_at DESC`).
			WillReturnRows(rows)

		got, total, err := repo.List(context.Background(), ListFilter{})
		require.NoError(t, err)
		assert.Equal(t, int64(2), total)
		assert.Len(t, got, 2)
		assert.Equal(t, id1, got[0].ID)
		assert.Equal(t, id2, got[1].ID)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("with status filter", func(t *testing.T) {
		t.Parallel()
		repo, mock := newMockDB(t)
		defer repo.db.Close()

		id := uuid.New()
		a := testApprovalRequest(id, appID, ApprovalStatusPending, now)
		jsBytes := mustMarshalJobSnapshot(t, a.JobSnapshot)

		// count with WHERE
		mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM approval_requests WHERE status = $1")).
			WithArgs(ApprovalStatusPending).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		// select with WHERE
		rows := sqlmock.NewRows(approvalRowColumns).
			AddRow(id.String(), appID.String(), jsBytes,
				*a.ResumePreviewPath, *a.CoverLetterPreview,
				a.Status, nil, a.CreatedAt, nil)

		mock.ExpectQuery(`SELECT\s+id,\s+application_id,\s+job_snapshot,.*FROM approval_requests WHERE status = \$1 ORDER BY created_at DESC`).
			WithArgs(ApprovalStatusPending).
			WillReturnRows(rows)

		filter := ListFilter{Status: ApprovalStatusPending}
		got, total, err := repo.List(context.Background(), filter)
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, got, 1)
		assert.Equal(t, ApprovalStatusPending, got[0].Status)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("with application_id filter", func(t *testing.T) {
		t.Parallel()
		repo, mock := newMockDB(t)
		defer repo.db.Close()

		id := uuid.New()
		a := testApprovalRequest(id, appID, ApprovalStatusPending, now)
		jsBytes := mustMarshalJobSnapshot(t, a.JobSnapshot)

		mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM approval_requests WHERE application_id = $1")).
			WithArgs(appID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		rows := sqlmock.NewRows(approvalRowColumns).
			AddRow(id.String(), appID.String(), jsBytes,
				*a.ResumePreviewPath, *a.CoverLetterPreview,
				a.Status, nil, a.CreatedAt, nil)

		mock.ExpectQuery(`SELECT\s+id,\s+application_id,\s+job_snapshot,.*FROM approval_requests WHERE application_id = \$1 ORDER BY created_at DESC`).
			WithArgs(appID).
			WillReturnRows(rows)

		filter := ListFilter{ApplicationID: appID}
		got, total, err := repo.List(context.Background(), filter)
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, got, 1)
		assert.Equal(t, appID, got[0].ApplicationID)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("with both filters", func(t *testing.T) {
		t.Parallel()
		repo, mock := newMockDB(t)
		defer repo.db.Close()

		id := uuid.New()
		a := testApprovalRequest(id, appID, ApprovalStatusPending, now)
		jsBytes := mustMarshalJobSnapshot(t, a.JobSnapshot)

		mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM approval_requests WHERE status = $1 AND application_id = $2")).
			WithArgs(ApprovalStatusPending, appID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		rows := sqlmock.NewRows(approvalRowColumns).
			AddRow(id.String(), appID.String(), jsBytes,
				*a.ResumePreviewPath, *a.CoverLetterPreview,
				a.Status, nil, a.CreatedAt, nil)

		mock.ExpectQuery(`SELECT\s+id,\s+application_id,\s+job_snapshot,.*FROM approval_requests WHERE status = \$1 AND application_id = \$2 ORDER BY created_at DESC`).
			WithArgs(ApprovalStatusPending, appID).
			WillReturnRows(rows)

		filter := ListFilter{Status: ApprovalStatusPending, ApplicationID: appID}
		got, total, err := repo.List(context.Background(), filter)
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, got, 1)
		assert.Equal(t, ApprovalStatusPending, got[0].Status)
		assert.Equal(t, appID, got[0].ApplicationID)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("with limit and offset", func(t *testing.T) {
		t.Parallel()
		repo, mock := newMockDB(t)
		defer repo.db.Close()

		id := uuid.New()
		a := testApprovalRequest(id, appID, ApprovalStatusPending, now)
		jsBytes := mustMarshalJobSnapshot(t, a.JobSnapshot)

		// count (no WHERE)
		mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM approval_requests")).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(50))

		// select with LIMIT and OFFSET
		expectedSQL := "SELECT " + approvalRequestColumns + " FROM approval_requests ORDER BY created_at DESC LIMIT 10 OFFSET 20"
		rows := sqlmock.NewRows(approvalRowColumns).
			AddRow(id.String(), appID.String(), jsBytes,
				*a.ResumePreviewPath, *a.CoverLetterPreview,
				a.Status, nil, a.CreatedAt, nil)

		mock.ExpectQuery(regexp.QuoteMeta(expectedSQL)).
			WillReturnRows(rows)

		filter := ListFilter{Limit: 10, Offset: 20}
		got, total, err := repo.List(context.Background(), filter)
		require.NoError(t, err)
		assert.Equal(t, int64(50), total)
		assert.Len(t, got, 1)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("count query error", func(t *testing.T) {
		t.Parallel()
		repo, mock := newMockDB(t)
		defer repo.db.Close()

		mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM approval_requests")).
			WillReturnError(sql.ErrConnDone)

		got, total, err := repo.List(context.Background(), ListFilter{})
		assert.Error(t, err)
		assert.Nil(t, got)
		assert.Equal(t, int64(0), total)
		assert.Contains(t, err.Error(), "count approvals")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("select query error", func(t *testing.T) {
		t.Parallel()
		repo, mock := newMockDB(t)
		defer repo.db.Close()

		// count succeeds
		mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM approval_requests")).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

		// select fails
		mock.ExpectQuery(`SELECT\s+id,\s+application_id,\s+job_snapshot,.*FROM approval_requests ORDER BY created_at DESC`).
			WillReturnError(sql.ErrConnDone)

		got, total, err := repo.List(context.Background(), ListFilter{})
		assert.Error(t, err)
		assert.Nil(t, got)
		assert.Equal(t, int64(0), total)
		assert.Contains(t, err.Error(), "list approvals")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestRepository_Create(t *testing.T) {
	t.Parallel()

	t.Run("default status pending", func(t *testing.T) {
		t.Parallel()
		repo, mock := newMockDB(t)
		defer repo.db.Close()

		id := uuid.New()
		appID := uuid.New()
		a := testApprovalRequest(id, appID, "", time.Now())
		// Clear status so Create defaults to pending
		a.Status = ""

		mock.ExpectExec(regexp.QuoteMeta(
			`INSERT INTO approval_requests (id, application_id, job_snapshot, resume_preview_path,
			                            cover_letter_preview, status, rejection_reason, created_at, reviewed_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`)).
			WithArgs(a.ID, a.ApplicationID, a.JobSnapshot,
				a.ResumePreviewPath, a.CoverLetterPreview,
				ApprovalStatusPending, // defaulted
				a.RejectionReason, sqlmock.AnyArg(), a.ReviewedAt).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Create(context.Background(), a)
		require.NoError(t, err)

		// Verify defaults were applied
		assert.Equal(t, ApprovalStatusPending, a.Status)
		assert.False(t, a.CreatedAt.IsZero())
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("explicit status preserved", func(t *testing.T) {
		t.Parallel()
		repo, mock := newMockDB(t)
		defer repo.db.Close()

		id := uuid.New()
		appID := uuid.New()
		a := testApprovalRequest(id, appID, ApprovalStatusApproved, time.Now())

		mock.ExpectExec(regexp.QuoteMeta(
			`INSERT INTO approval_requests (id, application_id, job_snapshot, resume_preview_path,
			                            cover_letter_preview, status, rejection_reason, created_at, reviewed_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`)).
			WithArgs(a.ID, a.ApplicationID, a.JobSnapshot,
				a.ResumePreviewPath, a.CoverLetterPreview,
				ApprovalStatusApproved,
				a.RejectionReason, sqlmock.AnyArg(), a.ReviewedAt).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Create(context.Background(), a)
		require.NoError(t, err)
		assert.Equal(t, ApprovalStatusApproved, a.Status)
		assert.False(t, a.CreatedAt.IsZero())
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		t.Parallel()
		repo, mock := newMockDB(t)
		defer repo.db.Close()

		id := uuid.New()
		appID := uuid.New()
		a := testApprovalRequest(id, appID, ApprovalStatusPending, time.Now())

		mock.ExpectExec(regexp.QuoteMeta(
			`INSERT INTO approval_requests (id, application_id, job_snapshot, resume_preview_path,
			                            cover_letter_preview, status, rejection_reason, created_at, reviewed_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`)).
			WillReturnError(sql.ErrConnDone)

		err := repo.Create(context.Background(), a)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "create approval")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// ---------------------------------------------------------------------------
// UpdateStatus
// ---------------------------------------------------------------------------

func TestRepository_UpdateStatus(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Microsecond)

	t.Run("pending to approved", func(t *testing.T) {
		t.Parallel()
		repo, mock := newMockDB(t)
		defer repo.db.Close()

		id := uuid.New()
		appID := uuid.New()
		a := testApprovalRequest(id, appID, ApprovalStatusPending, now)
		jsBytes := mustMarshalJobSnapshot(t, a.JobSnapshot)

		// GetByID returns pending approval
		rows := sqlmock.NewRows(approvalRowColumns).
			AddRow(id.String(), appID.String(), jsBytes,
				*a.ResumePreviewPath, *a.CoverLetterPreview,
				ApprovalStatusPending, nil, a.CreatedAt, nil)
		mock.ExpectQuery(regexp.QuoteMeta(getByIDSQL())).
			WithArgs(id).
			WillReturnRows(rows)

		// Update succeeds
		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE approval_requests
			 SET status = $1, rejection_reason = $2, reviewed_at = $3
			 WHERE id = $4`)).
			WithArgs(ApprovalStatusApproved, (*string)(nil), sqlmock.AnyArg(), id).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.UpdateStatus(context.Background(), id, ApprovalStatusApproved, nil)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("pending to rejected with reason", func(t *testing.T) {
		t.Parallel()
		repo, mock := newMockDB(t)
		defer repo.db.Close()

		id := uuid.New()
		appID := uuid.New()
		reason := "Salary expectations too high"
		a := testApprovalRequest(id, appID, ApprovalStatusPending, now)
		jsBytes := mustMarshalJobSnapshot(t, a.JobSnapshot)

		// GetByID returns pending approval
		rows := sqlmock.NewRows(approvalRowColumns).
			AddRow(id.String(), appID.String(), jsBytes,
				*a.ResumePreviewPath, *a.CoverLetterPreview,
				ApprovalStatusPending, nil, a.CreatedAt, nil)
		mock.ExpectQuery(regexp.QuoteMeta(getByIDSQL())).
			WithArgs(id).
			WillReturnRows(rows)

		// Update with rejection reason
		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE approval_requests
			 SET status = $1, rejection_reason = $2, reviewed_at = $3
			 WHERE id = $4`)).
			WithArgs(ApprovalStatusRejected, &reason, sqlmock.AnyArg(), id).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.UpdateStatus(context.Background(), id, ApprovalStatusRejected, &reason)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found on get", func(t *testing.T) {
		t.Parallel()
		repo, mock := newMockDB(t)
		defer repo.db.Close()

		id := uuid.New()

		// GetByID returns not found
		mock.ExpectQuery(regexp.QuoteMeta(getByIDSQL())).
			WithArgs(id).
			WillReturnError(sql.ErrNoRows)

		err := repo.UpdateStatus(context.Background(), id, ApprovalStatusApproved, nil)
		assert.ErrorIs(t, err, ErrNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invalid transition", func(t *testing.T) {
		t.Parallel()
		repo, mock := newMockDB(t)
		defer repo.db.Close()

		id := uuid.New()
		appID := uuid.New()
		// Already approved — terminal state
		a := testApprovalRequest(id, appID, ApprovalStatusApproved, now)
		jsBytes := mustMarshalJobSnapshot(t, a.JobSnapshot)

		// GetByID returns approved (terminal)
		rows := sqlmock.NewRows(approvalRowColumns).
			AddRow(id.String(), appID.String(), jsBytes,
				*a.ResumePreviewPath, *a.CoverLetterPreview,
				ApprovalStatusApproved, nil, a.CreatedAt, nil)
		mock.ExpectQuery(regexp.QuoteMeta(getByIDSQL())).
			WithArgs(id).
			WillReturnRows(rows)

		// No ExecContext should be called — rejected by CanTransition
		err := repo.UpdateStatus(context.Background(), id, ApprovalStatusPending, nil)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidStatus)
		assert.Contains(t, err.Error(), "approved → pending")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("exec error", func(t *testing.T) {
		t.Parallel()
		repo, mock := newMockDB(t)
		defer repo.db.Close()

		id := uuid.New()
		appID := uuid.New()
		a := testApprovalRequest(id, appID, ApprovalStatusPending, now)
		jsBytes := mustMarshalJobSnapshot(t, a.JobSnapshot)

		// GetByID returns pending
		rows := sqlmock.NewRows(approvalRowColumns).
			AddRow(id.String(), appID.String(), jsBytes,
				*a.ResumePreviewPath, *a.CoverLetterPreview,
				ApprovalStatusPending, nil, a.CreatedAt, nil)
		mock.ExpectQuery(regexp.QuoteMeta(getByIDSQL())).
			WithArgs(id).
			WillReturnRows(rows)

		// ExecContext fails
		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE approval_requests
			 SET status = $1, rejection_reason = $2, reviewed_at = $3
			 WHERE id = $4`)).
			WithArgs(ApprovalStatusApproved, (*string)(nil), sqlmock.AnyArg(), id).
			WillReturnError(sql.ErrConnDone)

		err := repo.UpdateStatus(context.Background(), id, ApprovalStatusApproved, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "update approval status")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("rows affected zero", func(t *testing.T) {
		t.Parallel()
		repo, mock := newMockDB(t)
		defer repo.db.Close()

		id := uuid.New()
		appID := uuid.New()
		a := testApprovalRequest(id, appID, ApprovalStatusPending, now)
		jsBytes := mustMarshalJobSnapshot(t, a.JobSnapshot)

		// GetByID returns pending
		rows := sqlmock.NewRows(approvalRowColumns).
			AddRow(id.String(), appID.String(), jsBytes,
				*a.ResumePreviewPath, *a.CoverLetterPreview,
				ApprovalStatusPending, nil, a.CreatedAt, nil)
		mock.ExpectQuery(regexp.QuoteMeta(getByIDSQL())).
			WithArgs(id).
			WillReturnRows(rows)

		// Update matches 0 rows
		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE approval_requests
			 SET status = $1, rejection_reason = $2, reviewed_at = $3
			 WHERE id = $4`)).
			WithArgs(ApprovalStatusApproved, (*string)(nil), sqlmock.AnyArg(), id).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.UpdateStatus(context.Background(), id, ApprovalStatusApproved, nil)
		assert.ErrorIs(t, err, ErrNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// ---------------------------------------------------------------------------
// ListFilter.buildWhere
// ---------------------------------------------------------------------------

func TestListFilter_buildWhere(t *testing.T) {
	t.Parallel()

	appID := uuid.New()

	t.Run("empty filter", func(t *testing.T) {
		t.Parallel()
		f := ListFilter{}
		where, args := f.buildWhere()
		assert.Empty(t, where)
		assert.Empty(t, args)
	})

	t.Run("status filter", func(t *testing.T) {
		t.Parallel()
		f := ListFilter{Status: ApprovalStatusPending}
		where, args := f.buildWhere()
		assert.Equal(t, " WHERE status = $1", where)
		require.Len(t, args, 1)
		assert.Equal(t, ApprovalStatusPending, args[0])
	})

	t.Run("application_id filter", func(t *testing.T) {
		t.Parallel()
		f := ListFilter{ApplicationID: appID}
		where, args := f.buildWhere()
		assert.Equal(t, " WHERE application_id = $1", where)
		require.Len(t, args, 1)
		assert.Equal(t, appID, args[0])
	})

	t.Run("both filters", func(t *testing.T) {
		t.Parallel()
		f := ListFilter{Status: ApprovalStatusRejected, ApplicationID: appID}
		where, args := f.buildWhere()
		assert.Equal(t, " WHERE status = $1 AND application_id = $2", where)
		require.Len(t, args, 2)
		assert.Equal(t, ApprovalStatusRejected, args[0])
		assert.Equal(t, appID, args[1])
	})

	t.Run("zero value uuid ignored", func(t *testing.T) {
		t.Parallel()
		f := ListFilter{Status: ApprovalStatusPending, ApplicationID: uuid.Nil}
		where, args := f.buildWhere()
		assert.Equal(t, " WHERE status = $1", where)
		require.Len(t, args, 1)
	})
}
