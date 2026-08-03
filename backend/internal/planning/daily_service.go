package planning

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const dailyPlanningPurpose = "daily_planning"
const dailyPlanningPromptVersion = "daily-planning-v3"

// DailyContextGatherer owns deterministic external I/O for planning evidence.
type DailyContextGatherer interface {
	GatherDailyPlanningContext(context.Context, string, time.Time, *time.Location) DailyPlanningContext
}

// DailyAI is the provider-neutral model boundary used by a planning session.
// Every turn returns one structured proposal: a natural-language reply plus the
// current draft of the plan.
type DailyAI interface {
	Propose(context.Context, string, DailyPlanningContext, []DailyPlanningMessage) (AIGeneration, DailyPlanProposal, error)
}

// AIGeneration carries a completed response only in memory; stores persist its metadata alone.
type AIGeneration struct {
	ID               string
	PromptTokens     int
	CompletionTokens int
	Response         string
}

// DailyService owns the state machine for a daily planning conversation.
type DailyService struct {
	store    *Store
	userID   string
	location *time.Location
	gatherer DailyContextGatherer
	ai       DailyAI
	now      func() time.Time
}

// NewDailyService constructs daily planning behavior from explicit boundaries.
func NewDailyService(store *Store, userID, timezone string, gatherer DailyContextGatherer, ai DailyAI) (*DailyService, error) {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("load timezone: %w", err)
	}
	return &DailyService{store: store, userID: userID, location: location, gatherer: gatherer, ai: ai, now: time.Now}, nil
}

// StartOrResume returns the active session for a date, gathering context once when newly created.
func (s *DailyService) StartOrResume(ctx context.Context, rawDate string) (DailyPlanningView, error) {
	date, err := s.parseDailyDate(rawDate)
	if err != nil {
		return DailyPlanningView{}, err
	}
	session, created, err := s.store.FindOrCreateDailySession(ctx, s.userID, date)
	if err != nil {
		return DailyPlanningView{}, err
	}
	if created {
		if err := s.gather(ctx, session.ID, date, false); err != nil {
			return DailyPlanningView{}, err
		}
		session, err = s.store.LoadDailySession(ctx, s.userID, session.ID)
		if err != nil {
			return DailyPlanningView{}, err
		}
	}
	return s.dailyView(ctx, date, &session)
}

// RefreshContext explicitly replaces session evidence and invalidates its preview.
func (s *DailyService) RefreshContext(ctx context.Context, rawDate, sessionID string) (DailyPlanningView, error) {
	date, err := s.parseDailyDate(rawDate)
	if err != nil {
		return DailyPlanningView{}, err
	}
	if _, err := s.store.LoadDailySession(ctx, s.userID, sessionID); err != nil {
		return DailyPlanningView{}, err
	}
	if err := s.gather(ctx, sessionID, date, true); err != nil {
		return DailyPlanningView{}, err
	}
	session, err := s.store.LoadDailySession(ctx, s.userID, sessionID)
	if err != nil {
		return DailyPlanningView{}, err
	}
	return s.dailyView(ctx, date, &session)
}

func (s *DailyService) gather(ctx context.Context, sessionID string, date time.Time, refreshed bool) error {
	if s.gatherer == nil {
		return fmt.Errorf("daily planning context gatherer is not configured")
	}
	value := s.gatherer.GatherDailyPlanningContext(ctx, s.userID, date, s.location)
	value.GatheredAt = s.now().UTC()
	status := DailySessionChatting
	if hasFailedSource(value) {
		status = DailySessionContextBlocked
	}
	note := ""
	if refreshed {
		note = "Context was refreshed. Any unconfirmed preview was cleared."
	}
	return s.store.ReplaceDailyContext(ctx, s.userID, sessionID, value, status, note)
}

