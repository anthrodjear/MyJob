package interviews

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// The Transcript type (defined in model.go) provides sql.Scanner / driver.Valuer
// for JSONB round-trips.  No additional wrapper types are needed in tests.
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// newMockDB creates a sqlx.DB backed by go-sqlmock.
func newMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	sqlxDB := sqlx.NewDb(db, "postgres")
	t.Cleanup(func() { _ = sqlxDB.Close() })

	return sqlxDB, mock
}

// newRepo creates a Repository backed by a sqlmock DB.
func newRepo(t *testing.T) (*Repository, *sqlx.DB, sqlmock.Sqlmock) {
	t.Helper()

	sqlxDB, mock := newMockDB(t)
	repo := NewRepository(sqlxDB)
	return repo, sqlxDB, mock
}

// nowTruncated returns time.Now() truncated to microsecond precision.
func nowTruncated() time.Time {
	return time.Now().UTC().Truncate(time.Microsecond)
}

// newTestSession returns a fully populated InterviewSession for use in tests.
func newTestSession() *InterviewSession {
	now := nowTruncated()
	extID := "RMtest123"
	score := 87.5
	feedback := json.RawMessage(`{"categories":{"communication":90},"summary":"Good"}`)
	startedAt := now.Add(-30 * time.Minute)
	endedAt := now

	return &InterviewSession{
		ID:                uuid.New(),
		ApplicationID:     uuid.New(),
		Mode:              ModeAssist,
		Status:            StatusPending,
		ExternalSessionID: &extID,
		Provider:          "openai_realtime",
		Model:             "gpt-4o-realtime-preview",
		Transcript: Transcript{
			{ID: uuid.New(), Speaker: SpeakerAI, Content: "Hello", Timestamp: now},
		},
		Score:     &score,
		Feedback:  feedback,
		StartedAt: &startedAt,
		EndedAt:   &endedAt,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// sessionRowValues returns the driver.Value slice for a full row of
// interview_sessions columns, using the optional field pointers from session.
// JSONB columns (transcript, feedback) are serialised so that the Scanner
// types in InterviewSessionJSONB can deserialise them.
func sessionRowValues(session *InterviewSession) []driver.Value {
	transcriptJSON, _ := json.Marshal(session.Transcript)

	var feedbackJSON []byte
	if session.Feedback != nil {
		feedbackJSON = []byte(session.Feedback)
	}

	var scoreVal interface{}
	if session.Score != nil {
		scoreVal = *session.Score
	}

	var startedAtVal interface{}
	if session.StartedAt != nil {
		startedAtVal = *session.StartedAt
	}

	var endedAtVal interface{}
	if session.EndedAt != nil {
		endedAtVal = *session.EndedAt
	}

	return []driver.Value{
		session.ID,
		session.ApplicationID,
		session.Mode,
		session.Status,
		session.ExternalSessionID,
		session.Provider,
		session.Model,
		transcriptJSON,
		scoreVal,
		feedbackJSON,
		startedAtVal,
		endedAtVal,
		session.CreatedAt,
		session.UpdatedAt,
	}
}

// newMockRows returns an sqlmock.Rows with the interview_sessions column schema.
func newMockRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "application_id", "mode", "status", "external_session_id",
		"provider", "model", "transcript", "score", "feedback",
		"started_at", "ended_at", "created_at", "updated_at",
	})
}

// ---------------------------------------------------------------------------
// Expected SQL query helpers
//
// These reproduce the exact query strings that the repository methods build,
// so that sqlmock expectations match what the code sends.
// ---------------------------------------------------------------------------

func selectCols() string {
	return `SELECT ` + interviewSessionColumns + `
		 FROM interview_sessions`
}

func getByIDQuery() string {
	return selectCols() + ` WHERE id = $1`
}

func getByExternalIDQuery() string {
	return selectCols() + ` WHERE external_session_id = $1`
}

func listCountQuery() string {
	return `SELECT COUNT(*) FROM interview_sessions`
}

func listSelectQueryWithLimit(limit int) string {
	return selectCols() + ` ORDER BY created_at DESC LIMIT ` + itoa(limit)
}

func itoa(i int) string {
	return strconv.Itoa(i)
}

// ---------------------------------------------------------------------------
// Repository — GetByID
// ---------------------------------------------------------------------------

