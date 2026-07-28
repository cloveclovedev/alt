// Package routine implements user-managed recurring routines.
package routine

import "time"

// RoutineStatus identifies whether a routine accepts future completions.
type RoutineStatus string

const (
	RoutineStatusActive   RoutineStatus = "active"
	RoutineStatusInactive RoutineStatus = "inactive"
)

// DueState is the derived current state of an active routine.
type DueState string

const (
	DueStateOverdue  DueState = "overdue"
	DueStateToday    DueState = "today"
	DueStateUpcoming DueState = "upcoming"
)

// Category groups routines for management and display.
type Category struct {
	ID       string
	Name     string
	Position int
	Active   bool
}

// Routine is a typed recurrence definition.
type Routine struct {
	ID                string
	CategoryID        string
	CategoryName      string
	Name              string
	Notes             string
	Status            RoutineStatus
	IntervalDays      int
	AvailableWeekdays []int16
	ActiveMonths      []int16
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Event records a completion on a local calendar date.
type Event struct {
	ID          string
	RoutineID   string
	CompletedOn time.Time
	Note        string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Status is the provider-neutral recurrence result for an active routine.
type Status struct {
	RoutineID       string
	Name            string
	CategoryID      string
	CategoryName    string
	State           DueState
	DueOn           *time.Time
	NextAvailableOn *time.Time
	Notes           string
}

// RoutineSummary combines a definition and its derived status for list pages.
type RoutineSummary struct {
	Routine
	Status Status
}

// CategorySection is a category and its routines for the list UI.
type CategorySection struct {
	Category Category
	Routines []RoutineSummary
}

// DueSection groups active routines with one derived state.
type DueSection struct {
	State      DueState
	Categories []CategorySection
}

// ListView is the routine management list page model.
type ListView struct {
	DueSections      []DueSection
	InactiveSections []CategorySection
	IncludeInactive  bool
	Today            time.Time
}

// NewRoutineView provides a creation form with a local default date.
type NewRoutineView struct {
	Categories []Category
	Today      time.Time
}

// DetailView is the routine definition, derived status, and history.
type DetailView struct {
	Routine    Routine
	Status     Status
	Events     []Event
	Today      time.Time
	Categories []Category
}

// CategoryInput is a transport-neutral category mutation.
type CategoryInput struct {
	Name     string
	Position int
	Active   bool
}

// RoutineInput is a transport-neutral routine definition mutation.
type RoutineInput struct {
	CategoryID        string
	Name              string
	Notes             string
	Status            RoutineStatus
	IntervalDays      int
	AvailableWeekdays []int16
	ActiveMonths      []int16
	LastCompletedOn   *time.Time
}

// CompletionInput is a transport-neutral completion event mutation.
type CompletionInput struct {
	CompletedOn time.Time
	Note        string
}

// EventInput is a transport-neutral event correction mutation.
type EventInput struct {
	CompletedOn time.Time
	Note        string
}
