package nutrition

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store persists the nutrition catalog, intake entries, and targets. protein_g is
// stored as numeric(6,1); it is read and summed as float8 so pgx scans it into a
// plain float64, and written as a float64 that PostgreSQL casts to numeric.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore constructs a PostgreSQL nutrition store.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// --- Catalog ---

// ListCatalog returns a user's catalog items ordered by name.
func (s *Store) ListCatalog(ctx context.Context, userID string) ([]CatalogItem, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, name, calories_kcal, protein_g::float8, kind::text, source, created_at, updated_at
		FROM nutrition_catalog
		WHERE user_id = $1
		ORDER BY name, id
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list nutrition catalog: %w", err)
	}
	defer rows.Close()
	var items []CatalogItem
	for rows.Next() {
		item, err := scanCatalogItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate nutrition catalog: %w", err)
	}
	return items, nil
}

// GetCatalogItem loads one owned catalog item.
func (s *Store) GetCatalogItem(ctx context.Context, userID, id string) (CatalogItem, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id::text, name, calories_kcal, protein_g::float8, kind::text, source, created_at, updated_at
		FROM nutrition_catalog
		WHERE id = $1 AND user_id = $2
	`, id, userID)
	item, err := scanCatalogItem(row)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		return CatalogItem{}, ErrNotFound
	}
	if err != nil {
		return CatalogItem{}, err
	}
	return item, nil
}

// CreateCatalogItem inserts a catalog item and returns it.
func (s *Store) CreateCatalogItem(ctx context.Context, userID string, input CatalogInput) (CatalogItem, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO nutrition_catalog (user_id, name, calories_kcal, protein_g, kind, source)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id::text, name, calories_kcal, protein_g::float8, kind::text, source, created_at, updated_at
	`, userID, input.Name, input.CaloriesKcal, input.ProteinG, string(input.Kind), input.Source)
	item, err := scanCatalogItem(row)
	if isUniqueViolation(err) {
		return CatalogItem{}, fmt.Errorf("%w: a catalog item with that name already exists", ErrConflict)
	}
	if err != nil {
		return CatalogItem{}, fmt.Errorf("create nutrition catalog item: %w", err)
	}
	return item, nil
}

// UpdateCatalogItem edits an owned catalog item. Existing entries keep their
// snapshots and are never rewritten.
func (s *Store) UpdateCatalogItem(ctx context.Context, userID, id string, input CatalogInput) error {
	command, err := s.pool.Exec(ctx, `
		UPDATE nutrition_catalog
		SET name = $3, calories_kcal = $4, protein_g = $5, kind = $6, updated_at = now()
		WHERE id = $1 AND user_id = $2
	`, id, userID, input.Name, input.CaloriesKcal, input.ProteinG, string(input.Kind))
	if isUniqueViolation(err) {
		return fmt.Errorf("%w: a catalog item with that name already exists", ErrConflict)
	}
	if err != nil {
		return fmt.Errorf("update nutrition catalog item: %w", err)
	}
	if command.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}

// DeleteCatalogItem removes an owned catalog item. Referencing entries keep their
// snapshots; their catalog_id is set to NULL by the foreign key.
func (s *Store) DeleteCatalogItem(ctx context.Context, userID, id string) error {
	command, err := s.pool.Exec(ctx, `DELETE FROM nutrition_catalog WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete nutrition catalog item: %w", err)
	}
	if command.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}

// --- Entries ---

// ListEntriesForDate returns the intake log for one local date, ordered by meal
// type then insertion time.
func (s *Store) ListEntriesForDate(ctx context.Context, userID string, date time.Time) ([]Entry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, logged_date, meal_type::text, name, calories_kcal, protein_g::float8, source::text, catalog_id::text, created_at
		FROM nutrition_entries
		WHERE user_id = $1 AND logged_date = $2
		ORDER BY meal_type, created_at, id
	`, userID, date.Format(time.DateOnly))
	if err != nil {
		return nil, fmt.Errorf("list nutrition entries: %w", err)
	}
	defer rows.Close()
	var entries []Entry
	for rows.Next() {
		entry, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate nutrition entries: %w", err)
	}
	return entries, nil
}

