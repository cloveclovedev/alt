package planning

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (s *Store) CreateGoogleOAuthState(ctx context.Context, userID string, hash, encryptedVerifier []byte, expiresAt time.Time) error {
	if _, err := s.pool.Exec(ctx, `DELETE FROM google_oauth_states WHERE expires_at <= now()`); err != nil {
		return fmt.Errorf("delete expired Google OAuth states: %w", err)
	}
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO google_oauth_states (state_hash, user_id, encrypted_code_verifier, expires_at)
		VALUES ($1, $2, $3, $4)
	`, hash, userID, encryptedVerifier, expiresAt); err != nil {
		return fmt.Errorf("save Google OAuth state: %w", err)
	}
	return nil
}

// ConsumeGoogleOAuthState deletes and returns a state in one statement so it cannot be reused.
func (s *Store) ConsumeGoogleOAuthState(ctx context.Context, userID string, hash []byte, now time.Time) ([]byte, error) {
	var verifier []byte
	err := s.pool.QueryRow(ctx, `
		DELETE FROM google_oauth_states
		WHERE state_hash = $1 AND user_id = $2 AND expires_at > $3
		RETURNING encrypted_code_verifier
	`, hash, userID, now).Scan(&verifier)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("%w: Google authorization request expired or was already used", ErrInvalidInput)
	}
	if err != nil {
		return nil, fmt.Errorf("consume Google OAuth state: %w", err)
	}
	return verifier, nil
}

// ListGitHubRepositories returns user-configured repositories in display order.
func (s *Store) ListGitHubRepositories(ctx context.Context, userID string) ([]GitHubRepository, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, owner, name, enabled, planning_instructions, created_at, updated_at
		FROM github_repositories
		WHERE user_id = $1
		ORDER BY owner, name, id
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list GitHub repositories: %w", err)
	}
	defer rows.Close()
	var repositories []GitHubRepository
	for rows.Next() {
		var repository GitHubRepository
		if err := rows.Scan(&repository.ID, &repository.Owner, &repository.Name, &repository.Enabled, &repository.PlanningInstructions, &repository.CreatedAt, &repository.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan GitHub repository: %w", err)
		}
		repositories = append(repositories, repository)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate GitHub repositories: %w", err)
	}
	return repositories, nil
}

// SaveGitHubRepository creates or updates one repository after upstream validation.
func (s *Store) SaveGitHubRepository(ctx context.Context, userID string, repository GitHubRepository) error {
	if repository.ID == "" {
		_, err := s.pool.Exec(ctx, `
			INSERT INTO github_repositories (user_id, owner, name, enabled, planning_instructions)
			VALUES ($1, $2, $3, $4, $5)
		`, userID, repository.Owner, repository.Name, repository.Enabled, repository.PlanningInstructions)
		if duplicatePlanningInput(err, "repository") != nil {
			return duplicatePlanningInput(err, "repository")
		}
		if err != nil {
			return fmt.Errorf("insert GitHub repository: %w", err)
		}
		return nil
	}
	command, err := s.pool.Exec(ctx, `
		UPDATE github_repositories
		SET owner = $3, name = $4, enabled = $5, planning_instructions = $6, updated_at = now()
		WHERE id = $1 AND user_id = $2
	`, repository.ID, userID, repository.Owner, repository.Name, repository.Enabled, repository.PlanningInstructions)
	if err != nil {
		return fmt.Errorf("update GitHub repository: %w", err)
	}
	if command.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteGitHubRepository(ctx context.Context, userID, repositoryID string) error {
	command, err := s.pool.Exec(ctx, `DELETE FROM github_repositories WHERE id = $1 AND user_id = $2`, repositoryID, userID)
	if err != nil {
		return fmt.Errorf("delete GitHub repository: %w", err)
	}
	if command.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}

// LoadCalendarConnection loads encrypted credential material only for a server-side adapter.
func (s *Store) LoadCalendarConnection(ctx context.Context, userID string) (CalendarConnection, []byte, error) {
	var connection CalendarConnection
	var encrypted []byte
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, granted_scopes, encrypted_refresh_token, created_at, updated_at
		FROM google_calendar_connections WHERE user_id = $1
	`, userID).Scan(&connection.ID, &connection.GrantedScopes, &encrypted, &connection.CreatedAt, &connection.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return CalendarConnection{}, nil, ErrNotFound
	}
	if err != nil {
		return CalendarConnection{}, nil, fmt.Errorf("load Calendar connection: %w", err)
	}
	return connection, encrypted, nil
}

