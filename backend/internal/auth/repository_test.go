package auth

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"backend/internal/config"
)

func TestRepository_seedIfNeeded(t *testing.T) {
	tests := []struct {
		name          string
		initialHash   string
		existingCount int
		expectError   bool
	}{
		{
			name:          "seeds user when table is empty and hash provided",
			initialHash:   "$2a$12$hashedpassword",
			existingCount: 0,
			expectError:   false,
		},
		{
			name:          "skips seeding when user already exists",
			initialHash:   "$2a$12$hashedpassword",
			existingCount: 1,
			expectError:   false,
		},
		{
			name:          "skips seeding when hash is empty",
			initialHash:   "",
			existingCount: 0,
			expectError:   false,
		},
		{
			name:          "error on db query",
			initialHash:   "$2a$12$hashedpassword",
			existingCount: -1, // simulate error
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			sqlxDB := sqlx.NewDb(db, "postgres")
			defer sqlxDB.Close()

			if tt.existingCount >= 0 {
				mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users WHERE id = 'local-user'`).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(tt.existingCount))
			} else {
				mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users WHERE id = 'local-user'`).
					WillReturnError(errors.New("db error"))
			}

			if tt.initialHash != "" && tt.existingCount == 0 && !tt.expectError {
				mock.ExpectExec(`INSERT INTO users \(id, password_hash, session_version, password_changed_at\)`).
					WithArgs(tt.initialHash).
					WillReturnResult(sqlmock.NewResult(1, 1))
			}

			_, err = NewRepository(sqlxDB, config.AuthConfig{
				PasswordHash: tt.initialHash,
				BCryptCost:   12,
			})

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_loadUser(t *testing.T) {
	tests := []struct {
		name         string
		setupMock    func(mock sqlmock.Sqlmock)
		expectError  bool
		expectedUser *User
	}{
		{
			name: "user found",
			setupMock: func(mock sqlmock.Sqlmock) {
				now := time.Now()
				rows := sqlmock.NewRows([]string{
					"id", "username", "email", "password_hash", "session_version",
					"last_login_at", "password_changed_at", "onboarding_completed_at",
					"onboarding_step", "created_at", "updated_at",
				}).AddRow(
					"local-user", "testuser", "test@example.com", "$2a$12$hash",
					1, nil, now, now, "llm", now, now,
				)
				mock.ExpectQuery(`SELECT id, username, email, password_hash, session_version,`).
					WillReturnRows(rows)
			},
			expectError: false,
			expectedUser: &User{
				ID:                "local-user",
				Username:          "testuser",
				Email:             "test@example.com",
				PasswordHash:      "$2a$12$hash",
				SessionVersion:    1,
				PasswordChangedAt: time.Time{},
				OnboardingStep:    "llm",
			},
		},
		{
			name: "user not found",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, username, email, password_hash, session_version,`).
					WillReturnError(sql.ErrNoRows)
			},
			expectError:  true,
			expectedUser: nil,
		},
		{
			name: "db error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, username, email, password_hash, session_version,`).
					WillReturnError(errors.New("db error"))
			},
			expectError:  true,
			expectedUser: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			sqlxDB := sqlx.NewDb(db, "postgres")
			defer sqlxDB.Close()

			tt.setupMock(mock)

			repo := &Repository{
				db: sqlxDB,
			}

			user, err := repo.loadUser(context.Background())

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, user)
			} else {
				require.NoError(t, err)
				require.NotNil(t, user)
				assert.Equal(t, tt.expectedUser.ID, user.ID)
				assert.Equal(t, tt.expectedUser.Username, user.Username)
				assert.Equal(t, tt.expectedUser.Email, user.Email)
				assert.Equal(t, tt.expectedUser.PasswordHash, user.PasswordHash)
				assert.Equal(t, tt.expectedUser.SessionVersion, user.SessionVersion)
				assert.Equal(t, tt.expectedUser.OnboardingStep, user.OnboardingStep)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_GetUser(t *testing.T) {
	t.Run("user loaded", func(t *testing.T) {
		now := time.Now()
		user := &User{
			ID:             "local-user",
			Username:       "testuser",
			Email:          "test@example.com",
			PasswordHash:   "hash",
			SessionVersion: 1,
			CreatedAt:      now,
			UpdatedAt:      now,
		}

		repo := &Repository{
			user: user,
		}

		result, err := repo.GetUser(context.Background())
		require.NoError(t, err)
		assert.Equal(t, user, result)
	})

	t.Run("user not loaded", func(t *testing.T) {
		repo := &Repository{
			user: nil,
		}

		_, err := repo.GetUser(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not loaded")
	})
}

func TestRepository_GetPasswordHash(t *testing.T) {
	t.Run("user loaded", func(t *testing.T) {
		repo := &Repository{
			user: &User{PasswordHash: "hashed-password"},
		}

		hash, err := repo.GetPasswordHash(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "hashed-password", hash)
	})

	t.Run("user not loaded - falls back to DB", func(t *testing.T) {
		// Create a mock repo with a mock DB
		mockDB, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		sqlxDB := sqlx.NewDb(mockDB, "postgres")
		repo := &Repository{
			db:   sqlxDB,
			user: nil,
		}

		mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, username, email, password_hash, session_version, last_login_at,
		       password_changed_at, onboarding_completed_at, onboarding_step,
		       created_at, updated_at
		FROM users WHERE id = 'local-user'
	`)).WillReturnRows(sqlmock.NewRows([]string{
			"id", "username", "email", "password_hash", "session_version",
			"last_login_at", "password_changed_at", "onboarding_completed_at", "onboarding_step",
			"created_at", "updated_at",
		}).AddRow("local-user", "test", "test@example.com", "db-hashed-password", 1,
			nil, time.Now(), time.Now(), time.Now(), time.Now(), time.Now()))

		hash, err := repo.GetPasswordHash(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "db-hashed-password", hash)
	})
}

func TestRepository_GetSessionVersion(t *testing.T) {
	t.Run("user loaded", func(t *testing.T) {
		repo := &Repository{
			user: &User{SessionVersion: 5},
		}

		version, err := repo.GetSessionVersion(context.Background())
		require.NoError(t, err)
		assert.Equal(t, 5, version)
	})

	t.Run("user not loaded - falls back to DB", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		sqlxDB := sqlx.NewDb(mockDB, "postgres")
		repo := &Repository{
			db:   sqlxDB,
			user: nil,
		}

		mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, username, email, password_hash, session_version, last_login_at,
		       password_changed_at, onboarding_completed_at, onboarding_step,
		       created_at, updated_at
		FROM users WHERE id = 'local-user'
	`)).WillReturnRows(sqlmock.NewRows([]string{
			"id", "username", "email", "password_hash", "session_version",
			"last_login_at", "password_changed_at", "onboarding_completed_at", "onboarding_step",
			"created_at", "updated_at",
		}).AddRow("local-user", "test", "test@example.com", "db-hashed-password", 3,
			nil, time.Now(), time.Now(), time.Now(), time.Now(), time.Now()))

		version, err := repo.GetSessionVersion(context.Background())
		require.NoError(t, err)
		assert.Equal(t, 3, version)
	})
}

func TestRepository_UpdatePasswordHash(t *testing.T) {
	tests := []struct {
		name        string
		newHash     string
		setupMock   func(mock sqlmock.Sqlmock)
		expectError bool
	}{
		{
			name:    "successful update",
			newHash: "$2a$12$newhash",
			setupMock: func(mock sqlmock.Sqlmock) {
				// UPDATE query
				mock.ExpectExec(`UPDATE users`).
					WithArgs("$2a$12$newhash", sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
				// loadUser query
				mock.ExpectQuery(`SELECT id, username, email, password_hash, session_version,`).
					WillReturnRows(sqlmock.NewRows([]string{
						"id", "username", "email", "password_hash", "session_version",
						"last_login_at", "password_changed_at", "onboarding_completed_at",
						"onboarding_step", "created_at", "updated_at",
					}).AddRow("local-user", "testuser", "test@example.com", "$2a$12$newhash",
						1, nil, time.Now(), time.Now(), "llm", time.Now(), time.Now()))
			},
			expectError: false,
		},
		{
			name:    "db update error",
			newHash: "$2a$12$newhash",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE users`).
					WillReturnError(errors.New("db error"))
			},
			expectError: true,
		},
		{
			name:    "reload user error",
			newHash: "$2a$12$newhash",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE users`).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectQuery(`SELECT id, username, email, password_hash, session_version,`).
					WillReturnError(errors.New("db error"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			sqlxDB := sqlx.NewDb(db, "postgres")
			defer sqlxDB.Close()

			tt.setupMock(mock)

			repo := &Repository{
				db: sqlxDB,
			}

			err = repo.UpdatePasswordHash(context.Background(), tt.newHash)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_UpdateLastLogin(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(mock sqlmock.Sqlmock)
		expectError bool
	}{
		{
			name: "successful update",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE users`).
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectError: false,
		},
		{
			name: "db error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE users`).
					WillReturnError(errors.New("db error"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			sqlxDB := sqlx.NewDb(db, "postgres")
			defer sqlxDB.Close()

			tt.setupMock(mock)

			repo := &Repository{
				db:   sqlxDB,
				user: &User{},
			}

			err = repo.UpdateLastLogin(context.Background())

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_IncrementSessionVersion(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(mock sqlmock.Sqlmock)
		expectError bool
	}{
		{
			name: "successful increment",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE users`).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectError: false,
		},
		{
			name: "db error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE users`).
					WillReturnError(errors.New("db error"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			sqlxDB := sqlx.NewDb(db, "postgres")
			defer sqlxDB.Close()

			tt.setupMock(mock)

			repo := &Repository{
				db:   sqlxDB,
				user: &User{},
			}

			err = repo.IncrementSessionVersion(context.Background())

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_IsSetupRequired(t *testing.T) {
	tests := []struct {
		name        string
		userCount   int
		expectError bool
		expected    bool
	}{
		{
			name:        "no users - setup required",
			userCount:   0,
			expectError: false,
			expected:    true,
		},
		{
			name:        "users exist - setup not required",
			userCount:   1,
			expectError: false,
			expected:    false,
		},
		{
			name:        "multiple users - setup not required",
			userCount:   5,
			expectError: false,
			expected:    false,
		},
		{
			name:        "db error",
			userCount:   -1,
			expectError: true,
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			sqlxDB := sqlx.NewDb(db, "postgres")
			defer sqlxDB.Close()

			if tt.userCount >= 0 {
				mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users`).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(tt.userCount))
			} else {
				mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users`).
					WillReturnError(errors.New("db error"))
			}

			repo := &Repository{db: sqlxDB}
			result, err := repo.IsSetupRequired(context.Background())

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_CreateAdminUser(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(mock sqlmock.Sqlmock)
		expectError bool
	}{
		{
			name: "successful creation",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users`).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
				mock.ExpectExec(`INSERT INTO users`).
					WithArgs("testuser", "test@example.com", "$2a$12$hash", sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectError: false,
		},
		{
			name: "users already exist",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users`).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			},
			expectError: true,
		},
		{
			name: "db error on count",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users`).
					WillReturnError(errors.New("db error"))
			},
			expectError: true,
		},
		{
			name: "db error on insert",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users`).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
				mock.ExpectExec(`INSERT INTO users`).
					WillReturnError(errors.New("db error"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			sqlxDB := sqlx.NewDb(db, "postgres")
			defer sqlxDB.Close()

			tt.setupMock(mock)

			repo := &Repository{db: sqlxDB}
			err = repo.CreateAdminUser(context.Background(), "testuser", "test@example.com", "$2a$12$hash")

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_IsOnboardingCompleted(t *testing.T) {
	tests := []struct {
		name        string
		completed   bool
		expectError bool
		expected    bool
	}{
		{
			name:        "onboarding completed",
			completed:   true,
			expectError: false,
			expected:    true,
		},
		{
			name:        "onboarding not completed",
			completed:   false,
			expectError: false,
			expected:    false,
		},
		{
			name:        "db error",
			completed:   false,
			expectError: true,
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			sqlxDB := sqlx.NewDb(db, "postgres")
			defer sqlxDB.Close()

			if tt.expectError {
				mock.ExpectQuery(`SELECT onboarding_completed_at IS NOT NULL FROM users WHERE id = 'local-user'`).
					WillReturnError(errors.New("db error"))
			} else {
				mock.ExpectQuery(`SELECT onboarding_completed_at IS NOT NULL FROM users WHERE id = 'local-user'`).
					WillReturnRows(sqlmock.NewRows([]string{"onboarding_completed_at IS NOT NULL"}).AddRow(tt.completed))
			}

			repo := &Repository{db: sqlxDB}
			result, err := repo.IsOnboardingCompleted(context.Background())

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_GetOnboardingStep(t *testing.T) {
	tests := []struct {
		name        string
		step        sql.NullString
		expectError bool
		expected    string
	}{
		{
			name:        "step set",
			step:        sql.NullString{String: "llm", Valid: true},
			expectError: false,
			expected:    "llm",
		},
		{
			name:        "step not set (NULL)",
			step:        sql.NullString{Valid: false},
			expectError: false,
			expected:    "account",
		},
		{
			name:        "db error",
			step:        sql.NullString{},
			expectError: true,
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			sqlxDB := sqlx.NewDb(db, "postgres")
			defer sqlxDB.Close()

			if tt.expectError {
				mock.ExpectQuery(`SELECT onboarding_step FROM users WHERE id = 'local-user'`).
					WillReturnError(errors.New("db error"))
			} else {
				mock.ExpectQuery(`SELECT onboarding_step FROM users WHERE id = 'local-user'`).
					WillReturnRows(sqlmock.NewRows([]string{"onboarding_step"}).AddRow(tt.step))
			}

			repo := &Repository{db: sqlxDB}
			result, err := repo.GetOnboardingStep(context.Background())

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_SetOnboardingCompleted(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(mock sqlmock.Sqlmock)
		expectError bool
	}{
		{
			name: "successful update",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE users SET onboarding_completed_at = \$1, updated_at = \$2 WHERE id = 'local-user'`).
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectError: false,
		},
		{
			name: "db error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE users SET onboarding_completed_at = \$1, updated_at = \$2 WHERE id = 'local-user'`).
					WillReturnError(errors.New("db error"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			sqlxDB := sqlx.NewDb(db, "postgres")
			defer sqlxDB.Close()

			tt.setupMock(mock)

			repo := &Repository{db: sqlxDB}
			err = repo.SetOnboardingCompleted(context.Background(), time.Now())

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_UpdateOnboardingStep(t *testing.T) {
	tests := []struct {
		name        string
		step        string
		setupMock   func(mock sqlmock.Sqlmock)
		expectError bool
	}{
		{
			name: "successful update",
			step: "llm",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE users SET onboarding_step = \$1, updated_at = \$2 WHERE id = 'local-user'`).
					WithArgs("llm", sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectError: false,
		},
		{
			name: "db error",
			step: "voice",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE users SET onboarding_step = \$1, updated_at = \$2 WHERE id = 'local-user'`).
					WillReturnError(errors.New("db error"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			sqlxDB := sqlx.NewDb(db, "postgres")
			defer sqlxDB.Close()

			tt.setupMock(mock)

			repo := &Repository{db: sqlxDB}
			err = repo.UpdateOnboardingStep(context.Background(), tt.step)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// Refresh Token Tests

func TestRepository_CreateRefreshToken(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(mock sqlmock.Sqlmock)
		expectError bool
	}{
		{
			name: "successful creation",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO refresh_tokens`).
					WithArgs("token-id", "user-id", "hash", sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectError: false,
		},
		{
			name: "db error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO refresh_tokens`).
					WillReturnError(errors.New("db error"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			sqlxDB := sqlx.NewDb(db, "postgres")
			defer sqlxDB.Close()

			tt.setupMock(mock)

			repo := &Repository{db: sqlxDB}
			err = repo.CreateRefreshToken(context.Background(), "token-id", "user-id", "hash", time.Now().Add(time.Hour))

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_GetRefreshTokenByHash(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(mock sqlmock.Sqlmock)
		expectError bool
		expected    *RefreshToken
	}{
		{
			name: "token found",
			setupMock: func(mock sqlmock.Sqlmock) {
				now := time.Now()
				rows := sqlmock.NewRows([]string{
					"id", "user_id", "token_hash", "expires_at", "created_at", "revoked_at", "updated_at",
				}).AddRow("token-id", "user-id", "hash", now.Add(time.Hour), now, nil, now)
				mock.ExpectQuery(`SELECT id, user_id, token_hash, expires_at, created_at, revoked_at, updated_at FROM refresh_tokens WHERE token_hash = \$1`).
					WillReturnRows(rows)
			},
			expectError: false,
			expected: &RefreshToken{
				ID:        "token-id",
				UserID:    "user-id",
				TokenHash: "hash",
				RevokedAt: nil,
			},
		},
		{
			name: "token not found",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, user_id, token_hash, expires_at, created_at, revoked_at, updated_at FROM refresh_tokens WHERE token_hash = \$1`).
					WillReturnError(sql.ErrNoRows)
			},
			expectError: true,
			expected:    nil,
		},
		{
			name: "db error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, user_id, token_hash, expires_at, created_at, revoked_at, updated_at FROM refresh_tokens WHERE token_hash = \$1`).
					WillReturnError(errors.New("db error"))
			},
			expectError: true,
			expected:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			sqlxDB := sqlx.NewDb(db, "postgres")
			defer sqlxDB.Close()

			tt.setupMock(mock)

			repo := &Repository{db: sqlxDB}
			result, err := repo.GetRefreshTokenByHash(context.Background(), "hash")

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.expected.ID, result.ID)
				assert.Equal(t, tt.expected.UserID, result.UserID)
				assert.Equal(t, tt.expected.TokenHash, result.TokenHash)
				if tt.expected.RevokedAt != nil {
					assert.Equal(t, *tt.expected.RevokedAt, *result.RevokedAt)
				} else {
					assert.Nil(t, result.RevokedAt)
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_RevokeRefreshToken(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(mock sqlmock.Sqlmock)
		expectError bool
	}{
		{
			name: "successful revocation",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE refresh_tokens`).
					WithArgs(sqlmock.AnyArg(), "token-id").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectError: false,
		},
		{
			name: "token not found or already revoked",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE refresh_tokens`).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectError: false, // No error, just 0 rows affected
		},
		{
			name: "db error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE refresh_tokens`).
					WillReturnError(errors.New("db error"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			sqlxDB := sqlx.NewDb(db, "postgres")
			defer sqlxDB.Close()

			tt.setupMock(mock)

			repo := &Repository{db: sqlxDB}
			err = repo.RevokeRefreshToken(context.Background(), "token-id")

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_RevokeAllRefreshTokens(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(mock sqlmock.Sqlmock)
		expectError bool
	}{
		{
			name: "successful revocation",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE refresh_tokens`).
					WithArgs(sqlmock.AnyArg(), "user-id").
					WillReturnResult(sqlmock.NewResult(5, 5))
			},
			expectError: false,
		},
		{
			name: "db error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE refresh_tokens`).
					WillReturnError(errors.New("db error"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			sqlxDB := sqlx.NewDb(db, "postgres")
			defer sqlxDB.Close()

			tt.setupMock(mock)

			repo := &Repository{db: sqlxDB}
			err = repo.RevokeAllRefreshTokens(context.Background(), "user-id")

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_DeleteExpiredRefreshTokens(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(mock sqlmock.Sqlmock)
		expectError bool
		expected    int64
	}{
		{
			name: "tokens deleted",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM refresh_tokens`).
					WithArgs(sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(10, 10))
			},
			expectError: false,
			expected:    10,
		},
		{
			name: "db error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM refresh_tokens`).
					WillReturnError(errors.New("db error"))
			},
			expectError: true,
			expected:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			sqlxDB := sqlx.NewDb(db, "postgres")
			defer sqlxDB.Close()

			tt.setupMock(mock)

			repo := &Repository{db: sqlxDB}
			deleted, err := repo.DeleteExpiredRefreshTokens(context.Background())

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, int64(0), deleted)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, deleted)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
