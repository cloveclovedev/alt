package planning

import (
	"errors"
	"testing"
	"time"
)

func TestParsePeriodAcceptsDailyDateAndWeeklyMonday(t *testing.T) {
	service, err := NewService(nil, "user-1", "Asia/Tokyo")
	if err != nil {
		t.Fatal(err)
	}

	dailyKind, dailyStart, err := service.parsePeriod("daily", "2026-07-28")
	if err != nil {
		t.Fatal(err)
	}
	if dailyKind != KindDaily || dailyStart.Format(time.DateOnly) != "2026-07-28" {
		t.Fatalf("daily period = %q %s", dailyKind, dailyStart)
	}

	weeklyKind, weeklyStart, err := service.parsePeriod("weekly", "2026-07-27")
	if err != nil {
		t.Fatal(err)
	}
	if weeklyKind != KindWeekly || weeklyStart.Weekday() != time.Monday {
		t.Fatalf("weekly period = %q %s", weeklyKind, weeklyStart)
	}
}

func TestParsePeriodRejectsWeeklyNonMonday(t *testing.T) {
	service, err := NewService(nil, "user-1", "Asia/Tokyo")
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = service.parsePeriod("weekly", "2026-07-28")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("error = %v, want ErrInvalidInput", err)
	}
}
