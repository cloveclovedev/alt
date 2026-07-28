package routine

import (
	"errors"
	"testing"
	"time"
)

func TestDeriveStatusUsesUnrestrictedInterval(t *testing.T) {
	location := time.UTC
	today := date(t, location, "2026-07-28")
	result := deriveStatus(Routine{ID: "routine", Status: RoutineStatusActive, IntervalDays: 8}, &Event{CompletedOn: date(t, location, "2026-07-20")}, today, location)

	if result.State != DueStateToday || result.DueOn == nil || result.DueOn.Format(time.DateOnly) != "2026-07-28" {
		t.Fatalf("status = %#v", result)
	}
}

func TestDeriveStatusShiftsToAllowedWeekday(t *testing.T) {
	location := time.UTC
	result := deriveStatus(Routine{Status: RoutineStatusActive, IntervalDays: 8, AvailableWeekdays: []int16{6, 7}}, &Event{CompletedOn: date(t, location, "2026-07-20")}, date(t, location, "2026-07-28"), location)

	if result.DueOn == nil || result.DueOn.Format(time.DateOnly) != "2026-08-01" || result.State != DueStateUpcoming {
		t.Fatalf("status = %#v", result)
	}
}

func TestDeriveStatusShiftsAcrossInactiveMonth(t *testing.T) {
	location := time.UTC
	result := deriveStatus(Routine{Status: RoutineStatusActive, IntervalDays: 1, ActiveMonths: []int16{3}}, &Event{CompletedOn: date(t, location, "2026-01-31")}, date(t, location, "2026-02-01"), location)

	if result.DueOn == nil || result.DueOn.Format(time.DateOnly) != "2026-03-01" {
		t.Fatalf("due date = %#v", result.DueOn)
	}
}

func TestDeriveStatusCombinesRestrictionsAcrossYearBoundary(t *testing.T) {
	location := time.UTC
	result := deriveStatus(Routine{Status: RoutineStatusActive, IntervalDays: 1, AvailableWeekdays: []int16{1}, ActiveMonths: []int16{1}}, &Event{CompletedOn: date(t, location, "2026-12-31")}, date(t, location, "2027-01-01"), location)

	if result.DueOn == nil || result.DueOn.Format(time.DateOnly) != "2027-01-04" {
		t.Fatalf("due date = %#v", result.DueOn)
	}
}

func TestDeriveStatusSupportsLeapYearMonthEnd(t *testing.T) {
	location := time.UTC
	result := deriveStatus(Routine{Status: RoutineStatusActive, IntervalDays: 1}, &Event{CompletedOn: date(t, location, "2028-02-28")}, date(t, location, "2028-02-29"), location)

	if result.DueOn == nil || result.DueOn.Format(time.DateOnly) != "2028-02-29" || result.State != DueStateToday {
		t.Fatalf("status = %#v", result)
	}
}

func TestDeriveStatusUsesUserTimezoneForState(t *testing.T) {
	location, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}
	result := deriveStatus(Routine{Status: RoutineStatusActive, IntervalDays: 1}, &Event{CompletedOn: time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC)}, time.Date(2026, 7, 29, 0, 0, 0, 0, location), location)

	if result.DueOn == nil || result.DueOn.Format(time.DateOnly) != "2026-07-29" || result.State != DueStateToday {
		t.Fatalf("status = %#v", result)
	}
}

func TestDeriveStatusWithoutHistoryIsOverdueAndShowsNextAvailability(t *testing.T) {
	location := time.UTC
	result := deriveStatus(Routine{Status: RoutineStatusActive, AvailableWeekdays: []int16{1}}, nil, date(t, location, "2026-07-28"), location)

	if result.State != DueStateOverdue || result.DueOn != nil || result.NextAvailableOn == nil || result.NextAvailableOn.Format(time.DateOnly) != "2026-08-03" {
		t.Fatalf("status = %#v", result)
	}
}

func TestDeriveStatusSkipsInactiveRoutine(t *testing.T) {
	result := deriveStatus(Routine{Status: RoutineStatusInactive, IntervalDays: 1}, nil, date(t, time.UTC, "2026-07-28"), time.UTC)
	if result.State != "" || result.DueOn != nil || result.NextAvailableOn != nil {
		t.Fatalf("status = %#v", result)
	}
}

func TestValidateRoutineInputNormalizesSelectionsAndRejectsFutureLastCompletion(t *testing.T) {
	service, err := NewService(nil, "user", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return date(t, time.UTC, "2026-07-28") }
	future := date(t, time.UTC, "2026-07-29")
	input := RoutineInput{CategoryID: "category", Name: " Routine ", Status: RoutineStatusActive, IntervalDays: 1, AvailableWeekdays: []int16{7, 1, 7}, ActiveMonths: []int16{12, 1, 12}, LastCompletedOn: &future}

	err = service.validateRoutineInput(&input)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("error = %v, want validation error", err)
	}
	future = date(t, time.UTC, "2026-07-28")
	input.LastCompletedOn = &future
	if err := service.validateRoutineInput(&input); err != nil {
		t.Fatal(err)
	}
	if got, want := input.AvailableWeekdays, []int16{1, 7}; !sameInt16(got, want) {
		t.Fatalf("weekdays = %v, want %v", got, want)
	}
	if got, want := input.ActiveMonths, []int16{1, 12}; !sameInt16(got, want) {
		t.Fatalf("months = %v, want %v", got, want)
	}
}

func date(t *testing.T, location *time.Location, value string) time.Time {
	t.Helper()
	parsed, err := time.ParseInLocation(time.DateOnly, value, location)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func sameInt16(left, right []int16) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