func TestRepository_GetByID_Success(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()
	session := newTestSession()

	rows := newMockRows().AddRow(sessionRowValues(session)...)

	mock.ExpectQuery(regexp.QuoteMeta(getByIDQuery())).
		WithArgs(session.ID).
		WillReturnRows(rows)

	got, err := repo.GetByID(ctx, session.ID)
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, session.ID, got.ID)
	assert.Equal(t, session.ApplicationID, got.ApplicationID)
	assert.Equal(t, session.Mode, got.Mode)
	assert.Equal(t, session.Status, got.Status)
	require.NotNil(t, got.ExternalSessionID)
	assert.Equal(t, *session.ExternalSessionID, *got.ExternalSessionID)
	assert.Equal(t, session.Provider, got.Provider)
	assert.Equal(t, session.Model, got.Model)
	require.Len(t, got.Transcript, 1)
	assert.Equal(t, session.Transcript[0].Speaker, got.Transcript[0].Speaker)
	assert.Equal(t, session.Transcript[0].Content, got.Transcript[0].Content)
	require.NotNil(t, got.Score)
	assert.Equal(t, *session.Score, *got.Score)
	require.NotNil(t, got.Feedback)
	assert.Equal(t, string(session.Feedback), string(got.Feedback))
	require.NotNil(t, got.StartedAt)
	assert.True(t, session.StartedAt.Equal(*got.StartedAt))
	require.NotNil(t, got.EndedAt)
	assert.True(t, session.EndedAt.Equal(*got.EndedAt))

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetByID_NotFound(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()
	id := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(getByIDQuery())).
		WithArgs(id).
		WillReturnError(sql.ErrNoRows)

	got, err := repo.GetByID(ctx, id)
	assert.ErrorIs(t, err, ErrNotFound)
	assert.Nil(t, got)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetByID_DBError(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()
	id := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(getByIDQuery())).
		WithArgs(id).
		WillReturnError(sql.ErrConnDone)

	got, err := repo.GetByID(ctx, id)
	assert.Error(t, err)
	assert.Nil(t, got)
	assert.NotContains(t, err.Error(), "not found") // must be wrapped, not ErrNotFound
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// Repository — GetByExternalSessionID
// ---------------------------------------------------------------------------

