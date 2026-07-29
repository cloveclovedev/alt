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

// DailyGatherer coordinates independent providers without exposing their DTOs to AI.
type DailyGatherer struct {
	store    *Store
	routines RoutineStatusSource
	github   *GitHubClient
	calendar CalendarPlanningReader
}

func NewDailyGatherer(store *Store, routines RoutineStatusSource, github *GitHubClient, calendar CalendarPlanningReader) *DailyGatherer {
	return &DailyGatherer{store: store, routines: routines, github: github, calendar: calendar}
}

func (g *DailyGatherer) GatherDailyPlanningContext(ctx context.Context, userID string, date time.Time, location *time.Location) DailyPlanningContext {
	value := DailyPlanningContext{CalendarStatus: SourceStatusUnavailable, GitHubStatus: SourceStatusUnavailable, RoutineStatus: SourceStatusUnavailable}
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
	return value
}

func safeSourceError(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("%s could not be loaded. Try again or continue without it.", sourceName(err))
}

func sourceName(err error) string { return "This source" }
