package planning

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cloveclovedev/alt/internal/routine"
)

// RoutineStatusSource is the routine feature boundary consumed by daily planning.
type RoutineStatusSource interface {
	Statuses(context.Context) ([]routine.Status, error)
}

// CalendarPlanningReader fetches already-normalized events for enabled sources.
type CalendarPlanningReader interface {
	PlanningEvents(context.Context, string, time.Time, *time.Location) ([]CalendarEvent, error)
}

// NutritionContextSource is the nutrition feature boundary consumed by daily
// planning. It returns compact, provider-neutral facts for a plan date, or nil
// when there is no nutrition data to report. Planning declares this port and
// never imports the nutrition package; the composition root supplies an adapter.
type NutritionContextSource interface {
	NutritionFacts(context.Context, time.Time) (*NutritionFacts, error)
}

// DailyGatherer coordinates independent providers without exposing their DTOs to AI.
type DailyGatherer struct {
	store     *Store
	routines  RoutineStatusSource
	github    *GitHubClient
	calendar  CalendarPlanningReader
	nutrition NutritionContextSource
}

func NewDailyGatherer(store *Store, routines RoutineStatusSource, github *GitHubClient, calendar CalendarPlanningReader, nutrition NutritionContextSource) *DailyGatherer {
	return &DailyGatherer{store: store, routines: routines, github: github, calendar: calendar, nutrition: nutrition}
}

func (g *DailyGatherer) GatherDailyPlanningContext(ctx context.Context, userID string, date time.Time, location *time.Location) DailyPlanningContext {
	value := DailyPlanningContext{CalendarStatus: SourceStatusUnavailable, GitHubStatus: SourceStatusUnavailable, RoutineStatus: SourceStatusUnavailable, NutritionStatus: SourceStatusUnavailable}
	if g.calendar != nil {
		events, err := g.calendar.PlanningEvents(ctx, userID, date, location)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				value.CalendarStatus = SourceStatusUnavailable
			} else {
				value.CalendarStatus, value.CalendarError = SourceStatusFailed, safeSourceError(err)
			}
		} else {
			value.Calendar, value.CalendarStatus = events, SourceStatusLoaded
		}
	}
	if g.github != nil {
		repositories, err := g.store.ListGitHubRepositories(ctx, userID)
		if err != nil {
			value.GitHubStatus, value.GitHubError = SourceStatusFailed, safeSourceError(err)
		} else {
			enabled := false
			for _, repository := range repositories {
				if !repository.Enabled {
					continue
				}
				enabled = true
				issues, issueErr := g.github.ListOpenIssues(ctx, repository)
				if issueErr != nil {
					value.GitHubStatus, value.GitHubError = SourceStatusFailed, safeSourceError(issueErr)
					break
				}
				value.GitHub = append(value.GitHub, issues...)
			}
			if value.GitHubStatus != SourceStatusFailed {
				if enabled {
					value.GitHubStatus = SourceStatusLoaded
				} else {
					value.GitHubStatus = SourceStatusUnavailable
				}
			}
		}
	}
	if g.routines != nil {
		statuses, err := g.routines.Statuses(ctx)
		if err != nil {
			value.RoutineStatus, value.RoutineError = SourceStatusFailed, safeSourceError(err)
		} else {
			for _, status := range statuses {
				value.Routines = append(value.Routines, RoutineCandidate{RoutineID: status.RoutineID, Name: status.Name, CategoryName: status.CategoryName, State: string(status.State), DueOn: status.DueOn, NextAvailableOn: status.NextAvailableOn, Notes: status.Notes})
			}
			value.RoutineStatus = SourceStatusLoaded
		}
	}
	// Nutrition is context, not a blocking source: a failure or absence records a
	// status and degrades gracefully rather than blocking the plan (unlike the
	// sources above, it is excluded from hasFailedSource).
	if g.nutrition != nil {
		facts, err := g.nutrition.NutritionFacts(ctx, date)
		switch {
		case err != nil:
			value.NutritionStatus, value.NutritionError = SourceStatusFailed, safeSourceError(err)
		case facts == nil:
			value.NutritionStatus = SourceStatusUnavailable
		default:
			value.Nutrition, value.NutritionStatus = facts, SourceStatusLoaded
		}
	}
	return value
}

func safeSourceError(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("%s could not be loaded. Try again or continue without it.", sourceName(err))
}

func sourceName(err error) string { return "This source" }
