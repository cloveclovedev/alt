package routine

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

var (
	// ErrInvalidInput marks a user-correctable validation failure.
	ErrInvalidInput = errors.New("invalid input")
	// ErrNotFound indicates that a user-owned resource does not exist.
	ErrNotFound = errors.New("not found")
	// ErrDeleteNotAllowed marks a routine or category that must be deactivated.
	ErrDeleteNotAllowed = errors.New("delete not allowed")
)

// Service owns recurrence behavior independently of HTTP.
type Service struct {
	store    *Store
	userID   string
	location *time.Location
	now      func() time.Time
}

// NewService constructs the routine application service.
func NewService(store *Store, userID, timezone string) (*Service, error) {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("load timezone: %w", err)
	}
	return &Service{store: store, userID: userID, location: location, now: time.Now}, nil
}

// List returns routines grouped by category, with inactive routines optional.
func (s *Service) List(ctx context.Context, includeInactive bool) (ListView, error) {
	categories, err := s.store.ListCategories(ctx, s.userID, true)
	if err != nil {
		return ListView{}, err
	}
	routines, err := s.store.ListRoutines(ctx, s.userID, includeInactive)
	if err != nil {
		return ListView{}, err
	}
	today := s.localToday()
	byStateAndCategory := make(map[DueState]map[string][]RoutineSummary)
	inactiveByCategory := make(map[string][]RoutineSummary)
	for _, record := range routines {
		summary := RoutineSummary{
			Routine: record.Routine,
			Status:  deriveStatus(record.Routine, record.LastEvent, today, s.location),
		}
		if record.Routine.Status == RoutineStatusInactive {
			inactiveByCategory[record.Routine.CategoryID] = append(inactiveByCategory[record.Routine.CategoryID], summary)
			continue
		}
		if byStateAndCategory[summary.Status.State] == nil {
			byStateAndCategory[summary.Status.State] = make(map[string][]RoutineSummary)
		}
		byStateAndCategory[summary.Status.State][record.Routine.CategoryID] = append(byStateAndCategory[summary.Status.State][record.Routine.CategoryID], summary)
	}
	view := ListView{IncludeInactive: includeInactive, Today: today}
	for _, state := range []DueState{DueStateOverdue, DueStateToday, DueStateUpcoming} {
		section := DueSection{State: state}
		for _, category := range categories {
			if items := byStateAndCategory[state][category.ID]; len(items) > 0 {
				section.Categories = append(section.Categories, CategorySection{Category: category, Routines: items})
			}
		}
		view.DueSections = append(view.DueSections, section)
	}
	if includeInactive {
		for _, category := range categories {
			if items := inactiveByCategory[category.ID]; len(items) > 0 {
				view.InactiveSections = append(view.InactiveSections, CategorySection{Category: category, Routines: items})
			}
		}
	}
	return view, nil
}

// NewForm returns category choices and the user's local date for a new routine.
func (s *Service) NewForm(ctx context.Context) (NewRoutineView, error) {
	categories, err := s.store.ListCategories(ctx, s.userID, false)
	if err != nil {
		return NewRoutineView{}, err
	}
	return NewRoutineView{Categories: categories, Today: s.localToday()}, nil
}

// Statuses returns active routine status views for future planning consumers.
func (s *Service) Statuses(ctx context.Context) ([]Status, error) {
	records, err := s.store.ListRoutines(ctx, s.userID, false)
	if err != nil {
		return nil, err
	}
	today := s.localToday()
	statuses := make([]Status, 0, len(records))
	for _, record := range records {
		statuses = append(statuses, deriveStatus(record.Routine, record.LastEvent, today, s.location))
	}
	return statuses, nil
}

// Detail returns a routine, its recurrence status, event history, and category choices.
func (s *Service) Detail(ctx context.Context, routineID string) (DetailView, error) {
	record, events, err := s.store.LoadRoutine(ctx, s.userID, strings.TrimSpace(routineID))
	if err != nil {
		return DetailView{}, err
	}
	categories, err := s.store.ListCategories(ctx, s.userID, true)
	if err != nil {
		return DetailView{}, err
	}
	return DetailView{
		Routine:    record.Routine,
		Status:     deriveStatus(record.Routine, record.LastEvent, s.localToday(), s.location),
		Events:     events,
		Today:      s.localToday(),
		Categories: categories,
	}, nil
}

