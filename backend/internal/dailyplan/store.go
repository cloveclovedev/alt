package dailyplan

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store persists the daily planning feature in PostgreSQL.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore constructs a PostgreSQL daily planning store.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// LoadToday loads one user's records for a local calendar day.
func (s *Store) LoadToday(ctx context.Context, userID string, date time.Time, location *time.Location) (Today, error) {
	start := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, location)
	end := start.AddDate(0, 0, 1)
	view := Today{Date: start, Timezone: location.String()}

	rows, err := s.pool.Query(ctx, `
		SELECT id::text, kind, body, occurred_at
		FROM journal_entries
		WHERE user_id = $1
		  AND occurred_at >= $2
		  AND occurred_at < $3
		ORDER BY occurred_at DESC
	`, userID, start, end)
	if err != nil {
		return Today{}, fmt.Errorf("list journal entries: %w", err)
	}
	for rows.Next() {
		var entry JournalEntry
		if err := rows.Scan(&entry.ID, &entry.Kind, &entry.Body, &entry.OccurredAt); err != nil {
			rows.Close()
			return Today{}, fmt.Errorf("scan journal entry: %w", err)
		}
		entry.OccurredAt = entry.OccurredAt.In(location)
		view.JournalEntries = append(view.JournalEntries, entry)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return Today{}, fmt.Errorf("iterate journal entries: %w", err)
	}
	rows.Close()

	rows, err = s.pool.Query(ctx, `
		SELECT id::text, title, COALESCE(notes, ''), status,
		       COALESCE(priority, ''), due_date
		FROM tasks
		WHERE user_id = $1
		  AND status IN ('active', 'backlog')
		ORDER BY
		  CASE priority
		    WHEN 'P0' THEN 0
		    WHEN 'P1' THEN 1
		    WHEN 'P2' THEN 2
		    WHEN 'P3' THEN 3
		    ELSE 4
		  END,
		  due_date ASC NULLS LAST,
		  created_at ASC
	`, userID)
	if err != nil {
		return Today{}, fmt.Errorf("list tasks: %w", err)
	}
	for rows.Next() {
		var task Task
		if err := rows.Scan(
			&task.ID,
			&task.Title,
			&task.Notes,
			&task.Status,
			&task.Priority,
			&task.DueDate,
		); err != nil {
			rows.Close()
			return Today{}, fmt.Errorf("scan task: %w", err)
		}
		view.Tasks = append(view.Tasks, task)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return Today{}, fmt.Errorf("iterate tasks: %w", err)
	}
	rows.Close()

	var plan Plan
	err = s.pool.QueryRow(ctx, `
		SELECT id::text, plan_date, revision, summary, created_at
		FROM daily_plans
		WHERE user_id = $1
		  AND plan_date = $2
		  AND status = 'accepted'
		ORDER BY revision DESC
		LIMIT 1
	`, userID, start.Format(time.DateOnly)).Scan(
		&plan.ID,
		&plan.Date,
		&plan.Revision,
		&plan.Summary,
		&plan.CreatedAt,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return Today{}, fmt.Errorf("load accepted daily plan: %w", err)
	}
	if err == nil {
		plan.CreatedAt = plan.CreatedAt.In(location)
		view.Plan = &plan
	}
	return view, nil
}

// AddJournalEntry appends a raw journal record.
func (s *Store) AddJournalEntry(ctx context.Context, userID, body string, occurredAt time.Time) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO journal_entries (user_id, kind, body, occurred_at, source)
		VALUES ($1, 'note', $2, $3, 'web')
	`, userID, body, occurredAt)
	if err != nil {
		return fmt.Errorf("insert journal entry: %w", err)
	}
	return nil
}

// AddTask creates an active task.
func (s *Store) AddTask(
	ctx context.Context,
	userID, title, notes, priority string,
	dueDate *time.Time,
) error {
	var nullablePriority any
	if priority != "" {
		nullablePriority = priority
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO tasks (user_id, title, notes, status, priority, due_date)
		VALUES ($1, $2, NULLIF($3, ''), 'active', $4, $5)
	`, userID, title, notes, nullablePriority, dueDate)
	if err != nil {
		return fmt.Errorf("insert task: %w", err)
	}
	return nil
}

// CompleteTask marks one owned actionable task as done.
func (s *Store) CompleteTask(ctx context.Context, userID, taskID string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE tasks
		SET status = 'done', updated_at = now()
		WHERE id = $1
		  AND user_id = $2
		  AND status IN ('active', 'backlog')
	`, taskID, userID)
	if err != nil {
		return fmt.Errorf("complete task: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// AcceptPlan stores a new accepted revision and archives the prior one.
func (s *Store) AcceptPlan(ctx context.Context, userID string, date time.Time, summary string) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin plan transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var lockedUserID string
	if err := tx.QueryRow(ctx, `
		SELECT id::text
		FROM users
		WHERE id = $1
		FOR UPDATE
	`, userID).Scan(&lockedUserID); err != nil {
		return fmt.Errorf("lock plan owner: %w", err)
	}

	planDate := date.Format(time.DateOnly)
	if _, err := tx.Exec(ctx, `
		UPDATE daily_plans
		SET status = 'archived', updated_at = now()
		WHERE user_id = $1
		  AND plan_date = $2
		  AND status = 'accepted'
	`, userID, planDate); err != nil {
		return fmt.Errorf("archive accepted plan: %w", err)
	}

	var revision int
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(revision), 0) + 1
		FROM daily_plans
		WHERE user_id = $1 AND plan_date = $2
	`, userID, planDate).Scan(&revision); err != nil {
		return fmt.Errorf("select next plan revision: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO daily_plans (user_id, plan_date, revision, status, summary)
		VALUES ($1, $2, $3, 'accepted', $4)
	`, userID, planDate, revision, summary); err != nil {
		return fmt.Errorf("insert accepted plan: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit accepted plan: %w", err)
	}
	return nil
}