// ContinueWithoutFailedSources records the user's explicit decision to proceed.
func (s *DailyService) ContinueWithoutFailedSources(ctx context.Context, rawDate, sessionID string) (DailyPlanningView, error) {
	date, err := s.parseDailyDate(rawDate)
	if err != nil {
		return DailyPlanningView{}, err
	}
	session, err := s.store.LoadDailySession(ctx, s.userID, sessionID)
	if err != nil {
		return DailyPlanningView{}, err
	}
	if session.Status != DailySessionContextBlocked {
		return DailyPlanningView{}, ErrInvalidSessionState
	}
	value := session.Context
	if value.CalendarStatus == SourceStatusFailed {
		value.CalendarStatus, value.CalendarError = SourceStatusUnavailable, ""
	}
	if value.GitHubStatus == SourceStatusFailed {
		value.GitHubStatus, value.GitHubError = SourceStatusUnavailable, ""
	}
	if value.RoutineStatus == SourceStatusFailed {
		value.RoutineStatus, value.RoutineError = SourceStatusUnavailable, ""
	}
	if err := s.store.ReplaceDailyContext(ctx, s.userID, sessionID, value, DailySessionChatting, "Continuing without the unavailable context source."); err != nil {
		return DailyPlanningView{}, err
	}
	session, err = s.store.LoadDailySession(ctx, s.userID, sessionID)
	if err != nil {
		return DailyPlanningView{}, err
	}
	return s.dailyView(ctx, date, &session)
}

// SendMessage saves a user turn before requesting a complete non-streaming assistant response.
func (s *DailyService) SendMessage(ctx context.Context, rawDate, sessionID, content string) (DailyPlanningView, error) {
	date, err := s.parseDailyDate(rawDate)
	if err != nil {
		return DailyPlanningView{}, err
	}
	content, err = validateMessage(content)
	if err != nil {
		return DailyPlanningView{}, err
	}
	if err := s.store.AppendDailyMessage(ctx, s.userID, sessionID, MessageRoleUser, content); err != nil {
		return DailyPlanningView{}, err
	}
	return s.completeAssistantTurn(ctx, date, sessionID)
}

// RetryLastMessage repeats a failed model request without adding another user message.
func (s *DailyService) RetryLastMessage(ctx context.Context, rawDate, sessionID string) (DailyPlanningView, error) {
	date, err := s.parseDailyDate(rawDate)
	if err != nil {
		return DailyPlanningView{}, err
	}
	session, err := s.store.LoadDailySession(ctx, s.userID, sessionID)
	if err != nil {
		return DailyPlanningView{}, err
	}
	if session.Status != DailySessionChatting || len(session.Messages) == 0 || session.Messages[len(session.Messages)-1].Role != MessageRoleUser {
		return DailyPlanningView{}, ErrInvalidSessionState
	}
	return s.completeAssistantTurn(ctx, date, sessionID)
}

func (s *DailyService) completeAssistantTurn(ctx context.Context, date time.Time, sessionID string) (DailyPlanningView, error) {
	session, err := s.store.LoadDailySession(ctx, s.userID, sessionID)
	if err != nil {
		return DailyPlanningView{}, err
	}
	if (session.Status != DailySessionChatting && session.Status != DailySessionReviewing) ||
		len(session.Messages) == 0 || session.Messages[len(session.Messages)-1].Role != MessageRoleUser {
		return DailyPlanningView{}, ErrInvalidSessionState
	}
	modelID, err := s.store.LoadModelAssignment(ctx, s.userID, dailyPlanningPurpose)
	if err != nil {
		return DailyPlanningView{}, err
	}
	if s.ai == nil {
		return DailyPlanningView{}, fmt.Errorf("daily planning AI is not configured")
	}
	generation, proposal, err := s.ai.Propose(ctx, modelID, session.Context, session.Messages)
	if err != nil {
		_ = s.store.RecordGeneration(ctx, session.ID, dailyPlanningPurpose, modelID, "", dailyPlanningPromptVersion, "failed", 0, 0)
		return DailyPlanningView{}, fmt.Errorf("request daily planning model: %w", err)
	}
	if strings.TrimSpace(generation.ID) == "" {
		generation.ID = "unavailable"
	}
	if err := s.store.RecordGeneration(ctx, session.ID, dailyPlanningPurpose, modelID, generation.ID, dailyPlanningPromptVersion, "succeeded", generation.PromptTokens, generation.CompletionTokens); err != nil {
		return DailyPlanningView{}, err
	}
	// The assistant's conversational reply is the saved chat message; the rest of
	// the proposal becomes the current unconfirmed preview.
	message := strings.TrimSpace(proposal.AssistantMessage)
	if message == "" {
		message = "(No reply text was returned. See the plan draft below.)"
	}
	if err := s.store.AppendDailyMessage(ctx, s.userID, sessionID, MessageRoleAssistant, message); err != nil {
		return DailyPlanningView{}, err
	}
	if err := s.store.SaveDailyPreview(ctx, s.userID, sessionID, modelID, dailyPlanningPromptVersion, sanitizeProposal(proposal, session.Context)); err != nil {
		return DailyPlanningView{}, err
	}
	updated, err := s.store.LoadDailySession(ctx, s.userID, sessionID)
	if err != nil {
		return DailyPlanningView{}, err
	}
	return s.dailyView(ctx, date, &updated)
}

