package planning

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	// ErrNotFound identifies a user-owned planning resource that is absent.
	ErrNotFound = errors.New("not found")
	// ErrInvalidSessionState identifies a lifecycle action attempted at the wrong time.
	ErrInvalidSessionState = errors.New("invalid session state")
)

type calendarContextEnvelope struct {
	Events []CalendarEvent `json:"events"`
	Error  string          `json:"error,omitempty"`
}

type githubContextEnvelope struct {
	Issues []GitHubIssue `json:"issues"`
	Error  string        `json:"error,omitempty"`
}

type routineContextEnvelope struct {
	Routines []RoutineCandidate `json:"routines"`
	Error    string             `json:"error,omitempty"`
}

// FindOrCreateDailySession resumes an active session or creates its gathering shell.
func (s *Store) FindOrCreateDailySession(ctx context.Context, userID string, date time.Time) (DailyPlanningSession, bool, error) {
	dateText := date.Format(time.DateOnly)
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO daily_planning_sessions (user_id, plan_date, status)
		VALUES ($1, $2, 'gathering')
		ON CONFLICT DO NOTHING
	`, userID, dateText); err != nil {
		return DailyPlanningSession{}, false, fmt.Errorf("ensure daily planning session: %w", err)
	}

	var sessionID string
	err := s.pool.QueryRow(ctx, `
		SELECT id::text
		FROM daily_planning_sessions
		WHERE user_id = $1 AND plan_date = $2
		  AND status IN ('gathering', 'context_blocked', 'chatting', 'reviewing')
		ORDER BY updated_at DESC, id DESC
		LIMIT 1
	`, userID, dateText).Scan(&sessionID)
	if err != nil {
		return DailyPlanningSession{}, false, fmt.Errorf("find daily planning session: %w", err)
	}
	session, err := s.LoadDailySession(ctx, userID, sessionID)
	if err != nil {
		return DailyPlanningSession{}, false, err
	}
	return session, session.Status == DailySessionGathering && session.Context.GatheredAt.IsZero(), nil
}

// LoadLatestFinalizedDailySession returns the newest final session for one local date.
func (s *Store) LoadLatestFinalizedDailySession(ctx context.Context, userID string, date time.Time) (DailyPlanningSession, error) {
	var sessionID string
	err := s.pool.QueryRow(ctx, `
		SELECT id::text
		FROM daily_planning_sessions
		WHERE user_id = $1 AND plan_date = $2 AND status = 'finalized'
		ORDER BY finalized_at DESC, id DESC
		LIMIT 1
	`, userID, date.Format(time.DateOnly)).Scan(&sessionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return DailyPlanningSession{}, ErrNotFound
	}
	if err != nil {
		return DailyPlanningSession{}, fmt.Errorf("load finalized daily session: %w", err)
	}
	return s.LoadDailySession(ctx, userID, sessionID)
}

// LoadDailySession returns persisted working state. Finalized sessions intentionally have no content.
func (s *Store) LoadDailySession(ctx context.Context, userID, sessionID string) (DailyPlanningSession, error) {
	var session DailyPlanningSession
	var previewJSON []byte
	var finalizedAt *time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, plan_date, status::text, preview_json,
		       COALESCE(confirmed_plan_revision_id::text, ''), model_id, prompt_version,
		       created_at, updated_at, finalized_at
		FROM daily_planning_sessions
		WHERE id = $1 AND user_id = $2
	`, strings.TrimSpace(sessionID), userID).Scan(
		&session.ID, &session.PlanDate, &session.Status, &previewJSON,
		&session.ConfirmedPlanRevisionID, &session.ModelID, &session.PromptVersion,
		&session.CreatedAt, &session.UpdatedAt, &finalizedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return DailyPlanningSession{}, ErrNotFound
	}
	if err != nil {
		return DailyPlanningSession{}, fmt.Errorf("load daily planning session: %w", err)
	}
	session.FinalizedAt = finalizedAt
	if len(previewJSON) > 0 {
		var preview DailyPlanProposal
		if err := json.Unmarshal(previewJSON, &preview); err != nil {
			return DailyPlanningSession{}, fmt.Errorf("decode daily planning preview: %w", err)
		}
		session.Preview = &preview
	}

	if err := s.loadDailyContext(ctx, &session); err != nil {
		return DailyPlanningSession{}, err
	}
	if err := s.loadDailyMessages(ctx, &session); err != nil {
		return DailyPlanningSession{}, err
	}
	return session, nil
}

