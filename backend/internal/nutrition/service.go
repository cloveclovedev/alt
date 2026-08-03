package nutrition

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

var (
	// ErrInvalidInput marks a user-correctable validation failure.
	ErrInvalidInput = errors.New("invalid input")
	// ErrNotFound indicates that a user-owned resource does not exist.
	ErrNotFound = errors.New("not found")
	// ErrConflict indicates a uniqueness conflict (for example a duplicate name).
	ErrConflict = errors.New("conflict")
)

const (
	maxNameRunes      = 200
	maxRationaleRunes = 2000
	maxCalories       = 1_000_000
	maxProtein        = 99_999.9 // fits numeric(6,1)
)

// Service owns nutrition business rules: validation, ownership, target
// derivation, and daily and trailing summaries.
type Service struct {
	store    *Store
	userID   string
	location *time.Location
	now      func() time.Time
}

// NewService constructs the nutrition service for one local user and timezone.
func NewService(store *Store, userID, timezone string) (*Service, error) {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("load timezone: %w", err)
	}
	return &Service{store: store, userID: userID, location: location, now: time.Now}, nil
}

// LocalToday returns the current local calendar date at midnight.
func (s *Service) LocalToday() time.Time {
	now := s.now().In(s.location)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, s.location)
}

// --- Catalog ---

// ListCatalog returns the user's catalog items.
func (s *Service) ListCatalog(ctx context.Context) ([]CatalogItem, error) {
	return s.store.ListCatalog(ctx, s.userID)
}

// GetCatalogItem loads one owned catalog item.
func (s *Service) GetCatalogItem(ctx context.Context, id string) (CatalogItem, error) {
	return s.store.GetCatalogItem(ctx, s.userID, strings.TrimSpace(id))
}

// CreateCatalogItem validates and stores a new catalog item.
func (s *Service) CreateCatalogItem(ctx context.Context, input CatalogInput) (CatalogItem, error) {
	normalized, err := s.validateCatalogInput(input)
	if err != nil {
		return CatalogItem{}, err
	}
	return s.store.CreateCatalogItem(ctx, s.userID, normalized)
}

// UpdateCatalogItem validates and edits an owned catalog item.
func (s *Service) UpdateCatalogItem(ctx context.Context, id string, input CatalogInput) error {
	normalized, err := s.validateCatalogInput(input)
	if err != nil {
		return err
	}
	return s.store.UpdateCatalogItem(ctx, s.userID, strings.TrimSpace(id), normalized)
}

// DeleteCatalogItem removes an owned catalog item; entries keep their snapshots.
func (s *Service) DeleteCatalogItem(ctx context.Context, id string) error {
	return s.store.DeleteCatalogItem(ctx, s.userID, strings.TrimSpace(id))
}

// --- Entries ---

// ListEntriesForDate returns the intake log for one local date.
func (s *Service) ListEntriesForDate(ctx context.Context, date time.Time) ([]Entry, error) {
	return s.store.ListEntriesForDate(ctx, s.userID, date)
}

// GetEntry loads one owned entry.
func (s *Service) GetEntry(ctx context.Context, id string) (Entry, error) {
	return s.store.GetEntry(ctx, s.userID, strings.TrimSpace(id))
}

// CreateEntry validates and records one intake entry. A catalog reference, when
// present, must belong to the caller.
func (s *Service) CreateEntry(ctx context.Context, input EntryInput) (Entry, error) {
	normalized, err := s.validateEntryInput(ctx, input)
	if err != nil {
		return Entry{}, err
	}
	return s.store.CreateEntry(ctx, s.userID, normalized)
}

// UpdateEntry validates and corrects an owned entry.
func (s *Service) UpdateEntry(ctx context.Context, id string, input EntryInput) error {
	normalized, err := s.validateEntryInput(ctx, input)
	if err != nil {
		return err
	}
	return s.store.UpdateEntry(ctx, s.userID, strings.TrimSpace(id), normalized)
}

// DeleteEntry removes an owned entry.
func (s *Service) DeleteEntry(ctx context.Context, id string) error {
	return s.store.DeleteEntry(ctx, s.userID, strings.TrimSpace(id))
}

// --- Targets ---

// SetTarget validates and appends (or replaces) the target for its effective date.
func (s *Service) SetTarget(ctx context.Context, input TargetInput) (Target, error) {
	normalized, err := s.validateTargetInput(input)
	if err != nil {
		return Target{}, err
	}
	return s.store.UpsertTarget(ctx, s.userID, normalized)
}

// ListTargets returns the user's target history, newest first.
func (s *Service) ListTargets(ctx context.Context) ([]Target, error) {
	return s.store.ListTargets(ctx, s.userID)
}

// ApplicableTarget returns the target in effect on the given date, or nil.
func (s *Service) ApplicableTarget(ctx context.Context, date time.Time) (*Target, error) {
	return s.store.ApplicableTarget(ctx, s.userID, date)
}

// --- Summaries ---

// DailySummary returns one date's entries, totals, and the applicable target.
func (s *Service) DailySummary(ctx context.Context, date time.Time) (DailySummary, error) {
	entries, err := s.store.ListEntriesForDate(ctx, s.userID, date)
	if err != nil {
		return DailySummary{}, err
	}
	target, err := s.store.ApplicableTarget(ctx, s.userID, date)
	if err != nil {
		return DailySummary{}, err
	}
	return DailySummary{Date: date, Entries: entries, Totals: sumEntries(entries), Target: target}, nil
}

