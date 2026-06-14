package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

type PostgresDB struct {
	DB     *sqlx.DB
	Logger *zap.Logger
}

func NewPostgresDB(databaseURL string, logger *zap.Logger) (*PostgresDB, error) {
	db, err := sqlx.Connect("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Connected to PostgreSQL database")

	return &PostgresDB{
		DB:     db,
		Logger: logger,
	}, nil
}

func (db *PostgresDB) Close() error {
	return db.DB.Close()
}

func (db *PostgresDB) Ping(ctx context.Context) error {
	return db.DB.PingContext(ctx)
}