func TestRepository_GetByExternalSessionID_Success(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()
	session := newTestSession()

	externalID := *session.ExternalSessionID
	rows := newMockRows().AddRow(sessionRowValues(session)...)

	mock.ExpectQuery(regexp.QuoteMeta(getByExternalIDQuery())).
		WithArgs(externalID).
		WillReturnRows(rows)

	got, err := repo.GetByExternalSessionID(ctx, externalID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, session.ID, got.ID)
	assert.Equal(t, externalID, *got.ExternalSessionID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetByExternalSessionID_NotFound(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()
	externalID := "RMnonexistent"

	mock.ExpectQuery(regexp.QuoteMeta(getByExternalIDQuery())).
		WithArgs(externalID).
		WillReturnError(sql.ErrNoRows)

	got, err := repo.GetByExternalSessionID(ctx, externalID)
	assert.ErrorIs(t, err, ErrNotFound)
	assert.Nil(t, got)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetByExternalSessionID_DBError(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()

	mock.ExpectQuery(regexp.QuoteMeta(getByExternalIDQuery())).
		WithArgs("sess_err").
		WillReturnError(sql.ErrConnDone)

	got, err := repo.GetByExternalSessionID(ctx, "sess_err")
	assert.Error(t, err)
	assert.Nil(t, got)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// Repository — List
// ---------------------------------------------------------------------------

func TestRepository_List_DefaultPagination(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()

	filter := ListFilter{}

	mock.ExpectQuery(regexp.QuoteMeta(listCountQuery())).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(2)))

	s1 := newTestSession()
	s2 := newTestSession()
	rows := newMockRows()
	rows.AddRow(sessionRowValues(s1)...)
	rows.AddRow(sessionRowValues(s2)...)

	mock.ExpectQuery(regexp.QuoteMeta(listSelectQueryWithLimit(20))).
		WillReturnRows(rows)

	sessions, total, err := repo.List(ctx, filter)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, sessions, 2)
	assert.Equal(t, s1.ID, sessions[0].ID)
	assert.Equal(t, s2.ID, sessions[1].ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_List_WithFilters(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()

	appID := uuid.New()
	filter := ListFilter{
		ApplicationID: appID,
		Status:        StatusActive,
		Mode:          ModeAssist,
		Limit:         10,
		Offset:        5,
	}

	whereClause := " WHERE application_id = $1 AND status = $2 AND mode = $3"
	countQuery := listCountQuery() + whereClause
	selectQuery := selectCols() + whereClause + " ORDER BY created_at DESC LIMIT 10 OFFSET 5"

	mock.ExpectQuery(regexp.QuoteMeta(countQuery)).
		WithArgs(appID, StatusActive, ModeAssist).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))

	session := newTestSession()
	mock.ExpectQuery(regexp.QuoteMeta(selectQuery)).
		WithArgs(appID, StatusActive, ModeAssist).
		WillReturnRows(newMockRows().AddRow(sessionRowValues(session)...))

	sessions, total, err := repo.List(ctx, filter)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, sessions, 1)
	assert.Equal(t, session.ID, sessions[0].ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_List_CountError(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()

	mock.ExpectQuery(regexp.QuoteMeta(listCountQuery())).
		WillReturnError(sql.ErrConnDone)

	sessions, total, err := repo.List(ctx, ListFilter{})
	assert.Error(t, err)
	assert.Nil(t, sessions)
	assert.Zero(t, total)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_List_SelectError(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()

	mock.ExpectQuery(regexp.QuoteMeta(listCountQuery())).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))

	mock.ExpectQuery(regexp.QuoteMeta(listSelectQueryWithLimit(20))).
		WillReturnError(sql.ErrConnDone)

	sessions, total, err := repo.List(ctx, ListFilter{})
	assert.Error(t, err)
	assert.Nil(t, sessions)
	assert.Zero(t, total)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_List_LimitCapping(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()

	filter := ListFilter{Limit: 200}

	mock.ExpectQuery(regexp.QuoteMeta(listCountQuery())).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))

	mock.ExpectQuery(regexp.QuoteMeta(selectCols() + ` ORDER BY created_at DESC LIMIT 100`)).
		WillReturnRows(newMockRows())

	sessions, total, err := repo.List(ctx, filter)
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, sessions)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_List_ZeroLimitDefaultsTo20(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()

	filter := ListFilter{Limit: 0}

	mock.ExpectQuery(regexp.QuoteMeta(listCountQuery())).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))

	mock.ExpectQuery(regexp.QuoteMeta(selectCols() + ` ORDER BY created_at DESC LIMIT 20`)).
		WillReturnRows(newMockRows())

	sessions, total, err := repo.List(ctx, filter)
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, sessions)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_List_NegativeOffset(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()

	filter := ListFilter{Limit: 10, Offset: -5}

	mock.ExpectQuery(regexp.QuoteMeta(listCountQuery())).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))

	mock.ExpectQuery(regexp.QuoteMeta(selectCols() + ` ORDER BY created_at DESC LIMIT 10`)).
		WillReturnRows(newMockRows())

	sessions, total, err := repo.List(ctx, filter)
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, sessions)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_List_NegativeLimit(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()

	filter := ListFilter{Limit: -1}

	mock.ExpectQuery(regexp.QuoteMeta(listCountQuery())).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))

	mock.ExpectQuery(regexp.QuoteMeta(selectCols() + ` ORDER BY created_at DESC LIMIT 20`)).
		WillReturnRows(newMockRows())

	sessions, total, err := repo.List(ctx, filter)
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, sessions)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_List_EmptyResult(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()

	mock.ExpectQuery(regexp.QuoteMeta(listCountQuery())).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))

	mock.ExpectQuery(regexp.QuoteMeta(selectCols() + ` ORDER BY created_at DESC LIMIT 20`)).
		WillReturnRows(newMockRows())

	sessions, total, err := repo.List(ctx, ListFilter{})
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, sessions)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// Repository — Create
// ---------------------------------------------------------------------------

