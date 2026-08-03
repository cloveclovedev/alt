package planning

import (
	"sort"
	"time"
)

// DailySessionStatus identifies the lifecycle phase of a daily planning session.
type DailySessionStatus string

const (
	DailySessionGathering      DailySessionStatus = "gathering"
	DailySessionContextBlocked DailySessionStatus = "context_blocked"
	DailySessionChatting       DailySessionStatus = "chatting"
	DailySessionReviewing      DailySessionStatus = "reviewing"
	DailySessionFinalized      DailySessionStatus = "finalized"
	DailySessionCancelled      DailySessionStatus = "cancelled"
)

// SourceStatus describes whether one deterministic context source is usable.
type SourceStatus string

const (
	SourceStatusLoaded      SourceStatus = "loaded"
	SourceStatusFailed      SourceStatus = "failed"
	SourceStatusUnavailable SourceStatus = "unavailable"
)

// MessageRole identifies the author of a persisted planning message.
type MessageRole string

const (
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
	MessageRoleSystem    MessageRole = "system"
)

// CalendarRole conveys how calendar events should influence a plan.
type CalendarRole string

const (
	CalendarRoleCommitment CalendarRole = "commitment"
	CalendarRoleOptional   CalendarRole = "optional"
	CalendarRoleContext    CalendarRole = "context"
)

// DailyPlanningMessage is one persisted chat turn.
type DailyPlanningMessage struct {
	ID        string
	Sequence  int
	Role      MessageRole
	Content   string
	CreatedAt time.Time
}