// TrailingSummary returns per-day totals for the `days`-day window ending on
// `end` (most recent last), each with the target that applied on that day, plus
// the window's mean intake (dividing by the full window length, so untracked days
// count as zero).
func (s *Service) TrailingSummary(ctx context.Context, end time.Time, days int) (TrailingSummary, error) {
	if days < 1 {
		return TrailingSummary{}, fmt.Errorf("%w: trailing window must be at least one day", ErrInvalidInput)
	}
	start := end.AddDate(0, 0, -(days - 1))
	totalsByDate, err := s.store.RangeTotals(ctx, s.userID, start, end)
	if err != nil {
		return TrailingSummary{}, err
	}
	targets, err := s.store.ListTargets(ctx, s.userID)
	if err != nil {
		return TrailingSummary{}, err
	}
	summary := TrailingSummary{Days: make([]DaySummary, 0, days)}
	var sumCalories int
	var sumProtein float64
	for offset := days - 1; offset >= 0; offset-- {
		day := end.AddDate(0, 0, -offset)
		totals := totalsByDate[dateKey(day)]
		sumCalories += totals.CaloriesKcal
		sumProtein += totals.ProteinG
		summary.Days = append(summary.Days, DaySummary{Date: day, Totals: totals, Target: applicableTargetFor(targets, day)})
	}
	summary.Average = Totals{CaloriesKcal: sumCalories / days, ProteinG: sumProtein / float64(days)}
	return summary, nil
}

// --- validation ---

func (s *Service) validateCatalogInput(input CatalogInput) (CatalogInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Source = strings.TrimSpace(input.Source)
	if input.Kind == "" {
		input.Kind = KindFood
	}
	if input.Source == "" {
		input.Source = "manual"
	}
	if !ValidCatalogKind(input.Kind) {
		return CatalogInput{}, fmt.Errorf("%w: invalid catalog kind", ErrInvalidInput)
	}
	if err := validateName(input.Name); err != nil {
		return CatalogInput{}, err
	}
	if err := validateMetrics(input.CaloriesKcal, input.ProteinG); err != nil {
		return CatalogInput{}, err
	}
	return input, nil
}

func (s *Service) validateEntryInput(ctx context.Context, input EntryInput) (EntryInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.LoggedDate.IsZero() {
		return EntryInput{}, fmt.Errorf("%w: logged date is required", ErrInvalidInput)
	}
	if !ValidMealType(input.MealType) {
		return EntryInput{}, fmt.Errorf("%w: invalid meal type", ErrInvalidInput)
	}
	if !ValidEntrySource(input.Source) {
		return EntryInput{}, fmt.Errorf("%w: invalid entry source", ErrInvalidInput)
	}
	if err := validateName(input.Name); err != nil {
		return EntryInput{}, err
	}
	if err := validateMetrics(input.CaloriesKcal, input.ProteinG); err != nil {
		return EntryInput{}, err
	}
	if input.CatalogID != nil {
		id := strings.TrimSpace(*input.CatalogID)
		if id == "" {
			input.CatalogID = nil
		} else {
			if _, err := s.store.GetCatalogItem(ctx, s.userID, id); err != nil {
				if errors.Is(err, ErrNotFound) {
					return EntryInput{}, fmt.Errorf("%w: catalog item does not exist", ErrInvalidInput)
				}
				return EntryInput{}, err
			}
			input.CatalogID = &id
		}
	}
	return input, nil
}

func (s *Service) validateTargetInput(input TargetInput) (TargetInput, error) {
	input.Rationale = strings.TrimSpace(input.Rationale)
	if input.EffectiveOn.IsZero() {
		return TargetInput{}, fmt.Errorf("%w: effective date is required", ErrInvalidInput)
	}
	if len([]rune(input.Rationale)) > maxRationaleRunes {
		return TargetInput{}, fmt.Errorf("%w: rationale is too long", ErrInvalidInput)
	}
	if err := validateMetrics(input.CaloriesKcal, input.ProteinG); err != nil {
		return TargetInput{}, err
	}
	return input, nil
}

func validateName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	if len([]rune(name)) > maxNameRunes {
		return fmt.Errorf("%w: name is too long", ErrInvalidInput)
	}
	return nil
}

func validateMetrics(calories int, protein float64) error {
	// Reject NaN/Inf first: NaN fails every range comparison silently, and
	// PostgreSQL numeric would accept it and poison SUM and trailing summaries.
	if math.IsNaN(protein) || math.IsInf(protein, 0) {
		return fmt.Errorf("%w: protein must be a finite number", ErrInvalidInput)
	}
	if calories < 0 || protein < 0 {
		return fmt.Errorf("%w: calories and protein must not be negative", ErrInvalidInput)
	}
	if calories > maxCalories {
		return fmt.Errorf("%w: calories is too large", ErrInvalidInput)
	}
	if protein > maxProtein {
		return fmt.Errorf("%w: protein is too large", ErrInvalidInput)
	}
	return nil
}

// --- helpers ---

func sumEntries(entries []Entry) Totals {
	var totals Totals
	for _, entry := range entries {
		totals.CaloriesKcal += entry.CaloriesKcal
		totals.ProteinG += entry.ProteinG
	}
	return totals
}

// applicableTargetFor returns the target in effect on day from a list ordered by
// effective date descending: the first whose effective date is on or before day.
func applicableTargetFor(targets []Target, day time.Time) *Target {
	key := dateKey(day)
	for i := range targets {
		if dateKey(targets[i].EffectiveOn) <= key {
			return &targets[i]
		}
	}
	return nil
}

// dateKey renders a time's calendar date in its own location as a sortable
// YYYY-MM-DD string, so dates stored at UTC midnight and dates built at local
// midnight compare by calendar day without a timezone offset error.
func dateKey(t time.Time) string {
	return t.Format(time.DateOnly)
}
