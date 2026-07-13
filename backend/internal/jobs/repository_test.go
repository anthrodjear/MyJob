package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
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

func newMockRepo(t *testing.T) (*Repository, sqlmock.Sqlmock, func()) {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	db := sqlx.NewDb(mockDB, "postgres")
	repo := NewRepository(db)
	return repo, mock, func() { mockDB.Close() }
}

// selectColumns returns the column names from the full GetByID / List SELECT.
func selectColumns() []string {
	return []string{
		"id", "source_id", "external_id", "title", "company", "location",
		"remote_type", "salary_min", "salary_max", "salary_currency",
		"description", "requirements", "url", "application_url", "company_url",
		"source", "posted_at", "scraped_at", "match_score", "match_details",
		"status", "created_at", "updated_at", "source_name",
	}
}

// makeJobRows creates a sqlmock row set from one or more Job pointers.
func makeJobRows(jobs ...*Job) *sqlmock.Rows {
	rows := sqlmock.NewRows(selectColumns())
	for _, j := range jobs {
		rows.AddRow(
			j.ID.String(),              // id
			j.SourceID.String(),         // source_id
			j.ExternalID,                // external_id
			j.Title,                     // title
			j.Company,                   // company
			j.Location,                  // location
			j.RemoteType,                // remote_type
			int64(j.SalaryMin),          // salary_min
			int64(j.SalaryMax),          // salary_max
			j.SalaryCurrency,            // salary_currency
			j.Description,               // description
			j.Requirements,              // requirements
			j.URL,                       // url
			j.ApplicationURL,            // application_url
			j.CompanyURL,                // company_url
			j.Source,                    // source
			j.PostedAt,                  // posted_at
			j.ScrapedAt,                 // scraped_at
			j.MatchScore,                // match_score
			[]byte(j.MatchDetails),      // match_details
			j.Status,                    // status
			j.CreatedAt,                 // created_at
			j.UpdatedAt,                 // updated_at
			j.SourceName,                // source_name
		)
	}
	return rows
}

