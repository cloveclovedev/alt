// Package ai is a foundational feature (a peer of identity) that owns shared
// inference concerns: purpose-based model assignment, non-content usage metadata,
// ZDR enforcement, and the /settings/ai page. Any feature depends on it to run a
// completion without importing another feature or the provider adapter directly.
package ai

import (
	"errors"

	"github.com/cloveclovedev/alt/internal/platform/openrouter"
)

// Sentinel errors let callers distinguish an unassigned model (an expected,
// user-fixable state) from provider or input failures.
var (
	// ErrNoModelAssigned means the user has not chosen a model for the purpose.
	ErrNoModelAssigned = errors.New("no model assigned for AI purpose")
	// ErrModelIncompatible means the assigned model no longer satisfies the
	// purpose's capability or ZDR requirements.
	ErrModelIncompatible = errors.New("assigned model is incompatible with AI purpose")
	// ErrInvalidInput marks a rejected caller input.
	ErrInvalidInput = errors.New("invalid input")
)

// Purpose keys are the shared inference vocabulary. Features reference these
// constants instead of raw strings when requesting generation or assigning models.
const (
	PurposeDailyPlanning     = "daily_planning"
	PurposeNutritionLogging  = "nutrition_logging"
	PurposeNutritionCoaching = "nutrition_coaching"
)

// Neutral boundary types are aliased from the provider adapter so features depend
// only on internal/ai. These carry no OpenRouter response DTO; they are the
// contract shared across the boundary.
type (
	// Message is one provider-neutral chat message (optionally multimodal).
	Message = openrouter.Message
	// Image is one image part of a multimodal message.
	Image = openrouter.Image
	// Schema requests strict structured JSON output.
	Schema = openrouter.Schema
	// Completion is returned content plus usage metadata.
	Completion = openrouter.Completion
	// Usage is non-content generation metadata.
	Usage = openrouter.Usage
	// Model is a capability view of a selectable model.
	Model = openrouter.Model
)

// Purpose describes one inference use-case and the model capabilities it requires.
// Every purpose requires a live ZDR endpoint and structured-output support; some
// also require image input (vision).
type Purpose struct {
	Key                string
	Label              string
	RequiresImageInput bool
}

// Purposes returns the registered purposes in display order. This is the canonical
// registry shown on /settings/ai; features reference the purpose keys above.
func Purposes() []Purpose {
	return []Purpose{
		{Key: PurposeDailyPlanning, Label: "Daily planning"},
		{Key: PurposeNutritionLogging, Label: "Nutrition logging (photo / text)", RequiresImageInput: true},
		{Key: PurposeNutritionCoaching, Label: "Nutrition coaching"},
	}
}

func lookupPurpose(key string) (Purpose, bool) {
	for _, purpose := range Purposes() {
		if purpose.Key == key {
			return purpose, true
		}
	}
	return Purpose{}, false
}

// compatible reports whether a model satisfies a purpose's requirements. Every
// purpose requires a live ZDR endpoint plus text and structured-output support;
// vision purposes additionally require image input. There is no fallback to a
// non-compliant model.
func compatible(model Model, purpose Purpose) bool {
	if !model.HasZDREndpoint || !model.SupportsStructuredOutput || !model.SupportsTextInput {
		return false
	}
	if purpose.RequiresImageInput && !model.SupportsImageInput {
		return false
	}
	return true
}