// Confirm creates one immutable revision after revalidating its preview.
func (s *DailyService) Confirm(ctx context.Context, rawDate, sessionID string) (DailyPlanningView, error) {
	date, err := s.parseDailyDate(rawDate)
	if err != nil {
		return DailyPlanningView{}, err
	}
	session, err := s.store.LoadDailySession(ctx, s.userID, sessionID)
	if err != nil {
		return DailyPlanningView{}, err
	}
	if session.Status != DailySessionReviewing || session.Preview == nil {
		return DailyPlanningView{}, ErrInvalidSessionState
	}
	if err := validateProposal(*session.Preview, session.Context); err != nil {
		return DailyPlanningView{}, err
	}
	if _, err := s.store.ConfirmDailySession(ctx, s.userID, sessionID, *session.Preview, session.Context); err != nil {
		return DailyPlanningView{}, err
	}
	return s.dailyView(ctx, date, nil)
}

func (s *DailyService) dailyView(ctx context.Context, date time.Time, session *DailyPlanningSession) (DailyPlanningView, error) {
	view := DailyPlanningView{Date: date, Timezone: s.location.String(), Session: session}
	planView, err := s.store.Load(ctx, s.userID, KindDaily, date, s.location)
	if err != nil {
		return DailyPlanningView{}, err
	}
	view.Plan = planView.Plan
	if session != nil && session.Preview != nil {
		components := previewComponents(*session.Preview, session.Context, s.location)
		view.PreviewComponents = &components
	}
	return view, nil
}

func (s *DailyService) parseDailyDate(raw string) (time.Time, error) {
	date, err := time.ParseInLocation(time.DateOnly, strings.TrimSpace(raw), s.location)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: date must use YYYY-MM-DD", ErrInvalidInput)
	}
	return date, nil
}

func (s *DailyService) localToday() time.Time {
	now := s.now().In(s.location)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, s.location)
}

func hasFailedSource(value DailyPlanningContext) bool {
	return value.CalendarStatus == SourceStatusFailed || value.GitHubStatus == SourceStatusFailed || value.RoutineStatus == SourceStatusFailed
}

func validateMessage(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%w: message is required", ErrInvalidInput)
	}
	if len([]rune(value)) > 20_000 {
		return "", fmt.Errorf("%w: message is too long", ErrInvalidInput)
	}
	return value, nil
}

