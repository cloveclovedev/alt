package planning

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeNutritionSource struct {
	facts *NutritionFacts
	err   error
}

func (f fakeNutritionSource) NutritionFacts(context.Context, time.Time) (*NutritionFacts, error) {
	return f.facts, f.err
}

func TestGatherNutritionIsNonBlocking(t *testing.T) {
	date := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	// Only the nutrition source is wired; the others are nil and skipped, so the
	// store is never touched.
	gather := func(src NutritionContextSource) DailyPlanningContext {
		return NewDailyGatherer(nil, nil, nil, nil, src).
			GatherDailyPlanningContext(context.Background(), "user-1", date, time.UTC)
	}

	loaded := gather(fakeNutritionSource{facts: &NutritionFacts{TodayCaloriesKcal: 500, TargetCaloriesKcal: 2000}})
	if loaded.NutritionStatus != SourceStatusLoaded || loaded.Nutrition == nil || loaded.Nutrition.TodayCaloriesKcal != 500 {
		t.Fatalf("loaded nutrition not carried: status=%s facts=%+v", loaded.NutritionStatus, loaded.Nutrition)
	}

	unavailable := gather(fakeNutritionSource{facts: nil})
	if unavailable.NutritionStatus != SourceStatusUnavailable || unavailable.Nutrition != nil {
		t.Fatalf("nil facts should be unavailable, got status=%s facts=%+v", unavailable.NutritionStatus, unavailable.Nutrition)
	}

	failed := gather(fakeNutritionSource{err: errors.New("boom")})
	if failed.NutritionStatus != SourceStatusFailed {
		t.Fatalf("error should mark nutrition failed, got %s", failed.NutritionStatus)
	}
	// The key contract: a nutrition failure must not block the plan.
	if hasFailedSource(failed) {
		t.Fatal("nutrition failure must not count as a blocking failed source")
	}
}
