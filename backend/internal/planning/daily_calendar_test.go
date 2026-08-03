package planning

import (
	"testing"
	"time"
)

func TestTodaysCalendarEventsFiltersAndSorts(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatal(err)
	}
	day := time.Date(2026, 8, 9, 0, 0, 0, 0, loc)
	at := func(hour int) *time.Time {
		v := time.Date(2026, 8, 9, hour, 0, 0, 0, loc)
		return &v
	}
	tomorrow := time.Date(2026, 8, 10, 9, 0, 0, 0, loc)
	startDay := day

	events := []CalendarEvent{
		{Title: "evening", StartsAt: at(22), EndsAt: at(23)},
		{Title: "morning", StartsAt: at(9), EndsAt: at(10)},
		{Title: "tomorrow", StartsAt: &tomorrow},
		{Title: "allday", AllDay: true, StartDate: &startDay},
	}

	got := todaysCalendarEvents(events, day, loc)
	if len(got) != 3 {
		t.Fatalf("want 3 events on the plan date, got %d", len(got))
	}
	if got[0].Title != "allday" || got[1].Title != "morning" || got[2].Title != "evening" {
		t.Fatalf("unexpected order: %s, %s, %s", got[0].Title, got[1].Title, got[2].Title)
	}
}

func TestPreviewComponentsShowsAllTodaysCalendarEvents(t *testing.T) {
	loc := time.UTC
	day := time.Date(2026, 8, 9, 0, 0, 0, 0, loc)
	a := time.Date(2026, 8, 9, 10, 0, 0, 0, loc)
	b := time.Date(2026, 8, 9, 15, 0, 0, 0, loc)
	value := DailyPlanningContext{Calendar: []CalendarEvent{
		{Title: "Standup", StartsAt: &a},
		{Title: "Review", StartsAt: &b},
	}}
	// The proposal selects no calendar events; the day's events are shown anyway.
	components := previewComponents(DailyPlanProposal{}, value, day, loc)
	if len(components.CalendarEvents) != 2 {
		t.Fatalf("want 2 calendar events from context, got %d", len(components.CalendarEvents))
	}
}
