package nutrition

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/cloveclovedev/alt/internal/ai"
)

// ErrAIUnavailable means AI logging was requested but no inference model is
// configured for the nutrition_logging purpose (or the AI service is absent).
var ErrAIUnavailable = errors.New("nutrition AI logging is unavailable")

// ErrPhotoStorageUnavailable means photo logging was requested without object
// storage configured.
var ErrPhotoStorageUnavailable = errors.New("nutrition photo storage is unavailable")

const loggingPromptVersion = "nutrition-logging-v1"

// Parser is the inference boundary for AI-assisted logging. *ai.Service satisfies
// it. Nutrition builds the prompt and schema; internal/ai enforces ZDR and the
// vision-capable model policy for the nutrition_logging purpose.
type Parser interface {
	Generate(ctx context.Context, purpose, promptVersion string, messages []ai.Message, schema *ai.Schema) (ai.Completion, error)
}

// PhotoStore is the transient object storage a meal photo passes through. The
// upload is stored, extracted, and then removed; only structured entries are kept.
type PhotoStore interface {
	Put(ctx context.Context, key, contentType string, body io.Reader) error
	Delete(ctx context.Context, key string) error
}

// ParseText turns a free-text description into candidate entries. Nothing is
// stored: the caller previews and confirms candidates, which are then written as
// entries.
func (s *Service) ParseText(ctx context.Context, text string) ([]Candidate, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("%w: describe what you ate", ErrInvalidInput)
	}
	if len([]rune(text)) > 5_000 {
		return nil, fmt.Errorf("%w: description is too long", ErrInvalidInput)
	}
	if s.ai == nil {
		return nil, ErrAIUnavailable
	}
	messages := []ai.Message{
		{Role: "system", Content: s.loggingPrompt()},
		{Role: "user", Content: text},
	}
	return s.parseCandidates(ctx, SourceAIText, messages)
}

// ParsePhoto stores a meal photo transiently, extracts candidate entries from it
// with vision, and removes the photo. The image bytes are also sent inline
// (base64) so extraction works regardless of whether the provider can reach the
// object store. Nothing is stored as an entry until the user confirms.
func (s *Service) ParsePhoto(ctx context.Context, image []byte, contentType string) ([]Candidate, error) {
	if len(image) == 0 {
		return nil, fmt.Errorf("%w: attach a photo", ErrInvalidInput)
	}
	if s.ai == nil {
		return nil, ErrAIUnavailable
	}
	if s.photos == nil {
		return nil, ErrPhotoStorageUnavailable
	}
	if contentType == "" {
		contentType = "image/jpeg"
	}
	key := fmt.Sprintf("nutrition/uploads/%s/%d", s.userID, s.now().UnixNano())
	if err := s.photos.Put(ctx, key, contentType, bytes.NewReader(image)); err != nil {
		return nil, fmt.Errorf("stage nutrition photo: %w", err)
	}
	// The photo is transient: remove it after extraction whatever the outcome.
	// WithoutCancel so cleanup still runs if the request context is cancelled.
	defer func() { _ = s.photos.Delete(context.WithoutCancel(ctx), key) }()

	dataURL := "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(image)
	messages := []ai.Message{
		{Role: "system", Content: s.loggingPrompt()},
		{Role: "user", Content: "Extract the food and drink items visible in this meal photo.", Images: []ai.Image{{URL: dataURL}}},
	}
	return s.parseCandidates(ctx, SourceAIPhoto, messages)
}

func (s *Service) parseCandidates(ctx context.Context, source EntrySource, messages []ai.Message) ([]Candidate, error) {
	completion, err := s.ai.Generate(ctx, ai.PurposeNutritionLogging, loggingPromptVersion, messages, &ai.Schema{Name: "nutrition_candidates", Body: candidateSchema()})
	if err != nil {
		if errors.Is(err, ai.ErrNoModelAssigned) || errors.Is(err, ai.ErrModelIncompatible) {
			return nil, fmt.Errorf("%w: assign a vision-capable ZDR model to nutrition logging in AI settings", ErrAIUnavailable)
		}
		return nil, fmt.Errorf("parse nutrition input: %w", err)
	}
	var parsed struct {
		Candidates []rawCandidate `json:"candidates"`
	}
	if err := json.Unmarshal([]byte(completion.Content), &parsed); err != nil {
		return nil, fmt.Errorf("decode nutrition candidates: %w", err)
	}

	catalog, err := s.store.ListCatalog(ctx, s.userID)
	if err != nil {
		return nil, err
	}
	return buildCandidates(parsed.Candidates, catalog, guessMealType(s.now().In(s.location)), source), nil
}

