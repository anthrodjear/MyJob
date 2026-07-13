package scoring

import (
	"context"
	"database/sql"
	"encoding/json"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper: convert a JSON string literal to []byte for sqlmock rows.
// sqlmock driver returns driver.Value as string, but json.RawMessage expects
// []byte. We must pass raw []byte so sqlx's intermediate conversion works.
func rawJSON(s string) []byte {
	return []byte(s)
}

// nilRawMessage is a nil []byte used in mock expectations where json.RawMessage is nil.
var nilRawMessage []byte // nil []byte

// newMockRepo creates a sqlmock DB wrapped in sqlx and a PostgresRepository.
// Returns the mock, the repo, and a cleanup func that asserts all expectations were met.
func newMockRepo(t *testing.T) (sqlmock.Sqlmock, *PostgresRepository, func()) {
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

// ---------------------------------------------------------------------------
// GetJob
// ---------------------------------------------------------------------------

func TestPostgresRepository_GetJob_Success(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	jobID := uuid.New()
	details := json.RawMessage(`{"skill_match":85}`)

	rows := sqlmock.NewRows([]string{
		"id", "title", "company", "description", "requirements", "location",
		"remote_type", "salary_min", "salary_max", "match_score", "score_tier",
		"match_details", "scoring_reasoning", "scoring_model", "scoring_source",
	}).AddRow(
		jobID, "Software Engineer", "Acme Corp", "Build things", "Go, SQL",
		"San Francisco", "hybrid", 100000, 150000, 92.5, "auto",
		details, "Strong match", "gpt-4o", "hybrid",
	)

	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT id, title, company, description, requirements, location, remote_type, salary_min, salary_max, match_score, score_tier, match_details, scoring_reasoning, scoring_model, scoring_source FROM jobs WHERE id = $1",
	)).WithArgs(jobID).WillReturnRows(rows)

	job, err := repo.GetJob(context.Background(), jobID)
	require.NoError(t, err)

	assert.Equal(t, jobID, job.ID)
	assert.Equal(t, "Software Engineer", job.Title)
	assert.Equal(t, "Acme Corp", job.Company)
	assert.Equal(t, "Build things", job.Description)
	assert.Equal(t, "Go, SQL", job.Requirements)
	assert.Equal(t, "San Francisco", job.Location)
	assert.Equal(t, "hybrid", job.RemoteType)
	assert.Equal(t, 100000, job.SalaryMin)
	assert.Equal(t, 150000, job.SalaryMax)
	assert.Equal(t, 92.5, job.MatchScore)
	assert.Equal(t, "auto", job.ScoreTier)
	assert.JSONEq(t, `{"skill_match":85}`, string(job.MatchDetails))
	assert.Equal(t, "Strong match", job.ScoringReasoning)
	assert.Equal(t, "gpt-4o", job.ScoringModel)
	assert.Equal(t, "hybrid", job.ScoringSource)
}

func TestPostgresRepository_GetJob_QuietFieldDefaults(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	jobID := uuid.New()

	rows := sqlmock.NewRows([]string{
		"id", "title", "company", "description", "requirements", "location",
		"remote_type", "salary_min", "salary_max", "match_score", "score_tier",
		"match_details", "scoring_reasoning", "scoring_model", "scoring_source",
	}).AddRow(
		jobID, "Engineer", "", "", "", "", "", 0, 0, 0.0, "",
		nilRawMessage, "", "", "",
	)

	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT id, title, company, description, requirements, location, remote_type, salary_min, salary_max, match_score, score_tier, match_details, scoring_reasoning, scoring_model, scoring_source FROM jobs WHERE id = $1",
	)).WithArgs(jobID).WillReturnRows(rows)

	job, err := repo.GetJob(context.Background(), jobID)
	require.NoError(t, err)

	assert.Equal(t, jobID, job.ID)
	assert.Equal(t, "Engineer", job.Title)
	assert.Empty(t, job.Company)
	assert.Empty(t, job.Description)
	assert.Empty(t, job.Requirements)
	assert.Empty(t, job.Location)
	assert.Empty(t, job.RemoteType)
	assert.Equal(t, 0, job.SalaryMin)
	assert.Equal(t, 0, job.SalaryMax)
	assert.Equal(t, 0.0, job.MatchScore)
	assert.Empty(t, job.ScoreTier)
	assert.Nil(t, job.MatchDetails)
	assert.Empty(t, job.ScoringReasoning)
	assert.Empty(t, job.ScoringModel)
	assert.Empty(t, job.ScoringSource)
}