func TestRepository_Create_Success(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()

	session := &InterviewSession{
		ID:                uuid.New(),
		ApplicationID:     uuid.New(),
		Mode:              ModeAutonomous,
		Status:            StatusPending,
		ExternalSessionID: nil,
		Provider:          "",
		Model:             "",
		Transcript:        Transcript{},
		Score:             nil,
		Feedback:          nil,
		StartedAt:         nil,
		EndedAt:           nil,
	}

	tJSON, err := json.Marshal(session.Transcript)
	require.NoError(t, err)

	mock.ExpectExec(regexp.QuoteMeta(
		`INSERT INTO interview_sessions
		    (id, application_id, mode, status, external_session_id,
		     provider, model, transcript, score, feedback,
		     started_at, ended_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`)).
		WithArgs(
			session.ID, session.ApplicationID, session.Mode, session.Status,
			session.ExternalSessionID, session.Provider, session.Model,
			tJSON, session.Score, session.Feedback,
			session.StartedAt, session.EndedAt,
			sqlmock.AnyArg(), // created_at
			sqlmock.AnyArg(), // updated_at
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Create(ctx, session)
	require.NoError(t, err)

	assert.False(t, session.CreatedAt.IsZero())
	assert.False(t, session.UpdatedAt.IsZero())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_Create_EmptyTranscript(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()

	session := &InterviewSession{
		ID:            uuid.New(),
		ApplicationID: uuid.New(),
		Mode:          ModeAssist,
		Status:        StatusPending,
		Transcript:    Transcript{},
	}

	tJSON, err := json.Marshal(session.Transcript)
	require.NoError(t, err)

	mock.ExpectExec(regexp.QuoteMeta(
		`INSERT INTO interview_sessions
		    (id, application_id, mode, status, external_session_id,
		     provider, model, transcript, score, feedback,
		     started_at, ended_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`)).
		WithArgs(
			session.ID, session.ApplicationID, session.Mode, session.Status,
			session.ExternalSessionID, session.Provider, session.Model,
			tJSON, session.Score, session.Feedback,
			session.StartedAt, session.EndedAt,
			sqlmock.AnyArg(), sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Create(ctx, session)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_Create_DBError(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()

	session := &InterviewSession{
		ID:            uuid.New(),
		ApplicationID: uuid.New(),
		Mode:          ModeAssist,
		Status:        StatusPending,
		Transcript:    Transcript{},
	}

	mock.ExpectExec(regexp.QuoteMeta(
		`INSERT INTO interview_sessions
		    (id, application_id, mode, status, external_session_id,
		     provider, model, transcript, score, feedback,
		     started_at, ended_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`)).
		WithArgs(
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(),
		).
		WillReturnError(sql.ErrConnDone)

	err := repo.Create(ctx, session)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// Repository — UpdateStatus
// ---------------------------------------------------------------------------

func TestRepository_UpdateStatus_PendingToStarting(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()
	id := uuid.New()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT status FROM interview_sessions WHERE id = $1`)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(StatusPending))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE interview_sessions SET status = $1, updated_at = $2 WHERE id = $3`)).
		WithArgs(StatusStarting, sqlmock.AnyArg(), id).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.UpdateStatus(ctx, id, StatusStarting)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateStatus_StartingToActive(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()
	id := uuid.New()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT status FROM interview_sessions WHERE id = $1`)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(StatusStarting))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE interview_sessions SET status = $1, updated_at = $2, started_at = $3 WHERE id = $4`)).
		WithArgs(StatusActive, sqlmock.AnyArg(), sqlmock.AnyArg(), id).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.UpdateStatus(ctx, id, StatusActive)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateStatus_ActiveToCompleted(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()
	id := uuid.New()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT status FROM interview_sessions WHERE id = $1`)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(StatusActive))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE interview_sessions SET status = $1, updated_at = $2, ended_at = $3 WHERE id = $4`)).
		WithArgs(StatusCompleted, sqlmock.AnyArg(), sqlmock.AnyArg(), id).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.UpdateStatus(ctx, id, StatusCompleted)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateStatus_ActiveToFailed(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()
	id := uuid.New()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT status FROM interview_sessions WHERE id = $1`)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(StatusActive))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE interview_sessions SET status = $1, updated_at = $2, ended_at = $3 WHERE id = $4`)).
		WithArgs(StatusFailed, sqlmock.AnyArg(), sqlmock.AnyArg(), id).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.UpdateStatus(ctx, id, StatusFailed)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateStatus_ActiveToCancelled(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()
	id := uuid.New()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT status FROM interview_sessions WHERE id = $1`)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(StatusActive))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE interview_sessions SET status = $1, updated_at = $2, ended_at = $3 WHERE id = $4`)).
		WithArgs(StatusCancelled, sqlmock.AnyArg(), sqlmock.AnyArg(), id).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.UpdateStatus(ctx, id, StatusCancelled)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateStatus_NotFound(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()
	id := uuid.New()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT status FROM interview_sessions WHERE id = $1`)).
		WithArgs(id).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	err := repo.UpdateStatus(ctx, id, StatusStarting)
	assert.ErrorIs(t, err, ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateStatus_InvalidTransition(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()
	id := uuid.New()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT status FROM interview_sessions WHERE id = $1`)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(StatusCompleted))
	mock.ExpectRollback()

	err := repo.UpdateStatus(ctx, id, StatusActive)
	assert.ErrorIs(t, err, ErrInvalidStatus)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateStatus_BeginTxError(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()
	id := uuid.New()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT status FROM interview_sessions WHERE id = $1`)).
		WithArgs(id).
		WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()

	err := repo.UpdateStatus(ctx, id, StatusStarting)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateStatus_UpdateReturnsZeroRows(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()
	id := uuid.New()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT status FROM interview_sessions WHERE id = $1`)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(StatusPending))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE interview_sessions SET status = $1, updated_at = $2 WHERE id = $3`)).
		WithArgs(StatusStarting, sqlmock.AnyArg(), id).
		WillReturnResult(sqlmock.NewResult(1, 0))
	mock.ExpectRollback()

	err := repo.UpdateStatus(ctx, id, StatusStarting)
	assert.ErrorIs(t, err, ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// Repository — UpdateExternalSessionID
// ---------------------------------------------------------------------------

func TestRepository_UpdateExternalSessionID_Success(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()
	id := uuid.New()
	externalID := "RMnewRoom"

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE interview_sessions
		 SET external_session_id = $1, updated_at = $2
		 WHERE id = $3`)).
		WithArgs(externalID, sqlmock.AnyArg(), id).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.UpdateExternalSessionID(ctx, id, externalID)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateExternalSessionID_NotFound(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()
	id := uuid.New()

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE interview_sessions
		 SET external_session_id = $1, updated_at = $2
		 WHERE id = $3`)).
		WithArgs("RMgone", sqlmock.AnyArg(), id).
		WillReturnResult(sqlmock.NewResult(1, 0))

	err := repo.UpdateExternalSessionID(ctx, id, "RMgone")
	assert.ErrorIs(t, err, ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateExternalSessionID_DBError(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()
	id := uuid.New()

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE interview_sessions
		 SET external_session_id = $1, updated_at = $2
		 WHERE id = $3`)).
		WithArgs("RMerr", sqlmock.AnyArg(), id).
		WillReturnError(sql.ErrConnDone)

	err := repo.UpdateExternalSessionID(ctx, id, "RMerr")
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// Repository — UpdateProvider
// ---------------------------------------------------------------------------

