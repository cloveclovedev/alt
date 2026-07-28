// Package identity owns the provider-independent user anchor and profile.
package identity

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store persists identity records in PostgreSQL.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore constructs an identity store.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// BootstrapLocalUser creates the configured private-deployment user and
// application profile. Authentication identities are added separately when an
// authentication provider is introduced.
func (s *Store) BootstrapLocalUser(
	ctx context.Context,
	id, displayName, timezone string,
) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin local user bootstrap: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO users (id)
		VALUES ($1)
		ON CONFLICT (id) DO NOTHING
	`, id); err != nil {
		return fmt.Errorf("bootstrap user: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, display_name, timezone)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO UPDATE
		SET display_name = EXCLUDED.display_name,
		    timezone = EXCLUDED.timezone,
		    updated_at = now()
	`, id, displayName, timezone); err != nil {
		return fmt.Errorf("bootstrap profile: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit local user bootstrap: %w", err)
	}
	return nil
}
