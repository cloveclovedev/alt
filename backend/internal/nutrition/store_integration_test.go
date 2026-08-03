package nutrition

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestStoreNutritionFlow(t *testing.T) {
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

	userID := integrationUser(t, ctx, pool, "Nutrition integration test")
	store := NewStore(pool)

	// Catalog: create, and reject a duplicate name.
	item, err := store.CreateCatalogItem(ctx, userID, CatalogInput{Name: "Protein shake", CaloriesKcal: 120, ProteinG: 24.5, Kind: KindSupplement, Source: "manual"})
	if err != nil {
		t.Fatal(err)
	}
	if item.ProteinG != 24.5 || item.Kind != KindSupplement {
		t.Fatalf("catalog round-trip lost data: %+v", item)
	}
	if _, err := store.CreateCatalogItem(ctx, userID, CatalogInput{Name: "Protein shake", CaloriesKcal: 1, ProteinG: 1, Kind: KindFood, Source: "manual"}); !isConflict(err) {
		t.Fatalf("duplicate name error = %v, want ErrConflict", err)
	}

	// Entry from the catalog snapshots the values and links catalog_id.
	loggedDate := integrationDate(t, "2026-08-03")
	entry, err := store.CreateEntry(ctx, userID, EntryInput{
		LoggedDate: loggedDate, MealType: MealSnack, Name: item.Name,
		CaloriesKcal: item.CaloriesKcal, ProteinG: item.ProteinG, Source: SourceCatalog, CatalogID: &item.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if entry.CatalogID == nil || *entry.CatalogID != item.ID || entry.ProteinG != 24.5 {
		t.Fatalf("entry did not capture catalog link/snapshot: %+v", entry)
	}

	// Editing the catalog must not rewrite the historical entry snapshot.
	if err := store.UpdateCatalogItem(ctx, userID, item.ID, CatalogInput{Name: "Protein shake", CaloriesKcal: 999, ProteinG: 99.9, Kind: KindSupplement, Source: "manual"}); err != nil {
		t.Fatal(err)
	}
	reloaded, err := store.GetEntry(ctx, userID, entry.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.CaloriesKcal != 120 || reloaded.ProteinG != 24.5 {
		t.Fatalf("entry snapshot changed after catalog edit: %+v", reloaded)
	}

	// Deleting the catalog nulls the reference but preserves the snapshot.
	if err := store.DeleteCatalogItem(ctx, userID, item.ID); err != nil {
		t.Fatal(err)
	}
	afterDelete, err := store.GetEntry(ctx, userID, entry.ID)
	if err != nil {
		t.Fatal(err)
	}
	if afterDelete.CatalogID != nil {
		t.Fatalf("catalog_id should be NULL after catalog delete, got %v", *afterDelete.CatalogID)
	}
	if afterDelete.Name != "Protein shake" || afterDelete.ProteinG != 24.5 {
		t.Fatalf("snapshot lost after catalog delete: %+v", afterDelete)
	}

	// Targets: effective-dated derivation for past, current, and future dates.
	if _, err := store.UpsertTarget(ctx, userID, TargetInput{EffectiveOn: integrationDate(t, "2026-07-01"), CaloriesKcal: 2000, ProteinG: 120}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpsertTarget(ctx, userID, TargetInput{EffectiveOn: integrationDate(t, "2026-08-01"), CaloriesKcal: 1800, ProteinG: 140, Rationale: "cut"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpsertTarget(ctx, userID, TargetInput{EffectiveOn: integrationDate(t, "2026-09-01"), CaloriesKcal: 2200, ProteinG: 150}); err != nil {
		t.Fatal(err)
	}
	// Upserting the same effective date replaces it rather than adding a row.
	if _, err := store.UpsertTarget(ctx, userID, TargetInput{EffectiveOn: integrationDate(t, "2026-08-01"), CaloriesKcal: 1750, ProteinG: 145}); err != nil {
		t.Fatal(err)
	}
	targets, err := store.ListTargets(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 3 {
		t.Fatalf("want 3 distinct effective dates, got %d", len(targets))
	}
	assertTarget(t, store, ctx, userID, "2026-07-15", 2000) // before Aug -> July target
	assertTarget(t, store, ctx, userID, "2026-08-03", 1750) // Aug target, replaced value
	assertTarget(t, store, ctx, userID, "2026-08-31", 1750) // future Sept target not yet in effect
	if applicable, err := store.ApplicableTarget(ctx, userID, integrationDate(t, "2026-06-30")); err != nil || applicable != nil {
		t.Fatalf("no target should apply before the earliest effective date: %v, %v", applicable, err)
	}

	// Daily totals sum the day's entries.
	if _, err := store.CreateEntry(ctx, userID, EntryInput{LoggedDate: loggedDate, MealType: MealBreakfast, Name: "Oats", CaloriesKcal: 300, ProteinG: 10.5, Source: SourceManual}); err != nil {
		t.Fatal(err)
	}
	totals, err := store.RangeTotals(ctx, userID, loggedDate, loggedDate)
	if err != nil {
		t.Fatal(err)
	}
	day := totals[loggedDate.Format(time.DateOnly)]
	if day.CaloriesKcal != 420 || day.ProteinG != 35.0 {
		t.Fatalf("daily totals = %+v, want {420, 35.0}", day)
	}

	// Coaching cache: absent, then upsert, then regenerate replaces in place.
	if cached, err := store.LoadCoaching(ctx, userID, loggedDate); err != nil || cached != nil {
		t.Fatalf("expected no coaching cached, got %+v, %v", cached, err)
	}
	if _, err := store.UpsertCoaching(ctx, userID, loggedDate, Coaching{
		EvaluationMarkdown: "On track.", SuggestionMarkdown: "Add protein at dinner.",
		ModelID: "openai/model", PromptVersion: "nutrition-coaching-v1",
	}); err != nil {
		t.Fatal(err)
	}
	regenerated, err := store.UpsertCoaching(ctx, userID, loggedDate, Coaching{
		EvaluationMarkdown: "Slightly under target.", SuggestionMarkdown: "A snack would help.",
		ModelID: "openai/model", PromptVersion: "nutrition-coaching-v1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if regenerated.EvaluationMarkdown != "Slightly under target." {
		t.Fatalf("regeneration should replace commentary, got %+v", regenerated)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM nutrition_coaching WHERE user_id = $1 AND coached_date = $2`, userID, loggedDate.Format(time.DateOnly)).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("regeneration must replace in place, not append: %d rows", count)
	}
}

func assertTarget(t *testing.T, store *Store, ctx context.Context, userID, day string, wantCalories int) {
	t.Helper()
	target, err := store.ApplicableTarget(ctx, userID, integrationDate(t, day))
	if err != nil {
		t.Fatal(err)
	}
	if target == nil || target.CaloriesKcal != wantCalories {
		t.Fatalf("applicable target for %s = %+v, want calories %d", day, target, wantCalories)
	}
}

func isConflict(err error) bool {
	return errors.Is(err, ErrConflict)
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
	parsed, err := time.Parse(time.DateOnly, value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
