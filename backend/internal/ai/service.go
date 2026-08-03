package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// provider is the transport boundary internal/ai depends on. The OpenRouter
// adapter satisfies it; tests can substitute a fake.
type provider interface {
	Complete(ctx context.Context, modelID string, messages []Message, schema *Schema) (Completion, error)
	ListModels(ctx context.Context) ([]Model, error)
}

// store is the persistence boundary. *Store satisfies it; tests can substitute a
// fake to exercise selection and recording without a database.
type store interface {
	AssignedModel(ctx context.Context, userID, purpose string) (string, error)
	SaveAssignment(ctx context.Context, userID, purpose, modelID string) error
	RecordGeneration(ctx context.Context, generation Generation) error
}

// Service is the shared inference layer: purpose-based model selection, ZDR
// enforcement, and usage-metadata recording. It is single-user, like the other
// feature services, holding the user id at construction.
type Service struct {
	store    store
	provider provider
	userID   string
}

// NewService constructs the AI foundation service.
func NewService(store store, provider provider, userID string) *Service {
	return &Service{store: store, provider: provider, userID: userID}
}

// Generate runs one completion for a purpose. It resolves the assigned model,
// verifies the model still satisfies the purpose (ZDR plus required
// capabilities), calls the provider under ZDR policy, and records non-content
// usage metadata for both success and failure. Prompt content and schema are
// supplied by the caller; this layer owns only the shared inference concerns.
func (s *Service) Generate(ctx context.Context, purpose, promptVersion string, messages []Message, schema *Schema) (Completion, error) {
	description, ok := lookupPurpose(purpose)
	if !ok {
		return Completion{}, fmt.Errorf("%w: unknown AI purpose %q", ErrInvalidInput, purpose)
	}
	modelID, err := s.store.AssignedModel(ctx, s.userID, purpose)
	if err != nil {
		return Completion{}, err
	}
	if err := s.verifyModel(ctx, modelID, description); err != nil {
		s.record(ctx, purpose, modelID, promptVersion, Usage{}, "failed")
		return Completion{}, err
	}
	completion, err := s.provider.Complete(ctx, modelID, messages, schema)
	if err != nil {
		s.record(ctx, purpose, modelID, promptVersion, Usage{}, "failed")
		return Completion{}, err
	}
	if err := s.store.RecordGeneration(ctx, s.generation(purpose, modelID, promptVersion, completion.Usage, "succeeded")); err != nil {
		return Completion{}, err
	}
	return completion, nil
}

func (s *Service) verifyModel(ctx context.Context, modelID string, purpose Purpose) error {
	models, err := s.provider.ListModels(ctx)
	if err != nil {
		return fmt.Errorf("verify model policy: %w", err)
	}
	for _, model := range models {
		if model.ID == modelID {
			if compatible(model, purpose) {
				return nil
			}
			return fmt.Errorf("%w: %s no longer has a ZDR endpoint with the required capabilities", ErrModelIncompatible, modelID)
		}
	}
	return fmt.Errorf("%w: %s is no longer available", ErrModelIncompatible, modelID)
}

// record persists usage metadata on a best-effort basis; a metadata write failure
// must not mask the underlying generation error being reported to the caller.
func (s *Service) record(ctx context.Context, purpose, modelID, promptVersion string, usage Usage, status string) {
	_ = s.store.RecordGeneration(ctx, s.generation(purpose, modelID, promptVersion, usage, status))
}

func (s *Service) generation(purpose, modelID, promptVersion string, usage Usage, status string) Generation {
	return Generation{
		UserID:               s.userID,
		Purpose:              purpose,
		ModelID:              modelID,
		ProviderGenerationID: strings.TrimSpace(usage.GenerationID),
		PromptVersion:        promptVersion,
		PromptTokens:         usage.PromptTokens,
		CompletionTokens:     usage.CompletionTokens,
		Status:               status,
	}
}

// PurposeSetting is one purpose's settings row: the models compatible with it and
// the current assignment (empty when none is chosen).
type PurposeSetting struct {
	Purpose         Purpose
	Models          []Model
	SelectedModelID string
}

// SettingsView returns per-purpose compatible models and current assignments for
// the /settings/ai page. It queries the provider once and filters per purpose.
func (s *Service) SettingsView(ctx context.Context) ([]PurposeSetting, error) {
	models, err := s.provider.ListModels(ctx)
	if err != nil {
		return nil, err
	}
	purposes := Purposes()
	settings := make([]PurposeSetting, 0, len(purposes))
	for _, purpose := range purposes {
		compatibleModels := make([]Model, 0, len(models))
		for _, model := range models {
			if compatible(model, purpose) {
				compatibleModels = append(compatibleModels, model)
			}
		}
		selected, err := s.store.AssignedModel(ctx, s.userID, purpose.Key)
		if err != nil && !errors.Is(err, ErrNoModelAssigned) {
			return nil, err
		}
		settings = append(settings, PurposeSetting{Purpose: purpose, Models: compatibleModels, SelectedModelID: selected})
	}
	return settings, nil
}

// Assign validates that a model is compatible with a purpose and stores it.
func (s *Service) Assign(ctx context.Context, purpose, modelID string) error {
	description, ok := lookupPurpose(purpose)
	if !ok {
		return fmt.Errorf("%w: unknown AI purpose %q", ErrInvalidInput, purpose)
	}
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return fmt.Errorf("%w: choose a model", ErrInvalidInput)
	}
	models, err := s.provider.ListModels(ctx)
	if err != nil {
		return err
	}
	for _, model := range models {
		if model.ID == modelID {
			if !compatible(model, description) {
				return fmt.Errorf("%w: choose a ZDR-compatible model with the required capabilities", ErrInvalidInput)
			}
			return s.store.SaveAssignment(ctx, s.userID, purpose, modelID)
		}
	}
	return fmt.Errorf("%w: choose a ZDR-compatible model", ErrInvalidInput)
}