func (s *Store) loadDailyContext(ctx context.Context, session *DailyPlanningSession) error {
	var calendarJSON, githubJSON, routineJSON []byte
	var calendarStatus, githubStatus, routineStatus string
	err := s.pool.QueryRow(ctx, `
		SELECT calendar_context, github_context, routine_context,
		       calendar_status, github_status, routine_status, gathered_at
		FROM daily_planning_contexts
		WHERE session_id = $1
	`, session.ID).Scan(
		&calendarJSON, &githubJSON, &routineJSON,
		&calendarStatus, &githubStatus, &routineStatus, &session.Context.GatheredAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("load daily planning context: %w", err)
	}
	var calendar calendarContextEnvelope
	var github githubContextEnvelope
	var routines routineContextEnvelope
	if err := json.Unmarshal(calendarJSON, &calendar); err != nil {
		return fmt.Errorf("decode Calendar planning context: %w", err)
	}
	if err := json.Unmarshal(githubJSON, &github); err != nil {
		return fmt.Errorf("decode GitHub planning context: %w", err)
	}
	if err := json.Unmarshal(routineJSON, &routines); err != nil {
		return fmt.Errorf("decode routine planning context: %w", err)
	}
	session.Context.Calendar = calendar.Events
	session.Context.GitHub = github.Issues
	session.Context.Routines = routines.Routines
	session.Context.CalendarStatus = SourceStatus(calendarStatus)
	session.Context.GitHubStatus = SourceStatus(githubStatus)
	session.Context.RoutineStatus = SourceStatus(routineStatus)
	session.Context.CalendarError = calendar.Error
	session.Context.GitHubError = github.Error
	session.Context.RoutineError = routines.Error
	return nil
}

func (s *Store) loadDailyMessages(ctx context.Context, session *DailyPlanningSession) error {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, sequence, role::text, content, created_at
		FROM daily_planning_messages
		WHERE session_id = $1
		ORDER BY sequence
	`, session.ID)
	if err != nil {
		return fmt.Errorf("list daily planning messages: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var message DailyPlanningMessage
		if err := rows.Scan(&message.ID, &message.Sequence, &message.Role, &message.Content, &message.CreatedAt); err != nil {
			return fmt.Errorf("scan daily planning message: %w", err)
		}
		session.Messages = append(session.Messages, message)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate daily planning messages: %w", err)
	}
	return nil
}

// ReplaceDailyContext atomically replaces working evidence and invalidates its preview.
func (s *Store) ReplaceDailyContext(ctx context.Context, userID, sessionID string, value DailyPlanningContext, status DailySessionStatus, systemNote string) error {
	calendarJSON, err := json.Marshal(calendarContextEnvelope{Events: value.Calendar, Error: value.CalendarError})
	if err != nil {
		return fmt.Errorf("encode Calendar planning context: %w", err)
	}
	githubJSON, err := json.Marshal(githubContextEnvelope{Issues: value.GitHub, Error: value.GitHubError})
	if err != nil {
		return fmt.Errorf("encode GitHub planning context: %w", err)
	}
	routineJSON, err := json.Marshal(routineContextEnvelope{Routines: value.Routines, Error: value.RoutineError})
	if err != nil {
		return fmt.Errorf("encode routine planning context: %w", err)
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin context transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := ensureActiveSession(ctx, tx, userID, sessionID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO daily_planning_contexts (
			session_id, calendar_context, github_context, routine_context,
			calendar_status, github_status, routine_status, gathered_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (session_id) DO UPDATE SET
			calendar_context = EXCLUDED.calendar_context,
			github_context = EXCLUDED.github_context,
			routine_context = EXCLUDED.routine_context,
			calendar_status = EXCLUDED.calendar_status,
			github_status = EXCLUDED.github_status,
			routine_status = EXCLUDED.routine_status,
			gathered_at = EXCLUDED.gathered_at
	`, sessionID, calendarJSON, githubJSON, routineJSON,
		value.CalendarStatus, value.GitHubStatus, value.RoutineStatus, value.GatheredAt); err != nil {
		return fmt.Errorf("save daily planning context: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE daily_planning_sessions
		SET status = $3, preview_json = NULL, updated_at = now()
		WHERE id = $1 AND user_id = $2
	`, sessionID, userID, status); err != nil {
		return fmt.Errorf("update daily planning session context: %w", err)
	}
	if strings.TrimSpace(systemNote) != "" {
		if err := appendDailyMessage(ctx, tx, sessionID, MessageRoleSystem, systemNote); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit daily planning context: %w", err)
	}
	return nil
}

// AppendDailyMessage persists one user, assistant, or system turn.
func (s *Store) AppendDailyMessage(ctx context.Context, userID, sessionID string, role MessageRole, content string) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin daily planning message transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := ensureActiveSession(ctx, tx, userID, sessionID); err != nil {
		return err
	}
	if err := appendDailyMessage(ctx, tx, sessionID, role, content); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE daily_planning_sessions SET updated_at = now() WHERE id = $1`, sessionID); err != nil {
		return fmt.Errorf("touch daily planning session: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit daily planning message: %w", err)
	}
	return nil
}