// defaultJob returns a fully populated Job for use in tests.
func defaultJob() *Job {
	now := time.Now()
	return &Job{
		ID:             uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		SourceID:       uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		ExternalID:     "ext-001",
		Title:          "Software Engineer",
		Company:        "Acme Corp",
		Location:       "Remote",
		RemoteType:     "remote",
		SalaryMin:      100000,
		SalaryMax:      150000,
		SalaryCurrency: "USD",
		Description:    "Great job",
		Requirements:   "Go, React",
		URL:            "https://example.com/job/1",
		ApplicationURL: "https://example.com/apply/1",
		CompanyURL:     "https://acme.com",
		Source:         "greenhouse",
		ScrapedAt:      now,
		MatchScore:     92.5,
		MatchDetails:   json.RawMessage(`{"skill_match":90}`),
		Status:         StatusDiscovered,
		SourceName:     "Greenhouse",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// defaultCtx is a convenience for tests that don't care about the context.
var defaultCtx = context.Background()

// reboundInsertSQL is the INSERT query after sqlx rebinds :param to $N.
// sqlx.NamedExecContext internally rebinds named parameters to positional
// bindvars and normalizes whitespace to a single line.
const reboundInsertSQL = `INSERT INTO jobs ( id, source_id, external_id, title, company, location, remote_type, salary_min, salary_max, salary_currency, description, requirements, url, application_url, company_url, source, posted_at, scraped_at, match_score, match_details, status, created_at, updated_at ) VALUES ( $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23 ) ON CONFLICT (source_id, external_id) DO NOTHING`

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestRepository_GetByID_Success(t *testing.T) {
	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	job := defaultJob()

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT
			j.id, j.source_id, j.external_id, j.title, j.company, j.location,
			j.remote_type, j.salary_min, j.salary_max, j.salary_currency,
			j.description, j.requirements, j.url, j.application_url, j.company_url, j.source,
			j.posted_at, j.scraped_at, j.match_score, j.match_details,
			j.status, j.created_at, j.updated_at,
			s.name as source_name
		FROM jobs j
		LEFT JOIN job_sources s ON j.source_id = s.id
		WHERE j.id = $1
	`)).
		WithArgs(job.ID.String()).
		WillReturnRows(makeJobRows(job))

	got, err := repo.GetByID(defaultCtx, job.ID)
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, job.ID, got.ID)
	assert.Equal(t, job.Title, got.Title)
	assert.Equal(t, job.Company, got.Company)
	assert.Equal(t, job.MatchScore, got.MatchScore)
	assert.Equal(t, job.Status, got.Status)
	assert.Equal(t, job.SourceName, got.SourceName)
	assert.NotNil(t, got.MatchDetails)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetByID_NotFound(t *testing.T) {
	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT
			j.id, j.source_id, j.external_id, j.title, j.company, j.location,
			j.remote_type, j.salary_min, j.salary_max, j.salary_currency,
			j.description, j.requirements, j.url, j.application_url, j.company_url, j.source,
			j.posted_at, j.scraped_at, j.match_score, j.match_details,
			j.status, j.created_at, j.updated_at,
			s.name as source_name
		FROM jobs j
		LEFT JOIN job_sources s ON j.source_id = s.id
		WHERE j.id = $1
	`)).
		WithArgs(id.String()).
		WillReturnError(sql.ErrNoRows)

	got, err := repo.GetByID(defaultCtx, id)
	assert.Nil(t, got)
	assert.Error(t, err)
	assert.ErrorIs(t, err, sql.ErrNoRows)
	assert.Contains(t, err.Error(), "jobs: get by id")

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetByID_DBError(t *testing.T) {
	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT
			j.id, j.source_id, j.external_id, j.title, j.company, j.location,
			j.remote_type, j.salary_min, j.salary_max, j.salary_currency,
			j.description, j.requirements, j.url, j.application_url, j.company_url, j.source,
			j.posted_at, j.scraped_at, j.match_score, j.match_details,
			j.status, j.created_at, j.updated_at,
			s.name as source_name
		FROM jobs j
		LEFT JOIN job_sources s ON j.source_id = s.id
		WHERE j.id = $1
	`)).
		WithArgs(id.String()).
		WillReturnError(errors.New("connection refused"))

	got, err := repo.GetByID(defaultCtx, id)
	assert.Nil(t, got)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection refused")

	require.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestRepository_List_NoFilters(t *testing.T) {
	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	limit, offset := 20, 0
	filter := ListFilter{Limit: limit, Offset: offset}

	job := defaultJob()

	// Count query
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM jobs j WHERE 1=1`)).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))

	// List query
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT
			j.id, j.source_id, j.external_id, j.title, j.company, j.location,
			j.remote_type, j.salary_min, j.salary_max, j.salary_currency,
			j.description, j.requirements, j.url, j.application_url, j.company_url, j.source,
			j.posted_at, j.scraped_at, j.match_score, j.match_details,
			j.status, j.created_at, j.updated_at,
			s.name as source_name
		FROM jobs j
		LEFT JOIN job_sources s ON j.source_id = s.id
		WHERE 1=1
		ORDER BY j.scraped_at DESC
		LIMIT $1 OFFSET $2
	`)).
		WithArgs(int64(limit), int64(offset)).
		WillReturnRows(makeJobRows(job))

	jobs, total, err := repo.List(defaultCtx, filter)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, jobs, 1)
	assert.Equal(t, job.ID, jobs[0].ID)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_List_StatusFilter(t *testing.T) {
	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	limit, offset := 10, 5
	status := StatusMatched
	filter := ListFilter{Status: status, Limit: limit, Offset: offset}

	job := defaultJob()
	job.Status = status

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM jobs j WHERE 1=1 AND j.status = $1`)).
		WithArgs(status).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT
			j.id, j.source_id, j.external_id, j.title, j.company, j.location,
			j.remote_type, j.salary_min, j.salary_max, j.salary_currency,
			j.description, j.requirements, j.url, j.application_url, j.company_url, j.source,
			j.posted_at, j.scraped_at, j.match_score, j.match_details,
			j.status, j.created_at, j.updated_at,
			s.name as source_name
		FROM jobs j
		LEFT JOIN job_sources s ON j.source_id = s.id
		WHERE 1=1 AND j.status = $1
		ORDER BY j.scraped_at DESC
		LIMIT $2 OFFSET $3
	`)).
		WithArgs(status, int64(limit), int64(offset)).
		WillReturnRows(makeJobRows(job))

	jobs, total, err := repo.List(defaultCtx, filter)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, jobs, 1)
	assert.Equal(t, StatusMatched, jobs[0].Status)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_List_AllFilters(t *testing.T) {
	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	limit, offset := 10, 0
	sourceID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	filter := ListFilter{
		Status:   StatusApplied,
		Company:  "Acme",
		SourceID: sourceID,
		MinScore: 80.0,
		Limit:    limit,
		Offset:   offset,
	}

	job := defaultJob()
	job.Status = StatusApplied

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM jobs j WHERE 1=1 AND j.status = $1 AND j.company ILIKE $2 AND j.source_id = $3 AND j.match_score >= $4`)).
		WithArgs(StatusApplied, "%Acme%", sourceID.String(), 80.0).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT
			j.id, j.source_id, j.external_id, j.title, j.company, j.location,
			j.remote_type, j.salary_min, j.salary_max, j.salary_currency,
			j.description, j.requirements, j.url, j.application_url, j.company_url, j.source,
			j.posted_at, j.scraped_at, j.match_score, j.match_details,
			j.status, j.created_at, j.updated_at,
			s.name as source_name
		FROM jobs j
		LEFT JOIN job_sources s ON j.source_id = s.id
		WHERE 1=1 AND j.status = $1 AND j.company ILIKE $2 AND j.source_id = $3 AND j.match_score >= $4
		ORDER BY j.scraped_at DESC
		LIMIT $5 OFFSET $6
	`)).
		WithArgs(StatusApplied, "%Acme%", sourceID.String(), 80.0, int64(limit), int64(offset)).
		WillReturnRows(makeJobRows(job))

	jobs, total, err := repo.List(defaultCtx, filter)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, jobs, 1)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_List_CountError(t *testing.T) {
	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	filter := ListFilter{Status: "invalid", Limit: 20, Offset: 0}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM jobs j WHERE 1=1 AND j.status = $1`)).
		WithArgs("invalid").
		WillReturnError(errors.New("count failed"))

	jobs, total, err := repo.List(defaultCtx, filter)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "count failed")
	assert.Equal(t, 0, total)
	assert.Nil(t, jobs)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_List_QueryError(t *testing.T) {
	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	filter := ListFilter{Status: StatusDiscovered, Limit: 20, Offset: 0}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM jobs j WHERE 1=1 AND j.status = $1`)).
		WithArgs(StatusDiscovered).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT
			j.id, j.source_id, j.external_id, j.title, j.company, j.location,
			j.remote_type, j.salary_min, j.salary_max, j.salary_currency,
			j.description, j.requirements, j.url, j.application_url, j.company_url, j.source,
			j.posted_at, j.scraped_at, j.match_score, j.match_details,
			j.status, j.created_at, j.updated_at,
			s.name as source_name
		FROM jobs j
		LEFT JOIN job_sources s ON j.source_id = s.id
		WHERE 1=1 AND j.status = $1
		ORDER BY j.scraped_at DESC
		LIMIT $2 OFFSET $3
	`)).
		WithArgs(StatusDiscovered, int64(20), int64(0)).
		WillReturnError(errors.New("select failed"))

	jobs, total, err := repo.List(defaultCtx, filter)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "select failed")
	assert.Equal(t, 0, total)
	assert.Nil(t, jobs)

	require.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestRepository_Create_Success(t *testing.T) {
	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	job := defaultJob()

	mock.ExpectExec(regexp.QuoteMeta(reboundInsertSQL)).
		WithArgs(
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Create(defaultCtx, job)
	require.NoError(t, err)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_Create_Error(t *testing.T) {
	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	job := defaultJob()

	mock.ExpectExec(regexp.QuoteMeta(reboundInsertSQL)).
		WithArgs(
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(),
		).
		WillReturnError(errors.New("insert failed"))

	err := repo.Create(defaultCtx, job)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insert failed")

	require.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// BulkCreate
// ---------------------------------------------------------------------------

func TestRepository_BulkCreate_Empty(t *testing.T) {
	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	// No DB calls expected for empty slice.
	count, err := repo.BulkCreate(defaultCtx, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	count, err = repo.BulkCreate(defaultCtx, []*Job{})
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_BulkCreate_Success(t *testing.T) {
	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	job1 := defaultJob()
	job2 := defaultJob()
	job2.ID = uuid.MustParse("44444444-4444-4444-4444-444444444444")
	job2.ExternalID = "ext-002"

	mock.ExpectBegin()

	mock.ExpectExec(regexp.QuoteMeta(reboundInsertSQL)).
		WithArgs(
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec(regexp.QuoteMeta(reboundInsertSQL)).
		WithArgs(
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectCommit()

	count, err := repo.BulkCreate(defaultCtx, []*Job{job1, job2})
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_BulkCreate_BeginError(t *testing.T) {
	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	mock.ExpectBegin().WillReturnError(errors.New("begin failed"))

	count, err := repo.BulkCreate(defaultCtx, []*Job{defaultJob()})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "begin failed")
	assert.Equal(t, 0, count)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_BulkCreate_ExecError(t *testing.T) {
	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	mock.ExpectBegin()

	mock.ExpectExec(regexp.QuoteMeta(reboundInsertSQL)).
		WithArgs(
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(),
		).
		WillReturnError(errors.New("exec failed"))

	mock.ExpectRollback()

	count, err := repo.BulkCreate(defaultCtx, []*Job{defaultJob()})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exec failed")
	assert.Equal(t, 0, count)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_BulkCreate_CommitError(t *testing.T) {
	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	mock.ExpectBegin()

	mock.ExpectExec(regexp.QuoteMeta(reboundInsertSQL)).
		WithArgs(
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectCommit().WillReturnError(errors.New("commit failed"))

	// After a failed commit the deferred tx.Rollback() is a no-op
	// in sqlmock, so we do NOT add ExpectRollback here.

	count, err := repo.BulkCreate(defaultCtx, []*Job{defaultJob()})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "commit failed")
	assert.Equal(t, 0, count)

	require.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// ExistsBySourceAndExternalID
// ---------------------------------------------------------------------------

func TestRepository_ExistsBySourceAndExternalID_True(t *testing.T) {
	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	sourceID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	externalID := "ext-001"

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT EXISTS(
			SELECT 1 FROM jobs
			WHERE source_id = $1 AND external_id = $2
		)
	`)).
		WithArgs(sourceID.String(), externalID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	exists, err := repo.ExistsBySourceAndExternalID(defaultCtx, sourceID, externalID)
	require.NoError(t, err)
	assert.True(t, exists)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_ExistsBySourceAndExternalID_False(t *testing.T) {
	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	sourceID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	externalID := "ext-unknown"

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT EXISTS(
			SELECT 1 FROM jobs
			WHERE source_id = $1 AND external_id = $2
		)
	`)).
		WithArgs(sourceID.String(), externalID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	exists, err := repo.ExistsBySourceAndExternalID(defaultCtx, sourceID, externalID)
	require.NoError(t, err)
	assert.False(t, exists)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_ExistsBySourceAndExternalID_Error(t *testing.T) {
	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	sourceID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT EXISTS(
			SELECT 1 FROM jobs
			WHERE source_id = $1 AND external_id = $2
		)
	`)).
		WithArgs(sourceID.String(), sqlmock.AnyArg()).
		WillReturnError(errors.New("query failed"))

	exists, err := repo.ExistsBySourceAndExternalID(defaultCtx, sourceID, "anything")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "query failed")
	assert.False(t, exists)

	require.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// UpdateStatus
// ---------------------------------------------------------------------------

func TestRepository_UpdateStatus_Success(t *testing.T) {
	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	status := StatusArchived

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE jobs
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`)).
		WithArgs(status, id.String()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateStatus(defaultCtx, id, status)
	require.NoError(t, err)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateStatus_NoRows(t *testing.T) {
	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE jobs
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`)).
		WithArgs(StatusArchived, id.String()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.UpdateStatus(defaultCtx, id, StatusArchived)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrNoRowsAffected)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateStatus_Error(t *testing.T) {
	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE jobs
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`)).
		WithArgs(StatusArchived, id.String()).
		WillReturnError(errors.New("update failed"))

	err := repo.UpdateStatus(defaultCtx, id, StatusArchived)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update failed")

	require.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// UpdateMatchScore
// ---------------------------------------------------------------------------

func TestRepository_UpdateMatchScore_Success(t *testing.T) {
	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	score := 95.0
	details := json.RawMessage(`{"skill_match":95,"location_match":100}`)

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE jobs
		SET match_score = $1, match_details = $2, updated_at = NOW()
		WHERE id = $3
	`)).
		WithArgs(score, []byte(details), id.String()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateMatchScore(defaultCtx, id, score, details)
	require.NoError(t, err)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateMatchScore_NoRows(t *testing.T) {
	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	score := 95.0
	details := json.RawMessage(`{"skill_match":95}`)

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE jobs
		SET match_score = $1, match_details = $2, updated_at = NOW()
		WHERE id = $3
	`)).
		WithArgs(score, []byte(details), id.String()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.UpdateMatchScore(defaultCtx, id, score, details)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrNoRowsAffected)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateMatchScore_Error(t *testing.T) {
	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	score := 95.0
	details := json.RawMessage(`{}`)

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE jobs
		SET match_score = $1, match_details = $2, updated_at = NOW()
		WHERE id = $3
	`)).
		WithArgs(score, []byte(details), id.String()).
		WillReturnError(errors.New("update match score failed"))

	err := repo.UpdateMatchScore(defaultCtx, id, score, details)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update match score failed")

	require.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// ListFilter.buildWhere (unit-level, no DB required)
// ---------------------------------------------------------------------------

func TestListFilterBuildWhere(t *testing.T) {
	tests := []struct {
		name    string
		filter  ListFilter
		wantSQL string
		wantLen int // number of args
	}{
		{
			name:    "empty filter",
			filter:  ListFilter{},
			wantSQL: "WHERE 1=1",
			wantLen: 0,
		},
		{
			name:    "status only",
			filter:  ListFilter{Status: "discovered"},
			wantSQL: "WHERE 1=1 AND j.status = $1",
			wantLen: 1,
		},
		{
			name:    "company only",
			filter:  ListFilter{Company: "Acme"},
			wantSQL: "WHERE 1=1 AND j.company ILIKE $1",
			wantLen: 1,
		},
		{
			name:    "source_id only",
			filter:  ListFilter{SourceID: uuid.MustParse("33333333-3333-3333-3333-333333333333")},
			wantSQL: "WHERE 1=1 AND j.source_id = $1",
			wantLen: 1,
		},
		{
			name:    "min_score only",
			filter:  ListFilter{MinScore: 80.0},
			wantSQL: "WHERE 1=1 AND j.match_score >= $1",
			wantLen: 1,
		},
		{
			name:    "status + company",
			filter:  ListFilter{Status: "matched", Company: "Tech"},
			wantSQL: "WHERE 1=1 AND j.status = $1 AND j.company ILIKE $2",
			wantLen: 2,
		},
		{
			name: "all filters",
			filter: ListFilter{
				Status:   "applied",
				Company:  "Corp",
				SourceID: uuid.MustParse("44444444-4444-4444-4444-444444444444"),
				MinScore: 50.0,
			},
			wantSQL: "WHERE 1=1 AND j.status = $1 AND j.company ILIKE $2 AND j.source_id = $3 AND j.match_score >= $4",
			wantLen: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSQL, gotArgs := tt.filter.buildWhere()
			assert.Equal(t, tt.wantSQL, gotSQL)
			assert.Len(t, gotArgs, tt.wantLen)
		})
	}
}


