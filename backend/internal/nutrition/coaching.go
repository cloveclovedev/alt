package nutrition

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/cloveclovedev/alt/internal/ai"
)

const coachingPromptVersion = "nutrition-coaching-v1"

const coachingPrompt = `You are a nutrition coach. From one day's recorded intake and the applicable daily target, write a short evaluation of how the day went and a single concrete adjustment suggestion.

Work only from the data provided. Never invent foods, amounts, calories, protein, or a target that is not present. If no target is set, say targets are not configured and comment on the intake alone. If there are no entries, say nothing was logged.

Return two fields:
- evaluation_markdown: a brief plain-Markdown assessment of calories and protein versus the target (achievement), no more than a few sentences.
- suggestion_markdown: one actionable adjustment for the rest of the day or tomorrow, in plain Markdown.

Do not restate the raw numbers as a table; the app already shows them.`

// Coaching returns the cached coaching for a date, or nil when none is generated.
// The generation timestamp is presented in the user's local timezone.
func (s *Service) Coaching(ctx context.Context, date time.Time) (*Coaching, error) {
	coaching, err := s.store.LoadCoaching(ctx, s.userID, date)
	if err != nil || coaching == nil {
		return coaching, err
	}
	coaching.GeneratedAt = coaching.GeneratedAt.In(s.location)
	return coaching, nil
}

// GenerateCoaching produces on-demand coaching for a date from stored entries and
// the applicable target, and caches it (replacing any prior commentary for the
// day). It is read-only over stored data: the model receives only recorded facts
// and cannot create entries.
func (s *Service) GenerateCoaching(ctx context.Context, date time.Time) (Coaching, error) {
	if s.ai == nil {
		return Coaching{}, ErrAIUnavailable
	}
	summary, err := s.DailySummary(ctx, date)
	if err != nil {
		return Coaching{}, err
	}
	contextJSON, err := json.Marshal(coachingContextOf(summary))
	if err != nil {
		return Coaching{}, fmt.Errorf("encode coaching context: %w", err)
	}
	messages := []ai.Message{
		{Role: "system", Content: coachingPrompt},
		{Role: "user", Content: "Recorded nutrition for the day:\n" + string(contextJSON)},
	}
	completion, err := s.ai.Generate(ctx, ai.PurposeNutritionCoaching, coachingPromptVersion, messages, &ai.Schema{Name: "nutrition_coaching", Body: coachingSchema()})
	if err != nil {
		if errors.Is(err, ai.ErrNoModelAssigned) || errors.Is(err, ai.ErrModelIncompatible) {
			return Coaching{}, fmt.Errorf("%w: assign a ZDR model to nutrition coaching in AI settings", ErrAIUnavailable)
		}
		return Coaching{}, fmt.Errorf("generate nutrition coaching: %w", err)
	}
	var parsed struct {
		EvaluationMarkdown string `json:"evaluation_markdown"`
		SuggestionMarkdown string `json:"suggestion_markdown"`
	}
	if err := json.Unmarshal([]byte(completion.Content), &parsed); err != nil {
		return Coaching{}, fmt.Errorf("decode nutrition coaching: %w", err)
	}
	stored, err := s.store.UpsertCoaching(ctx, s.userID, date, Coaching{
		EvaluationMarkdown: parsed.EvaluationMarkdown,
		SuggestionMarkdown: parsed.SuggestionMarkdown,
		ModelID:            completion.Model,
		PromptVersion:      coachingPromptVersion,
	})
	if err != nil {
		return Coaching{}, err
	}
	stored.GeneratedAt = stored.GeneratedAt.In(s.location)
	return stored, nil
}

// coachingContext is the compact, provider-neutral view of a day handed to the
// model. It contains only recorded facts so the model cannot fabricate intake.
type coachingContext struct {
	Date              string          `json:"date"`
	Entries           []coachingEntry `json:"entries"`
	TotalCaloriesKcal int             `json:"total_calories_kcal"`
	TotalProteinG     float64         `json:"total_protein_g"`
	Target            *coachingTarget `json:"target,omitempty"`
}

type coachingEntry struct {
	Name         string  `json:"name"`
	MealType     string  `json:"meal_type"`
	CaloriesKcal int     `json:"calories_kcal"`
	ProteinG     float64 `json:"protein_g"`
}

type coachingTarget struct {
	CaloriesKcal int     `json:"calories_kcal"`
	ProteinG     float64 `json:"protein_g"`
}

func coachingContextOf(summary DailySummary) coachingContext {
	context := coachingContext{
		Date:              summary.Date.Format(time.DateOnly),
		Entries:           make([]coachingEntry, 0, len(summary.Entries)),
		TotalCaloriesKcal: summary.Totals.CaloriesKcal,
		TotalProteinG:     summary.Totals.ProteinG,
	}
	for _, entry := range summary.Entries {
		context.Entries = append(context.Entries, coachingEntry{
			Name:         entry.Name,
			MealType:     string(entry.MealType),
			CaloriesKcal: entry.CaloriesKcal,
			ProteinG:     entry.ProteinG,
		})
	}
	if summary.Target != nil {
		context.Target = &coachingTarget{CaloriesKcal: summary.Target.CaloriesKcal, ProteinG: summary.Target.ProteinG}
	}
	return context
}

func coachingSchema() map[string]any {
	return map[string]any{
		"type": "object", "additionalProperties": false,
		"properties": map[string]any{
			"evaluation_markdown": map[string]any{"type": "string"},
			"suggestion_markdown": map[string]any{"type": "string"},
		},
		"required": []string{"evaluation_markdown", "suggestion_markdown"},
	}
}
