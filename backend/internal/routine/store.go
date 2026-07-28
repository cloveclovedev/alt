package routine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store persists routines, categories, and completion history in PostgreSQL.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore constructs a PostgreSQL routine store.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

type routineRecord struct {
	Routine   Routine
	LastEvent *Event
}

// ListCategories returns user-owned categories in deterministic display order.
func (s *Store) ListCategories(ctx context.Context, userID string, includeInactive bool) ([]Category, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, name, position, active
		FROM routine_categories
		WHERE user_id = $1
		  AND ($2 OR active)
		ORDER BY position, name, id
	`, userID, includeInactive)
	if err != nil {
		return nil, fmt.Errorf("list routine categories: %w", err)
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var category Category
		if err := rows.Scan(&category.ID, &category.Name, &category.Position, &category.Active); err != nil {
			return nil, fmt.Errorf("scan routine category: %w", err)
		}
		categories = append(categories, category)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate routine categories: %w", err)
	}
	return categories, nil
}

// ListRoutines returns user-owned routines and their latest event.
func (s *Store) ListRoutines(ctx context.Context, userID string, includeInactive bool) ([]routineRecord, error) {
	rows, err := s.pool.Query(ctx, routineSelect+`
		WHERE r.user_id = $1
		  AND ($2 OR r.status = 'active')
		ORDER BY c.position, c.name, c.id, r.name, r.id
	`, userID, includeInactive)
	if err != nil {
		return nil, fmt.Errorf("list routines: %w", err)
	}
	defer rows.Close()

	var records []routineRecord
	for rows.Next() {
		record, err := scanRoutineRecord(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate routines: %w", err)
	}
	return records, nil
}

// LoadRoutine returns one user-owned routine, latest anchor, and full event history.
func (s *Store) LoadRoutine(ctx context.Context, userID, routineID string) (routineRecord, []Event, error) {
	record, err := scanRoutineRecord(s.pool.QueryRow(ctx, routineSelect+`
		WHERE r.user_id = $1 AND r.id = $2
	`, userID, routineID))
	if errors.Is(err, pgx.ErrNoRows) {
		return routineRecord{}, nil, ErrNotFound
	}
	if err != nil {
		return routineRecord{}, nil, fmt.Errorf("load routine: %w", err)
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id::text, routine_id::text, completed_on, note, created_at, updated_at
		FROM routine_events
		WHERE routine_id = $1
		ORDER BY completed_on DESC, created_at DESC, id DESC
	`, routineID)
	if err != nil {
		return routineRecord{}, nil, fmt.Errorf("list routine events: %w", err)
	}
	defer rows.Close()
	var events []Event
	for rows.Next() {
		var event Event
		if err := rows.Scan(&event.ID, &event.RoutineID, &event.CompletedOn, &event.Note, &event.CreatedAt, &event.UpdatedAt); err != nil {
			return routineRecord{}, nil, fmt.Errorf("scan routine event: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return routineRecord{}, nil, fmt.Errorf("iterate routine events: %w", err)
	}
	return record, events, nil
}

// CreateRoutine creates a definition and optional initial event atomically.
func (s *Store) CreateRoutine(ctx context.Context, userID string, input RoutineInput) (string, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", fmt.Errorf("begin create routine transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var routineID string
	err = tx.QueryRow(ctx, `
		INSERT INTO routines (
			user_id, category_id, name, notes, status, interval_days,
			available_weekdays, active_months
		)
		SELECT $1, c.id, $3, $4, $5, $6, $7, $8
		FROM routine_categories AS c
		WHERE c.id = $2 AND c.user_id = $1
		RETURNING id::text
	`, userID, input.CategoryID, input.Name, input.Notes, input.Status, input.IntervalDays, arrayValues(input.AvailableWeekdays), arrayValues(input.ActiveMonths)).Scan(&routineID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("%w: category", ErrNotFound)
	}
	if err != nil {
		if duplicateInput(err, "routine name") != nil {
			return "", duplicateInput(err, "routine name")
		}
		return "", fmt.Errorf("insert routine: %w", err)
	}
	if input.LastCompletedOn != nil {
		if _, err := tx.Exec(ctx, `
			INSERT INTO routine_events (routine_id, completed_on, note)
			VALUES ($1, $2, '')
		`, routineID, input.LastCompletedOn.Format(time.DateOnly)); err != nil {
			return "", fmt.Errorf("insert initial routine event: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit create routine: %w", err)
	}
	return routineID, nil
}

// UpdateRoutine changes a definition while retaining its history.
func (s *Store) UpdateRoutine(ctx context.Context, userID, routineID string, input RoutineInput) error {
	result, err := s.pool.Exec(ctx, `
		UPDATE routines AS r
		SET category_id = c.id,
			name = $4,
			notes = $5,
			status = $6,
			interval_days = $7,
			available_weekdays = $8,
			active_months = $9,
			updated_at = now()
		FROM routine_categories AS c
		WHERE r.id = $1
		  AND r.user_id = $2
		  AND c.id = $3
		  AND c.user_id = $2
	`, routineID, userID, input.CategoryID, input.Name, input.Notes, input.Status, input.IntervalDays, arrayValues(input.AvailableWeekdays), arrayValues(input.ActiveMonths))
	if err != nil {
		if duplicateInput(err, "routine name") != nil {
			return duplicateInput(err, "routine name")
		}
		return fmt.Errorf("update routine: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteRoutine permanently removes a routine only if it has no events.
func (s *Store) DeleteRoutine(ctx context.Context, userID, routineID string) error {
	result, err := s.pool.Exec(ctx, `
		DELETE FROM routines AS r
		WHERE r.id = $1
		  AND r.user_id = $2
		  AND NOT EXISTS (SELECT 1 FROM routine_events AS e WHERE e.routine_id = r.id)
	`, routineID, userID)
	if err != nil {
		return fmt.Errorf("delete routine: %w", err)
	}
	if result.RowsAffected() == 1 {
		return nil
	}
	var exists bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM routines WHERE id = $1 AND user_id = $2)`, routineID, userID).Scan(&exists); err != nil {
		return fmt.Errorf("check routine deletion: %w", err)
	}
	if exists {
		return ErrDeleteNotAllowed
	}
	return ErrNotFound
}

// CreateEvent records a user-owned completion event.
func (s *Store) CreateEvent(ctx context.Context, userID, routineID string, input CompletionInput) error {
	result, err := s.pool.Exec(ctx, `
		INSERT INTO routine_events (routine_id, completed_on, note)
		SELECT r.id, $3, $4
		FROM routines AS r
		WHERE r.id = $1 AND r.user_id = $2
	`, routineID, userID, input.CompletedOn.Format(time.DateOnly), input.Note)
	if err != nil {
		return fmt.Errorf("insert routine event: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateEvent corrects a routine event.
func (s *Store) UpdateEvent(ctx context.Context, userID, routineID, eventID string, input EventInput) error {
	result, err := s.pool.Exec(ctx, `
		UPDATE routine_events AS e
		SET completed_on = $4, note = $5, updated_at = now()
		FROM routines AS r
		WHERE e.id = $1
		  AND e.routine_id = $2
		  AND r.id = e.routine_id
		  AND r.user_id = $3
	`, eventID, routineID, userID, input.CompletedOn.Format(time.DateOnly), input.Note)
	if err != nil {
		return fmt.Errorf("update routine event: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteEvent removes an event that was recorded by mistake.
func (s *Store) DeleteEvent(ctx context.Context, userID, routineID, eventID string) error {
	result, err := s.pool.Exec(ctx, `
		DELETE FROM routine_events AS e
		USING routines AS r
		WHERE e.id = $1
		  AND e.routine_id = $2
		  AND r.id = e.routine_id
		  AND r.user_id = $3
	`, eventID, routineID, userID)
	if err != nil {
		return fmt.Errorf("delete routine event: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateCategory inserts a user-owned category.
func (s *Store) CreateCategory(ctx context.Context, userID string, input CategoryInput) error {
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO routine_categories (user_id, name, position, active)
		VALUES ($1, $2, $3, $4)
	`, userID, input.Name, input.Position, input.Active); err != nil {
		if duplicateInput(err, "category name") != nil {
			return duplicateInput(err, "category name")
		}
		return fmt.Errorf("insert routine category: %w", err)
	}
	return nil
}

// UpdateCategory changes a category's editable fields.
func (s *Store) UpdateCategory(ctx context.Context, userID, categoryID string, input CategoryInput) error {
	result, err := s.pool.Exec(ctx, `
		UPDATE routine_categories
		SET name = $3, position = $4, active = $5, updated_at = now()
		WHERE id = $1 AND user_id = $2
	`, categoryID, userID, input.Name, input.Position, input.Active)
	if err != nil {
		if duplicateInput(err, "category name") != nil {
			return duplicateInput(err, "category name")
		}
		return fmt.Errorf("update routine category: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteCategory permanently removes an unreferenced category.
func (s *Store) DeleteCategory(ctx context.Context, userID, categoryID string) error {
	result, err := s.pool.Exec(ctx, `
		DELETE FROM routine_categories AS c
		WHERE c.id = $1
		  AND c.user_id = $2
		  AND NOT EXISTS (SELECT 1 FROM routines AS r WHERE r.category_id = c.id)
	`, categoryID, userID)
	if err != nil {
		return fmt.Errorf("delete routine category: %w", err)
	}
	if result.RowsAffected() == 1 {
		return nil
	}
	var exists bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM routine_categories WHERE id = $1 AND user_id = $2)`, categoryID, userID).Scan(&exists); err != nil {
		return fmt.Errorf("check category deletion: %w", err)
	}
	if exists {
		return ErrDeleteNotAllowed
	}
	return ErrNotFound
}

const routineSelect = `
	SELECT r.id::text,
	       r.category_id::text,
	       c.name,
	       r.name,
	       r.notes,
	       r.status::text,
	       r.interval_days,
	       r.available_weekdays,
	       r.active_months,
	       r.created_at,
	       r.updated_at,
	       e.id::text,
	       e.routine_id::text,
	       e.completed_on,
	       e.note,
	       e.created_at,
	       e.updated_at
	FROM routines AS r
	JOIN routine_categories AS c ON c.id = r.category_id
	LEFT JOIN LATERAL (
		SELECT id, routine_id, completed_on, note, created_at, updated_at
		FROM routine_events
		WHERE routine_id = r.id
		ORDER BY completed_on DESC, created_at DESC, id DESC
		LIMIT 1
	) AS e ON true
`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRoutineRecord(row rowScanner) (routineRecord, error) {
	var record routineRecord
	var eventID, eventRoutineID *string
	var completedOn, eventCreatedAt, eventUpdatedAt *time.Time
	var eventNote *string
	err := row.Scan(
		&record.Routine.ID,
		&record.Routine.CategoryID,
		&record.Routine.CategoryName,
		&record.Routine.Name,
		&record.Routine.Notes,
		&record.Routine.Status,
		&record.Routine.IntervalDays,
		&record.Routine.AvailableWeekdays,
		&record.Routine.ActiveMonths,
		&record.Routine.CreatedAt,
		&record.Routine.UpdatedAt,
		&eventID,
		&eventRoutineID,
		&completedOn,
		&eventNote,
		&eventCreatedAt,
		&eventUpdatedAt,
	)
	if err != nil {
		return routineRecord{}, err
	}
	if eventID != nil {
		record.LastEvent = &Event{
			ID: *eventID, RoutineID: *eventRoutineID,
			CompletedOn: *completedOn, Note: *eventNote,
			CreatedAt: *eventCreatedAt, UpdatedAt: *eventUpdatedAt,
		}
	}
	return record, nil
}

func arrayValues(values []int16) []int16 {
	if values == nil {
		return []int16{}
	}
	return values
}

func duplicateInput(err error, field string) error {
	var databaseError *pgconn.PgError
	if errors.As(err, &databaseError) && databaseError.Code == "23505" {
		return fmt.Errorf("%w: %s already exists", ErrInvalidInput, field)
	}
	return nil
}
