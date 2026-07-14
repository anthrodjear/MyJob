package database

import (
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// RunMigrations applies all pending database migrations.
// It uses golang-migrate with embedded migration files.
// The migrations are expected to be in the migrations/ directory
// with sequential numbering (001_initial.up.sql, 001_initial.down.sql, etc.)
func RunMigrations(databaseURL string) error {
	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("create migration source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", sourceDriver, databaseURL)
	if err != nil {
		return fmt.Errorf("create migrate instance: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}

// MigrationVersion returns the current migration version and dirty state.
// Returns (0, false, nil) if no migrations have been applied.
func MigrationVersion(databaseURL string) (version uint, dirty bool, err error) {
	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return 0, false, fmt.Errorf("create migration source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", sourceDriver, databaseURL)
	if err != nil {
		return 0, false, fmt.Errorf("create migrate instance: %w", err)
	}
	defer m.Close()

	v, d, err := m.Version()
	if err != nil {
		if errors.Is(err, migrate.ErrNilVersion) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("get migration version: %w", err)
	}
	return v, d, nil
}
