package routine

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestStoreRoutineFlow(t *testing.T) {
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

	userID := integrationUser(t, ctx, pool, "Routine integration test")
	otherUserID := integrationUser(t, ctx, pool, "Other routine integration test")
	store := NewStore(pool)
	if err := store.CreateCategory(ctx, userID, CategoryInput{Name: "Health", Position: 10, Active: true}); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateCategory(ctx, userID, CategoryInput{Name: " Health ", Position: 11, Active: true}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("trimmed duplicate category error = %v", err)
	}
	if err := store.CreateCategory(ctx, otherUserID, CategoryInput{Name: "Other", Position: 0, Active: true}); err != nil {
		t.Fatal(err)
	}
	categories, err := store.ListCategories(ctx, userID, true)
	if err != nil || len(categories) != 1 {
		t.Fatalf("categories = %#v, error = %v", categories, err)
	}
	otherCategories, err := store.ListCategories(ctx, otherUserID, true)
	if err != nil {
		t.Fatal(err)
	}
	baseline := integrationDate(t, "2026-07-20")
	routineID, err := store.CreateRoutine(ctx, userID, RoutineInput{
		CategoryID: categories[0].ID, Name: "Walk", Notes: "Outside", Status: RoutineStatusActive,
		IntervalDays: 7, AvailableWeekdays: []int16{1}, BaselineOn: &baseline,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateRoutine(ctx, userID, RoutineInput{CategoryID: otherCategories[0].ID, Name: "Foreign category", Status: RoutineStatusActive, IntervalDays: 1}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("foreign category error = %v", err)
	}

	record, events, err := store.LoadRoutine(ctx, userID, routineID)
	if err != nil {
		t.Fatal(err)
	}
	if record.Routine.Name != "Walk" || record.LastEvent == nil || len(events) != 1 || events[0].Kind != EventKindBaseline {
		t.Fatalf("routine = %#v, events = %#v", record, events)
	}
	if err := store.UpdateRoutine(ctx, userID, routineID, RoutineInput{CategoryID: categories[0].ID, Name: "Daily walk", Notes: "Outside", Status: RoutineStatusActive, IntervalDays: 7, AvailableWeekdays: []int16{1}}); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateRoutine(ctx, userID, routineID, RoutineInput{CategoryID: categories[0].ID, Name: "Daily walk", Notes: "Outside", Status: RoutineStatusInactive, IntervalDays: 7}); err != nil {
		t.Fatal(err)
	}
	activeRecords, err := store.ListRoutines(ctx, userID, false)
	if err != nil || len(activeRecords) != 0 {
		t.Fatalf("active records = %#v, error = %v", activeRecords, err)
	}
	if err := store.UpdateRoutine(ctx, userID, routineID, RoutineInput{CategoryID: categories[0].ID, Name: "Daily walk", Notes: "Outside", Status: RoutineStatusActive, IntervalDays: 7}); err != nil {
		t.Fatal(err)
	}
	_, events, err = store.LoadRoutine(ctx, userID, routineID)
	if err != nil || len(events) != 1 || events[0].RoutineID != routineID {
		t.Fatalf("renamed events = %#v, error = %v", events, err)
	}
	if err := store.DeleteCategory(ctx, userID, categories[0].ID); !errors.Is(err, ErrDeleteNotAllowed) {
		t.Fatalf("category delete error = %v", err)
	}
	if err := store.DeleteRoutine(ctx, userID, routineID); !errors.Is(err, ErrDeleteNotAllowed) {
		t.Fatalf("routine delete error = %v", err)
	}
	if err := store.DeleteEvent(ctx, userID, routineID, events[0].ID); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteRoutine(ctx, userID, routineID); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteCategory(ctx, userID, categories[0].ID); err != nil {
		t.Fatal(err)
	}
}

func integrationUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool, displayName string) string {
	t.Helper()
	var userID string
	if err := pool.QueryRow(ctx, `INSERT INTO users DEFAULT VALUES RETURNING id::text`).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO profiles (user_id, display_name, timezone) VALUES ($1, $2, 'Asia/Tokyo')`, userID, displayName); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID); err != nil {
			t.Errorf("delete integration user: %v", err)
		}
	})
	return userID
}

func integrationDate(t *testing.T, value string) time.Time {
	t.Helper()
	date, err := time.Parse(time.DateOnly, value)
	if err != nil {
		t.Fatal(err)
	}
	return date
}
