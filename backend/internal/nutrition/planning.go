package nutrition

import (
	"context"
	"time"
)

// PlanningFacts is compact nutrition context for daily planning. Its fields match
// planning.NutritionFacts exactly (names, types, and order) so the composition
// root converts one to the other without either feature importing the other.
// It carries no individual entries — only aggregates — so a consumer cannot
// reconstruct or fabricate the intake log from it.
type PlanningFacts struct {
	YesterdayCaloriesKcal       int
	YesterdayProteinG           float64
	YesterdayTargetCaloriesKcal int
	YesterdayTargetProteinG     float64
	TodayCaloriesKcal           int
	TodayProteinG               float64
	TrailingDays                int
	TrailingAvgCaloriesKcal     int
	TrailingAvgProteinG         float64
	TargetCaloriesKcal          int
	TargetProteinG              float64
}

// PlanningFacts returns compact nutrition context for a plan date: yesterday's
// totals and target, today's intake so far, and a trailing 7-day average. It
// returns nil when there is nothing to report (no targets and no entries in the
// window), so daily planning can treat nutrition as unavailable and proceed.
func (s *Service) PlanningFacts(ctx context.Context, date time.Time) (*PlanningFacts, error) {
	yesterday, err := s.DailySummary(ctx, date.AddDate(0, 0, -1))
	if err != nil {
		return nil, err
	}
	today, err := s.DailySummary(ctx, date)
	if err != nil {
		return nil, err
	}
	trailing, err := s.TrailingSummary(ctx, date, 7)
	if err != nil {
		return nil, err
	}

	hasEntries := len(yesterday.Entries) > 0 || len(today.Entries) > 0
	if !hasEntries {
		for _, day := range trailing.Days {
			if day.Totals.CaloriesKcal > 0 || day.Totals.ProteinG > 0 {
				hasEntries = true
				break
			}
		}
	}
	if !hasEntries && today.Target == nil {
		return nil, nil
	}

	facts := &PlanningFacts{
		YesterdayCaloriesKcal:   yesterday.Totals.CaloriesKcal,
		YesterdayProteinG:       yesterday.Totals.ProteinG,
		TodayCaloriesKcal:       today.Totals.CaloriesKcal,
		TodayProteinG:           today.Totals.ProteinG,
		TrailingDays:            len(trailing.Days),
		TrailingAvgCaloriesKcal: trailing.Average.CaloriesKcal,
		TrailingAvgProteinG:     trailing.Average.ProteinG,
	}
	if yesterday.Target != nil {
		facts.YesterdayTargetCaloriesKcal = yesterday.Target.CaloriesKcal
		facts.YesterdayTargetProteinG = yesterday.Target.ProteinG
	}
	if today.Target != nil {
		facts.TargetCaloriesKcal = today.Target.CaloriesKcal
		facts.TargetProteinG = today.Target.ProteinG
	}
	return facts, nil
}
