package nutrition

import (
	"context"
	"fmt"
	"html/template"
	"math"
	"net/http"
	"strings"
	"time"
)

// RegisterArea wires the nutrition management area and the home-card refresh
// routes. The entry-creation routes are registered by Register.
func (h *Handler) RegisterArea(mux *http.ServeMux) {
	mux.HandleFunc("GET /nutrition", h.area)
	mux.HandleFunc("GET /nutrition/entries/{id}/edit", h.editEntryForm)
	mux.HandleFunc("POST /nutrition/entries/{id}", h.updateEntry)
	mux.HandleFunc("POST /nutrition/entries/{id}/delete", h.deleteEntry)
	mux.HandleFunc("GET /nutrition/catalog", h.catalogPage)
	mux.HandleFunc("POST /nutrition/catalog", h.createCatalog)
	mux.HandleFunc("POST /nutrition/catalog/{id}", h.updateCatalog)
	mux.HandleFunc("POST /nutrition/catalog/{id}/delete", h.deleteCatalog)
	mux.HandleFunc("GET /nutrition/targets", h.targetsPage)
	mux.HandleFunc("POST /nutrition/targets", h.setTarget)
	mux.HandleFunc("POST /nutrition/card/catalog", h.cardQuickAdd)
}

// achievement is an intake total against its target, with a rounded percentage.
type achievement struct {
	Actual    float64
	Target    float64
	Percent   int
	HasTarget bool
}

func newAchievement(actual, target float64) achievement {
	a := achievement{Actual: actual, Target: target, HasTarget: target > 0}
	if a.HasTarget {
		a.Percent = int(math.Round(actual / target * 100))
	}
	return a
}

// summaryView presents a day's deterministic totals against the applicable
// target — no AI is involved in these numbers.
type summaryView struct {
	Date      string
	Calories  achievement
	Protein   achievement
	HasTarget bool
}

func summaryOf(summary DailySummary) summaryView {
	var calorieTarget, proteinTarget float64
	hasTarget := summary.Target != nil
	if hasTarget {
		calorieTarget = float64(summary.Target.CaloriesKcal)
		proteinTarget = summary.Target.ProteinG
	}
	return summaryView{
		Date:      summary.Date.Format(time.DateOnly),
		Calories:  newAchievement(float64(summary.Totals.CaloriesKcal), calorieTarget),
		Protein:   newAchievement(summary.Totals.ProteinG, proteinTarget),
		HasTarget: hasTarget,
	}
}

// mealGroup is the entries for one meal slot, in display order.
type mealGroup struct {
	MealType MealType
	Entries  []Entry
}

func groupByMeal(entries []Entry) []mealGroup {
	groups := make([]mealGroup, 0, len(MealTypes()))
	for _, meal := range MealTypes() {
		group := mealGroup{MealType: meal}
		for _, entry := range entries {
			if entry.MealType == meal {
				group.Entries = append(group.Entries, entry)
			}
		}
		if len(group.Entries) > 0 {
			groups = append(groups, group)
		}
	}
	return groups
}

type areaView struct {
	Date     string
	Summary  summaryView
	Meals    []mealGroup
	Coaching coachingView
	Message  string
}

func (h *Handler) area(w http.ResponseWriter, r *http.Request) {
	date := h.service.LocalToday()
	summary, err := h.service.DailySummary(r.Context(), date)
	if err != nil {
		h.fail(w, err)
		return
	}
	coaching, err := h.service.Coaching(r.Context(), date)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.renderPage(w, "area.html", areaView{
		Date:     date.Format(time.DateOnly),
		Summary:  summaryOf(summary),
		Meals:    groupByMeal(summary.Entries),
		Coaching: coachingView{Date: date.Format(time.DateOnly), Coaching: coaching, AIEnabled: h.service.ai != nil},
	})
}

// --- entry corrections ---