func TestPostgresRepository_GetJob_ErrNotFound(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	jobID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT id, title, company, description, requirements, location, remote_type, salary_min, salary_max, match_score, score_tier, match_details, scoring_reasoning, scoring_model, scoring_source FROM jobs WHERE id = $1",
	)).WithArgs(jobID).WillReturnError(sql.ErrNoRows)

	_, err := repo.GetJob(context.Background(), jobID)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestPostgresRepository_GetJob_DatabaseError(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	jobID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT id, title, company, description, requirements, location, remote_type, salary_min, salary_max, match_score, score_tier, match_details, scoring_reasoning, scoring_model, scoring_source FROM jobs WHERE id = $1",
	)).WithArgs(jobID).WillReturnError(sql.ErrConnDone)

	_, err := repo.GetJob(context.Background(), jobID)
	assert.ErrorIs(t, err, sql.ErrConnDone)
}

func TestPostgresRepository_GetJob_ContextCancelled(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	jobID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT id, title, company, description, requirements, location, remote_type, salary_min, salary_max, match_score, score_tier, match_details, scoring_reasoning, scoring_model, scoring_source FROM jobs WHERE id = $1",
	)).WithArgs(jobID).WillReturnError(context.Canceled)

	_, err := repo.GetJob(context.Background(), jobID)
	assert.ErrorIs(t, err, context.Canceled)
}

// ---------------------------------------------------------------------------
// GetProfile
// ---------------------------------------------------------------------------

func TestPostgresRepository_GetProfile_Success(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	profileJSON := `{
		"skills": ["Go", "Python"],
		"experience": [{"title": "SWE", "company": "Co", "skills_used": ["Go"]}],
		"preferences": {"preferred_locations": ["Remote"], "remote_only": true, "salary_min": 80000, "salary_max": 120000},
		"specializations": ["Backend"],
		"industries": ["Tech"],
		"career_goals": ["Senior Engineer"]
	}`

	rows := sqlmock.NewRows([]string{"data"}).AddRow(rawJSON(profileJSON))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT data FROM profiles LIMIT 1")).WillReturnRows(rows)

	profile, err := repo.GetProfile(context.Background())
	require.NoError(t, err)

	assert.Equal(t, []string{"Go", "Python"}, profile.Skills)
	require.Len(t, profile.Experience, 1)
	assert.Equal(t, "SWE", profile.Experience[0].Title)
	assert.Equal(t, "Co", profile.Experience[0].Company)
	assert.Equal(t, []string{"Go"}, profile.Experience[0].SkillsUsed)
	assert.True(t, profile.Preferences.RemoteOnly)
	assert.Equal(t, []string{"Remote"}, profile.Preferences.PreferredLocations)
	assert.Equal(t, 80000, profile.Preferences.SalaryMin)
	assert.Equal(t, 120000, profile.Preferences.SalaryMax)
	assert.Equal(t, []string{"Backend"}, profile.Specializations)
	assert.Equal(t, []string{"Tech"}, profile.Industries)
	assert.Equal(t, []string{"Senior Engineer"}, profile.CareerGoals)
}

func TestPostgresRepository_GetProfile_NoRowsReturnsEmptyProfile(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT data FROM profiles LIMIT 1")).WillReturnError(sql.ErrNoRows)

	profile, err := repo.GetProfile(context.Background())
	require.NoError(t, err)

	assert.Empty(t, profile.Skills)
	assert.Empty(t, profile.Experience)
	assert.Empty(t, profile.Specializations)
	assert.Empty(t, profile.Industries)
	assert.Empty(t, profile.CareerGoals)
	assert.Empty(t, profile.Preferences.PreferredLocations)
}

func TestPostgresRepository_GetProfile_DatabaseError(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT data FROM profiles LIMIT 1")).WillReturnError(sql.ErrConnDone)

	_, err := repo.GetProfile(context.Background())
	assert.ErrorIs(t, err, sql.ErrConnDone)
}

