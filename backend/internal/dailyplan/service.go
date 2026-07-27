package dailyplan

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
	// ErrNotFound marks a missing or inaccessible feature record.
	ErrNotFound = errors.New("not found")
)

// Service owns daily planning behavior independent of HTTP.
type Service struct {
	store    *Store
	userID   string
	location *time.Location
	now      func() time.Time
}

// NewService constructs the daily planning application service.
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

// Today returns today's planning view in the configured user timezone.
func (s *Service) Today(ctx context.Context) (Today, error) {
	return s.store.LoadToday(ctx, s.userID, s.now().In(s.location), s.location)
}

// AddJournalEntry validates and appends a note.
func (s *Service) AddJournalEntry(ctx context.Context, body string) error {
	body = strings.TrimSpace(body)
	if body == "" {
		return fmt.Errorf("%w: journal body is required", ErrInvalidInput)
	}
	if len([]rune(body)) > 10_000 {
		return fmt.Errorf("%w: journal body is too long", ErrInvalidInput)
	}
	return s.store.AddJournalEntry(ctx, s.userID, body, s.now())
}

// AddTask validates and creates an active task.
func (s *Service) AddTask(ctx context.Context, input AddTaskInput) error {
	input.Title = strings.TrimSpace(input.Title)
	input.Notes = strings.TrimSpace(input.Notes)
	input.Priority = strings.TrimSpace(input.Priority)
	input.DueDate = strings.TrimSpace(input.DueDate)

	if input.Title == "" {
		return fmt.Errorf("%w: task title is required", ErrInvalidInput)
	}
	if len([]rune(input.Title)) > 500 {
		return fmt.Errorf("%w: task title is too long", ErrInvalidInput)
	}
	switch input.Priority {
	case "", "P0", "P1", "P2", "P3":
	default:
		return fmt.Errorf("%w: unknown priority", ErrInvalidInput)
	}

	var dueDate *time.Time
	if input.DueDate != "" {
		parsed, err := time.ParseInLocation(time.DateOnly, input.DueDate, s.location)
		if err != nil {
			return fmt.Errorf("%w: due date must use YYYY-MM-DD", ErrInvalidInput)
		}
		dueDate = &parsed
	}
	return s.store.AddTask(
		ctx,
		s.userID,
		input.Title,
		input.Notes,
		input.Priority,
		dueDate,
	)
}

// CompleteTask marks one owned active or backlog task as done.
func (s *Service) CompleteTask(ctx context.Context, taskID string) error {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return fmt.Errorf("%w: task id is required", ErrInvalidInput)
	}
	return s.store.CompleteTask(ctx, s.userID, taskID)
}

// AcceptPlan validates and stores a new accepted plan revision for today.
func (s *Service) AcceptPlan(ctx context.Context, summary string) error {
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return fmt.Errorf("%w: plan summary is required", ErrInvalidInput)
	}
	if len([]rune(summary)) > 20_000 {
		return fmt.Errorf("%w: plan summary is too long", ErrInvalidInput)
	}
	return s.store.AcceptPlan(ctx, s.userID, s.now().In(s.location), summary)
}