func TestRepository_UpdateProvider_Success(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()
	id := uuid.New()

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE interview_sessions
		 SET provider = $1, model = $2, updated_at = $3
		 WHERE id = $4`)).
		WithArgs("elevenlabs", "eleven_turbo_v2", sqlmock.AnyArg(), id).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.UpdateProvider(ctx, id, "elevenlabs", "eleven_turbo_v2")
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateProvider_NotFound(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()
	id := uuid.New()

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE interview_sessions
		 SET provider = $1, model = $2, updated_at = $3
		 WHERE id = $4`)).
		WithArgs("p", "m", sqlmock.AnyArg(), id).
		WillReturnResult(sqlmock.NewResult(1, 0))

	err := repo.UpdateProvider(ctx, id, "p", "m")
	assert.ErrorIs(t, err, ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// Repository — AppendTranscript
// ---------------------------------------------------------------------------

func TestRepository_AppendTranscript_Success(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()
	id := uuid.New()

	entry := TranscriptEntry{
		ID:        uuid.New(),
		Speaker:   SpeakerCandidate,
		Content:   "I have 5 years of Go experience",
		Timestamp: nowTruncated(),
	}

	entryJSON, err := json.Marshal(entry)
	require.NoError(t, err)

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE interview_sessions
		 SET transcript = COALESCE(transcript, '[]'::jsonb) || $1::jsonb,
		     updated_at = $2
		 WHERE id = $3`)).
		WithArgs(string(entryJSON), sqlmock.AnyArg(), id).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.AppendTranscript(ctx, id, entry)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_AppendTranscript_NotFound(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()
	id := uuid.New()

	entry := TranscriptEntry{
		ID:        uuid.New(),
		Speaker:   SpeakerAI,
		Content:   "Tell me about yourself",
		Timestamp: nowTruncated(),
	}

	entryJSON, _ := json.Marshal(entry)

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE interview_sessions
		 SET transcript = COALESCE(transcript, '[]'::jsonb) || $1::jsonb,
		     updated_at = $2
		 WHERE id = $3`)).
		WithArgs(string(entryJSON), sqlmock.AnyArg(), id).
		WillReturnResult(sqlmock.NewResult(1, 0))

	err := repo.AppendTranscript(ctx, id, entry)
	assert.ErrorIs(t, err, ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_AppendTranscript_DBError(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()
	id := uuid.New()

	entry := TranscriptEntry{
		ID:        uuid.New(),
		Speaker:   SpeakerSystem,
		Content:   "Interview started",
		Timestamp: nowTruncated(),
	}

	entryJSON, _ := json.Marshal(entry)

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE interview_sessions
		 SET transcript = COALESCE(transcript, '[]'::jsonb) || $1::jsonb,
		     updated_at = $2
		 WHERE id = $3`)).
		WithArgs(string(entryJSON), sqlmock.AnyArg(), id).
		WillReturnError(sql.ErrConnDone)

	err := repo.AppendTranscript(ctx, id, entry)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// Repository — UpdateScore
// ---------------------------------------------------------------------------

func TestRepository_UpdateScore_Success(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()
	id := uuid.New()
	score := 92.5

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE interview_sessions
		 SET score = $1, updated_at = $2
		 WHERE id = $3`)).
		WithArgs(score, sqlmock.AnyArg(), id).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.UpdateScore(ctx, id, score)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateScore_NotFound(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()
	id := uuid.New()

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE interview_sessions
		 SET score = $1, updated_at = $2
		 WHERE id = $3`)).
		WithArgs(95.0, sqlmock.AnyArg(), id).
		WillReturnResult(sqlmock.NewResult(1, 0))

	err := repo.UpdateScore(ctx, id, 95.0)
	assert.ErrorIs(t, err, ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateScore_DBError(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()
	id := uuid.New()

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE interview_sessions
		 SET score = $1, updated_at = $2
		 WHERE id = $3`)).
		WithArgs(50.0, sqlmock.AnyArg(), id).
		WillReturnError(sql.ErrConnDone)

	err := repo.UpdateScore(ctx, id, 50.0)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// Repository — UpdateFeedback