func appendDailyMessage(ctx context.Context, tx pgx.Tx, sessionID string, role MessageRole, content string) error {
	var sequence int
	if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(sequence), 0) + 1 FROM daily_planning_messages WHERE session_id = $1`, sessionID).Scan(&sequence); err != nil {
		return fmt.Errorf("allocate daily planning message sequence: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO daily_planning_messages (session_id, sequence, role, content) VALUES ($1, $2, $3, $4)`, sessionID, sequence, role, content); err != nil {
		return fmt.Errorf("insert daily planning message: %w", err)
	}
	return nil
}

// SaveDailyPreview stores a validated structured proposal for explicit confirmation.
func (s *Store) SaveDailyPreview(ctx context.Context, userID, sessionID, modelID, promptVersion string, proposal DailyPlanProposal) error {
	encoded, err := json.Marshal(proposal)
	if err != nil {
		return fmt.Errorf("encode daily planning preview: %w", err)
	}
	command, err := s.pool.Exec(ctx, `
		UPDATE daily_planning_sessions
		SET status = 'reviewing', preview_json = $3, model_id = $4, prompt_version = $5, updated_at = now()
		WHERE id = $1 AND user_id = $2
		  AND status IN ('chatting', 'reviewing')
	`, sessionID, userID, encoded, modelID, promptVersion)
	if err != nil {
		return fmt.Errorf("save daily planning preview: %w", err)
	}
	if command.RowsAffected() != 1 {
		return ErrInvalidSessionState
	}
	return nil
}

// RecordGeneration stores non-content provider metadata for diagnostics.
func (s *Store) RecordGeneration(ctx context.Context, sessionID, purpose, modelID, generationID, promptVersion, status string, promptTokens, completionTokens int) error {
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO ai_generations (
			session_id, purpose, model_id, provider_generation_id, prompt_version,
			prompt_tokens, completion_tokens, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, sessionID, purpose, modelID, generationID, promptVersion, promptTokens, completionTokens, status); err != nil {
		return fmt.Errorf("record AI generation: %w", err)
	}
	return nil
}

