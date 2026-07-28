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
	CreatedAt       time.Time
}

// SavePlanInput is the transport-neutral plan revision request.
type SavePlanInput struct {
	Kind            string
	PeriodStart     string
	SummaryMarkdown string
	ContentMarkdown string
}
