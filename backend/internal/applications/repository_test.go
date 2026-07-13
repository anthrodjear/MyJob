package applications

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func setupRepo(t *testing.T) (sqlmock.Sqlmock, *Repository) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := &Repository{db: sqlxDB}

	t.Cleanup(func() {
		sqlxDB.Close()
		db.Close()
	})

	return mock, repo
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestRepository_GetByID(t *testing.T) {
	t.Parallel()

	appID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	jobID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	resumeID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	tests := []struct {
		name        string
		setupMock   func(mock sqlmock.Sqlmock)
		expectError bool
		expectNil   bool
		expectedErr string
		expectedID  uuid.UUID
	}{
		{
			name: "found",
			setupMock: func(mock sqlmock.Sqlmock) {
				now := time.Now().Truncate(time.Microsecond)
				rows := sqlmock.NewRows([]string{
					"id", "job_id", "resume_id", "cover_letter_id",
					"status", "approval_tier",
					"applied_at", "response_at", "interview_at",
					"notes", "portal_type", "portal_url",
					"form_data", "created_at", "updated_at",
				}).AddRow(
					appID.String(), jobID.String(), resumeID.String(), nil,
					"applied", "auto",
					nil, nil, nil,
					nil, nil, nil,
					nil, now, now,
				)
				mock.ExpectQuery(`(?s)SELECT id, job_id, resume_id, cover_letter_id, status, approval_tier,.*FROM applications WHERE id = \$1`).
					WithArgs(appID.String()).
					WillReturnRows(rows)
			},
			expectError: false,
			expectNil:   false,
			expectedID:  appID,
		},
		{
			name: "not found",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`(?s)SELECT id, job_id, resume_id, cover_letter_id, status, approval_tier,.*FROM applications WHERE id = \$1`).
					WithArgs(appID.String()).
					WillReturnError(sql.ErrNoRows)
			},
			expectError: true,
			expectNil:   true,
			expectedErr: ErrNotFound.Error(),
		},
		{
			name: "db error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`(?s)SELECT id, job_id, resume_id, cover_letter_id, status, approval_tier,.*FROM applications WHERE id = \$1`).
					WithArgs(appID.String()).
					WillReturnError(errors.New("connection refused"))
			},
			expectError: true,
			expectNil:   true,
			expectedErr: "get application by id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, repo := setupRepo(t)
			tt.setupMock(mock)

			app, err := repo.GetByID(context.Background(), appID)

			if tt.expectError {
				assert.Error(t, err)
				if tt.expectedErr != "" {
					assert.Contains(t, err.Error(), tt.expectedErr)
				}
				if tt.expectNil {
					assert.Nil(t, app)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, app)
				assert.Equal(t, tt.expectedID, app.ID)
				assert.Equal(t, jobID, app.JobID)
				assert.Equal(t, "applied", app.Status)
				assert.Equal(t, "auto", app.ApprovalTier)
				require.NotNil(t, app.ResumeID)
				assert.Equal(t, resumeID, *app.ResumeID)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestRepository_List(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Microsecond)
	jobID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	appID1 := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	appID2 := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

	appColumns := []string{
		"id", "job_id", "resume_id", "cover_letter_id",
		"status", "approval_tier",
		"applied_at", "response_at", "interview_at",
		"notes", "portal_type", "portal_url",
		"form_data", "created_at", "updated_at",
	}

	tests := []struct {
		name          string
		filter        ListFilter
		setupMock     func(mock sqlmock.Sqlmock)
		expectError   bool
		expectedTotal int64
		expectedLen   int
	}{
		{
			name:   "with filters",
			filter: ListFilter{Status: "applied", JobID: jobID, Limit: 10, Offset: 5},
			setupMock: func(mock sqlmock.Sqlmock) {
				// Count
				mock.ExpectQuery(`SELECT COUNT\(\*\) FROM applications WHERE status = \$1 AND job_id = \$2`).
					WithArgs("applied", jobID.String()).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
				// Data
				mock.ExpectQuery(`(?s)SELECT id, job_id, resume_id, cover_letter_id, status, approval_tier,.*FROM applications WHERE status = \$1 AND job_id = \$2 ORDER BY created_at DESC LIMIT 10 OFFSET 5`).
					WithArgs("applied", jobID.String()).
					WillReturnRows(sqlmock.NewRows(appColumns).
						AddRow(appID1.String(), jobID.String(), nil, nil, "applied", "auto", nil, nil, nil, nil, nil, nil, nil, now, now).
						AddRow(appID2.String(), jobID.String(), nil, nil, "draft", "review", nil, nil, nil, nil, nil, nil, nil, now, now))
			},
			expectError:   false,
			expectedTotal: 2,
			expectedLen:   2,
		},
		{
			name:   "without filters",
			filter: ListFilter{},
			setupMock: func(mock sqlmock.Sqlmock) {
				// Count
				mock.ExpectQuery(`SELECT COUNT\(\*\) FROM applications`).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
				// Data
				mock.ExpectQuery(`(?s)SELECT id, job_id, resume_id, cover_letter_id, status, approval_tier,.*FROM applications ORDER BY created_at DESC`).
					WillReturnRows(sqlmock.NewRows(appColumns).
						AddRow(appID1.String(), jobID.String(), nil, nil, "applied", "auto", nil, nil, nil, nil, nil, nil, nil, now, now))
			},
			expectError:   false,
			expectedTotal: 1,
			expectedLen:   1,
		},
		{
			name:   "empty results",
			filter: ListFilter{Status: "rejected"},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT COUNT\(\*\) FROM applications WHERE status = \$1`).
					WithArgs("rejected").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
				mock.ExpectQuery(`(?s)SELECT id, job_id, resume_id, cover_letter_id, status, approval_tier,.*FROM applications WHERE status = \$1 ORDER BY created_at DESC`).
					WithArgs("rejected").
					WillReturnRows(sqlmock.NewRows(appColumns))
			},
			expectError:   false,
			expectedTotal: 0,
			expectedLen:   0,
		},
		{
			name:   "db error on count",
			filter: ListFilter{},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT COUNT\(\*\) FROM applications`).
					WillReturnError(errors.New("disk full"))
			},
			expectError:   true,
			expectedTotal: 0,
			expectedLen:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, repo := setupRepo(t)
			tt.setupMock(mock)

			apps, total, err := repo.List(context.Background(), tt.filter)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, apps)
				assert.Equal(t, int64(0), total)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedTotal, total)
				assert.Len(t, apps, tt.expectedLen)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestRepository_Create(t *testing.T) {
	t.Parallel()

	appID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	jobID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

	tests := []struct {
		name        string
		app         *Application
		setupMock   func(mock sqlmock.Sqlmock)
		expectError bool
	}{
		{
			name: "success",
			app: &Application{
				ID:           appID,
				JobID:        jobID,
				Status:       "draft",
				ApprovalTier: "auto",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO applications`).
					WithArgs(
						appID.String(), jobID.String(),
						nil, nil, // resume_id, cover_letter_id
						"draft", "auto",
						nil, nil, nil, // portal_type, portal_url, notes
						sqlmock.AnyArg(), // created_at
						sqlmock.AnyArg(), // updated_at
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectError: false,
		},
		{
			name: "db error",
			app: &Application{
				ID:           appID,
				JobID:        jobID,
				Status:       "draft",
				ApprovalTier: "auto",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO applications`).
					WillReturnError(errors.New("constraint violation"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, repo := setupRepo(t)
			tt.setupMock(mock)

			// Before calling Create, timestamps are zero
			assert.True(t, tt.app.CreatedAt.IsZero())
			assert.True(t, tt.app.UpdatedAt.IsZero())

			err := repo.Create(context.Background(), tt.app)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				// Timestamps should be set
				assert.False(t, tt.app.CreatedAt.IsZero())
				assert.False(t, tt.app.UpdatedAt.IsZero())
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// ---------------------------------------------------------------------------
// UpdateStatus
// ---------------------------------------------------------------------------

func TestRepository_UpdateStatus(t *testing.T) {
	t.Parallel()

	appID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

	tests := []struct {
		name        string
		status      string
		notes       string
		setupMock   func(mock sqlmock.Sqlmock)
		expectError bool
		expectedErr string
	}{
		{
			name:   "success with notes and applied_at",
			status: StatusApplied,
			notes:  "submitted via portal",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				// Get current status
				mock.ExpectQuery(`SELECT status FROM applications WHERE id = \$1`).
					WithArgs(appID.String()).
					WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("draft"))
				// UPDATE status=$1, updated_at=$2, notes=$3, applied_at=$4 WHERE id=$5
				mock.ExpectExec(`UPDATE applications SET status = \$1, updated_at = \$2, notes = \$3, applied_at = \$4 WHERE id = \$5`).
					WithArgs("applied", sqlmock.AnyArg(), "submitted via portal", sqlmock.AnyArg(), appID.String()).
					WillReturnResult(sqlmock.NewResult(0, 1))
				// INSERT into application_events
				mock.ExpectExec(`INSERT INTO application_events`).
					WithArgs(sqlmock.AnyArg(), appID.String(), "draft", "applied", "submitted via portal", sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectError: false,
		},
		{
			name:   "success without notes with applied_at",
			status: StatusApplied,
			notes:  "",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`SELECT status FROM applications WHERE id = \$1`).
					WithArgs(appID.String()).
					WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("queued"))
				// UPDATE status=$1, updated_at=$2, applied_at=$3 WHERE id=$4
				mock.ExpectExec(`UPDATE applications SET status = \$1, updated_at = \$2, applied_at = \$3 WHERE id = \$4`).
					WithArgs("applied", sqlmock.AnyArg(), sqlmock.AnyArg(), appID.String()).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(`INSERT INTO application_events`).
					WithArgs(sqlmock.AnyArg(), appID.String(), "queued", "applied", "", sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectError: false,
		},
		{
			name:   "success with rejected_at",
			status: StatusRejected,
			notes:  "",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`SELECT status FROM applications WHERE id = \$1`).
					WithArgs(appID.String()).
					WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("technical"))
				// UPDATE status=$1, updated_at=$2, response_at=$3 WHERE id=$4
				mock.ExpectExec(`UPDATE applications SET status = \$1, updated_at = \$2, response_at = \$3 WHERE id = \$4`).
					WithArgs("rejected", sqlmock.AnyArg(), sqlmock.AnyArg(), appID.String()).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(`INSERT INTO application_events`).
					WithArgs(sqlmock.AnyArg(), appID.String(), "technical", "rejected", "", sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectError: false,
		},
		{
			name:   "success without timestamp (queued)",
			status: StatusQueued,
			notes:  "",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`SELECT status FROM applications WHERE id = \$1`).
					WithArgs(appID.String()).
					WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("draft"))
				// UPDATE status=$1, updated_at=$2 WHERE id=$3 (no timestamp column for queued)
				mock.ExpectExec(`UPDATE applications SET status = \$1, updated_at = \$2 WHERE id = \$3`).
					WithArgs("queued", sqlmock.AnyArg(), appID.String()).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(`INSERT INTO application_events`).
					WithArgs(sqlmock.AnyArg(), appID.String(), "draft", "queued", "", sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectError: false,
		},
		{
			name:   "success with offer (response_at)",
			status: StatusOffer,
			notes:  "congratulations!",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`SELECT status FROM applications WHERE id = \$1`).
					WithArgs(appID.String()).
					WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("final"))
				// UPDATE status=$1, updated_at=$2, notes=$3, response_at=$4 WHERE id=$5
				mock.ExpectExec(`UPDATE applications SET status = \$1, updated_at = \$2, notes = \$3, response_at = \$4 WHERE id = \$5`).
					WithArgs("offer", sqlmock.AnyArg(), "congratulations!", sqlmock.AnyArg(), appID.String()).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(`INSERT INTO application_events`).
					WithArgs(sqlmock.AnyArg(), appID.String(), "final", "offer", "congratulations!", sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectError: false,
		},
		{
			name:   "not found on select (ErrNoRows)",
			status: StatusApplied,
			notes:  "",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`SELECT status FROM applications WHERE id = \$1`).
					WithArgs(appID.String()).
					WillReturnError(sql.ErrNoRows)
				// Rollback happens via defer
				mock.ExpectRollback()
			},
			expectError: true,
			expectedErr: ErrNotFound.Error(),
		},
		{
			name:   "db error on begin",
			status: StatusApplied,
			notes:  "",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin().WillReturnError(errors.New("connection timeout"))
			},
			expectError: true,
			expectedErr: "begin tx",
		},
		{
			name:   "db error on get status",
			status: StatusApplied,
			notes:  "",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`SELECT status FROM applications WHERE id = \$1`).
					WithArgs(appID.String()).
					WillReturnError(errors.New("permission denied"))
				mock.ExpectRollback()
			},
			expectError: true,
			expectedErr: "get current status",
		},
		{
			name:   "db error on update",
			status: StatusApplied,
			notes:  "notes",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`SELECT status FROM applications WHERE id = \$1`).
					WithArgs(appID.String()).
					WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("draft"))
				mock.ExpectExec(`UPDATE applications SET status = \$1, updated_at = \$2, notes = \$3, applied_at = \$4 WHERE id = \$5`).
					WithArgs("applied", sqlmock.AnyArg(), "notes", sqlmock.AnyArg(), appID.String()).
					WillReturnError(errors.New("deadlock detected"))
				mock.ExpectRollback()
			},
			expectError: true,
			expectedErr: "update application status",
		},
		{
			name:   "db error on insert event",
			status: StatusApplied,
			notes:  "",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`SELECT status FROM applications WHERE id = \$1`).
					WithArgs(appID.String()).
					WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("draft"))
				mock.ExpectExec(`UPDATE applications SET status = \$1, updated_at = \$2, applied_at = \$3 WHERE id = \$4`).
					WithArgs("applied", sqlmock.AnyArg(), sqlmock.AnyArg(), appID.String()).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(`INSERT INTO application_events`).
					WithArgs(sqlmock.AnyArg(), appID.String(), "draft", "applied", "", sqlmock.AnyArg()).
					WillReturnError(errors.New("unique constraint violation"))
				mock.ExpectRollback()
			},
			expectError: true,
			expectedErr: "log application event",
		},
		{
			name:   "db error on commit",
			status: StatusQueued,
			notes:  "",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`SELECT status FROM applications WHERE id = \$1`).
					WithArgs(appID.String()).
					WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("draft"))
				mock.ExpectExec(`UPDATE applications SET status = \$1, updated_at = \$2 WHERE id = \$3`).
					WithArgs("queued", sqlmock.AnyArg(), appID.String()).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(`INSERT INTO application_events`).
					WithArgs(sqlmock.AnyArg(), appID.String(), "draft", "queued", "", sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit().WillReturnError(errors.New("disk full"))
				// Deferred Rollback after failed Commit returns sql.ErrTxDone
				// and does NOT consume an ExpectRollback.
			},
			expectError: true,
			expectedErr: "commit tx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		mock, repo := setupRepo(t)
			tt.setupMock(mock)

			err := repo.UpdateStatus(context.Background(), appID, tt.status, tt.notes)

			if tt.expectError {
				assert.Error(t, err)
				if tt.expectedErr != "" {
					assert.Contains(t, err.Error(), tt.expectedErr)
				}
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// ---------------------------------------------------------------------------
// GetEvents
// ---------------------------------------------------------------------------

func TestRepository_GetEvents(t *testing.T) {
	t.Parallel()

	appID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	eventID1 := uuid.MustParse("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeee1")
	eventID2 := uuid.MustParse("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeee2")
	now := time.Now().Truncate(time.Microsecond)
	later := now.Add(time.Hour)

	tests := []struct {
		name        string
		setupMock   func(mock sqlmock.Sqlmock)
		expectError bool
		expectedLen int
	}{
		{
			name: "found",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "application_id", "old_status", "new_status", "notes", "created_at",
				}).
					AddRow(eventID1.String(), appID.String(), "draft", "applied", "first submit", now).
					AddRow(eventID2.String(), appID.String(), "applied", "rejected", "not a fit", later)
				mock.ExpectQuery(`(?s)SELECT id, application_id, old_status, new_status, notes, created_at FROM application_events WHERE application_id = \$1 ORDER BY created_at ASC`).
					WithArgs(appID.String()).
					WillReturnRows(rows)
			},
			expectError: false,
			expectedLen: 2,
		},
		{
			name: "empty",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`(?s)SELECT id, application_id, old_status, new_status, notes, created_at FROM application_events WHERE application_id = \$1 ORDER BY created_at ASC`).
					WithArgs(appID.String()).
					WillReturnRows(sqlmock.NewRows([]string{
						"id", "application_id", "old_status", "new_status", "notes", "created_at",
					}))
			},
			expectError: false,
			expectedLen: 0,
		},
		{
			name: "db error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`(?s)SELECT id, application_id, old_status, new_status, notes, created_at FROM application_events WHERE application_id = \$1 ORDER BY created_at ASC`).
					WithArgs(appID.String()).
					WillReturnError(errors.New("table not found"))
			},
			expectError: true,
			expectedLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, repo := setupRepo(t)
			tt.setupMock(mock)

			events, err := repo.GetEvents(context.Background(), appID)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, events)
			} else {
				require.NoError(t, err)
				assert.Len(t, events, tt.expectedLen)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// ---------------------------------------------------------------------------
// UpdateNotes
// ---------------------------------------------------------------------------

func TestRepository_UpdateNotes(t *testing.T) {
	t.Parallel()

	appID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

	tests := []struct {
		name        string
		notes       string
		setupMock   func(mock sqlmock.Sqlmock)
		expectError bool
		expectedErr string
	}{
		{
			name:  "success",
			notes: "updated notes",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE applications SET notes = \$1, updated_at = \$2 WHERE id = \$3`).
					WithArgs("updated notes", sqlmock.AnyArg(), appID.String()).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectError: false,
		},
		{
			name:  "not found (zero rows affected)",
			notes: "updated notes",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE applications SET notes = \$1, updated_at = \$2 WHERE id = \$3`).
					WithArgs("updated notes", sqlmock.AnyArg(), appID.String()).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectError: true,
			expectedErr: ErrNotFound.Error(),
		},
		{
			name:  "db error",
			notes: "updated notes",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE applications SET notes = \$1, updated_at = \$2 WHERE id = \$3`).
					WithArgs("updated notes", sqlmock.AnyArg(), appID.String()).
					WillReturnError(errors.New("connection lost"))
			},
			expectError: true,
			expectedErr: "update notes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, repo := setupRepo(t)
			tt.setupMock(mock)

			err := repo.UpdateNotes(context.Background(), appID, tt.notes)

			if tt.expectError {
				assert.Error(t, err)
				if tt.expectedErr != "" {
					assert.Contains(t, err.Error(), tt.expectedErr)
				}
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// ---------------------------------------------------------------------------
// GetStats
// ---------------------------------------------------------------------------

func TestRepository_GetStats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		setupMock      func(mock sqlmock.Sqlmock)
		expectError    bool
		expectedErr    string
		expectedTotal  int64
		expectedStatus map[string]int64
		expectedTier   map[string]int64
	}{
		{
			name: "success",
			setupMock: func(mock sqlmock.Sqlmock) {
				// Total count
				mock.ExpectQuery(`SELECT COUNT\(\*\) FROM applications`).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(8))
				// By status
				statusRows := sqlmock.NewRows([]string{"status", "count"}).
					AddRow("applied", 4).
					AddRow("draft", 2).
					AddRow("rejected", 2)
				mock.ExpectQuery(`SELECT status, COUNT\(\*\) FROM applications GROUP BY status`).
					WillReturnRows(statusRows)
				// By tier
				tierRows := sqlmock.NewRows([]string{"approval_tier", "count"}).
					AddRow("auto", 5).
					AddRow("review", 2).
					AddRow("reject", 1)
				mock.ExpectQuery(`SELECT approval_tier, COUNT\(\*\) FROM applications GROUP BY approval_tier`).
					WillReturnRows(tierRows)
			},
			expectError:    false,
			expectedTotal:  8,
			expectedStatus: map[string]int64{"applied": 4, "draft": 2, "rejected": 2},
			expectedTier:   map[string]int64{"auto": 5, "review": 2, "reject": 1},
		},
		{
			name: "db error on count",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT COUNT\(\*\) FROM applications`).
					WillReturnError(errors.New("disk full"))
			},
			expectError: true,
			expectedErr: "count applications",
		},
		{
			name: "db error on status query",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT COUNT\(\*\) FROM applications`).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))
				mock.ExpectQuery(`SELECT status, COUNT\(\*\) FROM applications GROUP BY status`).
					WillReturnError(errors.New("permission denied"))
			},
			expectError: true,
			expectedErr: "group by status",
		},
		{
			name: "db error on tier query",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT COUNT\(\*\) FROM applications`).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
				mock.ExpectQuery(`SELECT status, COUNT\(\*\) FROM applications GROUP BY status`).
					WillReturnRows(sqlmock.NewRows([]string{"status", "count"}).
						AddRow("draft", 2).
						AddRow("applied", 1))
				mock.ExpectQuery(`SELECT approval_tier, COUNT\(\*\) FROM applications GROUP BY approval_tier`).
					WillReturnError(errors.New("table corrupted"))
			},
			expectError: true,
			expectedErr: "group by tier",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, repo := setupRepo(t)
			tt.setupMock(mock)

			stats, err := repo.GetStats(context.Background())

			if tt.expectError {
				assert.Error(t, err)
				if tt.expectedErr != "" {
					assert.Contains(t, err.Error(), tt.expectedErr)
				}
				assert.Nil(t, stats)
			} else {
				require.NoError(t, err)
				require.NotNil(t, stats)
				assert.Equal(t, tt.expectedTotal, stats.Total)
				assert.Equal(t, tt.expectedStatus, stats.ByStatus)
				assert.Equal(t, tt.expectedTier, stats.ByTier)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
