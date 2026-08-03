package ai

import (
	"context"
	"errors"
	"testing"
)

// fakeProvider is an in-memory provider for exercising selection and ZDR policy.
type fakeProvider struct {
	models      []Model
	completion  Completion
	completeErr error
	listErr     error
	calls       int
}

func (f *fakeProvider) Complete(context.Context, string, []Message, *Schema) (Completion, error) {
	f.calls++
	if f.completeErr != nil {
		return Completion{}, f.completeErr
	}
	return f.completion, nil
}

func (f *fakeProvider) ListModels(context.Context) ([]Model, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.models, nil
}

// fakeStore records assignments and generations in memory.
type fakeStore struct {
	assignments map[string]string
	generations []Generation
}

func newFakeStore() *fakeStore { return &fakeStore{assignments: map[string]string{}} }

func (s *fakeStore) AssignedModel(_ context.Context, userID, purpose string) (string, error) {
	modelID, ok := s.assignments[userID+"|"+purpose]
	if !ok {
		return "", ErrNoModelAssigned
	}
	return modelID, nil
}

func (s *fakeStore) SaveAssignment(_ context.Context, userID, purpose, modelID string) error {
	s.assignments[userID+"|"+purpose] = modelID
	return nil
}

func (s *fakeStore) RecordGeneration(_ context.Context, generation Generation) error {
	s.generations = append(s.generations, generation)
	return nil
}

func zdrTextModel(id string) Model {
	return Model{ID: id, Name: id, SupportsTextInput: true, SupportsStructuredOutput: true, HasZDREndpoint: true}
}

func zdrVisionModel(id string) Model {
	m := zdrTextModel(id)
	m.SupportsImageInput = true
	return m
}

func TestCompatibleEnforcesZDRAndCapabilities(t *testing.T) {
	planning := Purpose{Key: PurposeDailyPlanning}
	vision := Purpose{Key: PurposeNutritionLogging, RequiresImageInput: true}

	if !compatible(zdrTextModel("a"), planning) {
		t.Fatal("a ZDR structured text model should satisfy daily planning")
	}
	// No ZDR endpoint disqualifies the model regardless of other capabilities.
	noZDR := zdrTextModel("b")
	noZDR.HasZDREndpoint = false
	if compatible(noZDR, planning) {
		t.Fatal("a model without a ZDR endpoint must never be compatible")
	}
	// A text-only model cannot serve a vision purpose.
	if compatible(zdrTextModel("c"), vision) {
		t.Fatal("a non-vision model must not satisfy an image-input purpose")
	}
	if !compatible(zdrVisionModel("d"), vision) {
		t.Fatal("a ZDR vision model should satisfy an image-input purpose")
	}
}

func TestGenerateRecordsSuccess(t *testing.T) {
	fp := &fakeProvider{models: []Model{zdrTextModel("openai/model")}, completion: Completion{Content: "{}", Model: "openai/model", Usage: Usage{GenerationID: "gen-1", PromptTokens: 10, CompletionTokens: 5}}}
	fs := newFakeStore()
	fs.assignments["user-1|"+PurposeDailyPlanning] = "openai/model"
	service := NewService(fs, fp, "user-1")

	completion, err := service.Generate(context.Background(), PurposeDailyPlanning, "v1", []Message{{Role: "user", Content: "hi"}}, nil)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if completion.Content != "{}" {
		t.Fatalf("unexpected content %q", completion.Content)
	}
	if len(fs.generations) != 1 || fs.generations[0].Status != "succeeded" {
		t.Fatalf("expected one succeeded generation, got %+v", fs.generations)
	}
	if fs.generations[0].ProviderGenerationID != "gen-1" || fs.generations[0].PromptTokens != 10 {
		t.Fatalf("usage metadata not recorded: %+v", fs.generations[0])
	}
}

func TestGenerateRecordsFailureOnProviderError(t *testing.T) {
	fp := &fakeProvider{models: []Model{zdrTextModel("openai/model")}, completeErr: errors.New("boom")}
	fs := newFakeStore()
	fs.assignments["user-1|"+PurposeDailyPlanning] = "openai/model"
	service := NewService(fs, fp, "user-1")

	if _, err := service.Generate(context.Background(), PurposeDailyPlanning, "v1", nil, nil); err == nil {
		t.Fatal("expected provider error")
	}
	if len(fs.generations) != 1 || fs.generations[0].Status != "failed" {
		t.Fatalf("expected one failed generation, got %+v", fs.generations)
	}
}

func TestGenerateRejectsIncompatibleAssignedModel(t *testing.T) {
	// The assigned model lost its ZDR endpoint since assignment.
	stale := zdrTextModel("openai/model")
	stale.HasZDREndpoint = false
	fp := &fakeProvider{models: []Model{stale}}
	fs := newFakeStore()
	fs.assignments["user-1|"+PurposeDailyPlanning] = "openai/model"
	service := NewService(fs, fp, "user-1")

	_, err := service.Generate(context.Background(), PurposeDailyPlanning, "v1", nil, nil)
	if !errors.Is(err, ErrModelIncompatible) {
		t.Fatalf("expected ErrModelIncompatible, got %v", err)
	}
	if fp.calls != 0 {
		t.Fatal("provider Complete must not run for an incompatible model")
	}
	if len(fs.generations) != 1 || fs.generations[0].Status != "failed" {
		t.Fatalf("expected a failed generation record, got %+v", fs.generations)
	}
}

func TestGenerateWithoutAssignmentReturnsSentinel(t *testing.T) {
	fp := &fakeProvider{models: []Model{zdrTextModel("openai/model")}}
	service := NewService(newFakeStore(), fp, "user-1")

	_, err := service.Generate(context.Background(), PurposeDailyPlanning, "v1", nil, nil)
	if !errors.Is(err, ErrNoModelAssigned) {
		t.Fatalf("expected ErrNoModelAssigned, got %v", err)
	}
}

func TestSettingsViewFiltersModelsPerPurpose(t *testing.T) {
	fp := &fakeProvider{models: []Model{zdrTextModel("text/only"), zdrVisionModel("vision/model")}}
	service := NewService(newFakeStore(), fp, "user-1")

	view, err := service.SettingsView(context.Background())
	if err != nil {
		t.Fatalf("SettingsView error: %v", err)
	}
	byPurpose := map[string][]Model{}
	for _, setting := range view {
		byPurpose[setting.Purpose.Key] = setting.Models
	}
	if len(byPurpose[PurposeDailyPlanning]) != 2 {
		t.Fatalf("daily planning should offer both models, got %d", len(byPurpose[PurposeDailyPlanning]))
	}
	logging := byPurpose[PurposeNutritionLogging]
	if len(logging) != 1 || logging[0].ID != "vision/model" {
		t.Fatalf("nutrition logging should offer only the vision model, got %+v", logging)
	}
}

func TestAssignRejectsIncompatibleModel(t *testing.T) {
	fp := &fakeProvider{models: []Model{zdrTextModel("text/only")}}
	fs := newFakeStore()
	service := NewService(fs, fp, "user-1")

	// A text-only model cannot be assigned to the vision logging purpose.
	if err := service.Assign(context.Background(), PurposeNutritionLogging, "text/only"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
	if _, ok := fs.assignments["user-1|"+PurposeNutritionLogging]; ok {
		t.Fatal("incompatible model must not be stored")
	}
}
