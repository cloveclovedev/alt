package nutrition

import "testing"

func TestNewAchievementPercent(t *testing.T) {
	withTarget := newAchievement(1750, 2000)
	if !withTarget.HasTarget || withTarget.Percent != 88 { // 87.5 rounds to 88
		t.Fatalf("achievement = %+v, want 88%% with target", withTarget)
	}
	none := newAchievement(500, 0)
	if none.HasTarget || none.Percent != 0 {
		t.Fatalf("no target should yield HasTarget=false, Percent=0, got %+v", none)
	}
}

func TestSummaryOfReflectsTargetPresence(t *testing.T) {
	withTarget := summaryOf(DailySummary{
		Date:   date(2026, 8, 3),
		Totals: Totals{CaloriesKcal: 1800, ProteinG: 120},
		Target: &Target{CaloriesKcal: 2000, ProteinG: 140},
	})
	if !withTarget.HasTarget || withTarget.Calories.Percent != 90 {
		t.Fatalf("calories percent = %+v, want 90 with target", withTarget.Calories)
	}
	noTarget := summaryOf(DailySummary{Date: date(2026, 8, 3), Totals: Totals{CaloriesKcal: 500}})
	if noTarget.HasTarget {
		t.Fatalf("summary should report no target, got %+v", noTarget)
	}
}

func TestGroupByMealOrdersAndDropsEmpty(t *testing.T) {
	entries := []Entry{
		{Name: "Curry", MealType: MealDinner},
		{Name: "Oats", MealType: MealBreakfast},
		{Name: "Nuts", MealType: MealSnack},
		{Name: "Eggs", MealType: MealBreakfast},
	}
	groups := groupByMeal(entries)
	// No lunch entries, so lunch is dropped; order follows MealTypes().
	if len(groups) != 3 {
		t.Fatalf("want 3 non-empty meal groups, got %d", len(groups))
	}
	if groups[0].MealType != MealBreakfast || len(groups[0].Entries) != 2 {
		t.Fatalf("breakfast group wrong: %+v", groups[0])
	}
	if groups[1].MealType != MealDinner || groups[2].MealType != MealSnack {
		t.Fatalf("meal order should follow MealTypes(): %+v", groups)
	}
}
