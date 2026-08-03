package nutrition

import (
	"math"
	"testing"
	"time"
)

func TestBuildCandidatesPrefersCatalogValues(t *testing.T) {
	catalog := []CatalogItem{
		{ID: "cat-1", Name: "Protein shake", CaloriesKcal: 120, ProteinG: 24.5},
	}
	raws := []rawCandidate{
		{Name: "protein shake", CaloriesKcal: 200, ProteinG: 10, MealType: "snack"}, // matches catalog (case-insensitive)
		{Name: "Banana", CaloriesKcal: 100, ProteinG: 1.3, MealType: "breakfast"},   // no match
		{Name: "  ", CaloriesKcal: 50, ProteinG: 5, MealType: "lunch"},              // dropped (blank)
		{Name: "Rice", CaloriesKcal: -5, ProteinG: -2, MealType: "brunch"},          // sanitized + fallback meal
	}
	got := buildCandidates(raws, catalog, MealDinner, SourceAIPhoto)
	if len(got) != 3 {
		t.Fatalf("want 3 candidates, got %d: %+v", len(got), got)
	}

	shake := got[0]
	if shake.CaloriesKcal != 120 || shake.ProteinG != 24.5 {
		t.Fatalf("catalog match should override estimate: %+v", shake)
	}
	if shake.Name != "Protein shake" || shake.CatalogID == nil || *shake.CatalogID != "cat-1" {
		t.Fatalf("catalog match should set canonical name and link: %+v", shake)
	}
	if shake.Source != SourceAIPhoto {
		t.Fatalf("source should be preserved: %+v", shake)
	}

	banana := got[1]
	if banana.CatalogID != nil || banana.CaloriesKcal != 100 {
		t.Fatalf("non-match should keep the estimate and no link: %+v", banana)
	}

	rice := got[2]
	if rice.CaloriesKcal != 0 || rice.ProteinG != 0 {
		t.Fatalf("negative metrics should be clamped to zero: %+v", rice)
	}
	if rice.MealType != MealDinner {
		t.Fatalf("invalid meal type should fall back: %+v", rice)
	}
}

func TestSanitizeProteinRejectsNonFinite(t *testing.T) {
	cases := map[string]float64{"nan": math.NaN(), "inf": math.Inf(1), "negative": -3}
	for name, value := range cases {
		if got := sanitizeProtein(value); got != 0 {
			t.Fatalf("%s: sanitizeProtein(%v) = %v, want 0", name, value, got)
		}
	}
	if got := sanitizeProtein(maxProtein + 100); got != maxProtein {
		t.Fatalf("over-large protein should clamp to %v, got %v", maxProtein, got)
	}
	if got := sanitizeProtein(24.5); got != 24.5 {
		t.Fatalf("valid protein should pass through, got %v", got)
	}
}

func TestGuessMealTypeByHour(t *testing.T) {
	at := func(hour int) time.Time { return time.Date(2026, 8, 3, hour, 0, 0, 0, time.UTC) }
	cases := map[int]MealType{7: MealBreakfast, 12: MealLunch, 19: MealDinner, 23: MealSnack}
	for hour, want := range cases {
		if got := guessMealType(at(hour)); got != want {
			t.Fatalf("hour %d: guessMealType = %q, want %q", hour, got, want)
		}
	}
}