func (s *Store) UpsertCalendarConnection(ctx context.Context, userID string, encryptedToken []byte, scopes []string) (CalendarConnection, error) {
	var connection CalendarConnection
	err := s.pool.QueryRow(ctx, `
		INSERT INTO google_calendar_connections (user_id, encrypted_refresh_token, granted_scopes)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO UPDATE SET
			encrypted_refresh_token = EXCLUDED.encrypted_refresh_token,
			granted_scopes = EXCLUDED.granted_scopes,
			updated_at = now()
		RETURNING id::text, granted_scopes, created_at, updated_at
	`, userID, encryptedToken, scopes).Scan(&connection.ID, &connection.GrantedScopes, &connection.CreatedAt, &connection.UpdatedAt)
	if err != nil {
		return CalendarConnection{}, fmt.Errorf("save Calendar connection: %w", err)
	}
	return connection, nil
}

func (s *Store) ListCalendarSources(ctx context.Context, userID string) ([]CalendarSource, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT s.id::text, s.connection_id::text, s.external_calendar_id, s.display_name,
		       s.enabled, s.role::text, s.planning_instructions, s.timezone
		FROM calendar_sources AS s
		JOIN google_calendar_connections AS c ON c.id = s.connection_id
		WHERE c.user_id = $1
		ORDER BY s.display_name, s.id
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list Calendar sources: %w", err)
	}
	defer rows.Close()
	var sources []CalendarSource
	for rows.Next() {
		var source CalendarSource
		if err := rows.Scan(&source.ID, &source.ConnectionID, &source.ExternalCalendarID, &source.DisplayName, &source.Enabled, &source.Role, &source.PlanningInstructions, &source.Timezone); err != nil {
			return nil, fmt.Errorf("scan Calendar source: %w", err)
		}
		sources = append(sources, source)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Calendar sources: %w", err)
	}
	return sources, nil
}

func (s *Store) ReplaceCalendarSources(ctx context.Context, userID string, sources []CalendarSource) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin Calendar source transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var connectionID string
	if err := tx.QueryRow(ctx, `SELECT id::text FROM google_calendar_connections WHERE user_id = $1 FOR UPDATE`, userID).Scan(&connectionID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("lock Calendar connection: %w", err)
	}
	for _, source := range sources {
		if _, err := tx.Exec(ctx, `
			INSERT INTO calendar_sources (connection_id, external_calendar_id, display_name, enabled, role, planning_instructions, timezone)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (connection_id, external_calendar_id) DO UPDATE SET
				display_name = EXCLUDED.display_name, enabled = EXCLUDED.enabled, role = EXCLUDED.role,
				planning_instructions = EXCLUDED.planning_instructions, timezone = EXCLUDED.timezone, updated_at = now()
		`, connectionID, source.ExternalCalendarID, source.DisplayName, source.Enabled, source.Role, source.PlanningInstructions, source.Timezone); err != nil {
			return fmt.Errorf("save Calendar source: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit Calendar sources: %w", err)
	}
	return nil
}

func (s *Store) SaveModelAssignment(ctx context.Context, userID, purpose, modelID string) error {
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO ai_model_assignments (user_id, purpose, model_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, purpose) DO UPDATE SET model_id = EXCLUDED.model_id, updated_at = now()
	`, userID, purpose, modelID); err != nil {
		return fmt.Errorf("save AI model assignment: %w", err)
	}
	return nil
}

func duplicatePlanningInput(err error, field string) error {
	var databaseError *pgconn.PgError
	if errors.As(err, &databaseError) && databaseError.Code == "23505" {
		return fmt.Errorf("%w: %s already exists", ErrInvalidInput, field)
	}
	return nil
}

func normalizeRepository(repository GitHubRepository) (GitHubRepository, error) {
	repository.Owner = strings.TrimSpace(repository.Owner)
	repository.Name = strings.TrimSpace(repository.Name)
	repository.PlanningInstructions = strings.TrimSpace(repository.PlanningInstructions)
	if repository.Owner == "" || repository.Name == "" {
		return GitHubRepository{}, fmt.Errorf("%w: repository owner and name are required", ErrInvalidInput)
	}
	if len([]rune(repository.Owner)) > 100 || len([]rune(repository.Name)) > 100 || len([]rune(repository.PlanningInstructions)) > 10_000 {
		return GitHubRepository{}, fmt.Errorf("%w: repository input is too long", ErrInvalidInput)
	}
	return repository, nil
}
