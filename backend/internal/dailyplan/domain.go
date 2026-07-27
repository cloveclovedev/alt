package dailyplan

import "time"

// Today is the complete view required by the daily planning page.
type Today struct {
	Date           time.Time
	Timezone       string
	JournalEntries []JournalEntry
	Tasks          []Task
	Plan           *Plan
}

// JournalEntry is a raw user-authored record.
type JournalEntry struct {
	ID         string
	Kind       string
	Body       string
	OccurredAt time.Time
}

// Task is an actionable item with typed lifecycle fields.
type Task struct {
	ID       string
	Title    string
	Notes    string
	Status   string
	Priority string
	DueDate  *time.Time
}

// Plan is an accepted daily-plan revision.
type Plan struct {
	ID        string
	Date      time.Time
	Revision  int
	Summary   string
	CreatedAt time.Time
}

// AddTaskInput is the transport-neutral task creation request.
type AddTaskInput struct {
	Title    string
	Notes    string
	Priority string
	DueDate  string
}