// GetEntry loads one owned entry.
func (s *Store) GetEntry(ctx context.Context, userID, id string) (Entry, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id::text, logged_date, meal_type::text, name, calories_kcal, protein_g::float8, source::text, catalog_id::text, created_at
		FROM nutrition_entries
		WHERE id = $1 AND user_id = $2
	`, id, userID)
	entry, err := scanEntry(row)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		return Entry{}, ErrNotFound
	}
	if err != nil {
		return Entry{}, err
	}
	return entry, nil
}

// CreateEntry inserts one intake entry and returns it. A catalog_id, when set, is
// validated against the caller's own catalog by the service before this call.
func (s *Store) CreateEntry(ctx context.Context, userID string, input EntryInput) (Entry, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO nutrition_entries (user_id, logged_date, meal_type, name, calories_kcal, protein_g, source, catalog_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id::text, logged_date, meal_type::text, name, calories_kcal, protein_g::float8, source::text, catalog_id::text, created_at
	`, userID, input.LoggedDate.Format(time.DateOnly), string(input.MealType), input.Name, input.CaloriesKcal, input.ProteinG, string(input.Source), input.CatalogID)
	entry, err := scanEntry(row)
	if err != nil {
		return Entry{}, fmt.Errorf("create nutrition entry: %w", err)
	}
	return entry, nil
}

// UpdateEntry edits one owned entry (a correction). meal_type, name, and metrics
// are editable; the source and catalog link are not changed here.
func (s *Store) UpdateEntry(ctx context.Context, userID, id string, input EntryInput) error {
	command, err := s.pool.Exec(ctx, `
		UPDATE nutrition_entries
		SET logged_date = $3, meal_type = $4, name = $5, calories_kcal = $6, protein_g = $7
		WHERE id = $1 AND user_id = $2
	`, id, userID, input.LoggedDate.Format(time.DateOnly), string(input.MealType), input.Name, input.CaloriesKcal, input.ProteinG)
	if err != nil {
		return fmt.Errorf("update nutrition entry: %w", err)
	}
	if command.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}

// DeleteEntry removes one owned entry.
func (s *Store) DeleteEntry(ctx context.Context, userID, id string) error {
	command, err := s.pool.Exec(ctx, `DELETE FROM nutrition_entries WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete nutrition entry: %w", err)
	}
	if command.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}

// --- Targets ---

// UpsertTarget appends a target for its effective date, replacing any existing
// target for the same date.
func (s *Store) UpsertTarget(ctx context.Context, userID string, input TargetInput) (Target, error) {
	var rationale *string
	if input.Rationale != "" {
		rationale = &input.Rationale
	}
	row := s.pool.QueryRow(ctx, `
		INSERT INTO nutrition_targets (user_id, effective_on, calories_kcal, protein_g, rationale)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, effective_on) DO UPDATE
		SET calories_kcal = EXCLUDED.calories_kcal, protein_g = EXCLUDED.protein_g,
		    rationale = EXCLUDED.rationale, updated_at = now()
		RETURNING id::text, effective_on, calories_kcal, protein_g::float8, COALESCE(rationale, ''), created_at, updated_at
	`, userID, input.EffectiveOn.Format(time.DateOnly), input.CaloriesKcal, input.ProteinG, rationale)
	target, err := scanTarget(row)
	if err != nil {
		return Target{}, fmt.Errorf("upsert nutrition target: %w", err)
	}
	return target, nil
}

// ListTargets returns a user's targets, newest effective date first.
func (s *Store) ListTargets(ctx context.Context, userID string) ([]Target, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, effective_on, calories_kcal, protein_g::float8, COALESCE(rationale, ''), created_at, updated_at
		FROM nutrition_targets
		WHERE user_id = $1
		ORDER BY effective_on DESC, id
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list nutrition targets: %w", err)
	}
	defer rows.Close()
	var targets []Target
	for rows.Next() {
		target, err := scanTarget(rows)
		if err != nil {
			return nil, err
		}
		targets = append(targets, target)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate nutrition targets: %w", err)
	}
	return targets, nil
}

// ApplicableTarget returns the target with the greatest effective_on <= date, or
// nil when the user has no target effective by then.
func (s *Store) ApplicableTarget(ctx context.Context, userID string, date time.Time) (*Target, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id::text, effective_on, calories_kcal, protein_g::float8, COALESCE(rationale, ''), created_at, updated_at
		FROM nutrition_targets
		WHERE user_id = $1 AND effective_on <= $2
		ORDER BY effective_on DESC
		LIMIT 1
	`, userID, date.Format(time.DateOnly))
	target, err := scanTarget(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load applicable nutrition target: %w", err)
	}
	return &target, nil
}