// Create validates and creates a routine and optional baseline in one transaction.
func (s *Service) Create(ctx context.Context, input RoutineInput) (string, error) {
	if err := s.validateRoutineInput(&input); err != nil {
		return "", err
	}
	return s.store.CreateRoutine(ctx, s.userID, input)
}

// Update validates and updates a routine definition without changing history.
func (s *Service) Update(ctx context.Context, routineID string, input RoutineInput) error {
	input.BaselineOn = nil
	if err := s.validateRoutineInput(&input); err != nil {
		return err
	}
	return s.store.UpdateRoutine(ctx, s.userID, strings.TrimSpace(routineID), input)
}

// Delete deletes a routine only when it has no event history.
func (s *Service) Delete(ctx context.Context, routineID string) error {
	return s.store.DeleteRoutine(ctx, s.userID, strings.TrimSpace(routineID))
}

// Complete records an actual completion using a local calendar date.
func (s *Service) Complete(ctx context.Context, routineID string, input CompletionInput) error {
	if err := s.validateEventDate(input.CompletedOn); err != nil {
		return err
	}
	input.Note = strings.TrimSpace(input.Note)
	if len([]rune(input.Note)) > 10_000 {
		return fmt.Errorf("%w: event note is too long", ErrInvalidInput)
	}
	return s.store.CreateEvent(ctx, s.userID, strings.TrimSpace(routineID), EventKindCompleted, input)
}

// UpdateEvent corrects a historical completion date or note.
func (s *Service) UpdateEvent(ctx context.Context, routineID, eventID string, input EventInput) error {
	if err := s.validateEventDate(input.CompletedOn); err != nil {
		return err
	}
	input.Note = strings.TrimSpace(input.Note)
	if len([]rune(input.Note)) > 10_000 {
		return fmt.Errorf("%w: event note is too long", ErrInvalidInput)
	}
	return s.store.UpdateEvent(ctx, s.userID, strings.TrimSpace(routineID), strings.TrimSpace(eventID), input)
}

// DeleteEvent removes an erroneous completion or baseline event.
func (s *Service) DeleteEvent(ctx context.Context, routineID, eventID string) error {
	return s.store.DeleteEvent(ctx, s.userID, strings.TrimSpace(routineID), strings.TrimSpace(eventID))
}

// Categories returns categories, including inactive ones, for management.
func (s *Service) Categories(ctx context.Context) ([]Category, error) {
	return s.store.ListCategories(ctx, s.userID, true)
}

// CreateCategory validates and stores a category.
func (s *Service) CreateCategory(ctx context.Context, input CategoryInput) error {
	if err := validateCategoryInput(&input); err != nil {
		return err
	}
	return s.store.CreateCategory(ctx, s.userID, input)
}

// UpdateCategory validates and changes a category's name, status, or order.
func (s *Service) UpdateCategory(ctx context.Context, categoryID string, input CategoryInput) error {
	if err := validateCategoryInput(&input); err != nil {
		return err
	}
	return s.store.UpdateCategory(ctx, s.userID, strings.TrimSpace(categoryID), input)
}

// DeleteCategory deletes a category that is no longer referenced by a routine.
func (s *Service) DeleteCategory(ctx context.Context, categoryID string) error {
	return s.store.DeleteCategory(ctx, s.userID, strings.TrimSpace(categoryID))
}

