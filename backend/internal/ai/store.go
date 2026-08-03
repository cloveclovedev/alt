package ai

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store persists purpose-based model assignments and non-content usage metadata.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore constructs a PostgreSQL AI store.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// AssignedModel returns the model chosen for one purpose, or ErrNoModelAssigned.
func (s *Store) AssignedModel(ctx context.Context, userID, purpose string) (string, error) {
	var modelID string
	err := s.pool.QueryRow(ctx, `
		SELECT model_id FROM ai_model_assignments
		WHERE user_id = $1 AND purpose = $2
	`, userID, purpose).Scan(&modelID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNoModelAssigned
	}
	if err != nil {
		return "", fmt.Errorf("load AI model assignment: %w", err)
	}
	return modelID, nil
}

// SaveAssignment upserts the model for one purpose.
func (s *Store) SaveAssignment(ctx context.Context, userID, purpose, modelID string) error {
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO ai_model_assignments (user_id, purpose, model_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, purpose) DO UPDATE SET model_id = EXCLUDED.model_id, updated_at = now()
	`, userID, purpose, modelID); err != nil {
		return fmt.Errorf("save AI model assignment: %w", err)
	}
	return nil
}

// Generation is one non-content usage record: which purpose and model ran, the
// provider's generation id, token counts, and success or failure.
type Generation struct {
	UserID               string
	Purpose              string
	ModelID              string
	ProviderGenerationID string
	PromptVersion        string
	PromptTokens         int
	CompletionTokens     int
	Status               string
}

// RecordGeneration stores provider metadata for diagnostics. No prompt or
// response content is persisted.
func (s *Store) RecordGeneration(ctx context.Context, generation Generation) error {
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO ai_generations (
			user_id, purpose, model_id, provider_generation_id, prompt_version,
			prompt_tokens, completion_tokens, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, generation.UserID, generation.Purpose, generation.ModelID, generation.ProviderGenerationID, generation.PromptVersion, generation.PromptTokens, generation.CompletionTokens, generation.Status); err != nil {
		return fmt.Errorf("record AI generation: %w", err)
	}
	return nil
}