// ---------------------------------------------------------------------------

func TestRepository_UpdateFeedback_Success(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()
	id := uuid.New()
	feedback := json.RawMessage(`{"strengths":["communication"],"score":85}`)

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE interview_sessions
		 SET feedback = $1, updated_at = $2
		 WHERE id = $3`)).
		WithArgs(feedback, sqlmock.AnyArg(), id).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.UpdateFeedback(ctx, id, feedback)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateFeedback_NotFound(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()
	id := uuid.New()

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE interview_sessions
		 SET feedback = $1, updated_at = $2
		 WHERE id = $3`)).
		WithArgs(json.RawMessage(`[]`), sqlmock.AnyArg(), id).
		WillReturnResult(sqlmock.NewResult(1, 0))

	err := repo.UpdateFeedback(ctx, id, json.RawMessage(`[]`))
	assert.ErrorIs(t, err, ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateFeedback_DBError(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()
	id := uuid.New()

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE interview_sessions
		 SET feedback = $1, updated_at = $2
		 WHERE id = $3`)).
		WithArgs(json.RawMessage(`null`), sqlmock.AnyArg(), id).
		WillReturnError(sql.ErrConnDone)

	err := repo.UpdateFeedback(ctx, id, json.RawMessage(`null`))
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// Repository — StartSession
// ---------------------------------------------------------------------------

func TestRepository_StartSession_Success(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()
	id := uuid.New()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT status FROM interview_sessions WHERE id = $1`)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(StatusPending))
	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE interview_sessions
		 SET status = $1, external_session_id = $2, provider = $3, model = $4, updated_at = $5
		 WHERE id = $6`)).
		WithArgs(StatusStarting, "RMxyz", "openai_realtime", "gpt-4o-realtime-preview", sqlmock.AnyArg(), id).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.StartSession(ctx, id, "RMxyz", "openai_realtime", "gpt-4o-realtime-preview")
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_StartSession_NotFound(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()
	id := uuid.New()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT status FROM interview_sessions WHERE id = $1`)).
		WithArgs(id).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	err := repo.StartSession(ctx, id, "RMabc", "p", "m")
	assert.ErrorIs(t, err, ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_StartSession_InvalidStatus(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()
	id := uuid.New()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT status FROM interview_sessions WHERE id = $1`)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(StatusCompleted))
	mock.ExpectRollback()

	err := repo.StartSession(ctx, id, "RMabc", "p", "m")
	assert.ErrorIs(t, err, ErrInvalidStatus)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_StartSession_BeginTxError(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()
	id := uuid.New()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT status FROM interview_sessions WHERE id = $1`)).
		WithArgs(id).
		WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()

	err := repo.StartSession(ctx, id, "RMerr", "p", "m")
	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrNotFound)
	assert.NotErrorIs(t, err, ErrInvalidStatus)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_StartSession_UpdateError(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()
	id := uuid.New()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT status FROM interview_sessions WHERE id = $1`)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(StatusPending))
	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE interview_sessions
		 SET status = $1, external_session_id = $2, provider = $3, model = $4, updated_at = $5
		 WHERE id = $6`)).
		WithArgs(StatusStarting, "RMfail", "p", "m", sqlmock.AnyArg(), id).
		WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()

	err := repo.StartSession(ctx, id, "RMfail", "p", "m")
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// Edge case: GetByID with nil optional fields
// ---------------------------------------------------------------------------

