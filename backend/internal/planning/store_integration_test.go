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

	if err := store.Save(ctx, userID, KindDaily, date, SavePlanInput{
		SummaryMarkdown: "First summary",
		ContentMarkdown: "# First plan",
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, userID, KindDaily, date, SavePlanInput{
		SummaryMarkdown: "Second summary",
		ContentMarkdown: "# Second plan",
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, userID, KindWeekly, date, SavePlanInput{
		SummaryMarkdown: "Weekly summary",
		ContentMarkdown: "# Weekly plan",
	}); err != nil {
		t.Fatal(err)
	}

	view, err := store.Load(ctx, userID, KindDaily, date, location)
	if err != nil {
		t.Fatal(err)
	}
	if view.Plan == nil ||
		view.Plan.Revision != 2 ||
		view.Plan.SummaryMarkdown != "Second summary" ||
		view.Plan.ContentMarkdown != "# Second plan" {
		t.Fatalf("plan = %#v", view.Plan)
	}

	weeklyView, err := store.Load(ctx, userID, KindWeekly, date, location)
	if err != nil {
		t.Fatal(err)
	}
	if weeklyView.Plan == nil ||
		weeklyView.Plan.Revision != 1 ||
		weeklyView.Plan.SummaryMarkdown != "Weekly summary" ||
		weeklyView.Plan.ContentMarkdown != "# Weekly plan" {
		t.Fatalf("weekly plan = %#v", weeklyView.Plan)
	}
}
