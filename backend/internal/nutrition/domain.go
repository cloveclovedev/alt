// Package nutrition tracks daily calorie and protein intake: a reusable catalog,
// a mutable intake log, and effective-dated targets, with daily and trailing
// summaries against the applicable target. See docs/designs/nutrition-tracking.md.
package nutrition

import "time"

// MealType is the time slot a consumption event belongs to. It is a property of
// the event, not of a catalog item.
type MealType string

const (
	MealBreakfast MealType = "breakfast"
	MealLunch     MealType = "lunch"
	MealDinner    MealType = "dinner"
	MealSnack     MealType = "snack"
)

// MealTypes returns the meal types in display order.
func MealTypes() []MealType {
	return []MealType{MealBreakfast, MealLunch, MealDinner, MealSnack}
}

// ValidMealType reports whether value is a known meal type.
func ValidMealType(value MealType) bool {
	switch value {
	case MealBreakfast, MealLunch, MealDinner, MealSnack:
		return true
	}
	return false
}

// CatalogKind classifies a catalog item. Supplements are identifiable and
// filterable without a dedicated adherence mechanism.
type CatalogKind string

const (
	KindFood       CatalogKind = "food"
	KindSupplement CatalogKind = "supplement"
)

// ValidCatalogKind reports whether value is a known catalog kind.
func ValidCatalogKind(value CatalogKind) bool {
	return value == KindFood || value == KindSupplement
}

// EntrySource records how an entry was created.
type EntrySource string

const (
	SourceManual  EntrySource = "manual"
	SourceAIPhoto EntrySource = "ai_photo"
	SourceAIText  EntrySource = "ai_text"
	SourceCatalog EntrySource = "catalog"
)

// ValidEntrySource reports whether value is a known entry source.
func ValidEntrySource(value EntrySource) bool {
	switch value {
	case SourceManual, SourceAIPhoto, SourceAIText, SourceCatalog:
		return true
	}
	return false
}

// CatalogItem is a pre-registered reusable item. Calories and protein are per the
// registered serving.
type CatalogItem struct {
	ID           string
	Name         string
	CaloriesKcal int
	ProteinG     float64
	Kind         CatalogKind
	Source       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Entry is one consumed item in the mutable intake log. Name, calories, and
// protein are denormalized snapshots taken at logging time.
type Entry struct {
	ID           string
	LoggedDate   time.Time
	MealType     MealType
	Name         string
	CaloriesKcal int
	ProteinG     float64
	Source       EntrySource
	CatalogID    *string
	CreatedAt    time.Time
}

// Target is one effective-dated daily target.
type Target struct {
	ID           string
	EffectiveOn  time.Time
	CaloriesKcal int
	ProteinG     float64
	Rationale    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Totals is a day's summed intake.
type Totals struct {
	CaloriesKcal int
	ProteinG     float64
}

// DaySummary is one day's totals and the target that applied on that day.
type DaySummary struct {
	Date   time.Time
	Totals Totals
	// Target is the target in effect on Date, or nil when none applies.
	Target *Target
}

// DailySummary is a single day's full view: its entries, totals, and the
// applicable target.
type DailySummary struct {
	Date    time.Time
	Entries []Entry
	Totals  Totals
	Target  *Target
}

// TrailingSummary is a window of per-day summaries ending on the requested date,
// most recent last.
type TrailingSummary struct {
	Days    []DaySummary
	Average Totals
}

// CatalogInput is a transport-neutral catalog mutation.
type CatalogInput struct {
	Name         string
	CaloriesKcal int
	ProteinG     float64
	Kind         CatalogKind
	Source       string
}

// EntryInput is a transport-neutral intake-log mutation. CatalogID is set only
// when the entry was created from a catalog item.
type EntryInput struct {
	LoggedDate   time.Time
	MealType     MealType
	Name         string
	CaloriesKcal int
	ProteinG     float64
	Source       EntrySource
	CatalogID    *string
}

// TargetInput is a transport-neutral target mutation. Setting a target appends
// (or replaces) the row for its effective date.
type TargetInput struct {
	EffectiveOn  time.Time
	CaloriesKcal int
	ProteinG     float64
	Rationale    string
}

// Candidate is a proposed entry produced by AI parsing (photo or text). It is
// never stored on its own: the user reviews and edits candidates, and only on
// confirmation are they written as entries. This keeps mutation entirely with the
// user; the model only proposes.
type Candidate struct {
	Name         string
	CaloriesKcal int
	ProteinG     float64
	MealType     MealType
	Source       EntrySource
	// CatalogID is set when the candidate name matched a catalog item, whose
	// stored values are then preferred over the AI estimate.
	CatalogID *string
}