func TestRepository_GetByID_NilOptionalFields(t *testing.T) {
	repo, _, mock := newRepo(t)
	ctx := context.Background()

	id := uuid.New()
	now := nowTruncated()
	session := &InterviewSession{
		ID:                id,
		ApplicationID:     uuid.New(),
		Mode:              ModeAssist,
		Status:            StatusPending,
		ExternalSessionID: nil,
		Provider:          "",
		Model:             "",
		Transcript:        nil,
		Score:             nil,
		Feedback:          nil,
		StartedAt:         nil,
		EndedAt:           nil,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	rows := newMockRows().AddRow(
		session.ID,
		session.ApplicationID,
		session.Mode,
		session.Status,
		nil, // ExternalSessionID
		session.Provider,
		session.Model,
		nil,            // transcript ([]TranscriptEntry - slice, nil is fine)
		nil,            // score (*float64 - nil is fine)
		[]byte("null"), // feedback (json.RawMessage can't scan nil)
		nil,            // started_at
		nil,            // ended_at
		session.CreatedAt,
		session.UpdatedAt,
	)

	mock.ExpectQuery(regexp.QuoteMeta(getByIDQuery())).
		WithArgs(id).
		WillReturnRows(rows)

	got, err := repo.GetByID(ctx, id)
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Nil(t, got.ExternalSessionID)
	assert.Nil(t, got.Score)
	assert.Equal(t, json.RawMessage("null"), got.Feedback) // json.RawMessage can't scan nil, mock returns "null"
	assert.Nil(t, got.StartedAt)
	assert.Nil(t, got.EndedAt)
	assert.Nil(t, got.Transcript)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// Unit tests for ListFilter.buildWhere
// ---------------------------------------------------------------------------

func TestListFilter_BuildWhere_NoFilters(t *testing.T) {
	f := ListFilter{}
	where, args := f.buildWhere()
	assert.Empty(t, where)
	assert.Nil(t, args)
}

func TestListFilter_BuildWhere_ApplicationIDOnly(t *testing.T) {
	appID := uuid.New()
	f := ListFilter{ApplicationID: appID}
	where, args := f.buildWhere()
	assert.Equal(t, " WHERE application_id = $1", where)
	assert.Equal(t, []interface{}{appID}, args)
}

func TestListFilter_BuildWhere_StatusOnly(t *testing.T) {
	f := ListFilter{Status: StatusActive}
	where, args := f.buildWhere()
	assert.Equal(t, " WHERE status = $1", where)
	assert.Equal(t, []interface{}{StatusActive}, args)
}

func TestListFilter_BuildWhere_ModeOnly(t *testing.T) {
	f := ListFilter{Mode: ModeAssist}
	where, args := f.buildWhere()
	assert.Equal(t, " WHERE mode = $1", where)
	assert.Equal(t, []interface{}{ModeAssist}, args)
}

func TestListFilter_BuildWhere_AllFilters(t *testing.T) {
	appID := uuid.New()
	f := ListFilter{
		ApplicationID: appID,
		Status:        StatusActive,
		Mode:          ModeAutonomous,
	}
	where, args := f.buildWhere()
	assert.Equal(t, " WHERE application_id = $1 AND status = $2 AND mode = $3", where)
	assert.Equal(t, []interface{}{appID, StatusActive, ModeAutonomous}, args)
}
