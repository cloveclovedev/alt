package nutrition

import (
	"errors"
	"math"
	"testing"
	"time"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	service, err := NewService(nil, "user-1", "Asia/Tokyo", nil, nil, nil)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return service
}

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func TestApplicableTargetForDerivesGreatestNotAfterDay(t *testing.T) {
	// Targets ordered newest-effective first, as the store returns them, including
	// a future-dated target that must not take effect before its date.
	targets := []Target{
		{ID: "future", EffectiveOn: date(2026, 8, 10)},
		{ID: "current", EffectiveOn: date(2026, 8, 1)},
		{ID: "old", EffectiveOn: date(2026, 7, 1)},
	}
	cases := []struct {
		day  time.Time
		want string // target ID, or "" for none
	}{
		{date(2026, 6, 30), ""},       // before the earliest target
		{date(2026, 7, 1), "old"},     // exactly on an effective date
		{date(2026, 8, 5), "current"}, // future target not yet in effect
		{date(2026, 8, 10), "future"}, // on the future target's effective date
		{date(2026, 8, 20), "future"}, // after the latest target
	}
	for _, tc := range cases {
		got := applicableTargetFor(targets, tc.day)
		if tc.want == "" {
			if got != nil {
				t.Fatalf("day %s: want no target, got %s", tc.day.Format(time.DateOnly), got.ID)
			}
			continue
		}
		if got == nil || got.ID != tc.want {
			t.Fatalf("day %s: want %s, got %v", tc.day.Format(time.DateOnly), tc.want, got)
		}
	}
}

func TestSumEntriesAddsCaloriesAndProtein(t *testing.T) {
	entries := []Entry{
		{CaloriesKcal: 500, ProteinG: 30.5},
		{CaloriesKcal: 250, ProteinG: 12.0},
	}
	totals := sumEntries(entries)
	if totals.CaloriesKcal != 750 {
		t.Fatalf("calories = %d, want 750", totals.CaloriesKcal)
	}
	if totals.ProteinG != 42.5 {
		t.Fatalf("protein = %v, want 42.5", totals.ProteinG)
	}
}

func TestValidateCatalogInputRejectsBadValues(t *testing.T) {
	service := newTestService(t)
	cases := map[string]CatalogInput{
		"blank name":        {Name: "  ", CaloriesKcal: 100, ProteinG: 5},
		"negative calories": {Name: "Egg", CaloriesKcal: -1, ProteinG: 5},
		"negative protein":  {Name: "Egg", CaloriesKcal: 100, ProteinG: -0.1},
		"invalid kind":      {Name: "Egg", CaloriesKcal: 100, ProteinG: 5, Kind: CatalogKind("drink")},
		"protein too large": {Name: "Egg", CaloriesKcal: 100, ProteinG: 1_000_000},
		"protein NaN":       {Name: "Egg", CaloriesKcal: 100, ProteinG: math.NaN()},
		"protein Inf":       {Name: "Egg", CaloriesKcal: 100, ProteinG: math.Inf(1)},
	}
	for name, input := range cases {
		if _, err := service.validateCatalogInput(input); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("%s: expected ErrInvalidInput, got %v", name, err)
		}
	}
}

func TestValidateCatalogInputDefaultsKindAndSource(t *testing.T) {
	service := newTestService(t)
	got, err := service.validateCatalogInput(CatalogInput{Name: "  Chicken  ", CaloriesKcal: 200, ProteinG: 40})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "Chicken" {
		t.Fatalf("name = %q, want trimmed Chicken", got.Name)
	}
	if got.Kind != KindFood {
		t.Fatalf("kind = %q, want default food", got.Kind)
	}
	if got.Source != "manual" {
		t.Fatalf("source = %q, want default manual", got.Source)
	}
}

func TestValidateEntryInputRejectsBadEnums(t *testing.T) {
	service := newTestService(t)
	base := EntryInput{LoggedDate: date(2026, 8, 3), MealType: MealBreakfast, Name: "Oats", CaloriesKcal: 300, ProteinG: 10, Source: SourceManual}

	bad := base
	bad.MealType = MealType("brunch")
	if _, err := service.validateEntryInput(t.Context(), bad); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("invalid meal type: expected ErrInvalidInput, got %v", err)
	}

	bad = base
	bad.Source = EntrySource("import")
	if _, err := service.validateEntryInput(t.Context(), bad); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("invalid source: expected ErrInvalidInput, got %v", err)
	}

	bad = base
	bad.LoggedDate = time.Time{}
	if _, err := service.validateEntryInput(t.Context(), bad); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("zero date: expected ErrInvalidInput, got %v", err)
	}
}
