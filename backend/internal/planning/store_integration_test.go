package planning

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestStorePlanningFlow(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	var userID string
	err = pool.QueryRow(ctx, `
		INSERT INTO users DEFAULT VALUES
		RETURNING id::text
	`).Scan(&userID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO profiles (user_id, display_name, timezone)
		VALUES ($1, 'Daily plan integration test', 'Asia/Tokyo')
	`, userID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID); err != nil {
			t.Errorf("delete integration user: %v", err)
		}
	})

	store := NewStore(pool)
	location, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatal(err)
	}
	date := time.Date(2026, 7, 27, 9, 0, 0, 0, location)
	dateText := date.Format(time.DateOnly)

	var planID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO plans (user_id, kind, period_start)
		VALUES ($1, 'daily', $2)
		RETURNING id::text
	`, userID, dateText).Scan(&planID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO plan_revisions (plan_id, revision, summary_markdown, content_markdown)
		VALUES ($1, 1, 'First summary', '# First plan')
	`, planID); err != nil {
		t.Fatal(err)
	}
	var revisionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO plan_revisions (plan_id, revision, summary_markdown, content_markdown, notes_markdown)
		VALUES ($1, 2, 'Second summary', '# Second plan', 'Second notes')
		RETURNING id::text
	`, planID).Scan(&revisionID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO plan_revision_github_issues (plan_revision_id, repository_owner, repository_name, issue_number, title, html_url, position)
		VALUES ($1, 'cloveclovedev', 'alt', 61, 'Restructure daily planning UI', 'https://github.com/cloveclovedev/alt/issues/61', 0)
	`, revisionID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO plan_revision_action_items (plan_revision_id, title, note, position)
		VALUES ($1, 'Draft the release notes', 'Cover the UI changes', 0)
	`, revisionID); err != nil {
		t.Fatal(err)
	}

	var categoryID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO routine_categories (user_id, name, position)
		VALUES ($1, 'Health', 0)
		RETURNING id::text
	`, userID).Scan(&categoryID); err != nil {
		t.Fatal(err)
	}
	var completedRoutineID, pendingRoutineID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO routines (user_id, category_id, name, interval_days)
		VALUES ($1, $2, 'Stretch', 1)
		RETURNING id::text
	`, userID, categoryID).Scan(&completedRoutineID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO routines (user_id, category_id, name, interval_days)
		VALUES ($1, $2, 'Water the plants', 3)
		RETURNING id::text
	`, userID, categoryID).Scan(&pendingRoutineID); err != nil {
		t.Fatal(err)
	}
	today := time.Now().In(location)
	if _, err := pool.Exec(ctx, `
		INSERT INTO routine_events (routine_id, completed_on)
		VALUES ($1, $2)
	`, completedRoutineID, today.Format(time.DateOnly)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO plan_revision_routines (plan_revision_id, routine_id, position)
		VALUES ($1, $2, 0), ($1, $3, 1)
	`, revisionID, completedRoutineID, pendingRoutineID); err != nil {
		t.Fatal(err)
	}
	// Registered after the user-delete cleanup, so it runs first (t.Cleanup is
	// LIFO): a confirmed plan's routine references block deleting the routine,
	// which would otherwise fail user cascade deletion.
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM plan_revision_routines WHERE plan_revision_id = $1`, revisionID); err != nil {
			t.Errorf("delete plan revision routines: %v", err)
		}
	})

	view, err := store.Load(ctx, userID, KindDaily, date, location)
	if err != nil {
		t.Fatal(err)
	}
	if view.Plan == nil ||
		view.Plan.Revision != 2 ||
		view.Plan.SummaryMarkdown != "Second summary" ||
		view.Plan.ContentMarkdown != "# Second plan" ||
		view.Plan.NotesMarkdown != "Second notes" {
		t.Fatalf("plan = %#v", view.Plan)
	}

	components := view.Plan.Components
	if len(components.GitHubIssues) != 1 ||
		components.GitHubIssues[0].Number != 61 ||
		components.GitHubIssues[0].Repository() != "cloveclovedev/alt" {
		t.Fatalf("github issues = %#v", components.GitHubIssues)
	}
	if len(components.ActionItems) != 1 || components.ActionItems[0].Title != "Draft the release notes" {
		t.Fatalf("action items = %#v", components.ActionItems)
	}
	if len(components.Routines) != 2 ||
		components.Routines[0].RoutineID != completedRoutineID ||
		!components.Routines[0].CompletedToday ||
		components.Routines[1].RoutineID != pendingRoutineID ||
		components.Routines[1].CompletedToday {
		t.Fatalf("routines = %#v", components.Routines)
	}
}