func (s *Service) validateRoutineInput(input *RoutineInput) error {
	input.CategoryID = strings.TrimSpace(input.CategoryID)
	input.Name = strings.TrimSpace(input.Name)
	input.Notes = strings.TrimSpace(input.Notes)
	if input.CategoryID == "" {
		return fmt.Errorf("%w: category is required", ErrInvalidInput)
	}
	if input.Name == "" {
		return fmt.Errorf("%w: routine name is required", ErrInvalidInput)
	}
	if len([]rune(input.Name)) > 200 {
		return fmt.Errorf("%w: routine name is too long", ErrInvalidInput)
	}
	if len([]rune(input.Notes)) > 20_000 {
		return fmt.Errorf("%w: notes are too long", ErrInvalidInput)
	}
	if input.Status != RoutineStatusActive && input.Status != RoutineStatusInactive {
		return fmt.Errorf("%w: invalid routine status", ErrInvalidInput)
	}
	if input.IntervalDays < 1 {
		return fmt.Errorf("%w: interval must be at least one day", ErrInvalidInput)
	}
	var err error
	if input.AvailableWeekdays, err = normalizeMembers(input.AvailableWeekdays, 1, 7); err != nil {
		return fmt.Errorf("%w: invalid available weekday", ErrInvalidInput)
	}
	if input.ActiveMonths, err = normalizeMembers(input.ActiveMonths, 1, 12); err != nil {
		return fmt.Errorf("%w: invalid active month", ErrInvalidInput)
	}
	if input.BaselineOn != nil {
		if err := s.validateEventDate(*input.BaselineOn); err != nil {
			return fmt.Errorf("%w: baseline date", err)
		}
	}
	return nil
}

func (s *Service) validateEventDate(date time.Time) error {
	if date.IsZero() {
		return fmt.Errorf("%w: completion date is required", ErrInvalidInput)
	}
	if storedCalendarDate(date, s.location).After(s.localToday()) {
		return fmt.Errorf("%w: completion date cannot be in the future", ErrInvalidInput)
	}
	return nil
}

func (s *Service) localToday() time.Time {
	return localCalendarDate(s.now(), s.location)
}

func validateCategoryInput(input *CategoryInput) error {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return fmt.Errorf("%w: category name is required", ErrInvalidInput)
	}
	if len([]rune(input.Name)) > 100 {
		return fmt.Errorf("%w: category name is too long", ErrInvalidInput)
	}
	if input.Position < 0 {
		return fmt.Errorf("%w: category position cannot be negative", ErrInvalidInput)
	}
	return nil
}

func normalizeMembers(values []int16, min, max int16) ([]int16, error) {
	set := make(map[int16]struct{}, len(values))
	for _, value := range values {
		if value < min || value > max {
			return nil, errors.New("out of range")
		}
		set[value] = struct{}{}
	}
	normalized := make([]int16, 0, len(set))
	for value := range set {
		normalized = append(normalized, value)
	}
	sort.Slice(normalized, func(i, j int) bool { return normalized[i] < normalized[j] })
	return normalized, nil
}

func deriveStatus(routine Routine, lastEvent *Event, today time.Time, location *time.Location) Status {
	status := Status{
		RoutineID: routine.ID, Name: routine.Name, CategoryID: routine.CategoryID,
		CategoryName: routine.CategoryName, Notes: routine.Notes,
	}
	if routine.Status != RoutineStatusActive {
		return status
	}
	if lastEvent == nil {
		status.State = DueStateOverdue
		next := firstAvailableOn(today, routine, location)
		status.NextAvailableOn = &next
		return status
	}
	baseDue := storedCalendarDate(lastEvent.CompletedOn, location).AddDate(0, 0, routine.IntervalDays)
	due := firstAvailableOn(baseDue, routine, location)
	status.DueOn = &due
	switch {
	case due.Before(today):
		status.State = DueStateOverdue
	case due.Equal(today):
		status.State = DueStateToday
	default:
		status.State = DueStateUpcoming
	}
	return status
}

func firstAvailableOn(start time.Time, routine Routine, location *time.Location) time.Time {
	date := storedCalendarDate(start, location)
	for range 800 {
		if allowedMonth(date, routine.ActiveMonths) && allowedWeekday(date, routine.AvailableWeekdays) {
			return date
		}
		date = date.AddDate(0, 0, 1)
	}
	panic("routine availability could not be resolved")
}

func allowedWeekday(date time.Time, weekdays []int16) bool {
	if len(weekdays) == 0 {
		return true
	}
	day := int16(date.Weekday())
	if day == 0 {
		day = 7
	}
	for _, weekday := range weekdays {
		if weekday == day {
			return true
		}
	}
	return false
}

func allowedMonth(date time.Time, months []int16) bool {
	if len(months) == 0 {
		return true
	}
	for _, month := range months {
		if month == int16(date.Month()) {
			return true
		}
	}
	return false
}

func localCalendarDate(value time.Time, location *time.Location) time.Time {
	local := value.In(location)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
}

func storedCalendarDate(value time.Time, location *time.Location) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, location)
}
