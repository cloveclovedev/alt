package nutrition

import "testing"

func TestCoachingContextOmitsTargetWhenAbsentAndCopiesEntries(t *testing.T) {
	summary := DailySummary{
		Date: date(2026, 8, 3),
		Entries: []Entry{
			{Name: "Eggs", MealType: MealBreakfast, CaloriesKcal: 140, ProteinG: 12},
			{Name: "Rice", MealType: MealLunch, CaloriesKcal: 300, ProteinG: 6},
		},
		Totals: Totals{CaloriesKcal: 440, ProteinG: 18},
		Target: nil,
	}
	ctx := coachingContextOf(summary)
	if ctx.Target != nil {
		t.Fatalf("target should be omitted when none applies, got %+v", ctx.Target)
	}
	if ctx.TotalCaloriesKcal != 440 || ctx.TotalProteinG != 18 {
		t.Fatalf("totals not carried: %+v", ctx)
	}
	if len(ctx.Entries) != 2 || ctx.Entries[0].Name != "Eggs" || ctx.Entries[1].MealType != "lunch" {
		t.Fatalf("entries not copied faithfully: %+v", ctx.Entries)
	}
}

func TestCoachingContextIncludesTargetWhenPresent(t *testing.T) {
	summary := DailySummary{
		Date:   date(2026, 8, 3),
		Totals: Totals{CaloriesKcal: 1800, ProteinG: 120},
		Target: &Target{CaloriesKcal: 2000, ProteinG: 140},
	}
	ctx := coachingContextOf(summary)
	if ctx.Target == nil || ctx.Target.CaloriesKcal != 2000 || ctx.Target.ProteinG != 140 {
		t.Fatalf("target should be included: %+v", ctx.Target)
	}
	if len(ctx.Entries) != 0 {
		t.Fatalf("entries should be empty, got %+v", ctx.Entries)
	}
}