// sanitizeProposal drops selections that are not present in the session context,
// de-duplicates them, and removes empty or false entries, so an in-progress draft
// can be previewed and later confirmed without a stray hallucinated reference
// blocking confirmation. It does not enforce the length and required-prose rules;
// those remain strict checks applied only at confirmation.
func sanitizeProposal(proposal DailyPlanProposal, value DailyPlanningContext) DailyPlanProposal {
	knownIssues := make(map[string]struct{}, len(value.GitHub))
	for _, issue := range value.GitHub {
		knownIssues[githubIssueKey(issue.RepositoryOwner, issue.RepositoryName, issue.Number)] = struct{}{}
	}
	issues := make([]PlannedGitHubIssue, 0, len(proposal.GitHubIssues))
	seenIssue := make(map[string]struct{})
	for _, sel := range proposal.GitHubIssues {
		owner, name := strings.TrimSpace(sel.RepositoryOwner), strings.TrimSpace(sel.RepositoryName)
		key := githubIssueKey(owner, name, sel.Number)
		if _, ok := knownIssues[key]; !ok {
			continue
		}
		if _, dup := seenIssue[key]; dup {
			continue
		}
		seenIssue[key] = struct{}{}
		issues = append(issues, PlannedGitHubIssue{RepositoryOwner: owner, RepositoryName: name, Number: sel.Number})
	}
	proposal.GitHubIssues = issues

	knownRoutines := make(map[string]struct{}, len(value.Routines))
	for _, candidate := range value.Routines {
		knownRoutines[candidate.RoutineID] = struct{}{}
	}
	routineIDs := make([]string, 0, len(proposal.RoutineIDs))
	seenRoutine := make(map[string]struct{})
	for _, id := range proposal.RoutineIDs {
		id = strings.TrimSpace(id)
		if _, ok := knownRoutines[id]; !ok {
			continue
		}
		if _, dup := seenRoutine[id]; dup {
			continue
		}
		seenRoutine[id] = struct{}{}
		routineIDs = append(routineIDs, id)
	}
	proposal.RoutineIDs = routineIDs

	knownEvents := make(map[string]struct{}, len(value.Calendar))
	for _, event := range value.Calendar {
		knownEvents[calendarEventKey(event.CalendarSourceID, event.ExternalEventID)] = struct{}{}
	}
	events := make([]PlannedCalendarEvent, 0, len(proposal.CalendarEvents))
	seenEvent := make(map[string]struct{})
	for _, sel := range proposal.CalendarEvents {
		source, external := strings.TrimSpace(sel.CalendarSourceID), strings.TrimSpace(sel.ExternalEventID)
		key := calendarEventKey(source, external)
		if _, ok := knownEvents[key]; !ok {
			continue
		}
		if _, dup := seenEvent[key]; dup {
			continue
		}
		seenEvent[key] = struct{}{}
		events = append(events, PlannedCalendarEvent{CalendarSourceID: source, ExternalEventID: external})
	}
	proposal.CalendarEvents = events

	actions := make([]ActionItem, 0, len(proposal.ActionItems))
	for _, item := range proposal.ActionItems {
		if strings.TrimSpace(item.Title) == "" {
			continue
		}
		actions = append(actions, item)
	}
	proposal.ActionItems = actions

	unavailable := make([]string, 0, len(proposal.Unavailable))
	for _, source := range proposal.Unavailable {
		switch strings.TrimSpace(source) {
		case "calendar":
			if value.CalendarStatus == SourceStatusUnavailable {
				unavailable = append(unavailable, "calendar")
			}
		case "github":
			if value.GitHubStatus == SourceStatusUnavailable {
				unavailable = append(unavailable, "github")
			}
		case "routines":
			if value.RoutineStatus == SourceStatusUnavailable {
				unavailable = append(unavailable, "routines")
			}
		}
	}
	proposal.Unavailable = unavailable

	return proposal
}