func TestPostgresRepository_GetProfile_InvalidJSON(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	rows := sqlmock.NewRows([]string{"data"}).AddRow(rawJSON("not-valid-json-at-all"))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT data FROM profiles LIMIT 1")).WillReturnRows(rows)

	_, err := repo.GetProfile(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal profile")
}

func TestPostgresRepository_GetProfile_JSONNullLiteral(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	// JSON literal "null" is valid JSON — Unmarshal leaves profile as zero value.
	rows := sqlmock.NewRows([]string{"data"}).AddRow(rawJSON("null"))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT data FROM profiles LIMIT 1")).WillReturnRows(rows)

	profile, err := repo.GetProfile(context.Background())
	require.NoError(t, err)

	assert.Empty(t, profile.Skills)
	assert.Empty(t, profile.Experience)
	assert.Empty(t, profile.Specializations)
}

func TestPostgresRepository_GetProfile_MinimalJSON(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	rows := sqlmock.NewRows([]string{"data"}).AddRow(rawJSON(`{}`))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT data FROM profiles LIMIT 1")).WillReturnRows(rows)

	profile, err := repo.GetProfile(context.Background())
	require.NoError(t, err)

	assert.Empty(t, profile.Skills)
	assert.Empty(t, profile.Experience)
	assert.Empty(t, profile.Specializations)
	assert.Empty(t, profile.Preferences.PreferredLocations)
}

// ---------------------------------------------------------------------------
// PersistScore
// ---------------------------------------------------------------------------

func TestPostgresRepository_PersistScore_Success(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	jobID := uuid.New()
	details := json.RawMessage(`{"skill_match":90,"experience_match":85}`)

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE jobs
		 SET match_score = $1, score_tier = $2, match_details = $3, scored_at = NOW(), updated_at = NOW(),
		     scoring_reasoning = $4, scoring_model = $5, scoring_source = $6
		 WHERE id = $7`,
	)).WithArgs(87.5, "auto", details, "Great match", "gpt-4o", "hybrid", jobID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.PersistScore(context.Background(), jobID, 87.5, "auto", details, "Great match", "gpt-4o", "hybrid")
	assert.NoError(t, err)
}

func TestPostgresRepository_PersistScore_NilDetails(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	jobID := uuid.New()

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE jobs
		 SET match_score = $1, score_tier = $2, match_details = $3, scored_at = NOW(), updated_at = NOW(),
		     scoring_reasoning = $4, scoring_model = $5, scoring_source = $6
		 WHERE id = $7`,
	)).WithArgs(75.0, "review", nilRawMessage, "Decent", "gpt-4o-mini", "heuristic", jobID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.PersistScore(context.Background(), jobID, 75.0, "review", nil, "Decent", "gpt-4o-mini", "heuristic")
	assert.NoError(t, err)
}

func TestPostgresRepository_PersistScore_NoRowsAffected(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	jobID := uuid.New()
	details := json.RawMessage(`{}`)

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE jobs
		 SET match_score = $1, score_tier = $2, match_details = $3, scored_at = NOW(), updated_at = NOW(),
		     scoring_reasoning = $4, scoring_model = $5, scoring_source = $6
		 WHERE id = $7`,
	)).WithArgs(30.0, "reject", details, "Weak", "heuristic", "heuristic", jobID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.PersistScore(context.Background(), jobID, 30.0, "reject", details, "Weak", "heuristic", "heuristic")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestPostgresRepository_PersistScore_DatabaseError(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	jobID := uuid.New()
	details := json.RawMessage(`{}`)

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE jobs
		 SET match_score = $1, score_tier = $2, match_details = $3, scored_at = NOW(), updated_at = NOW(),
		     scoring_reasoning = $4, scoring_model = $5, scoring_source = $6
		 WHERE id = $7`,
	)).WithArgs(50.0, "review", details, "OK", "gpt-4o", "hybrid", jobID).
		WillReturnError(sql.ErrConnDone)

	err := repo.PersistScore(context.Background(), jobID, 50.0, "review", details, "OK", "gpt-4o", "hybrid")
	assert.ErrorIs(t, err, sql.ErrConnDone)
}

func TestPostgresRepository_PersistScore_RowsAffectedError(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	jobID := uuid.New()
	details := json.RawMessage(`{}`)

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE jobs
		 SET match_score = $1, score_tier = $2, match_details = $3, scored_at = NOW(), updated_at = NOW(),
		     scoring_reasoning = $4, scoring_model = $5, scoring_source = $6
		 WHERE id = $7`,
	)).WithArgs(90.0, "auto", details, "Excellent", "gpt-4o", "llm", jobID).
		WillReturnResult(sqlmock.NewErrorResult(sql.ErrConnDone))

	err := repo.PersistScore(context.Background(), jobID, 90.0, "auto", details, "Excellent", "gpt-4o", "llm")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rows affected")
}

func TestPostgresRepository_PersistScore_ZeroScoreEdgeCase(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	jobID := uuid.New()
	details := json.RawMessage(`{}`)

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE jobs
		 SET match_score = $1, score_tier = $2, match_details = $3, scored_at = NOW(), updated_at = NOW(),
		     scoring_reasoning = $4, scoring_model = $5, scoring_source = $6
		 WHERE id = $7`,
	)).WithArgs(0.0, "reject", details, "", "", "", jobID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.PersistScore(context.Background(), jobID, 0.0, "reject", details, "", "", "")
	assert.NoError(t, err)
}

func TestPostgresRepository_PersistScore_ContextCancelled(t *testing.T) {
	mock, repo, cleanup := newMockRepo(t)
	defer cleanup()

	jobID := uuid.New()
	details := json.RawMessage(`{}`)

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE jobs
		 SET match_score = $1, score_tier = $2, match_details = $3, scored_at = NOW(), updated_at = NOW(),
		     scoring_reasoning = $4, scoring_model = $5, scoring_source = $6
		 WHERE id = $7`,
	)).WithArgs(60.0, "review", details, "notes", "model", "source", jobID).
		WillReturnError(context.Canceled)

	err := repo.PersistScore(context.Background(), jobID, 60.0, "review", details, "notes", "model", "source")
	assert.ErrorIs(t, err, context.Canceled)
}


