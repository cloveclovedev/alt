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
		&plan.CreatedAt,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return View{}, fmt.Errorf("load latest plan revision: %w", err)
	}
	if err == nil {
		plan.CreatedAt = plan.CreatedAt.In(location)
		view.Plan = &plan
	}
	return view, nil
}

// Save creates the logical plan when needed and appends a new revision.
func (s *Store) Save(
	ctx context.Context,
	userID string,
	kind Kind,
	periodStart time.Time,
	input SavePlanInput,
) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin plan transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	periodStartText := periodStart.Format(time.DateOnly)
	if _, err := tx.Exec(ctx, `
		INSERT INTO plans (user_id, kind, period_start)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, kind, period_start) DO NOTHING
	`, userID, kind, periodStartText); err != nil {
		return fmt.Errorf("ensure plan: %w", err)
	}

	var planID string
	if err := tx.QueryRow(ctx, `
		SELECT id::text
		FROM plans
		WHERE user_id = $1
		  AND kind = $2
		  AND period_start = $3
		FOR UPDATE
	`, userID, kind, periodStartText).Scan(&planID); err != nil {
		return fmt.Errorf("lock plan: %w", err)
	}

	var revision int
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(revision), 0) + 1
		FROM plan_revisions
		WHERE plan_id = $1
	`, planID).Scan(&revision); err != nil {
		return fmt.Errorf("select next plan revision: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO plan_revisions (
			plan_id,
			revision,
			summary_markdown,
			content_markdown
		)
		VALUES ($1, $2, $3, $4)
	`, planID, revision, input.SummaryMarkdown, input.ContentMarkdown); err != nil {
		return fmt.Errorf("insert plan revision: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit plan revision: %w", err)
	}
	return nil
}
