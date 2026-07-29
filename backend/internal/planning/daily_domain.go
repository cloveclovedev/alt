package planning

import "time"

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

// DailyPlanningContext is saved working evidence, never a raw provider response.
type DailyPlanningContext struct {
	Calendar       []CalendarEvent    `json:"calendar"`
	GitHub         []GitHubIssue      `json:"github"`
	Routines       []RoutineCandidate `json:"routines"`
	CalendarStatus SourceStatus       `json:"calendar_status"`
	GitHubStatus   SourceStatus       `json:"github_status"`
	RoutineStatus  SourceStatus       `json:"routine_status"`
	CalendarError  string             `json:"calendar_error,omitempty"`
	GitHubError    string             `json:"github_error,omitempty"`
	RoutineError   string             `json:"routine_error,omitempty"`
	GatheredAt     time.Time          `json:"gathered_at"`
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

// DailyPlanProposal is strict structured model output and confirmation input.
type DailyPlanProposal struct {
	SummaryMarkdown string                 `json:"summary_markdown"`
	ContentMarkdown string                 `json:"content_markdown"`
	NotesMarkdown   string                 `json:"notes_markdown"`
	GitHubIssues    []PlannedGitHubIssue   `json:"github_issues"`
	RoutineIDs      []string               `json:"routine_ids"`
	CalendarEvents  []PlannedCalendarEvent `json:"calendar_events"`
	ActionItems     []ActionItem           `json:"action_items"`
	Unavailable     []string               `json:"unavailable_sources"`
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

// DailyPlanningView is the complete web view for one local date.
type DailyPlanningView struct {
	Date     time.Time
	Timezone string
	Session  *DailyPlanningSession
	Plan     *Plan
}