// GitHubIssue is application-owned normalized GitHub planning context.
type GitHubIssue struct {
	RepositoryOwner string    `json:"repository_owner"`
	RepositoryName  string    `json:"repository_name"`
	Number          int       `json:"number"`
	Title           string    `json:"title"`
	HTMLURL         string    `json:"html_url"`
	Labels          []string  `json:"labels"`
	Milestone       string    `json:"milestone,omitempty"`
	Assignees       []string  `json:"assignees"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// CalendarEvent is application-owned normalized Calendar planning context.
type CalendarEvent struct {
	CalendarSourceID string       `json:"calendar_source_id"`
	ExternalEventID  string       `json:"external_event_id"`
	CalendarName     string       `json:"calendar_name"`
	Role             CalendarRole `json:"role"`
	Instructions     string       `json:"instructions,omitempty"`
	Title            string       `json:"title"`
	Description      string       `json:"description,omitempty"`
	Location         string       `json:"location,omitempty"`
	StartsAt         *time.Time   `json:"starts_at,omitempty"`
	EndsAt           *time.Time   `json:"ends_at,omitempty"`
	StartDate        *time.Time   `json:"start_date,omitempty"`
	EndDate          *time.Time   `json:"end_date,omitempty"`
	AllDay           bool         `json:"all_day"`
	HTMLURL          string       `json:"html_url,omitempty"`
}

// RoutineCandidate is the normalized routine context used by daily planning.
type RoutineCandidate struct {
	RoutineID       string     `json:"routine_id"`
	Name            string     `json:"name"`
	CategoryName    string     `json:"category_name"`
	State           string     `json:"state"`
	DueOn           *time.Time `json:"due_on,omitempty"`
	NextAvailableOn *time.Time `json:"next_available_on,omitempty"`
	Notes           string     `json:"notes,omitempty"`
}

// NutritionFacts is compact, provider-neutral nutrition context for daily
// planning: yesterday's totals versus its target, today's intake so far, and a
// trailing-window average. It never contains individual entries, so the model
// cannot reconstruct or fabricate a log from it. Its fields mirror the nutrition
// feature's own planning-facts type so the composition root maps between them
// without either feature importing the other.
type NutritionFacts struct {
	YesterdayCaloriesKcal       int     `json:"yesterday_calories_kcal"`
	YesterdayProteinG           float64 `json:"yesterday_protein_g"`
	YesterdayTargetCaloriesKcal int     `json:"yesterday_target_calories_kcal,omitempty"`
	YesterdayTargetProteinG     float64 `json:"yesterday_target_protein_g,omitempty"`
	TodayCaloriesKcal           int     `json:"today_calories_kcal"`
	TodayProteinG               float64 `json:"today_protein_g"`
	TrailingDays                int     `json:"trailing_days"`
	TrailingAvgCaloriesKcal     int     `json:"trailing_avg_calories_kcal"`
	TrailingAvgProteinG         float64 `json:"trailing_avg_protein_g"`
	TargetCaloriesKcal          int     `json:"target_calories_kcal,omitempty"`
	TargetProteinG              float64 `json:"target_protein_g,omitempty"`
}

// DailyPlanningContext is saved working evidence, never a raw provider response.
type DailyPlanningContext struct {
	Calendar        []CalendarEvent    `json:"calendar"`
	GitHub          []GitHubIssue      `json:"github"`
	Routines        []RoutineCandidate `json:"routines"`
	Nutrition       *NutritionFacts    `json:"nutrition,omitempty"`
	CalendarStatus  SourceStatus       `json:"calendar_status"`
	GitHubStatus    SourceStatus       `json:"github_status"`
	RoutineStatus   SourceStatus       `json:"routine_status"`
	NutritionStatus SourceStatus       `json:"nutrition_status"`
	CalendarError   string             `json:"calendar_error,omitempty"`
	GitHubError     string             `json:"github_error,omitempty"`
	RoutineError    string             `json:"routine_error,omitempty"`
	NutritionError  string             `json:"nutrition_error,omitempty"`
	GatheredAt      time.Time          `json:"gathered_at"`
}

// PlannedGitHubIssue identifies a validated issue selected by the proposal.
type PlannedGitHubIssue struct {
	RepositoryOwner string `json:"repository_owner"`
	RepositoryName  string `json:"repository_name"`
	Number          int    `json:"number"`
}

// PlannedCalendarEvent identifies a validated Calendar event selected by a proposal.
type PlannedCalendarEvent struct {
	CalendarSourceID string `json:"calendar_source_id"`
	ExternalEventID  string `json:"external_event_id"`
}

// ActionItem is free-form work local to one plan revision.
type ActionItem struct {
	Title string `json:"title"`
	Note  string `json:"note"`
}

// DailyPlanProposal is the structured model output for every turn: a
// natural-language reply plus the current draft of the plan.
type DailyPlanProposal struct {
	AssistantMessage string                 `json:"assistant_message"`
	SummaryMarkdown  string                 `json:"summary_markdown"`
	ContentMarkdown  string                 `json:"content_markdown"`
	NotesMarkdown    string                 `json:"notes_markdown"`
	GitHubIssues     []PlannedGitHubIssue   `json:"github_issues"`
	RoutineIDs       []string               `json:"routine_ids"`
	CalendarEvents   []PlannedCalendarEvent `json:"calendar_events"`
	ActionItems      []ActionItem           `json:"action_items"`
	Unavailable      []string               `json:"unavailable_sources"`
}

// DailyPlanningSession is the transport-neutral session view.
type DailyPlanningSession struct {
	ID                      string
	PlanDate                time.Time
	Status                  DailySessionStatus
	Context                 DailyPlanningContext
	Messages                []DailyPlanningMessage
	Preview                 *DailyPlanProposal
	ConfirmedPlanRevisionID string
	ModelID                 string
	PromptVersion           string
	CreatedAt               time.Time
	UpdatedAt               time.Time
	FinalizedAt             *time.Time
}

// LatestMessage returns the most recent turn, or nil when the conversation is
// empty. It is shown expanded while earlier turns collapse, keeping the page
// short as the conversation grows.
func (s *DailyPlanningSession) LatestMessage() *DailyPlanningMessage {
	if len(s.Messages) == 0 {
		return nil
	}
	return &s.Messages[len(s.Messages)-1]
}

// EarlierMessages returns every turn before the latest, for the collapsed history.
func (s *DailyPlanningSession) EarlierMessages() []DailyPlanningMessage {
	if len(s.Messages) <= 1 {
		return nil
	}
	return s.Messages[:len(s.Messages)-1]
}

// DailyPlanningView is the complete web view for one local date.
type DailyPlanningView struct {
	Date     time.Time
	Timezone string
	Session  *DailyPlanningSession
	Plan     *Plan
	// PreviewComponents holds the reviewing session's proposal joined against
	// its context so titles and links can render. It is nil unless the session
	// is reviewing an unconfirmed preview.
	PreviewComponents *PlanComponents
}

// PlanComponents holds the structured selections of one revision for display.
// Prose lives in Markdown; these are rendered as their own lists beside it.
type PlanComponents struct {
	GitHubIssues   []PlanGitHubIssue
	Routines       []PlanRoutine
	ActionItems    []ActionItem
	CalendarEvents []PlanCalendarEvent
}

// Empty reports whether no structured selection exists.
func (c PlanComponents) Empty() bool {
	return len(c.GitHubIssues) == 0 && len(c.Routines) == 0 &&
		len(c.ActionItems) == 0 && len(c.CalendarEvents) == 0
}

// PlanGitHubIssue is a selected issue with its display snapshot.
type PlanGitHubIssue struct {
	RepositoryOwner string
	RepositoryName  string
	Number          int
	Title           string
	HTMLURL         string
}

// Repository is the "owner/name" label for a selected issue.
func (i PlanGitHubIssue) Repository() string {
	return i.RepositoryOwner + "/" + i.RepositoryName
}

// PlanRoutine is a selected routine's display snapshot.
type PlanRoutine struct {
	RoutineID      string
	Name           string
	CategoryName   string
	CompletedToday bool
}

// PlanCalendarEvent is a selected calendar event reduced to a minimal reference.
type PlanCalendarEvent struct {
	TimeLabel string
	Title     string
	Role      CalendarRole
	HTMLURL   string
}

// previewComponents joins a proposal's selections against session context so the
// draft can display titles, links, and times. Calendar events are not a model
// choice: the plan date's events from context are shown as the day's constraints.
func previewComponents(proposal DailyPlanProposal, value DailyPlanningContext, planDate time.Time, location *time.Location) PlanComponents {
	components := PlanComponents{ActionItems: proposal.ActionItems}

	issuesByKey := make(map[string]GitHubIssue, len(value.GitHub))
	for _, issue := range value.GitHub {
		issuesByKey[githubIssueKey(issue.RepositoryOwner, issue.RepositoryName, issue.Number)] = issue
	}
	for _, selected := range proposal.GitHubIssues {
		if issue, ok := issuesByKey[githubIssueKey(selected.RepositoryOwner, selected.RepositoryName, selected.Number)]; ok {
			components.GitHubIssues = append(components.GitHubIssues, PlanGitHubIssue{
				RepositoryOwner: issue.RepositoryOwner,
				RepositoryName:  issue.RepositoryName,
				Number:          issue.Number,
				Title:           issue.Title,
				HTMLURL:         issue.HTMLURL,
			})
		}
	}

	routinesByID := make(map[string]RoutineCandidate, len(value.Routines))
	for _, routine := range value.Routines {
		routinesByID[routine.RoutineID] = routine
	}
	for _, id := range proposal.RoutineIDs {
		if routine, ok := routinesByID[id]; ok {
			components.Routines = append(components.Routines, PlanRoutine{RoutineID: routine.RoutineID, Name: routine.Name, CategoryName: routine.CategoryName})
		}
	}

	for _, event := range todaysCalendarEvents(value.Calendar, planDate, location) {
		components.CalendarEvents = append(components.CalendarEvents, PlanCalendarEvent{
			TimeLabel: calendarTimeLabel(event.AllDay, event.StartsAt, event.EndsAt, location),
			Title:     event.Title,
			Role:      event.Role,
			HTMLURL:   event.HTMLURL,
		})
	}
	return components
}

// todaysCalendarEvents returns the context events that occur on the plan date,
// in start order. Calendar events are the day's constraints, shown in full and
// deterministically rather than selected by the model.
func todaysCalendarEvents(events []CalendarEvent, planDate time.Time, location *time.Location) []CalendarEvent {
	day := time.Date(planDate.Year(), planDate.Month(), planDate.Day(), 0, 0, 0, 0, location)
	var out []CalendarEvent
	for _, event := range events {
		start, end, ok := eventDayRange(event, location)
		if !ok {
			continue
		}
		if !day.Before(start) && day.Before(end) {
			out = append(out, event)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return calendarStart(out[i]).Before(calendarStart(out[j])) })
	return out
}

// eventDayRange returns the [start, end) day range an event covers, where end is
// exclusive. All-day end dates are already exclusive; timed events extend to the
// day after their end.
func eventDayRange(event CalendarEvent, location *time.Location) (start, end time.Time, ok bool) {
	dayOf := func(t time.Time) time.Time {
		t = t.In(location)
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, location)
	}
	if event.AllDay {
		if event.StartDate == nil {
			return time.Time{}, time.Time{}, false
		}
		start = dayOf(*event.StartDate)
		if event.EndDate != nil {
			end = dayOf(*event.EndDate)
		}
		if !end.After(start) {
			end = start.AddDate(0, 0, 1)
		}
		return start, end, true
	}
	if event.StartsAt == nil {
		return time.Time{}, time.Time{}, false
	}
	start = dayOf(*event.StartsAt)
	if event.EndsAt != nil {
		end = dayOf(*event.EndsAt).AddDate(0, 0, 1)
	}
	if !end.After(start) {
		end = start.AddDate(0, 0, 1)
	}
	return start, end, true
}

func calendarStart(event CalendarEvent) time.Time {
	if event.StartsAt != nil {
		return *event.StartsAt
	}
	if event.StartDate != nil {
		return *event.StartDate
	}
	return time.Time{}
}

// plannedCalendarEvents projects context events to the confirmation input shape.
func plannedCalendarEvents(events []CalendarEvent) []PlannedCalendarEvent {
	planned := make([]PlannedCalendarEvent, 0, len(events))
	for _, event := range events {
		planned = append(planned, PlannedCalendarEvent{CalendarSourceID: event.CalendarSourceID, ExternalEventID: event.ExternalEventID})
	}
	return planned
}

// calendarTimeLabel formats an event as a compact time reference in the local zone.
func calendarTimeLabel(allDay bool, startsAt, endsAt *time.Time, location *time.Location) string {
	if allDay || startsAt == nil {
		return "All day"
	}
	label := startsAt.In(location).Format("15:04")
	if endsAt != nil {
		label += "–" + endsAt.In(location).Format("15:04")
	}
	return label
}
