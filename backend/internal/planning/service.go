package planning

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	// ErrInvalidInput marks a user-correctable validation failure.
	ErrInvalidInput = errors.New("invalid input")
)

// Service owns planning behavior independent of HTTP.
type Service struct {
	store    *Store
	userID   string
	location *time.Location
	now      func() time.Time
}

// NewService constructs the planning application service.
func NewService(store *Store, userID, timezone string) (*Service, error) {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("load timezone: %w", err)
	}
	return &Service{
		store:    store,
		userID:   userID,
		location: location,
		now:      time.Now,
	}, nil
}

// Home returns the daily plan for the current local date.
func (s *Service) Home(ctx context.Context) (View, error) {
	now := s.now().In(s.location)
	periodStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, s.location)
	return s.store.Load(ctx, s.userID, KindDaily, periodStart, s.location)
}

// ViewPlan returns a plan for an explicit kind and period start.
func (s *Service) ViewPlan(ctx context.Context, kind, periodStart string) (View, error) {
	parsedKind, parsedStart, err := s.parsePeriod(kind, periodStart)
	if err != nil {
		return View{}, err
	}
	return s.store.Load(ctx, s.userID, parsedKind, parsedStart, s.location)
}

// SavePlan validates and stores a new immutable plan revision.
func (s *Service) SavePlan(ctx context.Context, input SavePlanInput) error {
	kind, periodStart, err := s.parsePeriod(input.Kind, input.PeriodStart)
	if err != nil {
		return err
	}

	input.SummaryMarkdown = strings.TrimSpace(input.SummaryMarkdown)
	input.ContentMarkdown = strings.TrimSpace(input.ContentMarkdown)
	if input.SummaryMarkdown == "" {
		return fmt.Errorf("%w: plan summary is required", ErrInvalidInput)
	}
	if input.ContentMarkdown == "" {
		return fmt.Errorf("%w: plan content is required", ErrInvalidInput)
	}
	if len([]rune(input.SummaryMarkdown)) > 5_000 {
		return fmt.Errorf("%w: plan summary is too long", ErrInvalidInput)
	}
	if len([]rune(input.ContentMarkdown)) > 50_000 {
		return fmt.Errorf("%w: plan content is too long", ErrInvalidInput)
	}
	return s.store.Save(ctx, s.userID, kind, periodStart, input)
}

func (s *Service) parsePeriod(rawKind, rawPeriodStart string) (Kind, time.Time, error) {
	var kind Kind
	switch strings.TrimSpace(rawKind) {
	case string(KindDaily):
		kind = KindDaily
	case string(KindWeekly):
		kind = KindWeekly
	default:
		return "", time.Time{}, fmt.Errorf("%w: unknown plan kind", ErrInvalidInput)
	}

	periodStart, err := time.ParseInLocation(
		time.DateOnly,
		strings.TrimSpace(rawPeriodStart),
		s.location,
	)
	if err != nil {
		return "", time.Time{}, fmt.Errorf(
			"%w: period start must use YYYY-MM-DD",
			ErrInvalidInput,
		)
	}
	if kind == KindWeekly && periodStart.Weekday() != time.Monday {
		return "", time.Time{}, fmt.Errorf(
			"%w: weekly period start must be a Monday",
			ErrInvalidInput,
		)
	}
	return kind, periodStart, nil
}
