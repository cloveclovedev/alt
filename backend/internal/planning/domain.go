package planning

import "time"

// Kind identifies the period represented by a plan.
type Kind string

const (
	// KindDaily identifies a plan for one local calendar date.
	KindDaily Kind = "daily"
	// KindWeekly identifies a plan for a Monday-based calendar week.
	KindWeekly Kind = "weekly"
)

// View is the complete page model for one planning period.
type View struct {
	Kind        Kind
	PeriodStart time.Time
	Timezone    string
	Plan        *Plan
	// Home marks the today entry card, which offers a start-or-resume action
	// and a compact summary. Read-only past-plan views leave it false.
	Home bool
}

// Plan is the latest immutable revision for a planning period.
type Plan struct {
	ID              string
	RevisionID      string
	Kind            Kind
	PeriodStart     time.Time
	Revision        int
	SummaryMarkdown string
	ContentMarkdown string
	NotesMarkdown   string
	CreatedAt       time.Time
	Components      PlanComponents
}
