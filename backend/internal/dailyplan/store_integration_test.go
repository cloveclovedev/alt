package dailyplan

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestStoreDailyPlanningFlow(t *testing.T) {
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
		INSERT INTO users (display_name, timezone)
		VALUES ('Daily plan integration test', 'Asia/Tokyo')
		RETURNING id::text
	`).Scan(&userID)
	if err != nil {
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
	dueDate := time.Date(2026, 7, 28, 0, 0, 0, 0, location)

	if err := store.AddJournalEntry(ctx, userID, "Integration note", date); err != nil {
		t.Fatal(err)
	}
	if err := store.AddTask(ctx, userID, "Integration task", "", "P1", &dueDate); err != nil {
		t.Fatal(err)
	}
	if err := store.AcceptPlan(ctx, userID, date, "First revision"); err != nil {
		t.Fatal(err)
	}
	if err := store.AcceptPlan(ctx, userID, date, "Second revision"); err != nil {
		t.Fatal(err)
	}

	view, err := store.LoadToday(ctx, userID, date, location)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.JournalEntries) != 1 || view.JournalEntries[0].Body != "Integration note" {
		t.Fatalf("journal entries = %#v", view.JournalEntries)
	}
	if len(view.Tasks) != 1 || view.Tasks[0].Priority != "P1" {
		t.Fatalf("tasks = %#v", view.Tasks)
	}
	if view.Plan == nil || view.Plan.Revision != 2 || view.Plan.Summary != "Second revision" {
		t.Fatalf("plan = %#v", view.Plan)
	}

	if err := store.CompleteTask(ctx, userID, view.Tasks[0].ID); err != nil {
		t.Fatal(err)
	}
	view, err = store.LoadToday(ctx, userID, date, location)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Tasks) != 0 {
		t.Fatalf("tasks after completion = %#v", view.Tasks)
	}
}