// rawCandidate mirrors one item of the structured model response.
type rawCandidate struct {
	Name         string  `json:"name"`
	CaloriesKcal int     `json:"calories_kcal"`
	ProteinG     float64 `json:"protein_g"`
	MealType     string  `json:"meal_type"`
}

// buildCandidates normalizes raw model output into candidate entries: it drops
// nameless items, sanitizes metrics, falls back to fallbackMeal for an invalid
// meal type, and prefers a matching catalog item's stored values (and link) over
// the estimate. It is pure so the matching logic is testable without I/O.
func buildCandidates(raws []rawCandidate, catalog []CatalogItem, fallbackMeal MealType, source EntrySource) []Candidate {
	byName := make(map[string]CatalogItem, len(catalog))
	for _, item := range catalog {
		byName[strings.ToLower(strings.TrimSpace(item.Name))] = item
	}
	candidates := make([]Candidate, 0, len(raws))
	for _, raw := range raws {
		name := strings.TrimSpace(raw.Name)
		if name == "" {
			continue
		}
		mealType := MealType(strings.TrimSpace(raw.MealType))
		if !ValidMealType(mealType) {
			mealType = fallbackMeal
		}
		candidate := Candidate{
			Name:         name,
			CaloriesKcal: maxInt(raw.CaloriesKcal, 0),
			ProteinG:     sanitizeProtein(raw.ProteinG),
			MealType:     mealType,
			Source:       source,
		}
		if item, ok := byName[strings.ToLower(name)]; ok {
			id := item.ID
			candidate.Name = item.Name
			candidate.CaloriesKcal = item.CaloriesKcal
			candidate.ProteinG = item.ProteinG
			candidate.CatalogID = &id
		}
		candidates = append(candidates, candidate)
	}
	return candidates
}

func (s *Service) loggingPrompt() string {
	now := s.now().In(s.location)
	return fmt.Sprintf(`You extract nutrition entries from a user's meal photo or free-text description.

Return a list of candidate entries. For each item you can identify, provide:
- name: a short food or drink name;
- calories_kcal: your best integer estimate of calories for the portion shown or described;
- protein_g: your best estimate of protein in grams;
- meal_type: one of breakfast, lunch, dinner, snack.

Guess meal_type from any time-of-day cues in the input; otherwise use the current local time, which is %s (%s). Only extract items that are actually present in the photo or text. Do not invent items, and do not add totals or commentary — return only the structured candidates. If nothing edible is present, return an empty list.`,
		now.Format("15:04"), now.Format("Monday"))
}

func candidateSchema() map[string]any {
	return map[string]any{
		"type": "object", "additionalProperties": false,
		"properties": map[string]any{
			"candidates": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object", "additionalProperties": false,
					"properties": map[string]any{
						"name":          map[string]any{"type": "string"},
						"calories_kcal": map[string]any{"type": "integer"},
						"protein_g":     map[string]any{"type": "number"},
						"meal_type":     map[string]any{"type": "string", "enum": []string{"breakfast", "lunch", "dinner", "snack"}},
					},
					"required": []string{"name", "calories_kcal", "protein_g", "meal_type"},
				},
			},
		},
		"required": []string{"candidates"},
	}
}

func guessMealType(now time.Time) MealType {
	switch hour := now.Hour(); {
	case hour < 10:
		return MealBreakfast
	case hour < 15:
		return MealLunch
	case hour < 21:
		return MealDinner
	default:
		return MealSnack
	}
}

func sanitizeProtein(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
		return 0
	}
	if value > maxProtein {
		return maxProtein
	}
	return value
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