func (h *Handler) editEntryForm(w http.ResponseWriter, r *http.Request) {
	entry, err := h.service.GetEntry(r.Context(), r.PathValue("id"))
	if err != nil {
		h.badRequest(w, err)
		return
	}
	h.renderPage(w, "edit_entry.html", map[string]any{"Entry": entry, "MealTypes": MealTypes()})
}

func (h *Handler) updateEntry(w http.ResponseWriter, r *http.Request) {
	calories, protein, err := parseMetrics(r)
	if err != nil {
		h.badRequest(w, err)
		return
	}
	input := EntryInput{
		LoggedDate:   h.service.LocalToday(),
		MealType:     MealType(strings.TrimSpace(r.FormValue("meal_type"))),
		Name:         r.FormValue("name"),
		CaloriesKcal: calories,
		ProteinG:     protein,
		Source:       SourceManual,
	}
	if err := h.service.UpdateEntry(r.Context(), r.PathValue("id"), input); err != nil {
		h.badRequest(w, err)
		return
	}
	http.Redirect(w, r, "/nutrition", http.StatusSeeOther)
}

func (h *Handler) deleteEntry(w http.ResponseWriter, r *http.Request) {
	if err := h.service.DeleteEntry(r.Context(), r.PathValue("id")); err != nil {
		h.badRequest(w, err)
		return
	}
	http.Redirect(w, r, "/nutrition", http.StatusSeeOther)
}

// --- catalog management ---

func (h *Handler) catalogPage(w http.ResponseWriter, r *http.Request) {
	h.renderCatalog(w, r, "")
}

func (h *Handler) renderCatalog(w http.ResponseWriter, r *http.Request, message string) {
	catalog, err := h.service.ListCatalog(r.Context())
	if err != nil {
		h.fail(w, err)
		return
	}
	h.renderPage(w, "catalog.html", map[string]any{"Catalog": catalog, "Kinds": []CatalogKind{KindFood, KindSupplement}, "Message": message})
}

func (h *Handler) createCatalog(w http.ResponseWriter, r *http.Request) {
	input, err := catalogInputFrom(r)
	if err == nil {
		_, err = h.service.CreateCatalogItem(r.Context(), input)
	}
	if err != nil {
		h.badRequest(w, err)
		return
	}
	http.Redirect(w, r, "/nutrition/catalog", http.StatusSeeOther)
}

func (h *Handler) updateCatalog(w http.ResponseWriter, r *http.Request) {
	input, err := catalogInputFrom(r)
	if err == nil {
		err = h.service.UpdateCatalogItem(r.Context(), r.PathValue("id"), input)
	}
	if err != nil {
		h.badRequest(w, err)
		return
	}
	http.Redirect(w, r, "/nutrition/catalog", http.StatusSeeOther)
}

func (h *Handler) deleteCatalog(w http.ResponseWriter, r *http.Request) {
	if err := h.service.DeleteCatalogItem(r.Context(), r.PathValue("id")); err != nil {
		h.badRequest(w, err)
		return
	}
	http.Redirect(w, r, "/nutrition/catalog", http.StatusSeeOther)
}

func catalogInputFrom(r *http.Request) (CatalogInput, error) {
	calories, protein, err := parseMetrics(r)
	if err != nil {
		return CatalogInput{}, err
	}
	return CatalogInput{
		Name:         r.FormValue("name"),
		CaloriesKcal: calories,
		ProteinG:     protein,
		Kind:         CatalogKind(strings.TrimSpace(r.FormValue("kind"))),
	}, nil
}

// --- targets ---

func (h *Handler) targetsPage(w http.ResponseWriter, r *http.Request) {
	targets, err := h.service.ListTargets(r.Context())
	if err != nil {
		h.fail(w, err)
		return
	}
	h.renderPage(w, "targets.html", map[string]any{"Targets": targets, "Today": h.service.LocalToday().Format(time.DateOnly)})
}