func validateProposal(proposal DailyPlanProposal, value DailyPlanningContext) error {
	proposal.SummaryMarkdown = strings.TrimSpace(proposal.SummaryMarkdown)
	proposal.ContentMarkdown = strings.TrimSpace(proposal.ContentMarkdown)
	if proposal.SummaryMarkdown == "" || proposal.ContentMarkdown == "" {
		return fmt.Errorf("%w: proposal summary and content are required", ErrInvalidInput)
	}
	if len([]rune(proposal.SummaryMarkdown)) > 5_000 || len([]rune(proposal.ContentMarkdown)) > 50_000 || len([]rune(proposal.NotesMarkdown)) > 20_000 {
		return fmt.Errorf("%w: proposal Markdown is too long", ErrInvalidInput)
	}
	if len(proposal.GitHubIssues) > 50 || len(proposal.RoutineIDs) > 50 || len(proposal.CalendarEvents) > 100 || len(proposal.ActionItems) > 100 {
		return fmt.Errorf("%w: proposal has too many selected items", ErrInvalidInput)
	}
	issues := make(map[string]struct{}, len(value.GitHub))
	for _, issue := range value.GitHub {
		issues[githubIssueKey(issue.RepositoryOwner, issue.RepositoryName, issue.Number)] = struct{}{}
	}
	seenIssues := make(map[string]struct{}, len(proposal.GitHubIssues))
	for _, issue := range proposal.GitHubIssues {
		key := githubIssueKey(strings.TrimSpace(issue.RepositoryOwner), strings.TrimSpace(issue.RepositoryName), issue.Number)
		if issue.Number < 1 || strings.TrimSpace(issue.RepositoryOwner) == "" || strings.TrimSpace(issue.RepositoryName) == "" {
			return fmt.Errorf("%w: invalid selected GitHub issue", ErrInvalidInput)
		}
		if _, ok := issues[key]; !ok {
			return fmt.Errorf("%w: proposal references a GitHub issue absent from session context", ErrInvalidInput)
		}
		if _, duplicate := seenIssues[key]; duplicate {
			return fmt.Errorf("%w: proposal selects a GitHub issue more than once", ErrInvalidInput)
		}
		seenIssues[key] = struct{}{}
	}
	routines := make(map[string]struct{}, len(value.Routines))
	for _, candidate := range value.Routines {
		routines[candidate.RoutineID] = struct{}{}
	}
	seenRoutines := make(map[string]struct{}, len(proposal.RoutineIDs))
	for _, routineID := range proposal.RoutineIDs {
		routineID = strings.TrimSpace(routineID)
		if _, ok := routines[routineID]; !ok {
			return fmt.Errorf("%w: proposal references a routine absent from session context", ErrInvalidInput)
		}
		if _, duplicate := seenRoutines[routineID]; duplicate {
			return fmt.Errorf("%w: proposal selects a routine more than once", ErrInvalidInput)
		}
		seenRoutines[routineID] = struct{}{}
	}
	events := make(map[string]struct{}, len(value.Calendar))
	for _, event := range value.Calendar {
		events[calendarEventKey(event.CalendarSourceID, event.ExternalEventID)] = struct{}{}
	}
	seenEvents := make(map[string]struct{}, len(proposal.CalendarEvents))
	for _, event := range proposal.CalendarEvents {
		key := calendarEventKey(strings.TrimSpace(event.CalendarSourceID), strings.TrimSpace(event.ExternalEventID))
		if _, ok := events[key]; !ok {
			return fmt.Errorf("%w: proposal references a Calendar event absent from session context", ErrInvalidInput)
		}
		if _, duplicate := seenEvents[key]; duplicate {
			return fmt.Errorf("%w: proposal selects a Calendar event more than once", ErrInvalidInput)
		}
		seenEvents[key] = struct{}{}
	}
	for _, item := range proposal.ActionItems {
		if strings.TrimSpace(item.Title) == "" || len([]rune(item.Title)) > 500 || len([]rune(item.Note)) > 5_000 {
			return fmt.Errorf("%w: invalid action item", ErrInvalidInput)
		}
	}
	for _, source := range proposal.Unavailable {
		switch strings.TrimSpace(source) {
		case "calendar":
			if value.CalendarStatus != SourceStatusUnavailable {
				return fmt.Errorf("%w: Calendar was available", ErrInvalidInput)
			}
		case "github":
			if value.GitHubStatus != SourceStatusUnavailable {
				return fmt.Errorf("%w: GitHub was available", ErrInvalidInput)
			}
		case "routines":
			if value.RoutineStatus != SourceStatusUnavailable {
				return fmt.Errorf("%w: routines were available", ErrInvalidInput)
			}
		default:
			return fmt.Errorf("%w: invalid unavailable source", ErrInvalidInput)
		}
	}
	return nil
}
