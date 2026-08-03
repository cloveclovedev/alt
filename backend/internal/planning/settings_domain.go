package planning

import "time"

// GitHubRepository is a user-configured public repository used for planning context.
type GitHubRepository struct {
	ID                   string
	Owner                string
	Name                 string
	Enabled              bool
	PlanningInstructions string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// CalendarConnection is a user-authorized Google Calendar account.
type CalendarConnection struct {
	ID            string
	GrantedScopes []string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// CalendarSource is one discovered Google Calendar and its planning interpretation.
type CalendarSource struct {
	ID                   string
	ConnectionID         string
	ExternalCalendarID   string
	DisplayName          string
	Enabled              bool
	Role                 CalendarRole
	PlanningInstructions string
	Timezone             string
}