// --- Summaries ---

// RangeTotals returns summed intake per local date over [start, end], keyed by
// the date's YYYY-MM-DD string. Days with no entries are absent from the map.
func (s *Store) RangeTotals(ctx context.Context, userID string, start, end time.Time) (map[string]Totals, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT logged_date, COALESCE(SUM(calories_kcal), 0)::bigint, COALESCE(SUM(protein_g), 0)::float8
		FROM nutrition_entries
		WHERE user_id = $1 AND logged_date BETWEEN $2 AND $3
		GROUP BY logged_date
	`, userID, start.Format(time.DateOnly), end.Format(time.DateOnly))
	if err != nil {
		return nil, fmt.Errorf("summarize nutrition intake: %w", err)
	}
	defer rows.Close()
	totals := make(map[string]Totals)
	for rows.Next() {
		var (
			day      time.Time
			calories int64
			protein  float64
		)
		if err := rows.Scan(&day, &calories, &protein); err != nil {
			return nil, fmt.Errorf("scan nutrition totals: %w", err)
		}
		totals[day.Format(time.DateOnly)] = Totals{CaloriesKcal: int(calories), ProteinG: protein}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate nutrition totals: %w", err)
	}
	return totals, nil
}

// --- scanning helpers ---

type scanner interface {
	Scan(dest ...any) error
}

func scanCatalogItem(row scanner) (CatalogItem, error) {
	var (
		item CatalogItem
		kind string
	)
	if err := row.Scan(&item.ID, &item.Name, &item.CaloriesKcal, &item.ProteinG, &kind, &item.Source, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return CatalogItem{}, err
	}
	item.Kind = CatalogKind(kind)
	return item, nil
}

func scanEntry(row scanner) (Entry, error) {
	var (
		entry     Entry
		mealType  string
		source    string
		catalogID *string
	)
	if err := row.Scan(&entry.ID, &entry.LoggedDate, &mealType, &entry.Name, &entry.CaloriesKcal, &entry.ProteinG, &source, &catalogID, &entry.CreatedAt); err != nil {
		return Entry{}, err
	}
	entry.MealType = MealType(mealType)
	entry.Source = EntrySource(source)
	entry.CatalogID = catalogID
	return entry, nil
}

func scanTarget(row scanner) (Target, error) {
	var target Target
	if err := row.Scan(&target.ID, &target.EffectiveOn, &target.CaloriesKcal, &target.ProteinG, &target.Rationale, &target.CreatedAt, &target.UpdatedAt); err != nil {
		return Target{}, err
	}
	return target, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// isInvalidUUID reports whether err is PostgreSQL's invalid-text-representation
// error (22P02), which a malformed id path parameter produces. Such an id cannot
// identify an existing row, so callers treat it as not found rather than a fault.
func isInvalidUUID(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "22P02"
}
