package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.uber.org/zap"

	"backend/internal/config"
)

type PostgresDB struct {
	DB     *sqlx.DB
	Logger *zap.Logger
}

func NewPostgresDB(cfg config.DatabaseConfig, logger *zap.Logger) (*PostgresDB, error) {
	db, err := sqlx.Connect("postgres", cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool from application config.
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Connected to PostgreSQL database",
		zap.Int("max_open_conns", cfg.MaxOpenConns),
		zap.Int("max_idle_conns", cfg.MaxIdleConns),
		zap.Duration("conn_max_lifetime", cfg.ConnMaxLifetime),
	)

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