func (h *Handler) setTarget(w http.ResponseWriter, r *http.Request) {
	calories, protein, err := parseMetrics(r)
	if err != nil {
		h.badRequest(w, err)
		return
	}
	effectiveOn, err := time.ParseInLocation(time.DateOnly, strings.TrimSpace(r.FormValue("effective_on")), h.service.location)
	if err != nil {
		h.badRequest(w, fmt.Errorf("%w: effective date must use YYYY-MM-DD", ErrInvalidInput))
		return
	}
	input := TargetInput{
		EffectiveOn:  effectiveOn,
		CaloriesKcal: calories,
		ProteinG:     protein,
		Rationale:    r.FormValue("rationale"),
	}
	if _, err := h.service.SetTarget(r.Context(), input); err != nil {
		h.badRequest(w, err)
		return
	}
	http.Redirect(w, r, "/nutrition/targets", http.StatusSeeOther)
}

// --- home card ---

type cardView struct {
	Date      string
	Summary   summaryView
	Catalog   []CatalogItem
	MealTypes []MealType
	Coaching  coachingView
	AIEnabled bool
	Message   string
}

// HomeCardHTML renders the nutrition home-card fragment. It satisfies the
// planning HomeCardSource port so the home page can compose nutrition without
// importing this package.
func (h *Handler) HomeCardHTML(ctx context.Context) (template.HTML, error) {
	view, err := h.cardData(ctx, "")
	if err != nil {
		return "", err
	}
	var builder strings.Builder
	if err := h.tmpl.ExecuteTemplate(&builder, "card.html", view); err != nil {
		return "", fmt.Errorf("render nutrition home card: %w", err)
	}
	return template.HTML(builder.String()), nil
}

func (h *Handler) cardData(ctx context.Context, message string) (cardView, error) {
	date := h.service.LocalToday()
	summary, err := h.service.DailySummary(ctx, date)
	if err != nil {
		return cardView{}, err
	}
	coaching, err := h.service.Coaching(ctx, date)
	if err != nil {
		return cardView{}, err
	}
	catalog, err := h.service.ListCatalog(ctx)
	if err != nil {
		return cardView{}, err
	}
	return cardView{
		Date:      date.Format(time.DateOnly),
		Summary:   summaryOf(summary),
		Catalog:   catalog,
		MealTypes: MealTypes(),
		Coaching:  coachingView{Date: date.Format(time.DateOnly), Coaching: coaching, AIEnabled: h.service.ai != nil},
		AIEnabled: h.service.ai != nil,
		Message:   message,
	}, nil
}

// cardQuickAdd records a catalog item from the home card and returns the
// refreshed card so the day's totals update in place.
func (h *Handler) cardQuickAdd(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.GetCatalogItem(r.Context(), r.FormValue("catalog_id"))
	if err != nil {
		h.renderCard(w, r, "Could not add that item.")
		return
	}
	id := item.ID
	_, err = h.service.CreateEntry(r.Context(), EntryInput{
		LoggedDate:   h.service.LocalToday(),
		MealType:     MealType(strings.TrimSpace(r.FormValue("meal_type"))),
		Name:         item.Name,
		CaloriesKcal: item.CaloriesKcal,
		ProteinG:     item.ProteinG,
		Source:       SourceCatalog,
		CatalogID:    &id,
	})
	if err != nil {
		h.renderCard(w, r, "Could not add that item.")
		return
	}
	h.renderCard(w, r, fmt.Sprintf("Added %s.", item.Name))
}

func (h *Handler) renderCard(w http.ResponseWriter, r *http.Request, message string) {
	view, err := h.cardData(r.Context(), message)
	if err != nil {
		h.fail(w, err)
		return
	}
	if r.Header.Get("HX-Request") == "true" {
		h.renderFragment(w, "card.html", view)
		return
	}
	http.Redirect(w, r, "/nutrition", http.StatusSeeOther)
}
