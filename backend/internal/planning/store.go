package planning

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store persists plans and their revisions in PostgreSQL.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore constructs a PostgreSQL plan store.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Load returns the latest revision for one planning period.
func (s *Store) Load(
	ctx context.Context,
	userID string,
	kind Kind,
	periodStart time.Time,
	location *time.Location,
) (View, error) {
	view := View{
		Kind:        kind,
		PeriodStart: periodStart,
		Timezone:    location.String(),
	}

	var plan Plan
	err := s.pool.QueryRow(ctx, `
		SELECT p.id::text,
		       r.id::text,
		       p.kind::text,
		       p.period_start,
		       r.revision,
		       r.summary_markdown,
		       r.content_markdown,
		       COALESCE(r.notes_markdown, ''),
		       r.created_at
		FROM plans AS p
		JOIN plan_revisions AS r ON r.plan_id = p.id
		WHERE p.user_id = $1
		  AND p.kind = $2
		  AND p.period_start = $3
		ORDER BY r.revision DESC
		LIMIT 1
	`, userID, kind, periodStart.Format(time.DateOnly)).Scan(
		&plan.ID,
		&plan.RevisionID,
		&plan.Kind,
		&plan.PeriodStart,
		&plan.Revision,
		&plan.SummaryMarkdown,
		&plan.ContentMarkdown,
		&plan.NotesMarkdown,
		&plan.CreatedAt,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return View{}, fmt.Errorf("load latest plan revision: %w", err)
	}
	if err == nil {
		plan.CreatedAt = plan.CreatedAt.In(location)
		components, err := s.loadRevisionComponents(ctx, plan.RevisionID, location)
		if err != nil {
			return View{}, err
		}
		plan.Components = components
		view.Plan = &plan
	}
	return view, nil
}

// loadRevisionComponents reads the structured selections stored with a revision
// so a confirmed plan can render them as lists beside its prose.
func (s *Store) loadRevisionComponents(ctx context.Context, revisionID string, location *time.Location) (PlanComponents, error) {
	var components PlanComponents

	issueRows, err := s.pool.Query(ctx, `
		SELECT repository_owner, repository_name, issue_number, title, html_url
		FROM plan_revision_github_issues
		WHERE plan_revision_id = $1
		ORDER BY position
	`, revisionID)
	if err != nil {
		return PlanComponents{}, fmt.Errorf("load revision GitHub issues: %w", err)
	}
	for issueRows.Next() {
		var issue PlanGitHubIssue
		if err := issueRows.Scan(&issue.RepositoryOwner, &issue.RepositoryName, &issue.Number, &issue.Title, &issue.HTMLURL); err != nil {
			issueRows.Close()
			return PlanComponents{}, fmt.Errorf("scan revision GitHub issue: %w", err)
		}
		components.GitHubIssues = append(components.GitHubIssues, issue)
	}
	issueRows.Close()
	if err := issueRows.Err(); err != nil {
		return PlanComponents{}, fmt.Errorf("iterate revision GitHub issues: %w", err)
	}

	routineRows, err := s.pool.Query(ctx, `
		SELECT r.id::text, r.name, c.name,
		       EXISTS(
		         SELECT 1 FROM routine_events AS e
		         WHERE e.routine_id = r.id AND e.completed_on = $2
		       )
		FROM plan_revision_routines AS pr
		JOIN routines AS r ON r.id = pr.routine_id
		JOIN routine_categories AS c ON c.id = r.category_id
		WHERE pr.plan_revision_id = $1
		ORDER BY pr.position
	`, revisionID, localToday(location).Format(time.DateOnly))
	if err != nil {
		return PlanComponents{}, fmt.Errorf("load revision routines: %w", err)
	}
	for routineRows.Next() {
		var routine PlanRoutine
		if err := routineRows.Scan(&routine.RoutineID, &routine.Name, &routine.CategoryName, &routine.CompletedToday); err != nil {
			routineRows.Close()
			return PlanComponents{}, fmt.Errorf("scan revision routine: %w", err)
		}
		components.Routines = append(components.Routines, routine)
	}
	routineRows.Close()
	if err := routineRows.Err(); err != nil {
		return PlanComponents{}, fmt.Errorf("iterate revision routines: %w", err)
	}

	actionRows, err := s.pool.Query(ctx, `
		SELECT title, note
		FROM plan_revision_action_items
		WHERE plan_revision_id = $1
		ORDER BY position
	`, revisionID)
	if err != nil {
		return PlanComponents{}, fmt.Errorf("load revision action items: %w", err)
	}
	for actionRows.Next() {
		var item ActionItem
		if err := actionRows.Scan(&item.Title, &item.Note); err != nil {
			actionRows.Close()
			return PlanComponents{}, fmt.Errorf("scan revision action item: %w", err)
		}
		components.ActionItems = append(components.ActionItems, item)
	}
	actionRows.Close()
	if err := actionRows.Err(); err != nil {
		return PlanComponents{}, fmt.Errorf("iterate revision action items: %w", err)
	}

	eventRows, err := s.pool.Query(ctx, `
		SELECT title, all_day, starts_at, ends_at, role::text, COALESCE(html_url, '')
		FROM plan_revision_calendar_events
		WHERE plan_revision_id = $1
		ORDER BY position
	`, revisionID)
	if err != nil {
		return PlanComponents{}, fmt.Errorf("load revision calendar events: %w", err)
	}
	for eventRows.Next() {
		var (
			event    PlanCalendarEvent
			allDay   bool
			startsAt *time.Time
			endsAt   *time.Time
			role     string
		)
		if err := eventRows.Scan(&event.Title, &allDay, &startsAt, &endsAt, &role, &event.HTMLURL); err != nil {
			eventRows.Close()
			return PlanComponents{}, fmt.Errorf("scan revision calendar event: %w", err)
		}
		event.Role = CalendarRole(role)
		event.TimeLabel = calendarTimeLabel(allDay, startsAt, endsAt, location)
		components.CalendarEvents = append(components.CalendarEvents, event)
	}
	eventRows.Close()
	if err := eventRows.Err(); err != nil {
		return PlanComponents{}, fmt.Errorf("iterate revision calendar events: %w", err)
	}

	return components, nil
}

// localToday returns the current calendar date in the given location.
func localToday(location *time.Location) time.Time {
	now := time.Now().In(location)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
}