// LoadModelAssignment returns the selected model for one owned planning purpose.
func (s *Store) LoadModelAssignment(ctx context.Context, userID, purpose string) (string, error) {
	var modelID string
	err := s.pool.QueryRow(ctx, `
		SELECT model_id FROM ai_model_assignments
		WHERE user_id = $1 AND purpose = $2
	`, userID, purpose).Scan(&modelID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("%w: choose a ZDR-compatible daily planning model in AI settings", ErrNotFound)
	}
	if err != nil {
		return "", fmt.Errorf("load AI model assignment: %w", err)
	}
	return modelID, nil
}

// ConfirmDailySession creates the next immutable revision and removes working content.
// The service validates the proposal against saved context before calling this method;
// ownership checks here protect the routine foreign references at the write boundary.
func (s *Store) ConfirmDailySession(ctx context.Context, userID, sessionID string, proposal DailyPlanProposal, value DailyPlanningContext) (string, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", fmt.Errorf("begin daily plan confirmation transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var date time.Time
	var status DailySessionStatus
	err = tx.QueryRow(ctx, `
		SELECT plan_date, status::text
		FROM daily_planning_sessions
		WHERE id = $1 AND user_id = $2
		FOR UPDATE
	`, sessionID, userID).Scan(&date, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("lock daily planning confirmation: %w", err)
	}
	if status != DailySessionReviewing {
		return "", ErrInvalidSessionState
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO plans (user_id, kind, period_start)
		VALUES ($1, 'daily', $2)
		ON CONFLICT (user_id, kind, period_start) DO NOTHING
	`, userID, date.Format(time.DateOnly)); err != nil {
		return "", fmt.Errorf("ensure daily plan: %w", err)
	}
	var planID string
	if err := tx.QueryRow(ctx, `
		SELECT id::text FROM plans
		WHERE user_id = $1 AND kind = 'daily' AND period_start = $2
		FOR UPDATE
	`, userID, date.Format(time.DateOnly)).Scan(&planID); err != nil {
		return "", fmt.Errorf("lock daily plan: %w", err)
	}
	var revision int
	if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(revision), 0) + 1 FROM plan_revisions WHERE plan_id = $1`, planID).Scan(&revision); err != nil {
		return "", fmt.Errorf("allocate daily plan revision: %w", err)
	}
	var revisionID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO plan_revisions (plan_id, revision, summary_markdown, content_markdown, notes_markdown)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''))
		RETURNING id::text
	`, planID, revision, proposal.SummaryMarkdown, proposal.ContentMarkdown, proposal.NotesMarkdown).Scan(&revisionID); err != nil {
		return "", fmt.Errorf("insert daily plan revision: %w", err)
	}
	if err := insertRevisionIssues(ctx, tx, revisionID, proposal.GitHubIssues, value.GitHub); err != nil {
		return "", err
	}
	if err := insertRevisionRoutines(ctx, tx, revisionID, userID, proposal.RoutineIDs); err != nil {
		return "", err
	}
	if err := insertRevisionActionItems(ctx, tx, revisionID, proposal.ActionItems); err != nil {
		return "", err
	}
	if err := insertRevisionCalendarEvents(ctx, tx, revisionID, proposal.CalendarEvents, value.Calendar); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM daily_planning_messages WHERE session_id = $1`, sessionID); err != nil {
		return "", fmt.Errorf("delete daily planning messages: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM daily_planning_contexts WHERE session_id = $1`, sessionID); err != nil {
		return "", fmt.Errorf("delete daily planning context: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE daily_planning_sessions
		SET status = 'finalized', preview_json = NULL, confirmed_plan_revision_id = $3,
		    finalized_at = now(), updated_at = now()
		WHERE id = $1 AND user_id = $2
	`, sessionID, userID, revisionID); err != nil {
		return "", fmt.Errorf("finalize daily planning session: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit daily plan confirmation: %w", err)
	}
	return revisionID, nil
}

func insertRevisionIssues(ctx context.Context, tx pgx.Tx, revisionID string, selected []PlannedGitHubIssue, contextIssues []GitHubIssue) error {
	byKey := make(map[string]GitHubIssue, len(contextIssues))
	for _, issue := range contextIssues {
		byKey[githubIssueKey(issue.RepositoryOwner, issue.RepositoryName, issue.Number)] = issue
	}
	for position, selectedIssue := range selected {
		issue, ok := byKey[githubIssueKey(selectedIssue.RepositoryOwner, selectedIssue.RepositoryName, selectedIssue.Number)]
		if !ok {
			return fmt.Errorf("%w: selected GitHub issue is no longer present in session context", ErrInvalidInput)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO plan_revision_github_issues (
				plan_revision_id, repository_owner, repository_name, issue_number, title, html_url, position
			) VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, revisionID, issue.RepositoryOwner, issue.RepositoryName, issue.Number, issue.Title, issue.HTMLURL, position); err != nil {
			return fmt.Errorf("insert confirmed GitHub issue: %w", err)
		}
	}
	return nil
}

func insertRevisionRoutines(ctx context.Context, tx pgx.Tx, revisionID, userID string, routineIDs []string) error {
	for position, routineID := range routineIDs {
		command, err := tx.Exec(ctx, `
			INSERT INTO plan_revision_routines (plan_revision_id, routine_id, position)
			SELECT $1, id, $3 FROM routines WHERE id = $2 AND user_id = $4
		`, revisionID, routineID, position, userID)
		if err != nil {
			return fmt.Errorf("insert confirmed routine: %w", err)
		}
		if command.RowsAffected() != 1 {
			return fmt.Errorf("%w: selected routine is not owned by this user", ErrInvalidInput)
		}
	}
	return nil
}

func insertRevisionActionItems(ctx context.Context, tx pgx.Tx, revisionID string, items []ActionItem) error {
	for position, item := range items {
		if _, err := tx.Exec(ctx, `
			INSERT INTO plan_revision_action_items (plan_revision_id, title, note, position)
			VALUES ($1, $2, $3, $4)
		`, revisionID, item.Title, item.Note, position); err != nil {
			return fmt.Errorf("insert confirmed action item: %w", err)
		}
	}
	return nil
}

func insertRevisionCalendarEvents(ctx context.Context, tx pgx.Tx, revisionID string, selected []PlannedCalendarEvent, contextEvents []CalendarEvent) error {
	byKey := make(map[string]CalendarEvent, len(contextEvents))
	for _, event := range contextEvents {
		byKey[calendarEventKey(event.CalendarSourceID, event.ExternalEventID)] = event
	}
	for position, selectedEvent := range selected {
		event, ok := byKey[calendarEventKey(selectedEvent.CalendarSourceID, selectedEvent.ExternalEventID)]
		if !ok {
			return fmt.Errorf("%w: selected Calendar event is no longer present in session context", ErrInvalidInput)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO plan_revision_calendar_events (
				plan_revision_id, calendar_source_id, external_event_id, title, starts_at, ends_at,
				start_date, end_date, all_day, role, html_url, position
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		`, revisionID, event.CalendarSourceID, event.ExternalEventID, event.Title, event.StartsAt, event.EndsAt,
			event.StartDate, event.EndDate, event.AllDay, event.Role, event.HTMLURL, position); err != nil {
			return fmt.Errorf("insert confirmed Calendar event: %w", err)
		}
	}
	return nil
}

func githubIssueKey(owner, name string, number int) string {
	return strings.ToLower(owner) + "/" + strings.ToLower(name) + "#" + fmt.Sprint(number)
}

func calendarEventKey(sourceID, eventID string) string { return sourceID + ":" + eventID }

func ensureActiveSession(ctx context.Context, tx pgx.Tx, userID, sessionID string) error {
	var status DailySessionStatus
	err := tx.QueryRow(ctx, `SELECT status::text FROM daily_planning_sessions WHERE id = $1 AND user_id = $2 FOR UPDATE`, sessionID, userID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("lock daily planning session: %w", err)
	}
	if status != DailySessionGathering && status != DailySessionContextBlocked && status != DailySessionChatting && status != DailySessionReviewing {
		return ErrInvalidSessionState
	}
	return nil
}
