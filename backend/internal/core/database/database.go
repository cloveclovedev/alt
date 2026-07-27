// Package database owns the shared PostgreSQL connection pool.
package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect creates and verifies a bounded PostgreSQL connection pool.
func Connect(ctx context.Context, databaseURL string, maxConnections int32) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database URL: %w", err)
	}
	cfg.MaxConns = maxConnections

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}

// BootstrapUser creates the private deployment's configured internal user.
func BootstrapUser(ctx context.Context, pool *pgxpool.Pool, id, displayName, timezone string) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO users (id, display_name, timezone)
		VALUES ($1, $2, $3)
		ON CONFLICT (id) DO UPDATE
		SET display_name = EXCLUDED.display_name,
		    timezone = EXCLUDED.timezone,
		    updated_at = now()
	`, id, displayName, timezone)
	if err != nil {
		return fmt.Errorf("bootstrap user: %w", err)
	}
	return nil
}
